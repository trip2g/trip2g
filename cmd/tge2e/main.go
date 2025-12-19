// cmd/tge2e - Telegram E2E test environment tool
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	dbPath string
)

func main() {
	// Global flags
	flag.StringVar(&dbPath, "db", "", "Path to database file (required)")

	// Custom usage
	flag.Usage = func() {
		printUsage()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	cmd := args[0]

	// Check if db is required for this command
	requiresDB := cmd != "help" && cmd != "-h" && cmd != "--help"
	if requiresDB && dbPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -db flag is required\n\n")
		printUsage()
		os.Exit(1)
	}

	var err error
	switch cmd {
	case "patch-db":
		err = runPatchDB()
	case "verify":
		err = runVerify()
	case "cleanup":
		err = runCleanup()
	case "dump":
		err = runDump()
	case "check":
		err = runCheck()
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`tge2e - Telegram E2E test environment tool

Usage: tge2e -db <database.sqlite> <command>

Commands:
  patch-db  Update database with credentials from .tg_e2e_session
            (for migrating from old workflow)

  verify    Check that database credentials are valid
            Returns exit code 0 if ready, 1 if not

  cleanup   Clear all messages from test channels

  dump      Save current channel messages to testdata/telegram/snapshots/

  check     Compare current channel messages with saved snapshots
            Returns exit code 0 if match, 1 if different

Flags:
  -db       Path to database file (required for all commands)

Environment:
  TELEGRAM_API_ID      Your Telegram API ID (from my.telegram.org)
  TELEGRAM_API_HASH    Your Telegram API hash (from my.telegram.org)

Snapshots: testdata/telegram/snapshots/*.json`)
}

// loadCredentials loads credentials from database with channel discovery.
func loadCredentials() (*Credentials, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	creds, err := LoadCredentialsFromDB(ctx, dbPath)
	if err != nil {
		return nil, err
	}

	// Find test channels via Telegram API
	fmt.Println("\nFinding test channels...")

	err = findChannelsWithSession(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to find channels: %w", err)
	}

	return creds, nil
}
