# Telegram Rich Messages

Findings from research and live probing, July 2026. Everything below marked **measured** came from real
Bot API calls against a throwaway bot; everything marked **seen** was confirmed by opening the client.
Documented-but-unprobed claims are marked as such — the docs were wrong often enough to matter.

## Status — 20 July 2026

Branch `feat/tgrich-account` carries the whole feature. **Both senders publish rich posts through the real
pipeline**, verified on the devstand by reading the blocks back off Telegram rather than trusting our own code:

| | State | Evidence |
|---|---|---|
| Bot path | works | `demo rich` message 107, 19 blocks, `post_type = 'rich'` |
| Account path (MTProto) | works, Premium required | `demo rich` message 116, 19 blocks, sent from account 1 |
| Edit safety | a rich post can no longer be flattened | message 109 survived two ordinary note edits with all 19 blocks |
| `gotd/td` | v0.161.0, layer 228 | account `dialogs` byte-identical before and after the bump |

A note opts in with `telegram_rich: on` in its frontmatter. `auto` deliberately means classic in V1 — the
classic converter only emits string warnings, which mix conversion loss with length and policy warnings and
so cannot be trusted as a predicate. `off` is classic.

### What is deliberately not built

- **Media in a rich post — the biggest gap, and it is a regression risk, not a missing extra.** The classic
  path sends images as an album. On the bot path the asset resolver is wired
  (`convertnoteviewtotgpost/rich.go`, via `NoteView.AssetReplaces`) but **has never been exercised**: no rich
  post with media has been published. An asset with no `AssetReplaces` entry becomes `LossUnresolvedMedia`
  and the image silently disappears. On the account path `ToPageBlocks` returns `ErrRichMediaUnsupported` and
  refuses the whole post — MTProto references media by id out of `InputRichMessage.Photos`/`.Documents` and
  does not ingest URLs, so it needs an upload step first. **Until this is closed, `telegram_rich: on` is a
  downgrade for any note with images.** Measure the bot path before building anything: it may already work.
- **e2e coverage.** `cmd/tge2e` cannot see rich messages: `extractNoteID` matches `^id: ` against message
  text, which is null for a rich post, and `MessageSnapshot` records only text/entities/media, so two
  structurally different rich posts snapshot identically. Adding a rich fixture without fixing this yields a
  test that passes while proving nothing. Now that gotd is on layer 228 the reader can see `rich_message`
  natively; before the bump the only route was forwarding a message through the bot to read it back.
- **Message splitting.** Nothing splits messages today; 32768 characters raises the ceiling roughly eightfold
  but a longer note still truncates. Needs durable multi-part identity first — the current idempotency key is
  `(note_path_id, chat_id)` with no part number, so a retry after part one either duplicates or drops.
- **Per-chat opt-in.** `tg_bot_chats.rich_messages_enabled` would add a kill switch. Frontmatter alone means
  any author can opt in on any bot chat.
- **Automatic table of contents.** Anchors are measured to defeat the *Show more* fold and heading IDs already
  exist, but inserting a TOC changes the author's post layout, which is a product decision.

### Next steps, in the order they are worth doing

1. **Publish a rich note with two or three images through the bot on the devstand.** Cheap, and it decides
   whether bot-side media is a test away or real work.
2. **Account-side media**: upload each asset via `uploader`, collect `InputPhoto`/`InputDocument` into the
   `InputRichMessage` slices, rewrite the block tree to carry the ids. Cache the ids per asset — a rich edit
   replaces the block tree rather than patching it, so an edit without replay loses the media.
3. **Make `cmd/tge2e` rich-aware**: note identity that does not depend on message text, and a snapshot that
   records the block-type sequence so a rich→plain regression fails.
4. Then splitting, the per-chat switch, and the TOC, in whatever order product wants.

## What it is

`sendRichMessage`, Bot API 10.1 (11 June 2026), with a typed `blocks` representation added in 10.2
(14 July 2026). Not an extension of `parse_mode`: a separate method taking an `InputRichMessage` with
exactly one of `markdown`, `html`, or `blocks[]`. No `parse_mode`, no `entities`, no `link_preview_options`.

Channels are supported targets. Premium is **not** required for bots — that was the one assumption capable
of killing the project, and it is refuted: a fresh bot on a non-Premium account sent ~40 rich messages,
including a full article into a channel on the first attempt.

## What we can rely on

