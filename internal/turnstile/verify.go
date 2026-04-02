package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultVerifyURL is the Cloudflare Turnstile siteverify endpoint.
const defaultVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// Config holds Turnstile configuration.
type Config struct {
	SiteKey   string
	SecretKey string
}

// Client verifies Cloudflare Turnstile tokens.
type Client struct {
	config    Config
	verifyURL string // defaults to defaultVerifyURL if empty; override for testing
}

// New creates a new Turnstile client.
func New(config Config) *Client {
	return &Client{config: config}
}

// VerifyCaptcha verifies a Turnstile token.
// Returns nil on success, error on failure.
// If SecretKey is empty, verification is skipped (dev mode).
func (c *Client) VerifyCaptcha(ctx context.Context, token, remoteIP string) error {
	if c.config.SecretKey == "" {
		return nil // skip in dev
	}

	if token == "" {
		return errors.New("captcha token is required")
	}

	verifyURL := c.verifyURL
	if verifyURL == "" {
		verifyURL = defaultVerifyURL
	}

	form := url.Values{
		"secret":   {c.config.SecretKey},
		"response": {token},
		"remoteip": {remoteIP},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("turnstile request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := &http.Client{Timeout: 10 * time.Second}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("turnstile request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return fmt.Errorf("turnstile decode failed: %w", decodeErr)
	}
	if !result.Success {
		return errors.New("turnstile verification failed")
	}
	return nil
}
