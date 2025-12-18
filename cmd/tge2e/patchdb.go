package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"trip2g/internal/dataencryption"
)

func runPatchDB() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: tge2e patch-db <path/to/database.sqlite>")
	}

	dbPath := os.Args[2]

	// Load credentials from .tg_e2e_session
	creds, err := LoadCredentials("")
	if err != nil {
		return err
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
		sessionData, err := creds.AccountSession()
		if err != nil {
			return fmt.Errorf("failed to decode session: %w", err)
		}

		// Encrypt session data
		encryptedSession, err := encryptor.EncryptData(sessionData)
		if err != nil {
			return fmt.Errorf("failed to encrypt session: %w", err)
		}

		res, err := db.Exec(`
			update telegram_accounts
			set phone = ?, session_data = ?, display_name = ?
			where id = 1
		`, creds.AccountPhone, encryptedSession, creds.AccountDisplayName)
		if err != nil {
			return fmt.Errorf("failed to update telegram_accounts: %w", err)
		}
		rows, _ := res.RowsAffected()
		fmt.Printf("telegram_accounts: updated %d row(s)\n", rows)
	}

	// Update tg_bots
	if creds.BotToken != "" {
		res, err := db.Exec(`
			update tg_bots
			set token = ?, name = ?
			where id = 1
		`, creds.BotToken, creds.BotUsername)
		if err != nil {
			return fmt.Errorf("failed to update tg_bots: %w", err)
		}
		rows, _ := res.RowsAffected()
		fmt.Printf("tg_bots: updated %d row(s)\n", rows)
	}

	fmt.Println("\nPatch completed!")
	return nil
}
