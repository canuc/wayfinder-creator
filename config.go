package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HCloudToken       string
	SSHKeyID          int64
	ServerType        string
	Image             string
	Location          string
	ListenAddr        string
	AnsibleDir        string
	SSHPrivateKey     string
	SSHPrivateKeyData string
}

func LoadConfig() (*Config, error) {
	// Load .env if present; doesn't override existing env vars
	_ = godotenv.Load()

	token := os.Getenv("HCLOUD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("HCLOUD_TOKEN is required")
	}

	sshKeyStr := os.Getenv("SSH_KEY_ID")
	if sshKeyStr == "" {
		return nil, fmt.Errorf("SSH_KEY_ID is required")
	}
	sshKeyID, err := strconv.ParseInt(sshKeyStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("SSH_KEY_ID must be an integer: %w", err)
	}

	return &Config{
		HCloudToken:       token,
		SSHKeyID:          sshKeyID,
		ServerType:        envOrDefault("SERVER_TYPE", "cpx11"),
		Image:             envOrDefault("IMAGE", "ubuntu-24.04"),
		Location:          envOrDefault("LOCATION", "fsn1"),
		ListenAddr:        envOrDefault("LISTEN_ADDR", ":8080"),
		AnsibleDir:        envOrDefault("ANSIBLE_DIR", "./ansible"),
		SSHPrivateKey:     envOrDefault("SSH_PRIVATE_KEY", "~/.ssh/id_ed25519"),
		SSHPrivateKeyData: os.Getenv("SSH_PRIVATE_KEY_DATA"),
	}, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
