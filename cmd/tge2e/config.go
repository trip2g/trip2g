package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	// Channel names for E2E testing
	ChannelBotScheduled     = "trip2g_test_bot"
	ChannelBotInstant       = "trip2g_test_bot_instant"
	ChannelAccountScheduled = "trip2g_test_account"
	ChannelAccountInstant   = "trip2g_test_account_inst"

	// Default output path
	DefaultCredentialsPath = ".tg_e2e_session"
)

// ChannelConfig holds info about a test channel.
type ChannelConfig struct {
	Title      string `json:"title"`
	Username   string `json:"username"`
	ID         int64  `json:"id"`
	AccessHash int64  `json:"access_hash"`
}

// Credentials holds all test environment credentials.
type Credentials struct {
	// API credentials
	APIID   int    `json:"api_id"`
	APIHash string `json:"api_hash"`

	// Test account
	AccountPhone       string `json:"account_phone"`
	AccountSessionB64  string `json:"account_session_b64"` // base64 encoded session
	AccountDisplayName string `json:"account_display_name"`

	// Test bot
	BotToken    string `json:"bot_token"`
	BotUsername string `json:"bot_username"`

	// Test channels
	Channels map[string]ChannelConfig `json:"channels"`
}

// AccountSession returns decoded session data.
func (c *Credentials) AccountSession() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.AccountSessionB64)
}

// SetAccountSession encodes and stores session data.
func (c *Credentials) SetAccountSession(data []byte) {
	c.AccountSessionB64 = base64.StdEncoding.EncodeToString(data)
}

// GetChannel returns channel config by name.
func (c *Credentials) GetChannel(name string) (ChannelConfig, bool) {
	ch, ok := c.Channels[name]
	return ch, ok
}

// LoadCredentials loads credentials from file.
func LoadCredentials(path string) (*Credentials, error) {
	if path == "" {
		path = DefaultCredentialsPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("credentials file not found: %s (run 'tge2e setup' first)", path)
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds Credentials
	err = json.Unmarshal(data, &creds)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// SaveCredentials saves credentials to file.
func SaveCredentials(creds *Credentials, path string) error {
	if path == "" {
		path = DefaultCredentialsPath
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	err = os.WriteFile(path, data, 0600) // restricted permissions
	if err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// LoadAPICredentials loads API ID and hash from environment.
func LoadAPICredentials() (int, string, error) {
	apiIDStr := os.Getenv("TELEGRAM_API_ID")
	apiHash := os.Getenv("TELEGRAM_API_HASH")

	if apiIDStr == "" {
		return 0, "", fmt.Errorf("TELEGRAM_API_ID environment variable is required")
	}
	if apiHash == "" {
		return 0, "", fmt.Errorf("TELEGRAM_API_HASH environment variable is required")
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid TELEGRAM_API_ID: %w", err)
	}

	return apiID, apiHash, nil
}

// AllChannelNames returns all test channel names.
func AllChannelNames() []string {
	return []string{
		ChannelBotScheduled,
		ChannelBotInstant,
		ChannelAccountScheduled,
		ChannelAccountInstant,
	}
}
