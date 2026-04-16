# Telegram Chat Username Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Resolve public Telegram post links via cached chat usernames with stale-while-revalidate behavior, while preserving `t.me/c/...` fallback for private or unresolved chats.

**Architecture:** Add a read-through cache table plus a focused `internal/case/gettelegramchatname` package that reads cache, resolves misses via bot/account credentials, and refreshes stale entries in the background. `rendernotepage` will call this case when building Telegram links, and a cron job will periodically revalidate stale cache rows through the same resolution logic.

**Tech Stack:** Go, SQLite/sqlc, Telegram Bot API, gotd/td, existing `internal/case/*` and `internal/case/cronjob/*` patterns.

---

### Task 1: Add cache persistence

**Files:**
- Create: `db/migrations/20260416000000_create_telegram_chat_usernames.sql`
- Modify: `db/schema.sql`
- Modify: `internal/db/queries.sql`
- Modify: `internal/db/queries.read.sql.go`
- Modify: `internal/db/queries.write.sql.go`
- Modify: `internal/db/models.go`

- [ ] Add a new `telegram_chat_usernames` table with `telegram_chat_id`, `username`, `title`, `refreshed_at`, `refresh_requested_at`, `last_error`, and timestamps.
- [ ] Add read/write queries for get, upsert-success, mark-refresh-requested, mark-error, and stale-row listing.
- [ ] Regenerate sqlc output or update generated files consistently.

### Task 2: Implement `gettelegramchatname`

**Files:**
- Create: `internal/case/gettelegramchatname/resolve.go`
- Create: `internal/case/gettelegramchatname/resolve_test.go`
- Create: `internal/case/gettelegramchatname/mocks_test.go`

- [ ] Write failing unit tests for fresh cache hit, stale cache hit with refresh scheduling, cache miss resolved by bot, cache miss resolved by account, and unresolved/private fallback.
- [ ] Implement `Resolve(ctx, env, telegramChatID)` returning cached/public username metadata plus fallback status.
- [ ] Reuse bot credentials first, then account credentials, and persist successful/negative results in cache.

### Task 3: Integrate render path

**Files:**
- Modify: `internal/case/rendernotepage/resolve.go`
- Modify: `internal/case/rendernotepage/resolve_test.go`

- [ ] Add failing tests covering username-based public link generation and unresolved fallback to `t.me/c/...`.
- [ ] Replace direct URL assembly with `gettelegramchatname.Resolve`.
- [ ] Keep current frontmatter `telegram_publish_message_link` path unchanged.

### Task 4: Add housekeeping refresh

**Files:**
- Create: `internal/case/cronjob/refreshtelegramchatusernames/job.go`
- Modify: `cmd/server/cronjobs.go`

- [ ] Add a cron job that periodically picks stale cache rows and revalidates them through the same lookup path.
- [ ] Keep the job best-effort and bounded so it cannot block the rest of cron execution.

### Task 5: Verify

**Files:**
- Modify as needed from previous tasks

- [ ] Run targeted unit tests for `gettelegramchatname`, `rendernotepage`, and any new cron/job code.
- [ ] Run any needed formatting/generation steps and confirm generated artifacts are in sync.
- [ ] Document residual risk around private channels and temporary Telegram API failures in the final report.
