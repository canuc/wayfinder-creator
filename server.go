package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

type Server struct {
	hetzner     *HetznerClient
	provisioner *Provisioner
	tracker     *ServerTracker
}

func NewServer(hetzner *HetznerClient, provisioner *Provisioner, tracker *ServerTracker) *Server {
	return &Server{
		hetzner:     hetzner,
		provisioner: provisioner,
		tracker:     tracker,
	}
}

func (s *Server) Router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /servers", s.handleCreateServer)
	mux.HandleFunc("GET /servers/{id}", s.handleGetServer)
	mux.HandleFunc("DELETE /servers/{id}", s.handleDeleteServer)
	return mux
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

	// Wait for SSH before returning
	if err := s.provisioner.WaitForSSH(info.IPv4); err != nil {
		slog.Error("SSH wait failed", "server_id", info.ID, "error", err)
		s.tracker.UpdateStatus(info.ID, "failed", false)
		writeJSON(w, http.StatusGatewayTimeout, ErrorResponse{Error: "server created but SSH not reachable: " + err.Error()})
		return
	}

	// Kick off Ansible provisioning in the background
	go func(id int64, opts ProvisionOpts) {
		if err := s.provisioner.RunPlaybook(opts); err != nil {
			slog.Error("provisioning failed", "server_id", id, "error", err)
			s.tracker.UpdateStatus(id, "failed", false)
			return
		}
		s.tracker.UpdateStatus(id, "ready", true)
	}(info.ID, ProvisionOpts{
		IP:              info.IPv4,
		SSHPublicKey:    req.SSHPublicKey,
		AnthropicAPIKey: req.AnthropicAPIKey,
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
		ID:          info.ID,
		Name:        info.Name,
		Status:      info.Status,
		IPv4:        info.IPv4,
		Provisioned: info.Provisioned,
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
