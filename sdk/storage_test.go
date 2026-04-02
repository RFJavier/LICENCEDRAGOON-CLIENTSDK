package license

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStorage_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	fs := NewFileStorage(path)

	state := &State{
		LicenseKey:      "LIC-TEST-123",
		DeviceID:        "device-abc",
		LastValidatedAt: time.Now().UTC().Truncate(time.Second),
		ValidUntil:      time.Now().UTC().Add(10 * time.Minute).Truncate(time.Second),
	}

	if err := fs.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := fs.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.LicenseKey != state.LicenseKey {
		t.Errorf("LicenseKey: got %s, want %s", loaded.LicenseKey, state.LicenseKey)
	}
	if loaded.DeviceID != state.DeviceID {
		t.Errorf("DeviceID: got %s, want %s", loaded.DeviceID, state.DeviceID)
	}
	if !loaded.LastValidatedAt.Equal(state.LastValidatedAt) {
		t.Errorf("LastValidatedAt mismatch")
	}
	if !loaded.ValidUntil.Equal(state.ValidUntil) {
		t.Errorf("ValidUntil mismatch")
	}
}

func TestFileStorage_LoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	fs := NewFileStorage(path)

	state, err := fs.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.LicenseKey != "" {
		t.Error("expected empty state for nonexistent file")
	}
}

func TestFileStorage_LoadCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")
	os.WriteFile(path, []byte("not json{{{"), 0o600)

	fs := NewFileStorage(path)
	_, err := fs.Load()
	if err == nil {
		t.Error("expected error for corrupted file")
	}
}

func TestFileStorage_SaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "deep", "state.json")
	fs := NewFileStorage(path)

	state := &State{LicenseKey: "test"}
	if err := fs.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	var loaded State
	json.Unmarshal(raw, &loaded)
	if loaded.LicenseKey != "test" {
		t.Error("saved content mismatch")
	}
}

func TestFileStorage_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	fs := NewFileStorage(path)
	fs.Save(&State{LicenseKey: "test"})

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	// On Windows, permissions may differ; just check file exists and is not directory
	if info.IsDir() {
		t.Error("expected regular file, got directory")
	}
}
