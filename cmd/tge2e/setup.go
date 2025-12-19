package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

// runCleanup clears all test channels.
func runCleanup() error {
	fmt.Println("=== Clearing Test Channels ===")
	fmt.Println()

	creds, err := loadCredentials()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = runWithClient(ctx, creds, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		return ClearAllTestChannels(ctx, api, creds.Channels)
	})
	if err != nil {
		return fmt.Errorf("failed to clear channels: %w", err)
	}

	fmt.Println()
	fmt.Println("All channels cleared.")

	return nil
}

// runVerify checks that the test environment is properly configured.
func runVerify() error {
	fmt.Println("=== Verifying Test Environment ===")
	fmt.Println()

	creds, err := loadCredentials()
	if err != nil {
		return err
	}

	issues := 0

	// Check account
	fmt.Print("Account session... ")
	_, err = creds.AccountSession()
	if err != nil {
		fmt.Println("FAIL: invalid session data")
		issues++
	} else {
		fmt.Println("OK")
	}

	// Check channels
	fmt.Println("Channels:")
	channelTitles := map[string]string{
		ChannelBotScheduled:     "Trip2G Test Bot",
		ChannelBotInstant:       "Trip2G Test Bot Instant",
		ChannelAccountScheduled: "Trip2G Test Account",
		ChannelAccountInstant:   "Trip2G Test Account Instant",
	}
	for _, name := range AllChannelNames() {
		title := channelTitles[name]
		fmt.Printf("  %s... ", title)
		ch, ok := creds.Channels[name]
		if !ok {
			fmt.Println("FAIL: not found in config")
			issues++
		} else if ch.ID == 0 {
			fmt.Println("FAIL: invalid ID")
			issues++
		} else {
			fmt.Printf("OK (ID: %d)\n", ch.ID)
		}
	}

	// Try to connect and verify
	fmt.Println()
	fmt.Print("Testing connection... ")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runWithClient(ctx, creds, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		// Verify we can get dialogs
		_, getErr := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			Limit:      10,
		})
		return getErr
	})
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		issues++
	} else {
		fmt.Println("OK")
	}

	fmt.Println()
	if issues > 0 {
		fmt.Printf("Found %d issue(s).\n", issues)
		os.Exit(1)
	}

	fmt.Println("All checks passed. Test environment is ready.")
	return nil
}

// runWithClient is a helper that runs a function with an authenticated client.
func runWithClient(ctx context.Context, creds *Credentials, fn func(ctx context.Context, client *telegram.Client, api *tg.Client) error) error {
	return RunWithClient(ctx, creds, fn)
}

// findChannelsWithSession finds test channels using account session.
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
