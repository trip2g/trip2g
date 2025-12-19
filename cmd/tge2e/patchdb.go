package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"trip2g/internal/dataencryption"
)

const legacyCredentialsPath = ".tg_e2e_session"

// LegacyCredentials is the old JSON format for credentials file.
type LegacyCredentials struct {
	APIID              int                        `json:"api_id"`
	APIHash            string                     `json:"api_hash"`
	AccountPhone       string                     `json:"account_phone"`
	AccountSessionB64  string                     `json:"account_session_b64"`
	AccountDisplayName string                     `json:"account_display_name"`
	BotToken           string                     `json:"bot_token"`
	BotUsername        string                     `json:"bot_username"`
	Channels           map[string]LegacyChannelCf `json:"channels"`
}

// LegacyChannelCf is the old JSON format for channel config.
type LegacyChannelCf struct {
	Title      string `json:"title"`
	Username   string `json:"username"`
	ID         int64  `json:"id"`
	AccessHash int64  `json:"access_hash"`
}

func runPatchDB() error {
	// Load credentials from legacy .tg_e2e_session file
	data, err := os.ReadFile(legacyCredentialsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("credentials file not found: %s (this command is for migrating from old workflow)", legacyCredentialsPath)
		}
		return fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds LegacyCredentials
	err = json.Unmarshal(data, &creds)
	if err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create encryption manager with default dev key
	encryptor, err := dataencryption.NewManager(dataencryption.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	// Update telegram_accounts
	if creds.AccountPhone != "" {
		// Create Credentials to use AccountSession method
		c := &Credentials{AccountSessionB64: creds.AccountSessionB64}
		sessionData, sessionErr := c.AccountSession()
		if sessionErr != nil {
			return fmt.Errorf("failed to decode session: %w", sessionErr)
		}

		// Encrypt session data
		encryptedSession, encryptErr := encryptor.EncryptData(sessionData)
		if encryptErr != nil {
			return fmt.Errorf("failed to encrypt session: %w", encryptErr)
		}

		res, execErr := db.Exec(`
			update telegram_accounts
			set phone = ?, session_data = ?, display_name = ?
			where id = 1
		`, creds.AccountPhone, encryptedSession, creds.AccountDisplayName)
		if execErr != nil {
			return fmt.Errorf("failed to update telegram_accounts: %w", execErr)
		}
		rows, _ := res.RowsAffected()
		fmt.Printf("telegram_accounts: updated %d row(s)\n", rows)
	}

	// Update tg_bots
	if creds.BotToken != "" {
		res, execErr := db.Exec(`
			update tg_bots
			set token = ?, name = ?
			where id = 1
		`, creds.BotToken, creds.BotUsername)
		if execErr != nil {
			return fmt.Errorf("failed to update tg_bots: %w", execErr)
		}
		rows, _ := res.RowsAffected()
		fmt.Printf("tg_bots: updated %d row(s)\n", rows)
	}

	fmt.Println("\nPatch completed!")
	return nil
}
