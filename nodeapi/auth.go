package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

const maxTimestampAge = 5 * time.Minute

func requireAuth(pubKey *ecdsa.PublicKey, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-Signature")
		ts := r.Header.Get("X-Signature-Timestamp")
		digest := r.Header.Get("X-Content-Digest")

		if sig == "" || ts == "" {
			writeError(w, http.StatusUnauthorized, "missing signature headers")
			return
		}

		// Verify timestamp freshness
		tsUnix, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid timestamp")
			return
		}
		age := time.Since(time.Unix(tsUnix, 0))
		if age < 0 {
			age = -age
		}
		if age > maxTimestampAge {
			writeError(w, http.StatusUnauthorized, "timestamp expired")
			return
		}

		// Reconstruct signing string: METHOD\nPATH\nTIMESTAMP\nDIGEST
		signingString := fmt.Sprintf("%s\n%s\n%s\n%s", r.Method, r.URL.Path, ts, digest)
		hash := sha256.Sum256([]byte(signingString))

		// Decode base64 signature (64 bytes raw IEEE P1363: r||s, 32 bytes each)
		sigBytes, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid signature encoding")
			return
		}

		byteLen := (elliptic.P256().Params().BitSize + 7) / 8 // 32
		if len(sigBytes) != 2*byteLen {
			writeError(w, http.StatusUnauthorized, "invalid signature length")
			return
		}

		rInt := new(big.Int).SetBytes(sigBytes[:byteLen])
		sInt := new(big.Int).SetBytes(sigBytes[byteLen:])

		if !ecdsa.Verify(pubKey, hash[:], rInt, sInt) {
			writeError(w, http.StatusUnauthorized, "invalid signature")
			return
		}

		next(w, r)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
