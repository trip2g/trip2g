package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func runExtract() error {
	fmt.Println("=== Extracting Credentials from Database ===")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	creds, err := LoadCredentialsFromDB(ctx, dbPath)
	if err != nil {
		return err
	}

	// Find test channels via Telegram API
	fmt.Println("\nFinding test channels...")

	err = findChannelsWithSession(ctx, creds)
	if err != nil {
		fmt.Printf("Warning: failed to find channels: %v\n", err)
	}

	// Convert to legacy format for saving
	legacy := LegacyCredentials{
		APIID:              creds.APIID,
		APIHash:            creds.APIHash,
		AccountPhone:       creds.AccountPhone,
		AccountSessionB64:  creds.AccountSessionB64,
		AccountDisplayName: creds.AccountDisplayName,
		BotToken:           creds.BotToken,
		BotUsername:        creds.BotUsername,
		Channels:           make(map[string]LegacyChannelCf),
	}

	for name, ch := range creds.Channels {
		legacy.Channels[name] = LegacyChannelCf{
			Title:      ch.Title,
			Username:   ch.Username,
			ID:         ch.ID,
			AccessHash: ch.AccessHash,
		}
	}

	// Save to file
	data, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	err = os.WriteFile(legacyCredentialsPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	fmt.Printf("\nSaved to %s\n", legacyCredentialsPath)
	fmt.Printf("  Account: %s (%s)\n", creds.AccountDisplayName, creds.AccountPhone)
	fmt.Printf("  Bot: @%s\n", creds.BotUsername)
	fmt.Printf("  Channels: %d\n", len(creds.Channels))

	return nil
}