**Measured**, in a private chat and again in a channel: headings 1–6 with the level preserved, GFM tables
with per-column alignment, task lists with checkbox state, nested lists with depth, fenced code with a
language tag, blockquotes, `marked` (real highlight, not the spoiler the classic path emits), photo and
video blocks in document order with server-side ingestion from plain HTTPS URLs, `<details>` with fold
state, anchors and footnotes, collage and slideshow, math in four notations.

**Seen** on screen: headings, tables, media in document order, collapsible sections, links and anchors.

**Documented, never probed**: maps, dividers, sub/superscript, audio, document and animation media,
16 levels of nesting, 20 table columns.

**Dead**: custom emoji. The server resolves the id and substitutes the sticker set's own fallback emoji;
only ordinary emoji appear on screen. Not channel-specific — a byte-identical payload collapses the same
way in a private chat.

## The account path: works, but requires Premium

Measured directly, not inferred. There is **no `sendRichMessage` in MTProto** — rich is a new optional field
`rich_message:flags.23?InputRichMessage` on the ordinary `messages.sendMessage#fef48f62` (same field on
`messages.editMessage` and `saveDraft`). Bots and users make the *same call*. Read from tdesktop's `api.tl`
(LAYER 228), byte-identical in TDLib's `telegram_api.tl`, and confirmed a third time by Telethon 1.44.0,
which exposes `rich_message` directly on `SendMessageRequest`. Telegram Desktop sends it from a plain user
session (`ApiWrap::sendRichMessage`, `apiwrap.cpp:4393`).

The block tree is Instant View's existing `PageBlock`/`RichText`, extended — so much of it is already in gotd
at layer 222. Input shape: `InputRichMessage(blocks, rtl, noautolink, photos, documents, users)`.

| Sender | Premium | Result |
|---|---|---|
| Bot | not required | works, channels included |
| User account | no | `400 RICH_MESSAGE_UNSUPPORTED` |
| User account | yes | works: saved messages, channel, and `send_as = channel` |

**The bot in that first row was created by the account in the second.** Both rows were measured in the same
session: `thailand_unknown` (id `7828312136`), non-Premium, was refused `RICH_MESSAGE_UNSUPPORTED` writing a
rich message to its own saved messages, and the bot it owns sent ~40 of them into a channel without a single
refusal. The entitlement is attached to the sending identity at send time and to nothing else — not to the
owner, not to the account that provisioned the bot. An owner therefore cannot post a rich message as himself
while his own bot posts them freely.

**The gate is Premium, enforced server-side**, and the entitlement follows the *sending user*, not the
posting identity: a non-Premium account is refused even writing to itself, while a Premium account succeeds
posting under the channel's identity. TDLib's "for bots only" annotation on the markdown/HTML source
variants is a comment, not a mechanism — `RichMessage.cpp:90` takes a `bool is_bot` and never reads it.

### Built and measured, July 2026

The account path is implemented and posted a real note through a Premium account session
(`internal/tgtd/richblocks.go`, `Client.SendRichMessage`). Three things were confirmed live and were not
knowable from the docs:

- **`rich_message_posting` is exactly `premium`**, read from a live `help.getAppConfig` on the devstand
  account, alongside all five limit keys at the values recorded below. The key names in this document are
  correct as written.
- **`PageBlockTable.Title` is required, not optional.** It is not a flag field, so a nil title fails to
  encode client-side and takes the whole message with it — surfacing as
  `unable to encode pageBlockTable#bf4dea82: field title is nil`, which names a field index rather than a
  block. Any `RichTextClass` field outside the flags needs an explicit `TextEmpty`. The cheap guard is a
  test that encodes a document exercising every block type; nothing else reads those fields.
- **The account path keys on the MTProto channel id**, not the Bot API form:
  `telegram_publish_account_chats.telegram_chat_id` must hold `4487679938`, not `-1004487679938`.
  `findChatInputPeer` compares against `tg.Channel.ID` directly.

Verified independently by forwarding the posted message with a bot that administers the channel and reading
`rich_message.blocks` off the Bot API response: 19 top-level blocks, headings at their original levels, a
4×3 table with per-cell `is_header`, a collapsed `details`, and `pre` with its language tag. `Message.text`
is null, as documented.

Two consequences for design. First, **capability is checkable before scheduling**: `help.getAppConfig`
returns `rich_message_posting` (`disabled` → `premium` since July 2026) and the account's own `premium` flag
is available — both reachable through calls that already exist in `internal/tgtd/client.go`. That turns
"account channels cannot do rich" into a precise per-account preflight with an honest error message.
Second, media differs: MTProto takes pre-uploaded `InputPhoto`/`InputDocument`, not the HTTPS URLs the Bot
API ingests server-side.

