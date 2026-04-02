package license

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func newTestSDK(t *testing.T, serverURL string, pub ed25519.PublicKey) *SDK {
	t.Helper()
	sdk, err := New(Config{
		APIURL:       serverURL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		StoragePath:  t.TempDir() + "/state.json",
		PublicKey:    hex.EncodeToString(pub),
		Interval:     100 * time.Millisecond,
		Timeout:      2 * time.Second,
		GracePeriod:  1 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return sdk
}

func makeHeartbeatServer(t *testing.T, priv ed25519.PrivateKey, status, action string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts := time.Now().Unix()
		payload := map[string]any{
			"status":   status,
			"action":   action,
			"interval": 30,
		}
		body := canonicalPayload(payload)
		msg := []byte(fmt.Sprintf("%d.%s", ts, body))
		sig := ed25519.Sign(priv, msg)

		resp := map[string]any{
			"status":    status,
			"action":    action,
			"interval":  30,
			"timestamp": ts,
			"signature": hex.EncodeToString(sig),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestRunHeartbeat_Success(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	server := makeHeartbeatServer(t, priv, "active", "continue")
	defer server.Close()

	sdk := newTestSDK(t, server.URL, pub)

	// Pre-populate storage (simulate activation)
	sdk.storage.Save(&State{
		LicenseKey: "LIC-TEST",
		DeviceID:   "dev-001",
		ValidUntil: time.Now().UTC().Add(10 * time.Minute),
	})

	err := sdk.runHeartbeat(context.Background())
	if err != nil {
		t.Fatalf("runHeartbeat failed: %v", err)
	}

	state, _ := sdk.storage.Load()
	if state.LastValidatedAt.IsZero() {
		t.Error("LastValidatedAt should be updated")
	}
}

func TestRunHeartbeat_NotActivated(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	server := makeHeartbeatServer(t, priv, "active", "continue")
	defer server.Close()

	sdk := newTestSDK(t, server.URL, pub)
	// Don't save any state - SDK not activated

	err := sdk.runHeartbeat(context.Background())
	if err == nil {
		t.Error("expected error when SDK not activated")
	}
}

func TestRunHeartbeat_Blocked(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	server := makeHeartbeatServer(t, priv, "blocked", "stop")
	defer server.Close()

	sdk := newTestSDK(t, server.URL, pub)
	sdk.storage.Save(&State{
		LicenseKey: "LIC-TEST",
		DeviceID:   "dev-001",
	})

	var blocked bool
	sdk.OnBlocked(func() { blocked = true })

	err := sdk.runHeartbeat(context.Background())
	if err == nil {
		t.Error("expected error for blocked license")
	}
	if !blocked {
		t.Error("onBlocked hook should have been called")
	}
}

func TestRunHeartbeat_InvalidSignature(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	_, wrongPriv, _ := ed25519.GenerateKey(nil)
	server := makeHeartbeatServer(t, wrongPriv, "active", "continue")
	defer server.Close()

	sdk := newTestSDK(t, server.URL, pub)
	sdk.storage.Save(&State{
		LicenseKey: "LIC-TEST",
		DeviceID:   "dev-001",
	})

	err := sdk.runHeartbeat(context.Background())
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestStartHeartbeat_CancelsOnContext(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	server := makeHeartbeatServer(t, priv, "active", "continue")
	defer server.Close()

	sdk := newTestSDK(t, server.URL, pub)
	sdk.storage.Save(&State{
		LicenseKey: "LIC-TEST",
		DeviceID:   "dev-001",
		ValidUntil: time.Now().UTC().Add(10 * time.Minute),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sdk.StartHeartbeat(ctx)
		close(done)
	}()

	select {
	case <-done:
		// OK, heartbeat loop ended
	case <-time.After(2 * time.Second):
		t.Error("StartHeartbeat did not stop on context cancel")
	}
}

func TestHandleOfflineFallback_GracePeriod(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	sdk := newTestSDK(t, "http://localhost:1", pub)

	// Set valid state with future ValidUntil
	sdk.storage.Save(&State{
		LicenseKey: "LIC-TEST",
		DeviceID:   "dev-001",
		ValidUntil: time.Now().UTC().Add(5 * time.Minute),
	})

	var graceCalled bool
	sdk.OnGracePeriodStart(func() { graceCalled = true })

	sdk.handleOfflineFallback()
	if !sdk.graceOn {
		t.Error("grace should be enabled")
	}
	if !graceCalled {
		t.Error("onGracePeriodStart should have been called")
	}
}

func TestHandleOfflineFallback_Expired(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	sdk := newTestSDK(t, "http://localhost:1", pub)

	// Expired validity
	sdk.storage.Save(&State{
		LicenseKey: "LIC-TEST",
		DeviceID:   "dev-001",
		ValidUntil: time.Now().UTC().Add(-1 * time.Minute),
	})

	var blocked bool
	sdk.OnBlocked(func() { blocked = true })

	sdk.handleOfflineFallback()
	if !blocked {
		t.Error("onBlocked should be called when grace period expired")
	}
}

func TestHooks_Registration(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	sdk := newTestSDK(t, "http://localhost:1", pub)

	var mu sync.Mutex
	calls := map[string]bool{}

	sdk.OnBlocked(func() {
		mu.Lock()
		calls["blocked"] = true
		mu.Unlock()
	})
	sdk.OnHeartbeatError(func(err error) {
		mu.Lock()
		calls["hb_error"] = true
		mu.Unlock()
	})
	sdk.OnGracePeriodStart(func() {
		mu.Lock()
		calls["grace"] = true
		mu.Unlock()
	})

	// Trigger hooks directly
	sdk.hooks.onBlocked()
	sdk.hooks.onHeartbeatError(fmt.Errorf("test"))
	sdk.hooks.onGracePeriodStart()

	mu.Lock()
	defer mu.Unlock()
	if !calls["blocked"] || !calls["hb_error"] || !calls["grace"] {
		t.Error("not all hooks were triggered")
	}
}
