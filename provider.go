package main

import "context"

// VPSProvider abstracts a cloud VPS provider (Hetzner, Vultr, etc.).
type VPSProvider interface {
	Name() string
	CreateServer(ctx context.Context, name string) (*ServerInfo, error)
	DeleteServer(ctx context.Context, providerID string) error
}
