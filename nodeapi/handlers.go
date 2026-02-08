package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const openclawBin = "/home/clawdbot/.local/bin/openclaw"

const maxBodyBytes = 4096 // 4 KB — plenty for JSON action requests

const maxConcurrentCLI = 4

var cliSem = make(chan struct{}, maxConcurrentCLI)

const cliTimeout = 30 * time.Second

var cmdEnv = []string{
	"HOME=/home/clawdbot",
	"PATH=/home/clawdbot/.local/bin:/home/clawdbot/.local/share/pnpm:/home/linuxbrew/.linuxbrew/bin:/usr/local/bin:/usr/bin:/bin",
	"PNPM_HOME=/home/clawdbot/.local/share/pnpm",
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleChannelsStatus(w http.ResponseWriter, r *http.Request) {
	out, err := runCLI(r.Context(), "channels", "status", "--json")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("channels status failed: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func handlePairingRequests(w http.ResponseWriter, r *http.Request) {
	channelsOut, err := runCLI(r.Context(), "channels", "status", "--json")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("channels status failed: %v", err))
		return
	}

	var channelsStatus struct {
		ChannelOrder []string `json:"channelOrder"`
	}
	if err := json.Unmarshal(channelsOut, &channelsStatus); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to parse channels: %v", err))
		return
	}

	type pairingRequest struct {
		ID        string          `json:"id"`
		Channel   string          `json:"channel"`
		Code      string          `json:"code,omitempty"`
		CreatedAt string          `json:"created_at"`
		LastSeenAt string         `json:"last_seen_at,omitempty"`
		Meta      json.RawMessage `json:"meta,omitempty"`
	}

	var allRequests []pairingRequest

	for _, chName := range channelsStatus.ChannelOrder {
		out, err := runCLI(r.Context(), "pairing", "list", chName, "--json")
		if err != nil {
			log.Printf("pairing list for %s failed: %v", chName, err)
			continue
		}
		var pairingResp struct {
			Channel  string `json:"channel"`
			Requests []struct {
				ID         string          `json:"id"`
				Code       string          `json:"code"`
				CreatedAt  string          `json:"createdAt"`
				LastSeenAt string          `json:"lastSeenAt"`
				Meta       json.RawMessage `json:"meta"`
			} `json:"requests"`
		}
		if err := json.Unmarshal(out, &pairingResp); err != nil {
			log.Printf("failed to parse pairing list for %s: %v", chName, err)
			continue
		}
		for _, req := range pairingResp.Requests {
			allRequests = append(allRequests, pairingRequest{
				ID:         req.ID,
				Channel:    chName,
				Code:       req.Code,
				CreatedAt:  req.CreatedAt,
				LastSeenAt: req.LastSeenAt,
				Meta:       req.Meta,
			})
		}
	}

	if allRequests == nil {
		allRequests = []pairingRequest{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allRequests)
}

func handlePairingApprove(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req struct {
		Channel string `json:"channel"`
		ID      string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Channel == "" || req.ID == "" {
		writeError(w, http.StatusBadRequest, "channel and id are required")
		return
	}

	out, err := runCLI(r.Context(), "pairing", "approve", req.Channel, req.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("approve failed: %v: %s", err, string(out)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

func handlePairingDeny(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req struct {
		Channel string `json:"channel"`
		ID      string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Channel == "" || req.ID == "" {
		writeError(w, http.StatusBadRequest, "channel and id are required")
		return
	}

	out, err := runCLI(r.Context(), "pairing", "deny", req.Channel, req.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deny failed: %v: %s", err, string(out)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "denied"})
}

func runCLI(ctx context.Context, args ...string) ([]byte, error) {
	// Sanitize args
	for _, arg := range args {
		if strings.ContainsAny(arg, ";|&`$(){}[]\\'\"\n\r") {
			return nil, fmt.Errorf("invalid argument: %q", arg)
		}
	}

	// Acquire semaphore slot (bounded concurrency)
	select {
	case cliSem <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("request cancelled waiting for CLI slot")
	}
	defer func() { <-cliSem }()

	ctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, openclawBin, args...)
	cmd.Env = cmdEnv
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}
	return out, nil
}
