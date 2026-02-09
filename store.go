package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Store struct {
	db *sql.DB
}

func NewStore(databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS servers (
			id BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			ipv4 TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'provisioning',
			provisioned BOOLEAN NOT NULL DEFAULT false,
			wallet_address TEXT NOT NULL DEFAULT '',
			default_key_removed BOOLEAN NOT NULL DEFAULT false,
			ssh_public_key TEXT NOT NULL DEFAULT '',
			anthropic_api_key TEXT NOT NULL DEFAULT '',
			openai_api_key TEXT NOT NULL DEFAULT '',
			gemini_api_key TEXT NOT NULL DEFAULT '',
			wayfinder_api_key TEXT NOT NULL DEFAULT '',
			channels JSONB NOT NULL DEFAULT '[]',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS server_logs (
			id BIGSERIAL PRIMARY KEY,
			server_id BIGINT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
			line TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_server_logs_server_id ON server_logs(server_id);
		ALTER TABLE servers ADD COLUMN IF NOT EXISTS public_key TEXT NOT NULL DEFAULT '';

		CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			approved BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at TIMESTAMPTZ NOT NULL
		);

		ALTER TABLE servers ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id);

		ALTER TABLE users ADD COLUMN IF NOT EXISTS address TEXT NOT NULL DEFAULT '';
		ALTER TABLE users ADD COLUMN IF NOT EXISTS public_key TEXT NOT NULL DEFAULT '';
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_address ON users(address);

		-- Drop legacy email/password auth constraints
		ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
		ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
		ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;

		ALTER TABLE users ADD COLUMN IF NOT EXISTS ssh_public_key TEXT NOT NULL DEFAULT '';
	`)
	return err
}

func (s *Store) FailStaleProvisioningServers() {
	result, err := s.db.Exec(`UPDATE servers SET status='failed' WHERE status='provisioning'`)
	if err != nil {
		slog.Error("failed to mark stale servers as failed", "error", err)
		return
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		slog.Info("marked stale provisioning servers as failed", "count", n)
		// Insert a log line for each affected server
		s.db.Exec(`
			INSERT INTO server_logs (server_id, line)
			SELECT id, 'Provisioning interrupted by server restart'
			FROM servers WHERE status='failed' AND id NOT IN (
				SELECT DISTINCT server_id FROM server_logs WHERE line='Provisioning interrupted by server restart'
			)
		`)
	}
}

func (s *Store) CreateServer(info *ServerInfo, opts ProvisionOpts, userID int64) error {
	channelsJSON, err := json.Marshal(opts.Channels)
	if err != nil {
		channelsJSON = []byte("[]")
	}
	_, err = s.db.Exec(`
		INSERT INTO servers (id, name, ipv4, status, provisioned, ssh_public_key, anthropic_api_key, openai_api_key, gemini_api_key, wayfinder_api_key, channels, public_key, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, info.ID, info.Name, info.IPv4, info.Status, info.Provisioned,
		opts.SSHPublicKey, opts.AnthropicAPIKey, opts.OpenAIAPIKey, opts.GeminiAPIKey, opts.WayfinderAPIKey, channelsJSON, opts.CreatorPublicKey, userID)
	return err
}

func (s *Store) GetServer(id, userID int64) (*ServerInfo, error) {
	var info ServerInfo
	var channelsJSON []byte
	err := s.db.QueryRow(`
		SELECT id, name, ipv4, status, provisioned, wallet_address, default_key_removed,
		       (public_key != '') AS has_node_api, created_at, channels
		FROM servers WHERE id=$1 AND user_id=$2
	`, id, userID).Scan(&info.ID, &info.Name, &info.IPv4, &info.Status, &info.Provisioned,
		&info.WalletAddress, &info.DefaultKeyRemoved, &info.HasNodeAPI, &info.CreatedAt, &channelsJSON)
	if err != nil {
		return nil, err
	}
	if len(channelsJSON) > 0 {
		var ch []any
		if json.Unmarshal(channelsJSON, &ch) == nil {
			info.ChannelCount = len(ch)
		}
	}
	return &info, nil
}

