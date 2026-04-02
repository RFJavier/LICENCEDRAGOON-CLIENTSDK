package license

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type ActivationResponse struct {
	DeviceID  string `json:"device_id"`
	Interval  int    `json:"interval"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

type HeartbeatResponse struct {
	Status    string `json:"status"`
	Action    string `json:"action"`
	Interval  int    `json:"interval"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

type SDK struct {
	cfg      Config
	storage  *FileStorage
	client   *httpClient
	hooks    Hooks
	pubKey   ed25519.PublicKey
	mu       sync.RWMutex
	graceOn  bool
	graceBeg time.Time
}

func New(cfg Config) (*SDK, error) {
	cfg = cfg.normalize()
	if cfg.APIURL == "" || cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.StoragePath == "" || cfg.PublicKey == "" {
		return nil, errors.New("invalid config: missing required fields")
	}
	pkBytes, err := hex.DecodeString(cfg.PublicKey)
	if err != nil || len(pkBytes) != ed25519.PublicKeySize {
		return nil, errors.New("invalid config: PublicKey must be a 64-char hex-encoded Ed25519 public key")
	}
	return &SDK{
		cfg:     cfg,
		storage: NewFileStorage(cfg.StoragePath),
		client:  newHTTPClient(cfg),
		pubKey:  ed25519.PublicKey(pkBytes),
	}, nil
}

func (s *SDK) Activate(licenseKey string) (*ActivationResponse, error) {
	var resp ActivationResponse
	err := s.client.post(context.Background(), "/v1/license/activate", map[string]any{
		"license_key": licenseKey,
		"device_name": "default-device",
		"device_hash": "default-device-hash",
	}, &resp)
	if err != nil {
		return nil, err
	}

	ok := verifySignature(map[string]any{
		"device_id": resp.DeviceID,
		"interval":  resp.Interval,
	}, resp.Timestamp, resp.Signature, s.pubKey)
	if !ok {
		return nil, errors.New("invalid activation signature")
	}

	state := &State{
		LicenseKey:      licenseKey,
		DeviceID:        resp.DeviceID,
		LastValidatedAt: time.Now().UTC(),
		ValidUntil:      time.Now().UTC().Add(s.cfg.GracePeriod),
	}
	if err := s.storage.Save(state); err != nil {
		return nil, err
	}
	return &resp, nil
}