## Limits

| Limit | Value | Notes |
|---|---|---|
| Text | 32768 | 32768 passes, 32769 fails |
| Top-level blocks | 500 | 500 passes, 501 fails. **List items do not count** — a 501-item list is one block |
| Media | 50 | 50 passes, 51 fails. Re-measured identically in a channel |
| Formatted runs | ~5849 | `RICH_MESSAGE_TOO_LARGE` past it |
| Formatted-run cost | ~35000 units | **Silently truncates** past it, see below |

A rich message with 50 media is **one** message for rate-limit purposes, where album-plus-text costs several.

**The server advertises these limits — read them, do not hardcode them.** `help.getAppConfig` returns
`rich_message_length_limit` 32768, `rich_message_max_blocks` 500, `rich_message_max_media` 50,
`rich_message_max_depth` 16, `rich_message_max_table_cols` 20. They match the probed values exactly, and the
last two were never probed at all — the server is the better source. `GetAppConfig` already exists in
`internal/tgtd/client.go`.

### The visible fold

On screen a long post collapses behind a *Show more*. Undocumented; no API probe can find it.

Measured once, on one client: the preview ended at roughly **7,350 characters of visible text, 19 blocks
in**, and it cut at a paragraph boundary rather than mid-sentence. The next block (~1,700 characters) was
held back whole, so the real threshold sits somewhere between where it cut and where that block would have
carried it — call it **7,500–9,000**. Unknown whether Telegram counts characters, blocks, or rendered
height, and whether desktop and mobile agree.

The consequence is a layout rule, not a mystery: keep what must be seen inside ~7,000 characters, and
watch large tables or code blocks near the boundary — they are not split, they are hidden entirely.

**Anchors defeat the fold.** Tapping a contents entry that points below the fold expands the post
(confirmed on screen). A generated table of contents at the top is therefore the answer to the fold, and
trip2g already computes the input: `extractHeadingsAndGenerateIDs` (`internal/model/note.go`) stamps an ID
onto every heading, and `assets/toc/src/index.ts` is the web-side subsystem built on it.

## Failure modes to code around

All of these return `ok:true`.

1. **Content can be discarded silently.** `**bold**` × 5000 came back with 4374 runs; 40000 emoji came back
   as 8749 characters. No error, no flag. **Verify after every send**: compare returned length and run
   count against what was submitted. There is no other way to notice.
2. **The edit path replaces, it never patches.** `editMessageText` with a `text` field keeps the
   `message_id` and drops every block; Telegram retains nothing, so our stored source is the only copy.
   A `rich_message` edit is no gentler — sending back heading-plus-paragraph on a post that carried a video
   silently dropped the video. Cache each asset's `file_id` and replay it in every edit; `file_id` is global
   across chats, so nothing re-uploads.
3. **Inline media loses its caption.** Media mid-sentence is accepted, splits the paragraph, and drops the
   caption text. Normalise media onto its own line.
4. **A line-leading `#tag` with no space becomes an H1.** Not GFM. In a vault where tag-first lines are
   routine this silently promotes them and eats the `#`. `\#` suppresses the heading but yields a hashtag
   entity, not literal text.
5. **Auto-detection is on by default** and broader than the classic parse modes: `$USD` becomes a cashtag,
   any bare 16-digit run becomes a bank card number. Set `skip_entity_detection: true`.
6. **Exactly-one-of is not enforced.** `markdown` + `html` + `blocks` all return `ok:true`; precedence is
   `blocks` > `markdown` > `html` and the losers are discarded. An empty `blocks: []` therefore silently
   swallows a populated `markdown`.
7. **Tabs inside fenced code collapse to one space.** This is a Go codebase and gofmt indents with tabs —
   expand tabs in the renderer.
8. **Dangling anchors degrade silently** to an ordinary url. Validate TOC targets ourselves.

### Error strings

All `400`, prefixed `Bad Request: `. Casing is inconsistent — do not assume SCREAMING_SNAKE:

```
RICH_MESSAGE_TEXT_TOO_LONG      RICH_MESSAGE_TOO_LARGE
RICH_MESSAGE_BLOCKS_TOO_MANY    RICH_MESSAGE_MEDIA_TOO_MANY
RICH_MESSAGE_EMPTY              RICH_MESSAGE_EMOJI_INVALID
RICH_MESSAGE_PHOTO_NO_MEDIA_FOUND   ← upstream fetch failed, not a validation error
rich message must be non-empty      ← distinct from RICH_MESSAGE_EMPTY
message is not modified             ← treat as a no-op success
```

