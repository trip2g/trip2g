# Telegram Rich Messages

Findings from research and live probing, July 2026. Everything below marked **measured** came from real
Bot API calls against a throwaway bot; everything marked **seen** was confirmed by opening the client.
Documented-but-unprobed claims are marked as such — the docs were wrong often enough to matter.

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

## Limits

| Limit | Value | Notes |
|---|---|---|
| Text | 32768 | 32768 passes, 32769 fails |
| Top-level blocks | 500 | 500 passes, 501 fails. **List items do not count** — a 501-item list is one block |
| Media | 50 | 50 passes, 51 fails. Re-measured identically in a channel |
| Formatted runs | ~5849 | `RICH_MESSAGE_TOO_LARGE` past it |
| Formatted-run cost | ~35000 units | **Silently truncates** past it, see below |

A rich message with 50 media is **one** message for rate-limit purposes, where album-plus-text costs several.

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

## What trip2g would build

**No library upgrade.** `go-telegram-bot-api/v5` is unmaintained since Bot API 6.0 and knows nothing about
rich messages — and does not need to. `BotAPI.MakeRequest(endpoint, params)` is what `Send`/`Request` already
call underneath; `sendRichMessage` is an ordinary JSON-over-HTTPS method. Rate-limit handling is inherited
unchanged, since `MakeRequest` returns the same `*tgbotapi.Error` that `telegram.HandleRateLimit` matches on.

Wrinkle: `tgbotapi.Chattable` cannot be implemented from outside the package (`params()` and `method()` are
unexported), so a rich send cannot drop into the existing `SendTelegramMessage(ctx, chatID, Chattable)`
signature. It needs its own `Env` method.

**gotd/td drops out of this project.** The six-layer upgrade (222 → 227+) is not a prerequisite for rich
messages. gotd stays for what it already does — publishing through user accounts, channel import — and note
that view statistics (`MessagesGetMessagesViews`, `StatsGetBroadcastStats`) exist only there; the Bot API
exposes no view counts at all.

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
- Whether the user-account path can send rich at all. TDLib marks the markdown/HTML source variants
  "for bots only" and does not mark the blocks variant, so a session serializing blocks by hand might work.
  Untested, and it needs the gotd upgrade first.
- Incoming rich messages via `getUpdates`/webhook — completely untested, and directly relevant to the
  inbox agent. Note `Message.text` is empty for a rich message, so any text-based readback sees nothing.
- Premium gating for human accounts (the composer shipped in Desktop 7.0; TDLib carries a
  `premiumFeatureRichMessages` constant).

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
