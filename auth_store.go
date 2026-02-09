package main

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"
)

// User operations

func (s *Store) CreateUser(address, publicKey string) (*User, error) {
	// Check if this is the first user
	count, err := s.CountUsers()
	if err != nil {
		return nil, err
	}

	role := "user"
	approved := false
	if count == 0 {
		role = "admin"
		approved = true
	}

	var user User
	err = s.db.QueryRow(`
		INSERT INTO users (address, public_key, role, approved)
		VALUES ($1, $2, $3, $4)
		RETURNING id, address, public_key, role, approved, ssh_public_key, created_at
	`, address, publicKey, role, approved).Scan(
		&user.ID, &user.Address, &user.PublicKey, &user.Role, &user.Approved, &user.SSHPublicKey, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// If first user, backfill any existing servers
	if count == 0 {
		s.BackfillFirstAdmin()
	}

	return &user, nil
}

func (s *Store) GetUserByAddress(address string) (*User, error) {
	var user User
	err := s.db.QueryRow(`
		SELECT id, address, public_key, role, approved, ssh_public_key, created_at
		FROM users WHERE address=$1
	`, address).Scan(&user.ID, &user.Address, &user.PublicKey, &user.Role, &user.Approved, &user.SSHPublicKey, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByID(id int64) (*User, error) {
	var user User
	err := s.db.QueryRow(`
		SELECT id, address, public_key, role, approved, ssh_public_key, created_at
		FROM users WHERE id=$1
	`, id).Scan(&user.ID, &user.Address, &user.PublicKey, &user.Role, &user.Approved, &user.SSHPublicKey, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) ListUsers() ([]*User, error) {
	rows, err := s.db.Query(`
		SELECT id, address, public_key, role, approved, ssh_public_key, created_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Address, &user.PublicKey, &user.Role, &user.Approved, &user.SSHPublicKey, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

func (s *Store) SetUserSSHKey(userID int64, sshPublicKey string) error {
	_, err := s.db.Exec(`UPDATE users SET ssh_public_key=$1 WHERE id=$2`, sshPublicKey, userID)
	return err
}

func (s *Store) ApproveUser(id int64) error {
	_, err := s.db.Exec(`UPDATE users SET approved=true WHERE id=$1`, id)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id=$1`, id)
	return err
}

func (s *Store) CountUsers() (int64, error) {
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// Session operations

func (s *Store) CreateSession(userID int64) (*Session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	sessionID := hex.EncodeToString(b)
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	var session Session
	err := s.db.QueryRow(`
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, created_at, expires_at
	`, sessionID, userID, expiresAt).Scan(&session.ID, &session.UserID, &session.CreatedAt, &session.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Store) GetSession(sessionID string) (*Session, error) {
	var session Session
	err := s.db.QueryRow(`
		SELECT id, user_id, created_at, expires_at
		FROM sessions WHERE id=$1 AND expires_at > now()
	`, sessionID).Scan(&session.ID, &session.UserID, &session.CreatedAt, &session.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Store) DeleteSession(sessionID string) {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id=$1`, sessionID)
	if err != nil {
		slog.Error("failed to delete session", "error", err)
	}
}

func (s *Store) CleanExpiredSessions() {
	result, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at < now()`)
	if err != nil {
		slog.Error("failed to clean expired sessions", "error", err)
		return
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		slog.Info("cleaned expired sessions", "count", n)
	}
}
