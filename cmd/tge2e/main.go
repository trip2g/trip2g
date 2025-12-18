// cmd/tge2e - Telegram E2E test environment tool
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
	case "extract":
		err = runExtract()
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

Usage: tge2e <command>

Commands:
  extract   Extract credentials from database to .tg_e2e_session
            Usage: tge2e extract <path/to/database.sqlite>

  patch-db  Update database with credentials from .tg_e2e_session
            Usage: tge2e patch-db <path/to/database.sqlite>

  verify    Check that .tg_e2e_session is valid
            Returns exit code 0 if ready, 1 if not

  cleanup   Clear all messages from test channels

  dump      Save current channel messages to testdata/telegram/snapshots/

  check     Compare current channel messages with saved snapshots
            Returns exit code 0 if match, 1 if different

Environment:
  TELEGRAM_API_ID      Your Telegram API ID (from my.telegram.org)
  TELEGRAM_API_HASH    Your Telegram API hash (from my.telegram.org)

Output:
  Credentials: .tg_e2e_session
  Snapshots:   testdata/telegram/snapshots/`)
}
