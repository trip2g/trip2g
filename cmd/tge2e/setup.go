package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

// runSetup creates the E2E test environment.
func runSetup() error {
	fmt.Println("=== Telegram E2E Test Environment Setup ===")
	fmt.Println()

	// Load API credentials from environment
	apiID, apiHash, err := LoadAPICredentials()
	if err != nil {
		return err
	}
	fmt.Printf("Using API ID: %d\n", apiID)
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Try to load existing credentials
	creds, err := LoadCredentials("")
	if err == nil && creds.AccountSessionB64 != "" {
		fmt.Println("Using existing session.")
	} else {
		// Initialize new credentials
		creds = &Credentials{
			APIID:    apiID,
			APIHash:  apiHash,
			Channels: make(map[string]ChannelConfig),
		}

		// Step 1: Authenticate account
		fmt.Println()
		fmt.Println("=== Step 1: Account Authentication ===")

		authResult, sessionData, authErr := AuthenticateAccount(ctx, apiID, apiHash)
		if authErr != nil {
			return fmt.Errorf("account authentication failed: %w", authErr)
		}

		creds.AccountPhone = authResult.Phone
		creds.SetAccountSession(sessionData)
		creds.AccountDisplayName = authResult.DisplayName

		// Save progress
		err = SaveCredentials(creds, "")
		if err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}
	}

	// Step 2: Find channels
	fmt.Println()
	fmt.Println("=== Step 2: Find Test Channels ===")

	err = runWithClient(ctx, creds, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		channels, findErr := FindTestChannels(ctx, api)
		if findErr != nil {
			return findErr
		}
		creds.Channels = channels
		return nil
	})
	if err != nil {
		return fmt.Errorf("channel lookup failed: %w", err)
	}

	// Save progress
	err = SaveCredentials(creds, "")
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	// Step 3: Bot setup
	fmt.Println()
	fmt.Println("=== Step 3: Bot Setup ===")

	if creds.BotToken == "" {
		creds.BotToken = readLine("Enter bot token: ")
		if creds.BotToken == "" {
			return fmt.Errorf("bot token is required")
		}
	} else {
		fmt.Printf("Using existing bot token.\n")
	}

	if creds.BotUsername == "" {
		creds.BotUsername = readLine("Enter bot username (without @): ")
		if creds.BotUsername == "" {
			return fmt.Errorf("bot username is required")
		}
	} else {
		fmt.Printf("Bot username: @%s\n", creds.BotUsername)
	}

	// Save progress
	err = SaveCredentials(creds, "")
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	// Step 4: Verify bot is in channels
	fmt.Println()
	fmt.Println("=== Step 4: Verify Bot in Channels ===")

	err = runWithClient(ctx, creds, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		return VerifyBotInChannels(ctx, api, creds.Channels, creds.BotUsername)
	})
	if err != nil {
		return fmt.Errorf("bot verification failed: %w", err)
	}

	// Step 5: Clear channels
	fmt.Println()
	fmt.Println("=== Step 5: Clear Channel Messages ===")

	err = runWithClient(ctx, creds, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		return ClearAllTestChannels(ctx, api, creds.Channels)
	})
	if err != nil {
		return fmt.Errorf("failed to clear channels: %w", err)
	}

	// Final save
	err = SaveCredentials(creds, "")
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	// Print summary
	fmt.Println()
	fmt.Println("=== Setup Complete ===")
	fmt.Printf("Credentials saved to: %s\n", DefaultCredentialsPath)
	fmt.Println()
	fmt.Println("Account:")
	fmt.Printf("  Phone: %s\n", creds.AccountPhone)
	fmt.Printf("  Name:  %s\n", creds.AccountDisplayName)
	fmt.Println()
	fmt.Println("Bot:")
	fmt.Printf("  Username: @%s\n", creds.BotUsername)
	fmt.Println()
	fmt.Println("Channels:")
	for name, ch := range creds.Channels {
		fmt.Printf("  %s: %s (ID: %d)\n", name, ch.Title, ch.ID)
	}
	fmt.Println()
	fmt.Println("Run 'tge2e verify' to validate the setup.")

	return nil
}

// runCleanup clears all test channels.
func runCleanup() error {
	fmt.Println("=== Clearing Test Channels ===")
	fmt.Println()

	creds, err := LoadCredentials("")
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

	creds, err := LoadCredentials("")
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
		fmt.Printf("Found %d issue(s). Run 'tge2e setup' to fix.\n", issues)
		os.Exit(1)
	}

	fmt.Println("All checks passed. Test environment is ready.")
	return nil
}

// runWithClient is a helper that runs a function with an authenticated client.
func runWithClient(ctx context.Context, creds *Credentials, fn func(ctx context.Context, client *telegram.Client, api *tg.Client) error) error {
	return RunWithClient(ctx, creds, fn)
}
