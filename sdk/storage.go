package license

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type State struct {
	LicenseKey      string    `json:"license_key"`
	DeviceID        string    `json:"device_id"`
	LastValidatedAt time.Time `json:"last_validated_at"`
	ValidUntil      time.Time `json:"valid_until"`
}

type FileStorage struct {
	path string
	mu   sync.RWMutex
}

func NewFileStorage(path string) *FileStorage {
	return &FileStorage{path: path}
}

func (s *FileStorage) Load() (*State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &State{}, nil
		}
		return nil, err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *FileStorage) Save(state *State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}
