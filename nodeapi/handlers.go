package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

const openclawBin = "/home/clawdbot/.local/bin/openclaw"

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
	out, err := runCLI("channels", "status", "--json")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("channels status failed: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func handlePairingRequests(w http.ResponseWriter, r *http.Request) {
	// Get channels list
	channelsOut, err := runCLI("channels", "status", "--json")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("channels status failed: %v", err))
		return
	}

	var channels []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(channelsOut, &channels); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse channels")
		return
	}

	type pairingRequest struct {
		ID        string `json:"id"`
		Channel   string `json:"channel"`
		User      string `json:"user"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}

	var allRequests []pairingRequest

	for _, ch := range channels {
		chName := ch.Name
		if chName == "" {
			chName = "default"
		}
		out, err := runCLI("pairing", "list", chName, "--json")
		if err != nil {
			log.Printf("pairing list for %s failed: %v", chName, err)
			continue
		}
		var requests []pairingRequest
		if err := json.Unmarshal(out, &requests); err != nil {
			log.Printf("failed to parse pairing list for %s: %v", chName, err)
			continue
		}
		for i := range requests {
			requests[i].Channel = chName
		}
		allRequests = append(allRequests, requests...)
	}

	if allRequests == nil {
		allRequests = []pairingRequest{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allRequests)
}

func handlePairingApprove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Channel string `json:"channel"`
		ID      string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Channel == "" || req.ID == "" {
		writeError(w, http.StatusBadRequest, "channel and id are required")
		return
	}

	out, err := runCLI("pairing", "approve", req.Channel, req.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("approve failed: %v: %s", err, string(out)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

func handlePairingDeny(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Channel string `json:"channel"`
		ID      string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Channel == "" || req.ID == "" {
		writeError(w, http.StatusBadRequest, "channel and id are required")
		return
	}

	out, err := runCLI("pairing", "deny", req.Channel, req.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deny failed: %v: %s", err, string(out)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "denied"})
}

func runCLI(args ...string) ([]byte, error) {
	// Sanitize args: reject anything that looks like shell injection
	for _, arg := range args {
		if strings.ContainsAny(arg, ";|&`$(){}[]\\'\"\n\r") {
			return nil, fmt.Errorf("invalid argument: %q", arg)
		}
	}

	cmd := exec.Command(openclawBin, args...)
	cmd.Env = cmdEnv
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}
	return out, nil
}
