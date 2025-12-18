package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/gotd/td/telegram"
	_ "github.com/mattn/go-sqlite3"
	"trip2g/internal/dataencryption"
)

func runExtract() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: tge2e extract <path/to/database.sqlite>")
	}

	dbPath := os.Args[2]

	// Load API credentials from environment
	apiID, apiHash, err := LoadAPICredentials()
	if err != nil {
		return err
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create decryption manager with default dev key
	decryptor, err := dataencryption.NewManager(dataencryption.DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to create decryptor: %w", err)
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
			fmt.Println("telegram_accounts: not found")
		} else {
			return fmt.Errorf("failed to query telegram_accounts: %w", err)
		}
	} else {
		// Decrypt session
		sessionData, err := decryptor.DecryptData(encryptedSession)
		if err != nil {
			return fmt.Errorf("failed to decrypt session: %w", err)
		}

		creds.AccountPhone = phone
		creds.AccountDisplayName = displayName
		creds.SetAccountSession(sessionData)
		fmt.Printf("telegram_accounts: extracted (phone: %s)\n", phone)
	}

	// Extract tg_bots
	var token, name string
	err = db.QueryRow(`
		select token, name
		from tg_bots
		where id = 1
	`).Scan(&token, &name)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("tg_bots: not found")
		} else {
			return fmt.Errorf("failed to query tg_bots: %w", err)
		}
	} else {
		creds.BotToken = token
		creds.BotUsername = name
		fmt.Printf("tg_bots: extracted (username: @%s)\n", name)
	}

	// Find test channels via Telegram API
	if creds.AccountSessionB64 != "" {
		fmt.Println("\nFinding test channels...")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		err = findChannelsWithSession(ctx, creds)
		if err != nil {
			fmt.Printf("Warning: failed to find channels: %v\n", err)
		}
	}

	// Save credentials
	err = SaveCredentials(creds, "")
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("\nSaved to %s\n", DefaultCredentialsPath)
	return nil
}

func findChannelsWithSession(ctx context.Context, creds *Credentials) error {
	sessionData, err := creds.AccountSession()
	if err != nil {
		return fmt.Errorf("failed to decode session: %w", err)
	}

	storage := &MemorySessionStorage{Data: sessionData}

	client := telegram.NewClient(creds.APIID, creds.APIHash, telegram.Options{
		SessionStorage: storage,
	})

	return client.Run(ctx, func(ctx context.Context) error {
		channels, err := FindTestChannels(ctx, client.API())
		if err != nil {
			return err
		}
		creds.Channels = channels
		return nil
	})
}
