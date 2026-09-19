# Instant Tags: why they exist and why we keep them

Status: decided 2026-09-19 — **no change**. This note records the reasoning so
the question does not get re-opened from memory alone.

## The question

Instant Tags are a second tag axis on a Telegram chat: a note carrying one of
them is posted to that chat immediately on sync, ignoring the schedule. The
documented use is a preview channel — see what the post will look like before
it reaches the real one (`docs/en/user/telegram.md:84`).

The objection: this could have been a convention instead of a feature. Keep one
note for the demo channel and a copy of it for the real one, both scheduled
normally. Copy-paste is acceptable; two code paths are not.

## What instant actually does today

- `handletgpublishviews/resolve.go:157` enqueues an instant send on **every**
  sync of a note that has publish metadata. The comment there calls it a preview.
- `sendtelegrampublishpost/resolve.go:44` resolves a **different chat list** for
  instant sends.
- `sendtelegrampublishpost/resolve.go:99` skips marking the note published, so
  the later scheduled post still fires.
- Instant messages are never edited on re-sync: the update path filters
  `instant = 0` (`queries.read.sql:1044`), and the uniqueness index is partial
  (`unique(chat_id, note_path_id) where instant = 0`). Every sync produces a new
  message in the preview channel.
- A note still needs **both** `telegram_publish_at` and `telegram_publish_tags`
  to enter the pipeline at all (`handletgpublishviews/resolve.go:84-90`). Instant
  does not exempt a note from having a schedule.

## Why it exists

Two reasons, and the second one is the load-bearing one.

1. **Publish state is per note, not per chat.** The scheduler gates on
   `published_at is null` (`queries.read.sql:1004`). Adding a second tag to an
   already-published note therefore never delivers it to the newly matched
   channel — silently, forever. The instant path sidesteps this by not touching
   the published flag at all.
2. **"Show me the post now" needs a second axis.** `telegram_publish_at` is a
   single field shared by all of a note's tags. With one note and two tags there
   is no way to say "now for the preview channel, tomorrow for the real one".
   Something outside the note has to carry that difference. Instant Tags are
   that something.

Reason 2 is what kills the obvious simplifications. It was not the reason the
feature was originally built, but it is the reason it cannot simply be deleted.

## What it costs

- Two junction tables: `telegram_publish_instant_chats`,
  `telegram_publish_account_instant_chats`.
- An `instant` column on both sent-message tables, an `instant = 0` filter in
  roughly a dozen queries, and two partial unique indexes.
- Two admin use-case packages: `internal/case/admin/settgchatpublishinstanttags`,
  `internal/case/admin/settelegramaccountchatpublishinstanttags`.
- Two UI components: `assets/ui/admin/tgbot/show/publishtags/instanttags`,
  `assets/ui/admin/telegramaccount/show/dialogs/instanttags`.
- An `if params.Instant` fork in the send path — all of it doubled, because bot
  publishing and account publishing are parallel implementations.
- The E2E harness depends on it: `testdata/e2e_seed.sql:614,653` seed instant
  chats, and `cmd/tge2e` reads from them.

## Alternatives considered

### Duplicate note per channel

One note tagged for the preview channel with a past `publish_at`, a copy tagged
for the real channel. No new code; the whole instant axis could be deleted.

Rejected because:

- The preview note becomes a **real page on the site** — search, sitemap, RSS,
  link graph. It cannot be hidden: `hidden_by` on `note_paths` excludes a note
  from the site *and* from Telegram publishing alike (`queries.read.sql:1002`).
- Re-syncing a scheduled note **edits** the existing message rather than sending
  a fresh one, and Telegram edits are limited (a media group cannot be
  recomposed). Iterating on a media-heavy post is worse than today.
- Two copies drift, and "one note, many render targets" is the product's own
  pitch.

It does have the cheapest code, and it answers "see it now" naturally. That is
the honest case for it.

### Per-(note, chat) publish state

Move "published" off the note and onto the pair — the state already exists as
`telegram_publish_sent_messages(chat_id, note_path_id)`. The scheduler would
look for pairs with no sent row. This fixes reason 1 properly, instead of
routing around it.

Rejected on its own because it does nothing for reason 2: with a single
`publish_at` per note, the preview channel would still have to wait for the real
publish time. Worth keeping in mind if the added-tag bug ever needs a real fix,
but it is not a replacement for instant.

### Boolean `instant` flag on the chat

Keep the ordinary publish tags (they decide *what* goes to a chat) and add one
column deciding *when*: ignore the schedule. Combined with per-pair publish
state, the scheduler becomes "pairs with no sent row where `chat.instant = 1` or
`publish_at <= now`".

This is the only alternative that is genuinely smaller than what we have: it
drops both junction tables, both admin use cases and both UI components in
favour of a checkbox, while keeping one note and immediate preview. It still
leaves the bot/account duplication, and it forces a choice about whether instant
sends record a sent row — recording one means re-sync edits instead of
re-posting, which loses the current fresh-render behaviour.

Not pursued: it is a migration on live data and a rewrite of the admin "Telegram
posts" screen, in exchange for deleting code that already works.

### Per-tag schedule in frontmatter

`telegram_publish_at` as a tag→time map. Solves it, but changes the
user-facing frontmatter contract. Most expensive of all; not seriously
considered.

## Decision

Leave it. The feature works, the E2E suite is built on it, and every cheaper
shape trades code size for either a dirtier site, a worse preview loop, or a
data migration.

## Known limitation, unfixed

Adding a publish tag to a note that is already published does not deliver it to
the newly matched channel — `published_at is null` gates the whole note
(`queries.read.sql:1004`). Workaround: reset the note's published state, or
publish from a fresh note. The real fix is per-(note, chat) publish state, above.
