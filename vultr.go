package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
)

type VultrClient struct {
	client *govultr.Client
	cfg    *Config
}

func NewVultrClient(cfg *Config) *VultrClient {
	tokenSrc := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.VultrAPIKey})
	httpClient := oauth2.NewClient(context.Background(), tokenSrc)
	client := govultr.NewClient(httpClient)
	return &VultrClient{client: client, cfg: cfg}
}

func (v *VultrClient) Name() string { return "vultr" }

func (v *VultrClient) CreateServer(ctx context.Context, name string) (*ServerInfo, error) {
	slog.Info("creating vultr instance", "name", name, "plan", v.cfg.VultrPlan, "region", v.cfg.VultrRegion)

	opts := &govultr.InstanceCreateReq{
		Label:    name,
		Region:   v.cfg.VultrRegion,
		Plan:     v.cfg.VultrPlan,
		OsID:     v.cfg.VultrOSID,
		SSHKeys:  []string{v.cfg.VultrSSHKeyID},
		Hostname: name,
	}

	instance, _, err := v.client.Instance.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("vultr create instance: %w", err)
	}

	slog.Info("vultr instance created, waiting for active status", "id", instance.ID)

	// Poll until instance is active with a real IP
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("vultr instance %s did not become active within 5 minutes", instance.ID)
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			inst, _, err := v.client.Instance.Get(ctx, instance.ID)
			if err != nil {
				slog.Warn("vultr poll error", "id", instance.ID, "error", err)
				continue
			}
			if inst.Status == "active" && inst.MainIP != "" && inst.MainIP != "0.0.0.0" {
				slog.Info("vultr instance active", "id", inst.ID, "ip", inst.MainIP)
				return &ServerInfo{
					ProviderID: inst.ID,
					Name:       inst.Label,
					IPv4:       inst.MainIP,
					Status:     "provisioning",
				}, nil
			}
			slog.Debug("vultr instance not ready yet", "id", inst.ID, "status", inst.Status, "ip", inst.MainIP)
		}
	}
}

func (v *VultrClient) DeleteServer(ctx context.Context, providerID string) error {
	slog.Info("deleting vultr instance", "provider_id", providerID)

	if err := v.client.Instance.Delete(ctx, providerID); err != nil {
		return fmt.Errorf("vultr delete instance: %w", err)
	}

	slog.Info("vultr instance deleted", "provider_id", providerID)
	return nil
}
