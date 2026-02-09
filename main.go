package main

import (
	"log/slog"
	"net/http"
	"os"
	"slices"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	store, err := NewStore(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := store.Migrate(); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// fly.toml release_command: openclaw-creator --migrate
	if slices.Contains(os.Args[1:], "--migrate") {
		slog.Info("migrations complete")
		return
	}

	store.FailStaleProvisioningServers()

	// Periodically clean expired sessions
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			store.CleanExpiredSessions()
		}
	}()

	providers := make(map[string]VPSProvider)
	if cfg.HCloudToken != "" {
		h := NewHetznerClient(cfg)
		providers[h.Name()] = h
		slog.Info("registered provider", "name", h.Name())
	}
	if cfg.VultrAPIKey != "" {
		v := NewVultrClient(cfg)
		providers[v.Name()] = v
		slog.Info("registered provider", "name", v.Name())
	}
	provisioner := NewProvisioner(cfg)
	hub := NewLogHub()

	srv := NewServer(cfg, providers, provisioner, store, hub)

	// Periodically clean expired challenges
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			srv.challenges.Cleanup()
		}
	}()

	slog.Info("starting server", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, srv.Router()); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
