# Telegram E2E Testing

End-to-end testing for Telegram publishing functionality using real Telegram channels.

## Overview

The `tge2e` tool manages the test environment for verifying that notes are correctly published to Telegram channels via both Bot API and MTProto Account pipelines.

## Prerequisites

1. **Telegram API credentials** from https://my.telegram.org:
   ```bash
   export TELEGRAM_API_ID=your_api_id
   export TELEGRAM_API_HASH=your_api_hash
   ```

2. **Four test channels** (create manually in Telegram):
   - `Trip2G Test Bot` - Bot API scheduled publishing
   - `Trip2G Test Bot Instant` - Bot API instant publishing
   - `Trip2G Test Account` - MTProto scheduled publishing
   - `Trip2G Test Account Instant` - MTProto instant publishing

3. **Test bot** created via @BotFather, added as admin to bot channels

## Installation

```bash
go build ./cmd/tge2e
```

## Commands

### setup

Initialize the test environment. Run once after creating channels.

```bash
./tge2e setup
```

Steps performed:
1. Authenticate with Telegram (or use existing session)
2. Find test channels by title
3. Configure bot token and username
4. Verify bot is admin in bot channels
5. Clear all channel messages

Session is saved to `.tg_e2e_session`.

### cleanup

Clear all messages from test channels. Run before each test.

```bash
./tge2e cleanup
```

### verify

Check that the test environment is properly configured.

```bash
./tge2e verify
```

Returns exit code 0 if ready, 1 if issues found.

### dump

Save current channel messages to snapshots.

```bash
./tge2e dump
```

Snapshots are saved to `testdata/telegram/snapshots/`:
- `trip2g_test_bot.json`
- `trip2g_test_bot_instant.json`
- `trip2g_test_account.json`
- `trip2g_test_account_inst.json`

### check

Compare current channel messages with saved snapshots.

```bash
./tge2e check
```

Returns exit code 0 if all channels match, 1 if different.

## Snapshot Format

```json
{
  "channel_name": "trip2g_test_bot",
  "channel_title": "Trip2G Test Bot",
  "messages": [
    {
      "id": 123,
      "text": "Message text",
      "has_media": false
    },
    {
      "id": 124,
      "text": "Caption",
      "has_media": true,
      "media_type": "photo"
    }
  ],
  "captured_at": "2025-01-15T12:00:00Z"
}
```

Comparison ignores message IDs and timestamps, only comparing:
- Message text
- Media presence (`has_media`)
- Media type (`photo`, `video`, `document`, etc.)

## Test Workflow

1. **Initial setup** (one time):
   ```bash
   # Create 4 channels in Telegram
   # Create bot via @BotFather
   # Add bot as admin to bot channels
   ./tge2e setup
   ```

2. **Before test run**:
   ```bash
   ./tge2e cleanup
   ```

3. **Run tests** that publish to channels

4. **Capture expected state** (first time or after intentional changes):
   ```bash
   ./tge2e dump
   git add testdata/telegram/snapshots/
   ```

5. **Verify published content**:
   ```bash
   ./tge2e check
   ```

## CI Integration

```bash
#!/bin/bash
set -e

# Verify environment
./tge2e verify

# Clean channels
./tge2e cleanup

# Run publishing tests
go test ./... -run TestTelegramPublish

# Trigger cron job or wait for scheduled publish
# ...

# Verify results
./tge2e check
```

## Files

| Path | Description |
|------|-------------|
| `.tg_e2e_session` | Telegram session and channel config (gitignored) |
| `testdata/telegram/snapshots/*.json` | Expected channel state |
| `cmd/tge2e/` | Tool source code |

## Troubleshooting

### "channel not found"

Ensure channels exist with exact titles:
- `Trip2G Test Bot`
- `Trip2G Test Bot Instant`
- `Trip2G Test Account`
- `Trip2G Test Account Instant`

### "bot not configured in channel"

Add the bot as admin to `Trip2G Test Bot` and `Trip2G Test Bot Instant` channels with posting permissions.

### "session expired"

Delete `.tg_e2e_session` and run `./tge2e setup` again.
