package main

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	config      *Config
	providers   map[string]VPSProvider
	provisioner *Provisioner
	store       *Store
	hub         *LogHub
	challenges  *ChallengeStore
	upgrader    websocket.Upgrader
}

func NewServer(cfg *Config, providers map[string]VPSProvider, provisioner *Provisioner, store *Store, hub *LogHub) *Server {
	return &Server{
		config:      cfg,
		providers:   providers,
		provisioner: provisioner,
		store:       store,
		hub:         hub,
		challenges:  NewChallengeStore(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Auth routes (public)
	mux.HandleFunc("POST /auth/challenge", s.handleChallenge)
	mux.HandleFunc("POST /auth/verify", s.handleVerify)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)
	mux.HandleFunc("GET /auth/me", s.handleMe)
	mux.HandleFunc("PUT /auth/ssh-key", s.requireApproved(s.handleSetSSHKey))

	// Admin routes (require admin)
	mux.HandleFunc("GET /admin/users", s.requireAdmin(s.handleListUsers))
	mux.HandleFunc("POST /admin/users/{id}/approve", s.requireAdmin(s.handleApproveUser))
	mux.HandleFunc("DELETE /admin/users/{id}", s.requireAdmin(s.handleDeleteUser))

	// Server routes (require approved user)
	mux.HandleFunc("POST /servers", s.requireApproved(s.handleCreateServer))
	mux.HandleFunc("GET /servers", s.requireApproved(s.handleListServers))
	mux.HandleFunc("GET /servers/{id}/ws", s.handleWebSocket) // WS auth handled inline
	mux.HandleFunc("POST /servers/{id}/public-key", s.requireApproved(s.handleSetPublicKey))
	mux.HandleFunc("GET /servers/{id}/pairing/requests", s.requireApproved(s.handlePairingRequests))
	mux.HandleFunc("POST /servers/{id}/pairing/approve", s.requireApproved(s.handlePairingApprove))
	mux.HandleFunc("POST /servers/{id}/pairing/deny", s.requireApproved(s.handlePairingDeny))
	mux.HandleFunc("GET /servers/{id}/channels/status", s.requireApproved(s.handleChannelsStatus))
	mux.HandleFunc("GET /servers/{id}", s.requireApproved(s.handleGetServer))
	mux.HandleFunc("DELETE /servers/{id}", s.requireApproved(s.handleDeleteServer))

	// Public config
	mux.HandleFunc("GET /config", s.handleConfig)

	// SPA static files
	mux.HandleFunc("GET /", s.handleSPA)

	return mux
}

func (s *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	var req CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Name == "" {
		req.Name = randomName()
	}
	if req.Provider == "" {
		// Default to first available provider
		for name := range s.providers {
			req.Provider = name
			break
		}
	}

	provider, ok := s.providers[req.Provider]
	if !ok {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("unsupported provider: %s", req.Provider)})
		return
	}

	info, err := provider.CreateServer(r.Context(), req.Name)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	info.Provider = req.Provider

	opts := ProvisionOpts{
		IP:               info.IPv4,
		SSHPublicKey:     req.SSHPublicKey,
		AnthropicAPIKey:  req.AnthropicAPIKey,
		OpenAIAPIKey:     req.OpenAIAPIKey,
		GeminiAPIKey:     req.GeminiAPIKey,
		WayfinderAPIKey:  req.WayfinderAPIKey,
		Channels:         req.Channels,
		CreatorPublicKey: req.PublicKeyPEM,
	}

	if err := s.store.CreateServer(info, opts, user.ID); err != nil {
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
		ID:       info.ID,
		Name:     info.Name,
		Status:   info.Status,
		IPv4:     info.IPv4,
		Provider: info.Provider,
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
		s.store.ClearChannelTokens(id)
		s.hub.Notify(id)
		return
	}

	result, err := s.provisioner.RunPlaybook(opts, logFn)
	if err != nil {
		slog.Error("provisioning failed", "server_id", id, "error", err)
		s.store.UpdateStatus(id, "failed", false)
		s.store.ClearChannelTokens(id)
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
	s.store.ClearChannelTokens(id)
	s.hub.Notify(id)
}

func (s *Server) handleGetServer(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid server id"})
		return
	}

	info, err := s.store.GetServer(id, user.ID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "server not found"})
		return
	}

	writeJSON(w, http.StatusOK, ServerStatusResponse{
		ID:                info.ID,
		Name:              info.Name,
		Status:            info.Status,
		IPv4:              info.IPv4,
		Provider:          info.Provider,
		Provisioned:       info.Provisioned,
		WalletAddress:     info.WalletAddress,
		DefaultKeyRemoved: info.DefaultKeyRemoved,
	})
}

