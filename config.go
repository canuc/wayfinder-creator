package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Hetzner (optional — set HCLOUD_TOKEN to enable)
	HCloudToken string
	SSHKeyID    int64
	ServerType  string
	Image       string
	Location    string

	// Vultr (optional — set VULTR_API_KEY to enable)
	VultrAPIKey   string
	VultrPlan     string
	VultrRegion   string
	VultrOSID     int
	VultrSSHKeyID string

	ListenAddr             string
	AnsibleDir             string
	SSHPrivateKey          string
	SSHPrivateKeyData      string
	SessionSecret          string
	DatabaseURL            string
	WalletConnectProjectID string
}

func LoadConfig() (*Config, error) {
	// Load .env if present; doesn't override existing env vars
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	hcloudToken := os.Getenv("HCLOUD_TOKEN")
	vultrAPIKey := os.Getenv("VULTR_API_KEY")

	if hcloudToken == "" && vultrAPIKey == "" {
		return nil, fmt.Errorf("at least one provider must be configured: set HCLOUD_TOKEN and/or VULTR_API_KEY")
	}

	// Parse Hetzner SSH key ID (only required when Hetzner is enabled)
	var sshKeyID int64
	if hcloudToken != "" {
		sshKeyStr := os.Getenv("SSH_KEY_ID")
		if sshKeyStr == "" {
			return nil, fmt.Errorf("SSH_KEY_ID is required when HCLOUD_TOKEN is set")
		}
		var err error
		sshKeyID, err = strconv.ParseInt(sshKeyStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("SSH_KEY_ID must be an integer: %w", err)
		}
	}

	// Parse Vultr OS ID
	vultrOSID := 2284 // Ubuntu 24.04
	if v := os.Getenv("VULTR_OS_ID"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("VULTR_OS_ID must be an integer: %w", err)
		}
		vultrOSID = parsed
	}

	return &Config{
		HCloudToken:            hcloudToken,
		SSHKeyID:               sshKeyID,
		ServerType:             envOrDefault("SERVER_TYPE", "cpx11"),
		Image:                  envOrDefault("IMAGE", "ubuntu-24.04"),
		Location:               envOrDefault("LOCATION", "fsn1"),
		VultrAPIKey:            vultrAPIKey,
		VultrPlan:              envOrDefault("VULTR_PLAN", "vc2-1c-1gb"),
		VultrRegion:            envOrDefault("VULTR_REGION", "ewr"),
		VultrOSID:              vultrOSID,
		VultrSSHKeyID:          os.Getenv("VULTR_SSH_KEY_ID"),
		ListenAddr:             envOrDefault("LISTEN_ADDR", ":8080"),
		AnsibleDir:             envOrDefault("ANSIBLE_DIR", "./ansible"),
		SSHPrivateKey:          envOrDefault("SSH_PRIVATE_KEY", "~/.ssh/id_ed25519"),
		SSHPrivateKeyData:      os.Getenv("SSH_PRIVATE_KEY_DATA"),
		SessionSecret:          envOrDefault("SESSION_SECRET", "openclaw-default-secret-change-me"),
		DatabaseURL:            dbURL,
		WalletConnectProjectID: os.Getenv("WALLETCONNECT_PROJECT_ID"),
	}, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
