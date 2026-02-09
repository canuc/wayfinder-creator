package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type contextKey string

const userContextKey contextKey = "user"

func userFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(userContextKey).(*User)
	return u
}

// ChallengeStore holds pending challenges in memory with expiration.
type ChallengeStore struct {
	mu         sync.Mutex
	challenges map[string]challengeEntry
}

type challengeEntry struct {
	address   string
	expiresAt time.Time
}

func NewChallengeStore() *ChallengeStore {
	return &ChallengeStore{
		challenges: make(map[string]challengeEntry),
	}
}

func (cs *ChallengeStore) Create(address string) string {
	b := make([]byte, 32)
	rand.Read(b)
	nonce := hex.EncodeToString(b)
	challenge := fmt.Sprintf("Sign in to openclaw creator\n\nNonce: %s", nonce)

	cs.mu.Lock()
	cs.challenges[challenge] = challengeEntry{
		address:   address,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	cs.mu.Unlock()

	return challenge
}

func (cs *ChallengeStore) Consume(challenge, address string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entry, ok := cs.challenges[challenge]
	if !ok {
		return false
	}
	delete(cs.challenges, challenge)

	if time.Now().After(entry.expiresAt) {
		return false
	}
	return entry.address == address
}

func (cs *ChallengeStore) Cleanup() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	now := time.Now()
	for k, v := range cs.challenges {
		if now.After(v.expiresAt) {
			delete(cs.challenges, k)
		}
	}
}

// verifyEthSignature recovers the Ethereum address from an EIP-191 personal_sign signature.
func verifyEthSignature(challenge, signatureHex string) (common.Address, error) {
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(signatureHex, "0x"))
	if err != nil {
		return common.Address{}, fmt.Errorf("decode signature: %w", err)
	}
	if len(sigBytes) != 65 {
		return common.Address{}, fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	// Normalize v: MetaMask returns v=27/28, ecrecover needs v=0/1
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	// EIP-191 prefix
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(challenge), challenge)
	hash := ethcrypto.Keccak256Hash([]byte(msg))

	pubBytes, err := ethcrypto.Ecrecover(hash.Bytes(), sigBytes)
	if err != nil {
		return common.Address{}, fmt.Errorf("ecrecover: %w", err)
	}

	pubKey, err := ethcrypto.UnmarshalPubkey(pubBytes)
	if err != nil {
		return common.Address{}, fmt.Errorf("unmarshal pubkey: %w", err)
	}

	return ethcrypto.PubkeyToAddress(*pubKey), nil
}

var addressRegex = regexp.MustCompile(`^0x[0-9a-f]{40}$`)

// sessionAuth extracts the user from the session cookie and adds it to context.
// Returns nil user if not authenticated (does NOT write error response).
func (s *Server) sessionAuth(r *http.Request) (*User, *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil, r
	}

	session, err := s.store.GetSession(cookie.Value)
	if err != nil {
		return nil, r
	}

	user, err := s.store.GetUserByID(session.UserID)
	if err != nil {
		return nil, r
	}

	ctx := context.WithValue(r.Context(), userContextKey, user)
	return user, r.WithContext(ctx)
}

// requireApproved wraps a handler to require an authenticated and approved user.
func (s *Server) requireApproved(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, r := s.sessionAuth(r)
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			return
		}
		if !user.Approved {
			writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "account not approved"})
			return
		}
		handler(w, r)
	}
}

// requireAdmin wraps a handler to require an admin user.
func (s *Server) requireAdmin(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, r := s.sessionAuth(r)
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			return
		}
		if user.Role != "admin" {
			writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "admin required"})
			return
		}
		handler(w, r)
	}
}

// Auth handlers

func (s *Server) handleChallenge(w http.ResponseWriter, r *http.Request) {
	var req ChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Address = strings.ToLower(req.Address)
	if !addressRegex.MatchString(req.Address) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid address format"})
		return
	}

	challenge := s.challenges.Create(req.Address)
	writeJSON(w, http.StatusOK, ChallengeResponse{Challenge: challenge})
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Address = strings.ToLower(req.Address)
	if !addressRegex.MatchString(req.Address) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid address format"})
		return
	}

	if req.Signature == "" || req.Challenge == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "signature and challenge are required"})
		return
	}

	// Verify challenge is valid and matches address
	if !s.challenges.Consume(req.Challenge, req.Address) {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid or expired challenge"})
		return
	}

	// Recover address from signature via ecrecover
	recovered, err := verifyEthSignature(req.Challenge, req.Signature)
	if err != nil {
		slog.Warn("signature verification failed", "address", req.Address, "error", err)
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid signature"})
		return
	}
	if strings.ToLower(recovered.Hex()) != req.Address {
		slog.Warn("recovered address mismatch", "expected", req.Address, "recovered", recovered.Hex())
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "signature does not match address"})
		return
	}

	// Look up or auto-create user
	user, err := s.store.GetUserByAddress(req.Address)
	if err == sql.ErrNoRows {
		user, err = s.store.CreateUser(req.Address, "")
		if err != nil {
			slog.Error("failed to create user", "address", req.Address, "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
			return
		}
		slog.Info("auto-created user", "address", req.Address, "role", user.Role)
	} else if err != nil {
		slog.Error("failed to look up user", "address", req.Address, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}

	// Create session
	session, err := s.store.CreateSession(user.ID)
	if err != nil {
		slog.Error("create session failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60,
	})

	writeJSON(w, http.StatusOK, AuthResponse{User: user})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		s.store.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, r := s.sessionAuth(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, AuthResponse{User: user})
}

func (s *Server) handleSetSSHKey(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	var req struct {
		SSHPublicKey string `json:"ssh_public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := s.store.SetUserSSHKey(user.ID, req.SSHPublicKey); err != nil {
		slog.Error("failed to set SSH key", "user_id", user.ID, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to save SSH key"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Admin handlers

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		slog.Error("failed to list users", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list users"})
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleApproveUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid user id"})
		return
	}

	if err := s.store.ApproveUser(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to approve user"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid user id"})
		return
	}

	// Don't allow deleting yourself
	user := userFromContext(r.Context())
	if user != nil && user.ID == id {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "cannot delete yourself"})
		return
	}

	if err := s.store.DeleteUser(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete user"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
