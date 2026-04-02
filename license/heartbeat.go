package license

import (
	"context"
	"errors"
	"time"
)

func (s *SDK) StartHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.runHeartbeat(ctx); err != nil {
				if s.hooks.onHeartbeatError != nil {
					s.hooks.onHeartbeatError(err)
				}
				s.handleOfflineFallback()
			}
		}
	}
}

func (s *SDK) runHeartbeat(ctx context.Context) error {
	state, err := s.storage.Load()
	if err != nil {
		return err
	}
	if state.LicenseKey == "" || state.DeviceID == "" {
		return errors.New("sdk not activated")
	}

	var resp HeartbeatResponse
	err = s.client.post(ctx, "/v1/license/heartbeat", map[string]any{
		"license_key": state.LicenseKey,
		"device_id":   state.DeviceID,
	}, &resp)
	if err != nil {
		return err
	}

	ok := verifySignature(map[string]any{
		"status":   resp.Status,
		"action":   resp.Action,
		"interval": resp.Interval,
	}, resp.Timestamp, resp.Signature, s.pubKey)
	if !ok {
		return errors.New("invalid heartbeat signature")
	}

	if resp.Status == "blocked" || resp.Action == "stop" {
		if s.hooks.onBlocked != nil {
			s.hooks.onBlocked()
		}
		return errors.New("license blocked")
	}

	state.LastValidatedAt = time.Now().UTC()
	state.ValidUntil = time.Now().UTC().Add(s.cfg.GracePeriod)
	return s.storage.Save(state)
}

func (s *SDK) handleOfflineFallback() {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.storage.Load()
	if err != nil {
		return
	}
	now := time.Now().UTC()
	if now.Before(state.ValidUntil) {
		if !s.graceOn {
			s.graceOn = true
			s.graceBeg = now
			if s.hooks.onGracePeriodStart != nil {
				s.hooks.onGracePeriodStart()
			}
		}
		return
	}

	if s.hooks.onBlocked != nil {
		s.hooks.onBlocked()
	}
}
