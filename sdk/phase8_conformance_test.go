package license

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type signatureVector struct {
	Name         string         `json:"name"`
	Timestamp    int64          `json:"timestamp"`
	Payload      map[string]any `json:"payload"`
	PublicKeyHex string         `json:"public_key_hex"`
	SignatureHex string         `json:"signature_hex"`
}

func loadVector(t *testing.T, file string) signatureVector {
	t.Helper()
	path := filepath.Join("..", "fixtures", "signature", file)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read vector %s: %v", file, err)
	}
	var v signatureVector
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("failed to decode vector %s: %v", file, err)
	}
	return v
}

func TestPhase8Conformance_ActivateVector(t *testing.T) {
	v := loadVector(t, "activate.json")
	pk, err := hex.DecodeString(v.PublicKeyHex)
	if err != nil {
		t.Fatalf("invalid public key hex: %v", err)
	}

	if !verifySignature(v.Payload, v.Timestamp, v.SignatureHex, ed25519.PublicKey(pk)) {
		t.Fatalf("vector %s must verify", v.Name)
	}

	// Negative check: payload tampering must fail.
	v.Payload["interval"] = 31
	if verifySignature(v.Payload, v.Timestamp, v.SignatureHex, ed25519.PublicKey(pk)) {
		t.Fatalf("tampered vector %s must fail", v.Name)
	}
}

func TestPhase8Conformance_HeartbeatVector(t *testing.T) {
	v := loadVector(t, "heartbeat.json")
	pk, err := hex.DecodeString(v.PublicKeyHex)
	if err != nil {
		t.Fatalf("invalid public key hex: %v", err)
	}

	if !verifySignature(v.Payload, v.Timestamp, v.SignatureHex, ed25519.PublicKey(pk)) {
		t.Fatalf("vector %s must verify", v.Name)
	}

	// Negative check: timestamp tampering must fail.
	if verifySignature(v.Payload, v.Timestamp+1, v.SignatureHex, ed25519.PublicKey(pk)) {
		t.Fatalf("tampered timestamp for %s must fail", v.Name)
	}
}
