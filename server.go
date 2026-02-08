package main

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed static/index.html
var indexHTML embed.FS

type Server struct {
	hetzner     *HetznerClient
	provisioner *Provisioner
	store       *Store
	hub         *LogHub
	username    string
	password    string
	upgrader    websocket.Upgrader
}

func NewServer(hetzner *HetznerClient, provisioner *Provisioner, store *Store, hub *LogHub, username, password string) *Server {
	return &Server{
		hetzner:     hetzner,
		provisioner: provisioner,
		store:       store,
		hub:         hub,
		username:    username,
		password:    password,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /servers", s.handleCreateServer)
	mux.HandleFunc("GET /servers", s.handleListServers)
	mux.HandleFunc("GET /servers/{id}/ws", s.handleWebSocket)
	mux.HandleFunc("POST /servers/{id}/reprovision", s.handleReprovision)
	mux.HandleFunc("GET /servers/{id}", s.handleGetServer)
	mux.HandleFunc("DELETE /servers/{id}", s.handleDeleteServer)
	mux.HandleFunc("GET /", s.handleIndex)
	return s.basicAuth(mux)
}

func (s *Server) basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(user), []byte(s.username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), []byte(s.password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="openclaw"`)
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	var req CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Name == "" {
		req.Name = randomName()
	}

	info, err := s.hetzner.CreateServer(r.Context(), req.Name)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	opts := ProvisionOpts{
		IP:              info.IPv4,
		SSHPublicKey:    req.SSHPublicKey,
		AnthropicAPIKey: req.AnthropicAPIKey,
		OpenAIAPIKey:    req.OpenAIAPIKey,
		GeminiAPIKey:    req.GeminiAPIKey,
		WayfinderAPIKey: req.WayfinderAPIKey,
		Channels:        req.Channels,
	}

	if err := s.store.CreateServer(info, opts); err != nil {
		slog.Error("failed to store server", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to persist server"})
		return
	}

	logFn := s.makeLogFn(info.ID, opts.SSHPublicKey != "")

	logFn("Creating server...")
	logFn(fmt.Sprintf("Server created: %s (%s)", info.Name, info.IPv4))
	logFn("Waiting for SSH to become available...")

	go s.runProvision(info.ID, opts, logFn)

	writeJSON(w, http.StatusAccepted, CreateServerResponse{
		ID:     info.ID,
		Name:   info.Name,
		Status: info.Status,
		IPv4:   info.IPv4,
	})
}

func (s *Server) makeLogFn(id int64, hasSSHKey bool) func(string) {
	return func(line string) {
		s.store.AppendLog(id, line)
		s.hub.Notify(id)
		if strings.Contains(line, "Hetzner provisioning key removed") {
			s.store.SetDefaultKeyRemoved(id, true)
			s.hub.Notify(id)
		}
	}
}

func (s *Server) runProvision(id int64, opts ProvisionOpts, logFn func(string)) {
	if err := s.provisioner.WaitForSSH(opts.IP, logFn); err != nil {
		slog.Error("SSH wait failed", "server_id", id, "error", err)
		logFn("SSH wait failed: " + err.Error())
		s.store.UpdateStatus(id, "failed", false)
		s.hub.Notify(id)
		return
	}

	result, err := s.provisioner.RunPlaybook(opts, logFn)
	if err != nil {
		slog.Error("provisioning failed", "server_id", id, "error", err)
		s.store.UpdateStatus(id, "failed", false)
		s.hub.Notify(id)
		return
	}
	if result.WalletAddress != "" {
		s.store.SetWalletAddress(id, result.WalletAddress)
	}
	if opts.SSHPublicKey != "" {
		s.store.SetDefaultKeyRemoved(id, true)
	}
	s.store.UpdateStatus(id, "ready", true)
	s.hub.Notify(id)
}

func (s *Server) handleGetServer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid server id"})
		return
	}

	info, err := s.store.GetServer(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "server not found"})
		return
	}

	writeJSON(w, http.StatusOK, ServerStatusResponse{
		ID:                info.ID,
		Name:              info.Name,
		Status:            info.Status,
		IPv4:              info.IPv4,
		Provisioned:       info.Provisioned,
		WalletAddress:     info.WalletAddress,
		DefaultKeyRemoved: info.DefaultKeyRemoved,
	})
}

func (s *Server) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid server id"})
		return
	}

	if err := s.hetzner.DeleteServer(r.Context(), id); err != nil {
		slog.Error("failed to delete server", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	s.store.DeleteServer(id)
	s.hub.Remove(id)

	writeJSON(w, http.StatusOK, DeleteServerResponse{
		ID:      id,
		Deleted: true,
	})
}

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := s.store.ListServers()
	if err != nil {
		slog.Error("failed to list servers", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list servers"})
		return
	}
	type item struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		Status            string `json:"status"`
		IPv4              string `json:"ipv4"`
		Provisioned       bool   `json:"provisioned"`
		WalletAddress     string `json:"wallet_address,omitempty"`
		DefaultKeyRemoved bool   `json:"default_key_removed"`
	}
	out := make([]item, len(servers))
	for i, info := range servers {
		out[i] = item{
			ID:                info.ID,
			Name:              info.Name,
			Status:            info.Status,
			IPv4:              info.IPv4,
			Provisioned:       info.Provisioned,
			WalletAddress:     info.WalletAddress,
			DefaultKeyRemoved: info.DefaultKeyRemoved,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid server id", http.StatusBadRequest)
		return
	}

	info, err := s.store.GetServer(id)
	if err != nil {
		http.Error(w, "server not found", http.StatusNotFound)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	// Read pump: drain reads, detect disconnect
	ctx := r.Context()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Helper to send JSON
	sendJSON := func(v any) error {
		return conn.WriteJSON(v)
	}

	// Send init message
	sendJSON(map[string]any{
		"type": "init",
		"server": map[string]any{
			"id":                  info.ID,
			"name":                info.Name,
			"ipv4":                info.IPv4,
			"status":              info.Status,
			"default_key_removed": info.DefaultKeyRemoved,
		},
	})

	// Replay all logs
	var lastLogID int64
	logs, err := s.store.GetLogsSince(id, 0)
	if err == nil {
		for _, entry := range logs {
			if err := sendJSON(map[string]any{"type": "log", "line": entry.Line}); err != nil {
				return
			}
			lastLogID = entry.ID
		}
	}

	// If already done, send final status and return
	isDone := func(status string) bool {
		return status == "ready" || status == "failed"
	}

	if isDone(info.Status) {
		sendJSON(map[string]any{
			"type":               "status",
			"status":             info.Status,
			"default_key_removed": info.DefaultKeyRemoved,
		})
		return
	}

	// Live streaming loop
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		waitCh := s.hub.WaitChan(id)
		select {
		case <-waitCh:
			// New logs available
			newLogs, err := s.store.GetLogsSince(id, lastLogID)
			if err != nil {
				continue
			}
			for _, entry := range newLogs {
				if err := sendJSON(map[string]any{"type": "log", "line": entry.Line}); err != nil {
					return
				}
				lastLogID = entry.ID
			}
			// Check if done
			info, err = s.store.GetServer(id)
			if err != nil {
				return
			}
			if isDone(info.Status) {
				sendJSON(map[string]any{
					"type":               "status",
					"status":             info.Status,
					"default_key_removed": info.DefaultKeyRemoved,
				})
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) handleReprovision(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid server id"})
		return
	}

	if err := s.store.ResetForReprovision(id); err != nil {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
		return
	}

	opts, err := s.store.GetProvisionOpts(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to load provision options"})
		return
	}

	logFn := s.makeLogFn(id, opts.SSHPublicKey != "")
	s.hub.Notify(id)

	go func() {
		logFn("Re-provisioning server...")
		if err := s.provisioner.CheckSSH(opts.IP, logFn); err != nil {
			slog.Error("SSH check failed during re-provision", "server_id", id, "error", err)
			logFn("SSH check failed: " + err.Error())
			s.store.UpdateStatus(id, "failed", false)
			s.hub.Notify(id)
			return
		}

		result, err := s.provisioner.RunPlaybook(*opts, logFn)
		if err != nil {
			slog.Error("re-provisioning failed", "server_id", id, "error", err)
			s.store.UpdateStatus(id, "failed", false)
			s.hub.Notify(id)
			return
		}
		if result.WalletAddress != "" {
			s.store.SetWalletAddress(id, result.WalletAddress)
		}
		if opts.SSHPublicKey != "" {
			s.store.SetDefaultKeyRemoved(id, true)
		}
		s.store.UpdateStatus(id, "ready", true)
		s.hub.Notify(id)
	}()

	writeJSON(w, http.StatusOK, map[string]string{"status": "reprovisioning"})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, _ := indexHTML.ReadFile("static/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func randomName() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("claw-%x", b)
}