`404 Not Found` vs `400 Bad Request: <reason>` cleanly separates "method missing on this deployment" from
"method exists, input rejected" — useful for a capability probe at startup.

## Where the docs are wrong

Worth knowing, because a reviewer reading the docs will reach the wrong conclusion:

- The docs say entities cannot nest. In the **classic** HTML path, `<pre>` inside `<blockquote>` is accepted
  with both entities intact, and `<blockquote>` inside `<blockquote>` is silently flattened rather than
  rejected. Neither returns an error. (Measured.)
- `sendRichMessage` silently ignores every unknown top-level parameter, so you cannot probe it for supported
  parameters by looking for errors. Only observable effects in the response count.
- The wire discriminator for a heading is `"heading"`, not the reference type name `RichBlockSectionHeading`.

## The block contract, measured

Probed field by field in July 2026 against the live bot, after a first implementation built from the
documented names was rejected outright. Everything below is echoed back by the server.

| Construct | Type | Fields |
|---|---|---|
| Heading | `heading` | `text`, `size` 1–6 |
| Paragraph | `paragraph` | `text` |
| Fenced code | `pre` | `text` carries the code, `language` |
| Quote | `blockquote` | `blocks` |
| Collapsible | `details` | `summary`, `blocks`, `is_open` |
| Table | `table` | `cells` — row-major, `caption` |
| List | `list` | `items`, each `{blocks, checked}` |

Three names in the earlier draft were wrong and rejected with
`can't parse InputRichBlock: type "…" is unsupported`: `code` is `pre`, `quote` is `blockquote`, and
`collapsible` is `details`. Fold state is `is_open` — `open`, `collapsed`, `expanded`, `folded`,
`is_collapsed`, `default_open` and `closed` are all accepted and silently ignored, and absent means
collapsed.

**A table is not rows of cells.** `cells` sits on the block as a row-major grid, `[[cell, cell], …]`. There is
no column descriptor at all, so alignment is per cell; a row object carrying its own `cells` is rejected. The
header marker is `is_header` on the cell — `header` is ignored — and it switches the server's default
alignment to centre.

**Rich text is a tagged union too, and this is the trap.** A text object without a `type` is parsed as a
*block*, and the error names the block: `can't parse InputRichBlock: Can't find field "type"`, pointing at
entirely the wrong part of the payload. The forms are a bare string for plain text, a JSON array for
concatenation, `{"type":"bold","text":…}` for a mark nesting through `text`, `{"type":"url","text":…,"url":…}`
for a link, and `{"type":"anchor","name":…}` for an anchor target. Marks nest rather than combine, and the
strikethrough mark is `strikethrough`, not `strike`.

**Anchors exist, but not where the earlier draft looked for them.** A heading takes no `anchor`, `name` or
`id` — all three are accepted and none is echoed. An anchor is a rich-text node of its own, and it carries a
name and no text. The table-of-contents answer to the visible fold is therefore still reachable, but it is
built from anchor nodes placed in text, not from a field on the heading.

**Ordered lists do not survive.** `ordered`, `start`, `style`, `is_ordered`, `list_type` and `numbered` are
all ignored, and a per-item `label` is overwritten: every item comes back labelled `•`. A numbered list
renders as bullets.

### The echo check needs to count the right thing

The server merges adjacent plain runs — `["a","b"]` comes back as `"ab"` — so a raw run count drops on almost
every message that submitted a run of unformatted spans, with nothing lost. Comparing it reports truncation
that did not happen: the first real post came back 83 runs against 65 with byte-identical text length.
Compare **blocks, formatted runs and text units**, where a formatted run is one carrying a mark or a link.
That is immune to the merge and still catches the documented failure, which is formatted runs disappearing
past the run-cost ceiling.

## What trip2g would build

**No library upgrade.** `go-telegram-bot-api/v5` is unmaintained since Bot API 6.0 and knows nothing about
rich messages — and does not need to. `BotAPI.MakeRequest(endpoint, params)` is what `Send`/`Request` already
call underneath; `sendRichMessage` is an ordinary JSON-over-HTTPS method. Rate-limit handling is inherited
unchanged, since `MakeRequest` returns the same `*tgbotapi.Error` that `telegram.HandleRateLimit` matches on.

