package main

import "sync"

// Request/response types

type CreateServerRequest struct {
	Name            string `json:"name"`
	SSHPublicKey    string `json:"ssh_public_key,omitempty"`
	AnthropicAPIKey string `json:"anthropic_api_key,omitempty"`
	WayfinderAPIKey string `json:"wayfinder_api_key,omitempty"`
}

type CreateServerResponse struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	IPv4   string `json:"ipv4"`
}

type ServerStatusResponse struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	IPv4          string `json:"ipv4"`
	Provisioned   bool   `json:"provisioned"`
	WalletAddress string `json:"wallet_address,omitempty"`
}

type DeleteServerResponse struct {
	ID      int64 `json:"id"`
	Deleted bool  `json:"deleted"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// In-memory state

type ServerInfo struct {
	ID            int64
	Name          string
	IPv4          string
	Status        string // "provisioning", "ready", "failed"
	Provisioned   bool
	WalletAddress string
}

type ServerTracker struct {
	mu      sync.RWMutex
	servers map[int64]*ServerInfo
}

func NewServerTracker() *ServerTracker {
	return &ServerTracker{
		servers: make(map[int64]*ServerInfo),
	}
}

func (t *ServerTracker) Add(info *ServerInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.servers[info.ID] = info
}

func (t *ServerTracker) Get(id int64) (*ServerInfo, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	info, ok := t.servers[id]
	return info, ok
}

func (t *ServerTracker) UpdateStatus(id int64, status string, provisioned bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if info, ok := t.servers[id]; ok {
		info.Status = status
		info.Provisioned = provisioned
	}
}

func (t *ServerTracker) SetWalletAddress(id int64, addr string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if info, ok := t.servers[id]; ok {
		info.WalletAddress = addr
	}
}

func (t *ServerTracker) Remove(id int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.servers, id)
}
