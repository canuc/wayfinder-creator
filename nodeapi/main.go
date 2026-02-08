package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	listen := flag.String("listen", ":8443", "listen address")
	pubkeyPath := flag.String("pubkey", "/home/clawdbot/.clawdbot/creator-public-key.pem", "path to creator ECDSA public key PEM")
	flag.Parse()

	pubKey, err := loadPublicKey(*pubkeyPath)
	if err != nil {
		log.Fatalf("failed to load public key from %s: %v", *pubkeyPath, err)
	}
	log.Printf("loaded creator public key from %s", *pubkeyPath)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /pairing/requests", requireAuth(pubKey, handlePairingRequests))
	mux.HandleFunc("POST /pairing/approve", requireAuth(pubKey, handlePairingApprove))
	mux.HandleFunc("POST /pairing/deny", requireAuth(pubKey, handlePairingDeny))
	mux.HandleFunc("GET /channels/status", requireAuth(pubKey, handleChannelsStatus))

	log.Printf("starting node API on %s", *listen)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func loadPublicKey(path string) (*ecdsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key")
	}

	return ecdsaPub, nil
}
