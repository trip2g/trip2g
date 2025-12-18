// cmd/tge2e - Telegram E2E test environment setup tool
//
// Commands:
//
//	setup   - Authenticate and find existing test channels
//	cleanup - Clear all messages from test channels
//	verify  - Check that test environment is properly configured
//
// Required channels (create manually before running setup):
//   - Trip2G Test Bot          (Bot API scheduled publish)
//   - Trip2G Test Bot Instant  (Bot API instant publish)
//   - Trip2G Test Account      (MTProto scheduled publish)
//   - Trip2G Test Account Instant (MTProto instant publish)
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	var err error
	switch cmd {
	case "setup":
		err = runSetup()
	case "cleanup":
		err = runCleanup()
	case "verify":
		err = runVerify()
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
	fmt.Println(`tge2e - Telegram E2E test environment setup

Usage: tge2e <command>

Commands:
  setup     Authenticate account, find test channels, verify bot, clear channels
            Interactive, run once to initialize test environment

  cleanup   Clear all messages from test channels
            Run before E2E tests to ensure clean state

  verify    Check that test environment is properly configured
            Returns exit code 0 if ready, 1 if not

  dump      Save current channel messages to testdata/telegram/snapshots/
            Run after publishing to capture expected state

  check     Compare current channel messages with saved snapshots
            Returns exit code 0 if match, 1 if different

Environment:
  TELEGRAM_API_ID      Your Telegram API ID (from my.telegram.org)
  TELEGRAM_API_HASH    Your Telegram API hash (from my.telegram.org)

Required channels (create manually before setup):
  - Trip2G Test Bot
  - Trip2G Test Bot Instant
  - Trip2G Test Account
  - Trip2G Test Account Instant

Output:
  Credentials: .tg_e2e_session
  Snapshots:   testdata/telegram/snapshots/`)
}
