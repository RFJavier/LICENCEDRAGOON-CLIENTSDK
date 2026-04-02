package license

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

func TestNew_ValidConfig(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	cfg := Config{
		APIURL:       "http://localhost:8080",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		StoragePath:  t.TempDir() + "/state.json",
		PublicKey:    hex.EncodeToString(pub),
	}

	sdk, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sdk == nil {
		t.Fatal("SDK should not be nil")
	}
}

func TestNew_MissingAPIURL(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	cfg := Config{
		ClientID:     "id",
		ClientSecret: "secret",
		StoragePath:  "/tmp/state.json",
		PublicKey:    hex.EncodeToString(pub),
	}
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for missing APIURL")
	}
}

func TestNew_MissingClientID(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	cfg := Config{
		APIURL:       "http://localhost",
		ClientSecret: "secret",
		StoragePath:  "/tmp/state.json",
		PublicKey:    hex.EncodeToString(pub),
	}
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for missing ClientID")
	}
}

func TestNew_MissingPublicKey(t *testing.T) {
	cfg := Config{
		APIURL:       "http://localhost",
		ClientID:     "id",
		ClientSecret: "secret",
		StoragePath:  "/tmp/state.json",
	}
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for missing PublicKey")
	}
}

func TestNew_InvalidPublicKey(t *testing.T) {
	cfg := Config{
		APIURL:       "http://localhost",
		ClientID:     "id",
		ClientSecret: "secret",
		StoragePath:  "/tmp/state.json",
		PublicKey:    "not-valid-hex",
	}
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for invalid PublicKey")
	}
}

func TestNew_WrongLengthPublicKey(t *testing.T) {
	cfg := Config{
		APIURL:       "http://localhost",
		ClientID:     "id",
		ClientSecret: "secret",
		StoragePath:  "/tmp/state.json",
		PublicKey:    hex.EncodeToString([]byte("tooshort")),
	}
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for wrong-length PublicKey")
	}
}

func TestNew_DefaultsApplied(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	cfg := Config{
		APIURL:       "http://localhost:8080",
		ClientID:     "id",
		ClientSecret: "secret",
		StoragePath:  t.TempDir() + "/state.json",
		PublicKey:    hex.EncodeToString(pub),
	}

	sdk, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sdk.cfg.Interval <= 0 {
		t.Error("Interval default should be applied")
	}
	if sdk.cfg.Timeout <= 0 {
		t.Error("Timeout default should be applied")
	}
	if sdk.cfg.GracePeriod <= 0 {
		t.Error("GracePeriod default should be applied")
	}
	if sdk.cfg.MaxRetries <= 0 {
		t.Error("MaxRetries default should be applied")
	}
}

func TestConfig_Normalize(t *testing.T) {
	c := Config{}
	n := c.normalize()

	if n.Interval <= 0 {
		t.Error("Interval should have a default")
	}
	if n.Timeout <= 0 {
		t.Error("Timeout should have a default")
	}
	if n.GracePeriod <= 0 {
		t.Error("GracePeriod should have a default")
	}
	if n.MaxRetries <= 0 {
		t.Error("MaxRetries should have a default")
	}
}