func (s *Server) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid server id"})
		return
	}

	// Look up provider before deleting from DB
	info, err := s.store.GetServer(id, user.ID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "server not found"})
		return
	}

	if err := s.store.DeleteServer(id, user.ID); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "server not found"})
		return
	}

	if provider, ok := s.providers[info.Provider]; ok {
		if err := provider.DeleteServer(r.Context(), info.ProviderID); err != nil {
			slog.Error("failed to delete server from provider", "provider", info.Provider, "error", err)
			// Server already deleted from DB, log the error but don't fail
		}
	} else {
		slog.Error("unknown provider for server deletion", "provider", info.Provider, "server_id", id)
	}

	s.hub.Remove(id)

	writeJSON(w, http.StatusOK, DeleteServerResponse{
		ID:      id,
		Deleted: true,
	})
}

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	servers, err := s.store.ListServers(user.ID)
	if err != nil {
		slog.Error("failed to list servers", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list servers"})
		return
	}
	type item struct {
		ID                int64  `json:"id"`
		ProviderID        string `json:"provider_id"`
		Name              string `json:"name"`
		Status            string `json:"status"`
		IPv4              string `json:"ipv4"`
		Provider          string `json:"provider"`
		Provisioned       bool   `json:"provisioned"`
		WalletAddress     string `json:"wallet_address,omitempty"`
		DefaultKeyRemoved bool   `json:"default_key_removed"`
		HasNodeAPI        bool   `json:"has_node_api"`
		CreatedAt         string `json:"created_at,omitempty"`
		ChannelCount      int    `json:"channel_count"`
	}
	out := make([]item, len(servers))
	for i, info := range servers {
		out[i] = item{
			ID:                info.ID,
			ProviderID:        info.ProviderID,
			Name:              info.Name,
			Status:            info.Status,
			IPv4:              info.IPv4,
			Provider:          info.Provider,
			Provisioned:       info.Provisioned,
			WalletAddress:     info.WalletAddress,
			DefaultKeyRemoved: info.DefaultKeyRemoved,
			HasNodeAPI:        info.HasNodeAPI,
			CreatedAt:         info.CreatedAt,
			ChannelCount:      info.ChannelCount,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// WebSocket auth: check session cookie
	user, r := s.sessionAuth(r)
	if user == nil || !user.Approved {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid server id", http.StatusBadRequest)
		return
	}

	info, err := s.store.GetServer(id, user.ID)
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
			info, err = s.store.GetServerAny(id)
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

func (s *Server) handleSetPublicKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid server id"})
		return
	}

	var req struct {
		PublicKeyPEM string `json:"public_key_pem"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PublicKeyPEM == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "public_key_pem is required"})
		return
	}

	if err := s.store.SetPublicKey(id, req.PublicKeyPEM); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to set public key"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePairingRequests(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	s.proxyToNode(w, r, id, "GET", "/pairing/requests", nil)
}

func (s *Server) handlePairingApprove(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	body, _ := io.ReadAll(r.Body)
	s.proxyToNode(w, r, id, "POST", "/pairing/approve", body)
}

func (s *Server) handlePairingDeny(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	body, _ := io.ReadAll(r.Body)
	s.proxyToNode(w, r, id, "POST", "/pairing/deny", body)
}

func (s *Server) handleChannelsStatus(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	s.proxyToNode(w, r, id, "GET", "/channels/status", nil)
}

func (s *Server) proxyToNode(w http.ResponseWriter, r *http.Request, serverID int64, method, path string, body []byte) {
	user := userFromContext(r.Context())
	info, err := s.store.GetServer(serverID, user.ID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "server not found"})
		return
	}
	if info.Status != "ready" {
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "server is not ready"})
		return
	}
	if !info.HasNodeAPI {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "node API not deployed on this server"})
		return
	}

	url := fmt.Sprintf("http://%s:8443%s", info.IPv4, path)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	}

	proxyReq, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create proxy request"})
		return
	}

	// Forward signature headers unchanged
	for _, h := range []string{"X-Signature", "X-Signature-Timestamp", "X-Content-Digest", "X-Signature-Method", "Content-Type"} {
		if v := r.Header.Get(h); v != "" {
			proxyReq.Header.Set(h, v)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		slog.Error("proxy to node failed", "server_id", serverID, "url", url, "error", err)
		writeJSON(w, http.StatusBadGateway, ErrorResponse{Error: "node unreachable"})
		return
	}
	defer resp.Body.Close()

	// Forward response
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		slog.Warn("node returned error", "server_id", serverID, "path", path, "status", resp.StatusCode, "body", string(bodyBytes))
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		w.Write(bodyBytes)
		return
	}
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"walletconnect_project_id": s.config.WalletConnectProjectID,
		"providers":                providers,
	})
}

func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	// Serve static files from embedded FS
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Try to serve the exact file first
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	f, err := sub.Open(strings.TrimPrefix(path, "/"))
	if err == nil {
		f.Close()
		http.FileServerFS(sub).ServeHTTP(w, r)
		return
	}

	// SPA fallback: serve index.html for non-file routes
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
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
