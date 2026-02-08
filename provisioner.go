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
	keyPath := expandHome(cfg.SSHPrivateKey)

	// If raw key data is provided via env, write it to a temp file and use that instead.
	// This avoids needing to mount/copy an SSH key file into a container.
	if cfg.SSHPrivateKeyData != "" {
		f, err := os.CreateTemp("", "ssh-private-key-*")
		if err != nil {
			slog.Error("failed to create temp file for SSH key data", "error", err)
		} else {
			data := cfg.SSHPrivateKeyData + "\n"
			if err := os.WriteFile(f.Name(), []byte(data), 0600); err != nil {
				slog.Error("failed to write SSH key data", "error", err)
			} else {
				keyPath = f.Name()
				slog.Info("using SSH key from SSH_PRIVATE_KEY_DATA env")
			}
			f.Close()
		}
	}

	return &Provisioner{
		ansibleDir:    cfg.AnsibleDir,
		sshPrivateKey: keyPath,
	}
}

type ProvisionOpts struct {
	IP              string
	SSHPublicKey    string
	AnthropicAPIKey string
	OpenAIAPIKey    string
	GeminiAPIKey    string
	WayfinderAPIKey string
	Channels        []ChannelConfig
}

func (p *Provisioner) WaitForSSH(ip string, logFn func(string)) error {
	addr := net.JoinHostPort(ip, "22")
	slog.Info("waiting for server boot", "addr", addr)
	logFn("Waiting 60s for server to boot...")
	time.Sleep(60 * time.Second)

	slog.Info("polling for SSH", "addr", addr)
	logFn("Polling for SSH on " + addr + "...")
	for attempt := range 60 {
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err == nil {
			conn.Close()
			slog.Info("SSH is ready", "addr", addr, "attempts", attempt+1)
			logFn(fmt.Sprintf("SSH is ready (after %d attempts)", attempt+1))
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	logFn("SSH not ready after 5 minutes — giving up")
	return fmt.Errorf("SSH not ready after 5m at %s", addr)
}

// ProvisionResult holds outputs extracted from provisioning.
type ProvisionResult struct {
	WalletAddress string
}

func (p *Provisioner) RunPlaybook(opts ProvisionOpts, logFn func(string)) (*ProvisionResult, error) {
	slog.Info("starting provisioning",
		"ip", opts.IP,
		"ansible_dir", p.ansibleDir,
		"ssh_key", p.sshPrivateKey,
		"has_ssh_public_key", opts.SSHPublicKey != "",
		"has_anthropic_key", opts.AnthropicAPIKey != "",
		"has_openai_key", opts.OpenAIAPIKey != "",
		"has_gemini_key", opts.GeminiAPIKey != "",
		"has_wayfinder_key", opts.WayfinderAPIKey != "",
		"channels", len(opts.Channels),
	)
	logFn("Starting Ansible provisioning...")

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

	slog.Info("inventory written", "ip", opts.IP, "file", inventoryFile.Name())

	args := []string{"-i", inventoryFile.Name(), "playbook.yml", "-vv"}

	// Build extra vars from provision options
	extraVars := p.buildExtraVars(opts)
	if extraVars != "" {
		args = append(args, "--extra-vars", extraVars)
		slog.Info("extra vars configured", "ip", opts.IP)
	}

	// Collect secret values to redact from output
	secrets := collectSecrets(opts)

	slog.Info("launching ansible-playbook", "ip", opts.IP, "cwd", p.ansibleDir)
	logFn("Running: ansible-playbook playbook.yml")

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = p.ansibleDir

	// Stream output in real-time via a pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ansible-playbook: %w", err)
	}
	slog.Info("ansible-playbook started", "ip", opts.IP, "pid", cmd.Process.Pid)

	var output strings.Builder
	buf := make([]byte, 4096)
	lastLogTime := time.Now()
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			output.WriteString(chunk)
			// Log each line for real-time visibility
			for _, line := range strings.Split(strings.TrimRight(chunk, "\n"), "\n") {
				if line != "" {
					redacted := redactLine(line, secrets)
					slog.Info("ansible", "ip", opts.IP, "out", redacted)
					logFn(redacted)
				}
			}
			lastLogTime = time.Now()
		}
		if readErr != nil {
			break
		}
		// Warn if no output for 60 seconds (likely stuck on interactive prompt)
		if time.Since(lastLogTime) > 60*time.Second {
			elapsed := time.Since(startTime).Round(time.Second)
			slog.Warn("ansible appears stalled - no output for 60s (may be stuck on interactive prompt)",
				"ip", opts.IP,
				"elapsed", elapsed.String(),
				"last_output_ago", time.Since(lastLogTime).Round(time.Second).String(),
			)
			lastLogTime = time.Now() // reset so we don't spam
		}
	}

	elapsed := time.Since(startTime).Round(time.Second)
	if err := cmd.Wait(); err != nil {
		slog.Error("provisioning failed", "ip", opts.IP, "error", err, "elapsed", elapsed.String(), "output_bytes", output.Len())
		logFn(fmt.Sprintf("Provisioning FAILED after %s: %v", elapsed, err))
		return nil, fmt.Errorf("ansible-playbook: %w\n%s", err, output.String())
	}

	result := &ProvisionResult{
		WalletAddress: parseWalletAddress(output.String()),
	}

	logFn(fmt.Sprintf("Provisioning completed successfully in %s", elapsed))
	slog.Info("provisioning completed", "ip", opts.IP, "wallet_address", result.WalletAddress, "elapsed", elapsed.String())
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
	if opts.OpenAIAPIKey != "" {
		vars["openai_api_key"] = opts.OpenAIAPIKey
	}
	if opts.GeminiAPIKey != "" {
		vars["gemini_api_key"] = opts.GeminiAPIKey
	}
	if opts.WayfinderAPIKey != "" {
		vars["wayfinder_api_key"] = opts.WayfinderAPIKey
	}
	if len(opts.Channels) > 0 {
		// Pass channels as a JSON-serializable list for Ansible
		channels := make([]map[string]string, len(opts.Channels))
		for i, ch := range opts.Channels {
			m := map[string]string{"type": ch.Type}
			if ch.Token != "" {
				m["token"] = ch.Token
			}
			if ch.Name != "" {
				m["name"] = ch.Name
			}
			if ch.Account != "" {
				m["account"] = ch.Account
			}
			channels[i] = m
		}
		vars["channels"] = channels
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

func (p *Provisioner) CheckSSH(ip string, logFn func(string)) error {
	addr := net.JoinHostPort(ip, "22")
	logFn("Checking SSH connectivity on " + addr + "...")
	for attempt := range 5 {
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err == nil {
			conn.Close()
			logFn(fmt.Sprintf("SSH is ready (attempt %d)", attempt+1))
			return nil
		}
		logFn(fmt.Sprintf("SSH attempt %d/5 failed: %v", attempt+1, err))
		if attempt < 4 {
			time.Sleep(5 * time.Second)
		}
	}
	return fmt.Errorf("SSH not reachable after 5 attempts at %s", addr)
}

func collectSecrets(opts ProvisionOpts) []string {
	var secrets []string
	for _, s := range []string{
		opts.AnthropicAPIKey,
		opts.OpenAIAPIKey,
		opts.GeminiAPIKey,
		opts.WayfinderAPIKey,
	} {
		if len(s) > 3 {
			secrets = append(secrets, s)
		}
	}
	for _, ch := range opts.Channels {
		if len(ch.Token) > 3 {
			secrets = append(secrets, ch.Token)
		}
	}
	return secrets
}

func redactLine(line string, secrets []string) string {
	for _, s := range secrets {
		if strings.Contains(line, s) {
			line = strings.ReplaceAll(line, s, s[:3]+"***")
		}
	}
	return line
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
