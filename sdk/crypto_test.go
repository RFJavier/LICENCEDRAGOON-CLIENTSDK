package license

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func testKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func signTestPayload(payload map[string]any, ts int64, priv ed25519.PrivateKey) string {
	body := canonicalPayload(payload)
	msg := []byte(fmt.Sprintf("%d.%s", ts, body))
	sig := ed25519.Sign(priv, msg)
	return hex.EncodeToString(sig)
}

func TestVerifySignature_Valid(t *testing.T) {
	pub, priv := testKeyPair(t)
	ts := time.Now().Unix()
	payload := map[string]any{"device_id": "abc", "interval": 30}

	sig := signTestPayload(payload, ts, priv)
	if !verifySignature(payload, ts, sig, pub) {
		t.Error("valid signature should verify")
	}
}

func TestVerifySignature_TamperedPayload(t *testing.T) {
	pub, priv := testKeyPair(t)
	ts := time.Now().Unix()
	original := map[string]any{"device_id": "abc", "interval": 30}
	tampered := map[string]any{"device_id": "xyz", "interval": 30}

	sig := signTestPayload(original, ts, priv)
	if verifySignature(tampered, ts, sig, pub) {
		t.Error("tampered payload should not verify")
	}
}

func TestVerifySignature_WrongTimestamp(t *testing.T) {
	pub, priv := testKeyPair(t)
	ts := time.Now().Unix()
	payload := map[string]any{"status": "active"}

	sig := signTestPayload(payload, ts, priv)
	if verifySignature(payload, ts+100, sig, pub) {
		t.Error("wrong timestamp should not verify")
	}
}

func TestVerifySignature_WrongKey(t *testing.T) {
	_, priv := testKeyPair(t)
	otherPub, _ := testKeyPair(t)
	ts := time.Now().Unix()
	payload := map[string]any{"test": true}

	sig := signTestPayload(payload, ts, priv)
	if verifySignature(payload, ts, sig, otherPub) {
		t.Error("wrong public key should not verify")
	}
}

func TestVerifySignature_InvalidHex(t *testing.T) {
	pub, _ := testKeyPair(t)
	if verifySignature(map[string]any{}, 1234, "not-hex!", pub) {
		t.Error("invalid hex should not verify")
	}
}

func TestVerifySignature_EmptySignature(t *testing.T) {
	pub, _ := testKeyPair(t)
	if verifySignature(map[string]any{}, 1234, "", pub) {
		t.Error("empty signature should not verify")
	}
}

func TestCanonicalPayload_Deterministic(t *testing.T) {
	p := map[string]any{"b": 2, "a": 1}
	a := canonicalPayload(p)
	b := canonicalPayload(p)
	if a != b {
		t.Errorf("canonical payload should be deterministic: %q != %q", a, b)
	}
}
