package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type HetznerClient struct {
	client *hcloud.Client
	cfg    *Config
}

func NewHetznerClient(cfg *Config) *HetznerClient {
	return &HetznerClient{
		client: hcloud.NewClient(hcloud.WithToken(cfg.HCloudToken)),
		cfg:    cfg,
	}
}

func (h *HetznerClient) CreateServer(ctx context.Context, name string) (*ServerInfo, error) {
	slog.Info("creating server", "name", name, "type", h.cfg.ServerType, "image", h.cfg.Image, "location", h.cfg.Location)

	result, _, err := h.client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name: name,
		ServerType: &hcloud.ServerType{
			Name: h.cfg.ServerType,
		},
		Image: &hcloud.Image{
			Name: h.cfg.Image,
		},
		Location: &hcloud.Location{
			Name: h.cfg.Location,
		},
		SSHKeys: []*hcloud.SSHKey{
			{ID: h.cfg.SSHKeyID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}

	slog.Info("waiting for server action to complete", "server_id", result.Server.ID)
	if err := h.client.Action.WaitFor(ctx, result.Action); err != nil {
		return nil, fmt.Errorf("wait for server creation: %w", err)
	}

	// Re-fetch to get the assigned IP
	server, _, err := h.client.Server.GetByID(ctx, result.Server.ID)
	if err != nil {
		return nil, fmt.Errorf("get server by id: %w", err)
	}

	ipv4 := ""
	if server.PublicNet.IPv4.IP != nil {
		ipv4 = server.PublicNet.IPv4.IP.String()
	}

	slog.Info("server created", "id", server.ID, "name", server.Name, "ipv4", ipv4)

	return &ServerInfo{
		ID:     server.ID,
		Name:   server.Name,
		IPv4:   ipv4,
		Status: "provisioning",
	}, nil
}

func (h *HetznerClient) DeleteServer(ctx context.Context, id int64) error {
	slog.Info("deleting server", "id", id)

	server := &hcloud.Server{ID: id}
	_, _, err := h.client.Server.DeleteWithResult(ctx, server)
	if err != nil {
		return fmt.Errorf("delete server: %w", err)
	}

	slog.Info("server deleted", "id", id)
	return nil
}
