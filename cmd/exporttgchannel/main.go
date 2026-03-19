// cmd/exporttgchannel - export a Telegram channel to Obsidian-compatible markdown files.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	sessionPath := flag.String("session", defaultSessionPath(), "session file path")
	timeOffset := flag.Duration("time-offset", 0, "correct system clock skew (e.g. +1h or -30m); use if auth hangs due to wrong system time")
	flag.Usage = func() { printUsage() }
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	var err error
	switch args[0] {
	case "auth":
		err = runAuth(ctx, *sessionPath, *timeOffset, args[1:])
	case "channels":
		err = runChannels(ctx, *sessionPath, args[1:])
	case "export":
		err = runExport(ctx, *sessionPath, args[1:])
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`exporttgchannel - export a Telegram channel to Obsidian-compatible markdown files

USAGE
  exporttgchannel [--session FILE] <command> [flags]

COMMANDS

  auth
    Interactive one-time setup. Reads TELEGRAM_API_ID / TELEGRAM_API_HASH from
    environment (or prompts if unset), asks for phone number and the SMS/app code,
    handles 2FA, then saves a session file to disk. Must be run by a human.

  channels [--filter NAME]
    Lists all channels the authenticated account can see, with numeric IDs.
    --filter NAME  case-insensitive substring match on channel title

  export --channel ID --out DIR [--media] [--no-skip-exists]
    Downloads every message from the channel and writes one .md file per post.
    --channel ID         numeric channel ID from the 'channels' command (required)
    --out DIR            output directory; created automatically if missing (required)
    --media              also download photos/videos into DIR/assets/
    --no-skip-exists     overwrite files that already exist (default: skip)

    Output layout:
      DIR/_index.md           chronological index with [[wikilinks]] to all posts
      DIR/<title>.md          one file per message group, with YAML frontmatter
      DIR/assets/<id>_<n>.jpg media files (only with --media)

    Frontmatter written per note:
      telegram_publish_channel_id, telegram_publish_message_id,
      telegram_publish_message_link, telegram_publish_at,
      telegram_import_allow_override: true

GLOBAL FLAGS
  --session FILE         path to session file
                         (default: ~/.config/exporttgchannel/session.json)
  --time-offset DURATION correct system clock skew, e.g. +1h or -30m
                         (use if 'auth' hangs — symptom: wrong system time)

ENVIRONMENT (read during 'auth')
  TELEGRAM_API_ID    integer API ID from https://my.telegram.org/apps
  TELEGRAM_API_HASH  string  API hash from https://my.telegram.org/apps

WORKFLOW FOR CLAUDE CODE
  Step 1 — human runs auth once:
    exporttgchannel auth

  Step 2 — Claude Code finds the channel ID:
    exporttgchannel channels
    exporttgchannel channels --filter "mychannel"

  Step 3 — Claude Code exports:
    exporttgchannel export --channel 1234567890 --out ./vault/import
    exporttgchannel export --channel 1234567890 --out ./vault/import --media

  The session file persists; Step 2-3 can be repeated without human interaction.
`)
}
