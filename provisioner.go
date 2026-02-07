package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Provisioner struct {
	ansibleDir    string
	sshPrivateKey string
}

func NewProvisioner(cfg *Config) *Provisioner {
	return &Provisioner{
		ansibleDir:    cfg.AnsibleDir,
		sshPrivateKey: expandHome(cfg.SSHPrivateKey),
	}
}

type ProvisionOpts struct {
	IP              string
	SSHPublicKey    string
	AnthropicAPIKey string
	WayfinderAPIKey string
}

func (p *Provisioner) WaitForSSH(ip string) error {
	addr := net.JoinHostPort(ip, "22")
	slog.Info("waiting for server boot", "addr", addr)
	time.Sleep(60 * time.Second)

	slog.Info("polling for SSH", "addr", addr)
	for attempt := range 60 {
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err == nil {
			conn.Close()
			slog.Info("SSH is ready", "addr", addr, "attempts", attempt+1)
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("SSH not ready after 5m at %s", addr)
}

// ProvisionResult holds outputs extracted from provisioning.
type ProvisionResult struct {
	WalletAddress string
}

func (p *Provisioner) RunPlaybook(opts ProvisionOpts) (*ProvisionResult, error) {
	slog.Info("starting provisioning", "ip", opts.IP)

	// Write a temporary inventory file
	inventoryContent := fmt.Sprintf("[openclaw]\n%s ansible_user=root ansible_ssh_private_key_file=%s ansible_ssh_common_args='-o StrictHostKeyChecking=no'\n", opts.IP, p.sshPrivateKey)

	inventoryFile, err := os.CreateTemp("", "inventory-*.ini")
	if err != nil {
		return nil, fmt.Errorf("create temp inventory: %w", err)
	}
	defer os.Remove(inventoryFile.Name())

	if _, err := inventoryFile.WriteString(inventoryContent); err != nil {
		inventoryFile.Close()
		return nil, fmt.Errorf("write inventory: %w", err)
	}
	inventoryFile.Close()

	args := []string{"-i", inventoryFile.Name(), "playbook.yml"}

	// Build extra vars from provision options
	extraVars := p.buildExtraVars(opts)
	if extraVars != "" {
		args = append(args, "--extra-vars", extraVars)
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = p.ansibleDir

	// Stream output in real-time via a pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ansible-playbook: %w", err)
	}

	var output strings.Builder
	buf := make([]byte, 4096)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			output.WriteString(chunk)
			// Log each line for real-time visibility
			for _, line := range strings.Split(strings.TrimRight(chunk, "\n"), "\n") {
				if line != "" {
					slog.Info("ansible", "ip", opts.IP, "out", line)
				}
			}
		}
		if readErr != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		slog.Error("provisioning failed", "ip", opts.IP, "error", err)
		return nil, fmt.Errorf("ansible-playbook: %w\n%s", err, output.String())
	}

	result := &ProvisionResult{
		WalletAddress: parseWalletAddress(output.String()),
	}

	slog.Info("provisioning completed", "ip", opts.IP, "wallet_address", result.WalletAddress)
	return result, nil
}

func (p *Provisioner) buildExtraVars(opts ProvisionOpts) string {
	vars := make(map[string]any)

	if opts.SSHPublicKey != "" {
		vars["clawdbot_ssh_keys"] = []string{opts.SSHPublicKey}
	}
	if opts.AnthropicAPIKey != "" {
		vars["anthropic_api_key"] = opts.AnthropicAPIKey
	}
	if opts.WayfinderAPIKey != "" {
		vars["wayfinder_api_key"] = opts.WayfinderAPIKey
	}
	if len(vars) == 0 {
		return ""
	}

	// Encode as JSON for --extra-vars
	b, err := json.Marshal(vars)
	if err != nil {
		slog.Error("failed to marshal extra vars", "error", err)
		return ""
	}
	return string(b)
}

func parseWalletAddress(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.Index(line, "WALLET_ADDRESS="); idx >= 0 {
			addr := strings.TrimSpace(line[idx+len("WALLET_ADDRESS="):])
			// Strip trailing quote if Ansible wraps it
			addr = strings.Trim(addr, "\"'")
			return addr
		}
	}
	return ""
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
