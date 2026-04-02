package license

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPClient_PostSuccess(t *testing.T) {
	expected := map[string]any{"device_id": "abc-123", "interval": float64(30)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type header")
		}
		if r.Header.Get("X-CLIENT-ID") == "" {
			t.Error("missing X-CLIENT-ID header")
		}
		if r.Header.Get("X-CLIENT-SECRET") == "" {
			t.Error("missing X-CLIENT-SECRET header")
		}
		if r.Header.Get("X-TIMESTAMP") == "" {
			t.Error("missing X-TIMESTAMP header")
		}
		if r.Header.Get("X-NONCE") == "" {
			t.Error("missing X-NONCE header")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := newHTTPClient(Config{
		APIURL:       server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Timeout:      5 * time.Second,
		MaxRetries:   1,
	})

	var result map[string]any
	err := client.post(context.Background(), "/test", map[string]string{"key": "val"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["device_id"] != "abc-123" {
		t.Errorf("expected device_id abc-123, got %v", result["device_id"])
	}
}

func TestHTTPClient_Post4xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	client := newHTTPClient(Config{
		APIURL:       server.URL,
		ClientID:     "id",
		ClientSecret: "secret",
		Timeout:      5 * time.Second,
		MaxRetries:   0,
	})

	var result map[string]any
	err := client.post(context.Background(), "/test", nil, &result)
	if err == nil {
		t.Error("expected error for 400 response")
	}
}

func TestHTTPClient_Post5xxRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer server.Close()

	client := newHTTPClient(Config{
		APIURL:       server.URL,
		ClientID:     "id",
		ClientSecret: "secret",
		Timeout:      5 * time.Second,
		MaxRetries:   3,
	})

	var result map[string]any
	err := client.post(context.Background(), "/test", nil, &result)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempts)
	}
}

func TestHTTPClient_PostConnectionRefused(t *testing.T) {
	client := newHTTPClient(Config{
		APIURL:       "http://127.0.0.1:1", // unlikely to be open
		ClientID:     "id",
		ClientSecret: "secret",
		Timeout:      500 * time.Millisecond,
		MaxRetries:   0,
	})

	var result map[string]any
	err := client.post(context.Background(), "/test", nil, &result)
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestHTTPClient_NonceUniqueness(t *testing.T) {
	nonces := make(map[string]bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := r.Header.Get("X-NONCE")
		if nonces[nonce] {
			t.Error("duplicate nonce detected")
		}
		nonces[nonce] = true
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer server.Close()

	client := newHTTPClient(Config{
		APIURL:       server.URL,
		ClientID:     "id",
		ClientSecret: "secret",
		Timeout:      5 * time.Second,
		MaxRetries:   0,
	})

	for i := 0; i < 10; i++ {
		var result map[string]any
		client.post(context.Background(), "/test", nil, &result)
	}

	if len(nonces) != 10 {
		t.Errorf("expected 10 unique nonces, got %d", len(nonces))
	}
}

func TestIsRetriableNetErr(t *testing.T) {
	if isRetriableNetErr(fmt.Errorf("connection refused")) {
		// Basic non-net.Error with "connection refused" in message
	}
	if isRetriableNetErr(fmt.Errorf("some random error")) {
		t.Error("random errors should not be retriable")
	}
}

func TestSDK_Activate_Success(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts := time.Now().Unix()
		payload := map[string]any{
			"device_id": "dev-001",
			"interval":  30,
		}
		body := canonicalPayload(payload)
		msg := []byte(fmt.Sprintf("%d.%s", ts, body))
		sig := ed25519.Sign(priv, msg)

		resp := map[string]any{
			"device_id": "dev-001",
			"interval":  30,
			"timestamp": ts,
			"signature": hex.EncodeToString(sig),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	sdk, err := New(Config{
		APIURL:       server.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		StoragePath:  t.TempDir() + "/state.json",
		PublicKey:    hex.EncodeToString(pub),
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	resp, err := sdk.Activate("LIC-TEST-123")
	if err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	if resp.DeviceID != "dev-001" {
		t.Errorf("expected device_id dev-001, got %s", resp.DeviceID)
	}

	// Verify state was saved
	state, _ := sdk.storage.Load()
	if state.LicenseKey != "LIC-TEST-123" {
		t.Error("license key not saved")
	}
	if state.DeviceID != "dev-001" {
		t.Error("device_id not saved")
	}
}

func TestSDK_Activate_InvalidSignature(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	_, otherPriv, _ := ed25519.GenerateKey(nil) // different key

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts := time.Now().Unix()
		payload := map[string]any{"device_id": "dev-001", "interval": 30}
		body := canonicalPayload(payload)
		msg := []byte(fmt.Sprintf("%d.%s", ts, body))
		sig := ed25519.Sign(otherPriv, msg) // signed with wrong key

		resp := map[string]any{
			"device_id": "dev-001",
			"interval":  30,
			"timestamp": ts,
			"signature": hex.EncodeToString(sig),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	sdk, _ := New(Config{
		APIURL:       server.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		StoragePath:  t.TempDir() + "/state.json",
		PublicKey:    hex.EncodeToString(pub),
	})

	_, err := sdk.Activate("LIC-TEST-123")
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}
