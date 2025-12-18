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

4. **E2E seed database** with configured account, bot and publish tags (see [e2e_seed.md](e2e_seed.md))

## Installation

```bash
go build ./cmd/tge2e
```

## Commands

### extract

Extract credentials from database to `.tg_e2e_session`.

```bash
./tge2e extract <path/to/database.sqlite>
```

Steps performed:
1. Extract telegram account session (decrypts with dev key)
2. Extract bot token
3. Connect to Telegram and find test channels by title
4. Save to `.tg_e2e_session`

### patch-db

Update database with credentials from `.tg_e2e_session`.

```bash
./tge2e patch-db <path/to/database.sqlite>
```

Updates:
- `telegram_accounts` (phone, session_data, display_name)
- `tg_bots` (token, name)

### cleanup

Clear all messages from test channels.

```bash
./tge2e cleanup
```

### verify

Check that `.tg_e2e_session` is valid and test environment is configured.

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

### Initial setup (one time)

1. Create 4 test channels in Telegram
2. Create bot via @BotFather, add as admin to bot channels
3. Create E2E seed database (see [e2e_seed.md](e2e_seed.md))
4. Extract credentials:
   ```bash
   ./tge2e extract data.sqlite3
   ```

### Running E2E tests

```bash
# 1. Prepare test database from seed
sqlite3 test.db < testdata/e2e_seed.sql
./tge2e patch-db test.db

# 2. Clean channels
./tge2e cleanup

# 3. Start server with test database
go run ./cmd/server -db-file=test.db -dev

# 4. Run tests that publish to channels
go test ./... -run TestTelegramPublish

# 5. Verify results
./tge2e check
```

### Updating snapshots

After intentional changes to publishing:

```bash
./tge2e dump
git add testdata/telegram/snapshots/
```

## Files

| Path | Description |
|------|-------------|
| `.tg_e2e_session` | Telegram session and channel config (gitignored) |
| `testdata/e2e_seed.sql` | Database seed with placeholders |
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

### "failed to decrypt session"

Ensure the database was created with dev encryption key (`please-change-me-to-32-byte-key!`).

### Reset sent messages

```sql
delete from telegram_publish_sent_messages;
delete from telegram_publish_sent_account_messages;
update telegram_publish_notes set published_at = null, published_version_id = null, error_count = 0, last_error = null;
```