Wrinkle: `tgbotapi.Chattable` cannot be implemented from outside the package (`params()` and `method()` are
unexported), so a rich send cannot drop into the existing `SendTelegramMessage(ctx, chatID, Chattable)`
signature. It needs its own `Env` method.

**gotd/td is not a prerequisite for the bot path**, and the upgrade is far cheaper than first estimated.
An earlier draft called it "six layers of breaking churn across ~1800 lines, XL, its own project". Measured
instead of guessed — the repo was built against gotd v0.161.0 (layer 228) in a scratch copy — it is
**4 compile errors in 3 files, all in `internal/tgtd`**: `telegram.Options.Logger` became a `gotd/log`
interface (2 sites) and `tg.PollAnswer` became `tg.PollAnswerClass` (2 sites). The other gotd-importing
packages compile clean. Rich types first appear in v0.150.0 (layer 227).

gotd also remains the only route to view statistics (`MessagesGetMessagesViews`,
`StatsGetBroadcastStats`) — the Bot API exposes no view counts at all.

**Emit `blocks[]`, not markdown.** Both produce byte-identical output on the same document, but they fail in
opposite directions: markdown fails silently (paragraph splits, dropped captions, `#tag` promotion), while
`blocks[]` fails loudly and names the offending field. For a pipeline rendering someone else's notes, loud
is the only acceptable property. We already walk a goldmark AST, so AST → typed blocks is shorter,
type-safe, and deletes an escaper we would otherwise have to write and test.

Shape: a package `internal/tgrich` for wire types, limits and transport, plus `rich_converter.go` as a
sibling of `HTMLConverter`, which stays untouched — the classic path, the navigation bot and canvas all
still need it.

**Obsidian covers most of the block model already.** Roughly 80% of the rich types are reachable from plain
markdown with no new syntax; the gap is converter work against an AST that already holds everything.
Collapse-by-default is fully native (`> [!x]-` / `> [!x]+` → `Foldable`/`Expanded`, already parsed, already
mapped to `<details>` on the web). An image gallery has no Obsidian syntax; the cheapest convention is a
`> [!gallery]` callout, which parses today with no parser change and degrades to a titled box in vanilla
Obsidian.

## Decisions taken

- **Frontmatter key: `telegram_rich: auto | on | off`**, `auto` by default. `true`/`false` also accepted
  (frontmatter lands in an untyped `RawMeta`, so tolerating both costs a helper). `enable`/`disable` rejected
  as spellings nobody types.
- **`auto` means "the classic conversion lost something."** The converter already emits a warning whenever it
  drops a node it cannot represent, and those warnings already reach `post.Warnings` and the admin dashboard.
  The predicate should read a **typed** set of dropped node types, not match on warning strings — the string
  list mixes in length-limit and unsupported-image warnings.
- **`on` against a chat that publishes through a user account is an error**, not a silent downgrade. It
  writes `telegram_publish_notes.last_error` via `SetTelegramPublishNoteLastError`, which both surfaces in
  the admin (`errorCount`/`lastError` in the GraphQL schema) and removes the note from scheduling — both
  publish queues filter on `last_error is null`. Correct semantics for a config error that will not fix
  itself by retrying.
- **Rollout is per-chat opt-in**, not a global switch, because old-client rendering is undocumented.
- Migration is kind: plain → rich upgrade works in place, same `message_id`, so nothing needs reposting and
  no permalink breaks. Forward and copy both preserve the block tree.

## Open questions

- The exact fold threshold and what it counts. One measurement, one client.
- Old-client fallback. Telegram Web showed a hard "not supported" card in June 2026; nothing newer.
- Incoming rich messages via `getUpdates`/webhook — completely untested, and directly relevant to the
  inbox agent. Note `Message.text` is empty for a rich message, so any text-based readback sees nothing.

## Known latent bug

`renderCustomEmoji` in `internal/markdownv2/html_converter.go` writes the Obsidian image alt text straight
into the `<tg-emoji>` tag body, and three cases in `html_converter_test.go` assert that shape. A *word* in
that slot returns `400 RICH_MESSAGE_EMOJI_INVALID` and kills the entire post. Empty alt is fine, a real emoji
character is fine. **Latent, not live** — the classic HTML path accepts that payload and ships today. It
fires the moment that alt text reaches `sendRichMessage`, so the rich renderer must not inherit the pattern,
nor the assertion with it.

## Raw evidence

The full probe transcripts (~250 live calls) and the Obsidian coverage matrix were working artifacts and are
not committed. The numbers above are the distilled result; re-probing is cheap if a claim needs recheck.
