package main

import (
	"log/slog"
	"net/http"
	"os"
	"slices"
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

	hetzner := NewHetznerClient(cfg)
	provisioner := NewProvisioner(cfg)
	hub := NewLogHub()

	srv := NewServer(hetzner, provisioner, store, hub, cfg.APIUsername, cfg.APIPassword)

	slog.Info("starting server", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, srv.Router()); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
