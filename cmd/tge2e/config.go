package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
	"trip2g/internal/appconfig"
	"trip2g/internal/dataencryption"
)

func init() {
	// Load .env file if it exists
	_ = appconfig.LoadDotEnvFromPath(".env")
}

const (
	// Channel names for E2E testing
	ChannelBotScheduled     = "trip2g_test_bot"
	ChannelBotInstant       = "trip2g_test_bot_instant"
	ChannelAccountScheduled = "trip2g_test_account"
	ChannelAccountInstant   = "trip2g_test_account_inst"

	// Default output path for snapshots
	SnapshotDir = "testdata/telegram/snapshots"
)

// ChannelConfig holds info about a test channel.
type ChannelConfig struct {
	Title      string `yaml:"title"`
	Username   string `yaml:"username"`
	ID         int64  `yaml:"id"`
	AccessHash int64  `yaml:"access_hash"`
}

// Credentials holds all test environment credentials.
type Credentials struct {
	// API credentials
	APIID   int    `yaml:"api_id"`
	APIHash string `yaml:"api_hash"`

	// Test account
	AccountPhone       string `yaml:"account_phone"`
	AccountSessionB64  string `yaml:"account_session_b64"` // base64 encoded session
	AccountDisplayName string `yaml:"account_display_name"`

	// Test bot
	BotToken    string `yaml:"bot_token"`
	BotUsername string `yaml:"bot_username"`

	// Test channels
	Channels map[string]ChannelConfig `yaml:"channels"`
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

// LoadCredentialsFromDB loads credentials directly from the database.
func LoadCredentialsFromDB(ctx context.Context, dbPath string) (*Credentials, error) {
	// Load API credentials from environment
	apiID, apiHash, err := LoadAPICredentials()
	if err != nil {
		return nil, err
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create decryption manager with default dev key
	decryptor, err := dataencryption.NewManager(dataencryption.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create decryptor: %w", err)
	}

	creds := &Credentials{
		APIID:    apiID,
		APIHash:  apiHash,
		Channels: make(map[string]ChannelConfig),
	}

	// Extract telegram_accounts
	var phone, displayName string
	var encryptedSession []byte
	err = db.QueryRow(`
		select phone, session_data, display_name
		from telegram_accounts
		where id = 1
	`).Scan(&phone, &encryptedSession, &displayName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("telegram_accounts: not found (id=1)")
		}
		return nil, fmt.Errorf("failed to query telegram_accounts: %w", err)
	}

	// Decrypt session
	sessionData, err := decryptor.DecryptData(encryptedSession)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session: %w", err)
	}

	creds.AccountPhone = phone
	creds.AccountDisplayName = displayName
	creds.SetAccountSession(sessionData)
	fmt.Printf("Loaded account: %s (%s)\n", displayName, phone)

	// Extract tg_bots
	var token, name string
	err = db.QueryRow(`
		select token, name
		from tg_bots
		where id = 1
	`).Scan(&token, &name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tg_bots: not found (id=1)")
		}
		return nil, fmt.Errorf("failed to query tg_bots: %w", err)
	}

	creds.BotToken = token
	creds.BotUsername = name
	fmt.Printf("Loaded bot: @%s\n", name)

	return creds, nil
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
