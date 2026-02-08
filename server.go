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
)

//go:embed static/index.html
var indexHTML embed.FS

type Server struct {
	hetzner     *HetznerClient
	provisioner *Provisioner
	tracker     *ServerTracker
	username    string
	password    string
}

func NewServer(hetzner *HetznerClient, provisioner *Provisioner, tracker *ServerTracker, username, password string) *Server {
	return &Server{
		hetzner:     hetzner,
		provisioner: provisioner,
		tracker:     tracker,
		username:    username,
		password:    password,
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /servers", s.handleCreateServer)
	mux.HandleFunc("GET /servers", s.handleListServers)
	mux.HandleFunc("GET /servers/{id}/logs", s.handleGetServerLogs)
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

	s.tracker.Add(info)

	logFn := func(line string) {
		s.tracker.AppendLog(info.ID, line)
	}

	logFn("Creating server...")
	logFn(fmt.Sprintf("Server created: %s (%s)", info.Name, info.IPv4))
	logFn("Waiting for SSH to become available...")

	// Kick off SSH wait + Ansible provisioning in the background
	go func(id int64, opts ProvisionOpts) {
		if err := s.provisioner.WaitForSSH(info.IPv4, logFn); err != nil {
			slog.Error("SSH wait failed", "server_id", id, "error", err)
			logFn("SSH wait failed: " + err.Error())
			s.tracker.UpdateStatus(id, "failed", false)
			return
		}

		result, err := s.provisioner.RunPlaybook(opts, logFn)
		if err != nil {
			slog.Error("provisioning failed", "server_id", id, "error", err)
			s.tracker.UpdateStatus(id, "failed", false)
			return
		}
		if result.WalletAddress != "" {
			s.tracker.SetWalletAddress(id, result.WalletAddress)
		}
		s.tracker.UpdateStatus(id, "ready", true)
	}(info.ID, ProvisionOpts{
		IP:              info.IPv4,
		SSHPublicKey:    req.SSHPublicKey,
		AnthropicAPIKey: req.AnthropicAPIKey,
		OpenAIAPIKey:    req.OpenAIAPIKey,
		GeminiAPIKey:    req.GeminiAPIKey,
		WayfinderAPIKey: req.WayfinderAPIKey,
		Channels:        req.Channels,
	})

	writeJSON(w, http.StatusAccepted, CreateServerResponse{
		ID:     info.ID,
		Name:   info.Name,
		Status: info.Status,
		IPv4:   info.IPv4,
	})
}

func (s *Server) handleGetServer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid server id"})
		return
	}

	info, ok := s.tracker.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "server not found"})
		return
	}

	writeJSON(w, http.StatusOK, ServerStatusResponse{
		ID:            info.ID,
		Name:          info.Name,
		Status:        info.Status,
		IPv4:          info.IPv4,
		Provisioned:   info.Provisioned,
		WalletAddress: info.WalletAddress,
	})
}

func (s *Server) handleGetServerLogs(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid server id"})
		return
	}

	info, ok := s.tracker.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "server not found"})
		return
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	lines, nextOffset := s.tracker.GetLogs(id, offset)
	done := info.Status == "ready" || info.Status == "failed"

	writeJSON(w, http.StatusOK, LogsResponse{
		Lines:  lines,
		Offset: nextOffset,
		Status: info.Status,
		Done:   done,
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

	s.tracker.Remove(id)

	writeJSON(w, http.StatusOK, DeleteServerResponse{
		ID:      id,
		Deleted: true,
	})
}

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers := s.tracker.List()
	type item struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		Status        string `json:"status"`
		IPv4          string `json:"ipv4"`
		Provisioned   bool   `json:"provisioned"`
		WalletAddress string `json:"wallet_address,omitempty"`
	}
	out := make([]item, len(servers))
	for i, info := range servers {
		out[i] = item{
			ID:            info.ID,
			Name:          info.Name,
			Status:        info.Status,
			IPv4:          info.IPv4,
			Provisioned:   info.Provisioned,
			WalletAddress: info.WalletAddress,
		}
	}
	writeJSON(w, http.StatusOK, out)
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
