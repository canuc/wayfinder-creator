package main

import "sync"

// Request/response types

type ChannelConfig struct {
	Type    string `json:"type"`              // telegram, discord, slack, whatsapp, signal, googlechat, mattermost
	Token   string `json:"token,omitempty"`   // bot token (telegram, discord, slack)
	Name    string `json:"name,omitempty"`    // display name for the account
	Account string `json:"account,omitempty"` // account id (default: "default")
}

type CreateServerRequest struct {
	Name            string          `json:"name"`
	SSHPublicKey    string          `json:"ssh_public_key,omitempty"`
	AnthropicAPIKey string          `json:"anthropic_api_key,omitempty"`
	OpenAIAPIKey    string          `json:"openai_api_key,omitempty"`
	GeminiAPIKey    string          `json:"gemini_api_key,omitempty"`
	WayfinderAPIKey string          `json:"wayfinder_api_key,omitempty"`
	Channels        []ChannelConfig `json:"channels,omitempty"`
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
	Logs          []string
}

type LogsResponse struct {
	Lines  []string `json:"lines"`
	Offset int      `json:"offset"`
	Status string   `json:"status"`
	Done   bool     `json:"done"`
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

func (t *ServerTracker) AppendLog(id int64, line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if info, ok := t.servers[id]; ok {
		info.Logs = append(info.Logs, line)
	}
}

func (t *ServerTracker) GetLogs(id int64, offset int) (lines []string, nextOffset int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	info, ok := t.servers[id]
	if !ok {
		return nil, offset
	}
	if offset >= len(info.Logs) {
		return nil, offset
	}
	lines = info.Logs[offset:]
	return lines, len(info.Logs)
}

func (t *ServerTracker) Remove(id int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.servers, id)
}

func (t *ServerTracker) List() []*ServerInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	list := make([]*ServerInfo, 0, len(t.servers))
	for _, info := range t.servers {
		list = append(list, info)
	}
	return list
}
