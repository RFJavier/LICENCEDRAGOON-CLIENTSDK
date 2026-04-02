package license

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strings"
	"time"
)

type httpClient struct {
	baseURL string
	cfg     Config
	client  *http.Client
}

func newHTTPClient(cfg Config) *httpClient {
	return &httpClient{
		baseURL: strings.TrimRight(cfg.APIURL, "/"),
		cfg:     cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *httpClient) post(ctx context.Context, path string, payload any, out any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CLIENT-ID", c.cfg.ClientID)
		req.Header.Set("X-CLIENT-SECRET", c.cfg.ClientSecret)
		req.Header.Set("X-TIMESTAMP", fmt.Sprintf("%d", time.Now().Unix()))
		nonce := make([]byte, 16)
		_, _ = rand.Read(nonce)
		req.Header.Set("X-NONCE", hex.EncodeToString(nonce))

		resp, err := c.client.Do(req)
		if err != nil {
			if isRetriableNetErr(err) && attempt < c.cfg.MaxRetries {
				sleepBackoff(attempt)
				lastErr = err
				continue
			}
			return err
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return readErr
		}

		if resp.StatusCode >= 500 && attempt < c.cfg.MaxRetries {
			sleepBackoff(attempt)
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("request failed: %s", strings.TrimSpace(string(body)))
		}

		if err := json.Unmarshal(body, out); err != nil {
			return err
		}
		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("request failed after retries")
}

func isRetriableNetErr(err error) bool {
	if nErr, ok := err.(net.Error); ok {
		return nErr.Timeout() || nErr.Temporary()
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such host") || strings.Contains(msg, "connection refused")
}

func sleepBackoff(attempt int) {
	base := 200 * time.Millisecond
	factor := math.Pow(2, float64(attempt))
	time.Sleep(time.Duration(factor) * base)
}

func canonicalPayload(payload map[string]any) string {
	raw, _ := json.Marshal(payload)
	return string(raw)
}
