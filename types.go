package main

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
	PublicKeyPEM    string          `json:"public_key_pem,omitempty"`
}

type CreateServerResponse struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	IPv4   string `json:"ipv4"`
}

type ServerStatusResponse struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	IPv4              string `json:"ipv4"`
	Provisioned       bool   `json:"provisioned"`
	WalletAddress     string `json:"wallet_address,omitempty"`
	DefaultKeyRemoved bool   `json:"default_key_removed"`
}

type DeleteServerResponse struct {
	ID      int64 `json:"id"`
	Deleted bool  `json:"deleted"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Server state (persisted in PostgreSQL)

type ServerInfo struct {
	ID                int64
	Name              string
	IPv4              string
	Status            string // "provisioning", "ready", "failed"
	Provisioned       bool
	WalletAddress     string
	DefaultKeyRemoved bool
}

type LogEntry struct {
	ID   int64  `json:"id"`
	Line string `json:"line"`
}

type PairingRequest struct {
	ID         string `json:"id"`
	Channel    string `json:"channel"`
	User       string `json:"user"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	ServerID   int64  `json:"server_id"`
	ServerName string `json:"server_name"`
}

type PairingActionRequest struct {
	Channel string `json:"channel"`
	ID      string `json:"id"`
}