// GetServerAny retrieves a server without user scoping (for WebSocket, internal use)
func (s *Store) GetServerAny(id int64) (*ServerInfo, error) {
	var info ServerInfo
	err := s.db.QueryRow(`
		SELECT id, name, ipv4, status, provisioned, wallet_address, default_key_removed, (public_key != '') AS has_node_api
		FROM servers WHERE id=$1
	`, id).Scan(&info.ID, &info.Name, &info.IPv4, &info.Status, &info.Provisioned, &info.WalletAddress, &info.DefaultKeyRemoved, &info.HasNodeAPI)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *Store) ListServers(userID int64) ([]*ServerInfo, error) {
	rows, err := s.db.Query(`
		SELECT id, name, ipv4, status, provisioned, wallet_address, default_key_removed,
		       (public_key != '') AS has_node_api, created_at, channels
		FROM servers WHERE user_id=$1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*ServerInfo
	for rows.Next() {
		var info ServerInfo
		var channelsJSON []byte
		if err := rows.Scan(&info.ID, &info.Name, &info.IPv4, &info.Status, &info.Provisioned,
			&info.WalletAddress, &info.DefaultKeyRemoved, &info.HasNodeAPI, &info.CreatedAt, &channelsJSON); err != nil {
			return nil, err
		}
		if len(channelsJSON) > 0 {
			var ch []any
			if json.Unmarshal(channelsJSON, &ch) == nil {
				info.ChannelCount = len(ch)
			}
		}
		servers = append(servers, &info)
	}
	return servers, rows.Err()
}

func (s *Store) UpdateStatus(id int64, status string, provisioned bool) {
	_, err := s.db.Exec(`UPDATE servers SET status=$1, provisioned=$2 WHERE id=$3`, status, provisioned, id)
	if err != nil {
		slog.Error("failed to update server status", "server_id", id, "error", err)
	}
}

func (s *Store) SetWalletAddress(id int64, addr string) {
	_, err := s.db.Exec(`UPDATE servers SET wallet_address=$1 WHERE id=$2`, addr, id)
	if err != nil {
		slog.Error("failed to set wallet address", "server_id", id, "error", err)
	}
}

func (s *Store) SetDefaultKeyRemoved(id int64, removed bool) {
	_, err := s.db.Exec(`UPDATE servers SET default_key_removed=$1 WHERE id=$2`, removed, id)
	if err != nil {
		slog.Error("failed to set default_key_removed", "server_id", id, "error", err)
	}
}

func (s *Store) SetPublicKey(id int64, pem string) error {
	_, err := s.db.Exec(`UPDATE servers SET public_key=$1 WHERE id=$2`, pem, id)
	if err != nil {
		slog.Error("failed to set public_key", "server_id", id, "error", err)
	}
	return err
}

func (s *Store) GetPublicKey(id int64) (string, error) {
	var key string
	err := s.db.QueryRow(`SELECT public_key FROM servers WHERE id=$1`, id).Scan(&key)
	return key, err
}

func (s *Store) AppendLog(id int64, line string) error {
	_, err := s.db.Exec(`INSERT INTO server_logs (server_id, line) VALUES ($1, $2)`, id, line)
	if err != nil {
		slog.Error("failed to append log", "server_id", id, "error", err)
	}
	return err
}

func (s *Store) GetLogsSince(serverID, afterID int64) ([]LogEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, line FROM server_logs
		WHERE server_id=$1 AND id>$2
		ORDER BY id
	`, serverID, afterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var entry LogEntry
		if err := rows.Scan(&entry.ID, &entry.Line); err != nil {
			return nil, err
		}
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}

func (s *Store) ClearLogs(id int64) {
	_, err := s.db.Exec(`DELETE FROM server_logs WHERE server_id=$1`, id)
	if err != nil {
		slog.Error("failed to clear logs", "server_id", id, "error", err)
	}
}

func (s *Store) DeleteServer(id, userID int64) error {
	result, err := s.db.Exec(`DELETE FROM servers WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server not found")
	}
	return nil
}

// ClearChannelTokens strips token fields from the channels JSONB, keeping type/name/account.
func (s *Store) ClearChannelTokens(id int64) {
	_, err := s.db.Exec(`
		UPDATE servers SET channels = (
			SELECT COALESCE(jsonb_agg(ch - 'token'), '[]'::jsonb)
			FROM jsonb_array_elements(channels) AS ch
		) WHERE id=$1
	`, id)
	if err != nil {
		slog.Error("failed to clear channel tokens", "server_id", id, "error", err)
	}
}

// BackfillFirstAdmin assigns all servers without a user_id to the first admin user
func (s *Store) BackfillFirstAdmin() {
	var adminID int64
	err := s.db.QueryRow(`SELECT id FROM users WHERE role='admin' ORDER BY id LIMIT 1`).Scan(&adminID)
	if err != nil {
		return
	}
	result, err := s.db.Exec(`UPDATE servers SET user_id=$1 WHERE user_id IS NULL`, adminID)
	if err != nil {
		slog.Error("failed to backfill server user_id", "error", err)
		return
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		slog.Info("backfilled servers to first admin", "count", n, "admin_id", adminID)
	}
}
