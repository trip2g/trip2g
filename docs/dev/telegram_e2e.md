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

## Live TG↔subgraph access e2e

A separate, real end-to-end test drives an actual authenticated Telegram **user
account** against a running trip2g bot and asserts the TG↔subgraph access flow
and the two fixes in PR #208 (chat_member updates in `AllowedUpdates`;
`sendContentMenu` id-space) from the client side.

- Test: `internal/case/handletgupdate/e2e_live_client_test.go`
  (`TestE2ELiveTgSubgraphAccess`).
- Client driver: `internal/case/handletgupdate/testdata/tg_e2e/tg_driver.py`.

Unlike the publishing `tge2e` tool above, this test does **not** touch the DB. It
shells out to the Python driver, which reuses the
[telegram-mcp](https://github.com/chigwell/telegram-mcp) server's own tool
functions (`send_message`, `get_messages`, `list_inline_buttons`,
`join_chat_by_link`, `leave_chat`). The account authenticates
**non-interactively** from telegram-mcp's pre-generated Telethon
`TELEGRAM_SESSION_STRING` — a Python-only session string that a Go MTProto client
cannot reuse, hence the Python driver.

### What it exercises end-to-end

Real client → real Telegram → trip2g bot (long-polling) → back:

1. **Connectivity** — the driver connects the real account and returns its id.
2. **Bug 1 (chat_member → access)** — the account joins the dedicated throwaway
   test group via its invite link. Telegram fires a real `chat_member` update
   that the bot now receives (the `AllowedUpdates` fix); the bot inserts the
   membership and grants access to the linked subgraph.
3. **Bug 2 (content menu id-space)** — the account DMs the bot `/content`. The bot
   replies with the accessible subgraph as an **inline button** instead of
   "Ничего не найдено" (`sendContentMenu` now resolves content by system user id).
   The driver reads the reply's buttons via `list_inline_buttons`.
4. **Revocation** — the account leaves the group; `/content` again returns
   "Ничего не найдено".

Direction A (subgraph access → group access) is covered at the DB contract level
by `TestE2ETgSubgraphBidirectionalAccess`; client-side observation of it requires
the bot to hold invite rights and offer a group link, which is environment-specific.

### Prerequisites (manual setup — not provisioned automatically)

1. **telegram-mcp checkout** with a valid `.env`
   (`TELEGRAM_API_ID`/`TELEGRAM_API_HASH`/`TELEGRAM_SESSION_STRING`) and its deps
   installed (`uv sync` / `poetry install`). Default location
   `~/projects/telegram-mcp`; override with `TG_E2E_MCP_DIR`. Point
   `TG_E2E_PYTHON` at that env's interpreter (e.g. `.venv/bin/python`).
2. **A running trip2g instance** with a configured Telegram **bot** (token added in
   admin → Telegram Bots). The bot long-polls, so no public webhook is needed.
3. **A dedicated throwaway test group** the bot is a member/admin of, **linked to a
   subgraph** (`tg_bot_chat_subgraph_invites` + `tg_bot_chat_subgraph_accesses`,
   configured via the admin UI). Use a throwaway group — the test joins and leaves
   it and messages the bot; keep it away from real chats.
4. The Telegram account behind the session string must have a **linked trip2g
   system user** (`users.tg_user_id`) so `sendContentMenu` can resolve its content.

### Gate / env vars

The test **skips** unless all four required vars are set, so `go test ./...` and CI
skip it cleanly:

| Var | Req | Meaning |
|-----|-----|---------|
| `TG_E2E_BOT` | yes | `@username` or numeric id of the trip2g bot to DM |
| `TG_E2E_GROUP_INVITE` | yes | invite link (`t.me/+hash`) of the throwaway group linked to the subgraph |
| `TG_E2E_GROUP_ID` | yes | numeric telegram id of that group (used to leave it) |
| `TG_E2E_SUBGRAPH` | yes | subgraph name expected as an inline-button label |
| `TG_E2E_PYTHON` | no | interpreter with telegram-mcp deps (default `python3`) |
| `TG_E2E_MCP_DIR` | no | telegram-mcp checkout dir (default `~/projects/telegram-mcp`) |

### Run

```bash
TG_E2E_BOT=@my_test_bot \
TG_E2E_GROUP_INVITE='https://t.me/+abcdEFGH' \
TG_E2E_GROUP_ID=-1002529281698 \
TG_E2E_SUBGRAPH=e2e-subgraph \
TG_E2E_PYTHON="$HOME/projects/telegram-mcp/.venv/bin/python" \
go test ./internal/case/handletgupdate/ -run TestE2ELiveTgSubgraphAccess -v
```

The driver is independently smoke-testable without the bot:

```bash
"$TG_E2E_PYTHON" internal/case/handletgupdate/testdata/tg_e2e/tg_driver.py whoami
```

### Proven live

This test has been run to green against a real Telegram account and a real
(isolated, branch-built) instance:

```
=== RUN   TestE2ELiveTgSubgraphAccess
    e2e_live_client_test.go:71: connected real Telegram account id=7828312136
    e2e_live_client_test.go:75: join: Joined chat via invite hash.
    e2e_live_client_test.go:78: content menu granted access to subgraph "premium"
    e2e_live_client_test.go:82: leave: Left basic group trip2g e2e live test 1783931218 (ID: -5576940310).
    e2e_live_client_test.go:83: content menu empty after leaving group (access revoked)
--- PASS: TestE2ELiveTgSubgraphAccess (69.42s)
PASS
```

Setup notes from that run (see PR #208 description for the full account):

- Build and run this branch's own binary as an **isolated** instance (separate
  `--listen-addr`, separate copy of the `.sqlite3` file) rather than reusing a
  shared dev stand running older code — otherwise the test proves nothing about
  the fix.
- Leave `--public-url` **empty** on that isolated instance. A non-empty
  `https://...` `PublicURL` switches `internal/tgbots` into webhook mode
  (`bots.go`: `strings.HasPrefix(publicURL, "https")`), which stops the
  long-poll loop the bot needs for this test. An empty `PublicURL` also makes
  `GenerateTgAuthURL` fall back to `https://example.com` for the inline auth
  button, which is syntactically valid for Telegram without needing a
  reachable public host.
- Give the "latest" noteloader a few seconds after boot to finish indexing
  before sending `/content` — `sendContentMenu` reads `LatestNoteViews()`,
  which is nil until the initial index pass completes (visible as
  `latest noteloader: notes indexed ... total=N` in the server log). Hitting
  it before that log line is an unrelated nil-pointer crash, not a bug in this
  PR.
- If reusing a DB copy that already has fixture rows (e.g. from a previous
  manual setup), check `user_subgraph_accesses` for a **direct** grant — it
  bypasses the TG-membership path entirely and will make `/content` show
  content regardless of group join/leave, masking the very thing this test
  proves.

