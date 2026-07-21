# Telegram Rich Messages: code review and delivery plan

This is a repository-grounded review of `telegram_rich.md`.  “Observed” means
verified in this checkout; Telegram protocol behaviour in the research note is
not independently verifiable here because the raw probes are deliberately not
committed.  File references below are to this checkout.

## Part 1 — critique

### Critical findings

1. **The proposed in-place upgrade is incompatible with the implemented edit
   path.**  The document says a plain post can be upgraded to rich “in place”
   with the same `message_id`.  Observed: an existing published bot message is
   re-rendered through the classic converter, compared by a SHA-256 of
   `post.Content`, and then edited solely with `editMessageText` or
   `editMessageCaption`, both with `ParseMode = HTML`
   ([updatetelegrampublishpost/resolve.go:46](/home/alexes/projects2/trip2g/internal/case/updatetelegrampublishpost/resolve.go:46),
   [updatetelegrammessage/resolve.go:105](/home/alexes/projects2/trip2g/internal/case/backjob/updatetelegrammessage/resolve.go:105),
   [updatetelegrammessage/resolve.go:132](/home/alexes/projects2/trip2g/internal/case/backjob/updatetelegrammessage/resolve.go:132)).
   It has neither a rich edit request nor enough persisted state to select one.
   The research note itself reports that this classic edit destroys blocks.  Do
   not promise migration-in-place until a real rich edit endpoint is probed and
   implemented, including media replay semantics.  V1 should leave old classic
   records classic; rich applies only to new messages (or an explicit reset and
   repost).

2. **“Per-chat opt-in” does not exist in the schema or the current selection
   model.** `tg_bot_chats` contains identity, title, lifecycle and bot ID only
   ([schema.sql:389](/home/alexes/projects2/trip2g/db/schema.sql:389)); the
   publishing mapping is tag-to-chat, not a place for a mode
   ([schema.sql:475](/home/alexes/projects2/trip2g/db/schema.sql:475)).  The
   conversion source does receive the internal chat ID
   ([sendtelegrampublishpost/resolve.go:63](/home/alexes/projects2/trip2g/internal/case/sendtelegrampublishpost/resolve.go:63)),
   so it is feasible, but it requires a migration and generated query work.
   The document calls no migration out and gives no admin/config surface.

3. **The existing idempotency key is insufficient for multi-message splitting.**
   A send checks only `(note_path_id, chat_id)` and treats any record as “already
   sent” ([queries.read.sql.go:1116](/home/alexes/projects2/trip2g/internal/db/queries.read.sql.go:1116),
   [sendtelegrammessage/resolve.go:74](/home/alexes/projects2/trip2g/internal/case/backjob/sendtelegrammessage/resolve.go:74)).
   The scheduled unique index uses the same pair
   ([schema.sql:499](/home/alexes/projects2/trip2g/db/schema.sql:499)).  There
   is no splitting today: classic text is truncated to one message
   ([sendtelegrammessage/resolve.go:104](/home/alexes/projects2/trip2g/internal/case/backjob/sendtelegrammessage/resolve.go:104)).
   Rich’s 32k limit is not a solution for longer notes.  A retry after sending
   part 1 but before recording the group needs an ordered part identity and a
   transactional/outbox strategy, otherwise it either duplicates or drops the
   remainder.

### High findings

4. **`auto = classic conversion lost something` is not a reliable predicate.**
   The classic `ConverterResult` contains only string warnings, content and
   assets ([converter.go:23](/home/alexes/projects2/trip2g/internal/markdownv2/converter.go:23)).
   `HTMLConverter` does not produce a typed dropped-node set; it also silently
   skips embedded wikilinks ([html_converter.go:380](/home/alexes/projects2/trip2g/internal/markdownv2/html_converter.go:380)).
   Its warnings include conversion failures plus policy/length warnings appended
   later ([convertnoteviewtotgpost/resolve.go:205](/home/alexes/projects2/trip2g/internal/case/convertnoteviewtotgpost/resolve.go:205),
   [convertnoteviewtotgpost/resolve.go:217](/home/alexes/projects2/trip2g/internal/case/convertnoteviewtotgpost/resolve.go:217)).
   The claim that converter warnings “already reach `post.Warnings`” is true;
   the claim that they already reach an admin dashboard is unverified from the
   named code.  The code shown maps only note-loader warnings into push/commit
   responses, not post warnings ([commitnotes/resolve.go:29](/home/alexes/projects2/trip2g/internal/case/commitnotes/resolve.go:29)).

   Recommendation: default `auto` to **classic** in V1, and make `on` explicit
   for rich-capable bot chats.  Later, define a separate AST capability analysis
   with typed `Loss` values and expose its result; do not infer it from warning
   strings.  This is safer and avoids format changes merely because a classic
   renderer warning happened to be emitted.

5. **Making `on` a permanent scheduling error for a user-account destination
   has the wrong granularity.** It is correct not to silently downgrade an
   explicit request.  But `last_error` belongs to the note, not a destination:
   setting it after one account chat fails blocks *every* scheduled destination.
   Both scheduled bot and account selectors filter `last_error is null`
   ([queries.read.sql.go:5501](/home/alexes/projects2/trip2g/internal/db/queries.read.sql.go:5501),
   [queries.read.sql.go:5462](/home/alexes/projects2/trip2g/internal/db/queries.read.sql.go:5462)).
   A single unsupported account tag could therefore suppress valid bot publishes.
   V1 should reject `telegram_rich:on` during dispatch with a clear per-chat
   result, but should not use `telegram_publish_notes.last_error` for this until
   errors are destination-scoped.  If product requires a hard authoring error,
   validate it before scheduling against all selected destinations.

6. **`blocks[]` is a sound target representation, but “shorter/type-safe” is
   overstated.** It avoids rich-Markdown ambiguity, but the existing renderer is
   not a simple node-to-string walk: it buffers blockquotes/callouts
   ([html_converter.go:278](/home/alexes/projects2/trip2g/internal/markdownv2/html_converter.go:278)),
   resolves wiki links with scheduled-post footer side effects
   ([html_converter.go:380](/home/alexes/projects2/trip2g/internal/markdownv2/html_converter.go:380)),
   and separates all note assets from the AST then attaches them as a caption or
   media group ([convertnoteviewtotgpost/resolve.go:211](/home/alexes/projects2/trip2g/internal/case/convertnoteviewtotgpost/resolve.go:211)).
   A rich renderer must deliberately replace, preserve, or reject each of those
   semantics.  Use `blocks[]`, but build a standalone rich converter with a
   typed result and an explicit link resolver; do not pretend it can be a
   sibling copy of `HTMLConverter`.

7. **“No library upgrade” is only partly verified.** The app sends through
   `handlerIO.Send/Request`, not directly through a visible `MakeRequest`
   ([cmd/server/telegram.go:262](/home/alexes/projects2/trip2g/cmd/server/telegram.go:262)).
   The research assertion about unexported `Chattable` methods is plausible but
   unverified here.  The implementation must add an app-layer method that calls
   the bot client’s supported raw request facility, serializes JSON exactly
   once, and decodes the response.  Do not make the transport package depend on
   `tgbotapi.Chattable`.

### Medium findings and stale/unverified references

8. **The heading reference is substantially right but incomplete.** Heading IDs
   are stamped onto AST heading nodes in
   `extractHeadingsAndGenerateIDs` ([note.go:850](/home/alexes/projects2/trip2g/internal/model/note.go:850)),
   after the loader has built the note ([note.go:589](/home/alexes/projects2/trip2g/internal/model/note.go:589)).
   A rich TOC is nevertheless not “the answer” yet: there is no Telegram TOC
   generator, no validation of anchors, and the client fold claim is external
   evidence only.  Generated IDs are transliterated and normalised; preserve the
   existing `id` attribute, do not recompute it ([note.go:862](/home/alexes/projects2/trip2g/internal/model/note.go:862)).

9. **Callout facts are correct, gallery semantics are not designed.** The
   parser really rewrites `[!type]-/+` blockquotes to `callout.Node` with
   `Foldable` and `Expanded` ([callout/parser.go:75](/home/alexes/projects2/trip2g/internal/mdloader/callout/parser.go:75),
   [callout/node.go:13](/home/alexes/projects2/trip2g/internal/mdloader/callout/node.go:13)).
   But every callout type currently has the same renderer behaviour; no
   `gallery` convention exists in code ([html_converter.go:299](/home/alexes/projects2/trip2g/internal/markdownv2/html_converter.go:299)).
   Do not ship a magic `gallery` callout until its asset ordering, mixed text,
   validation, and classic fallback are specified.

10. **Custom-emoji warning is accurately located but not automatically a rich
    send failure.** `renderCustomEmoji` emits `<tg-emoji>` with stripped alt
    text ([html_converter.go:347](/home/alexes/projects2/trip2g/internal/markdownv2/html_converter.go:347)).
    Rich blocks will not reuse that HTML output if the recommended blocks design
    is followed.  The stated Bot API result remains unverified from the repo;
    V1 should render custom emoji as the ordinary, stripped alt/fallback text
    (or return a typed loss), never send its ID.

11. **The proposed package boundary is conflated.** `internal/tgrich` should
    hold pure wire types, validation and JSON helpers; app-layer network IO
    belongs in `cmd/server/telegram.go` under the project rule in
    `app_patterns.md` ([app_patterns.md:74](/home/alexes/projects2/trip2g/docs/dev/app_patterns.md:74)).
    The AST converter belongs in `internal/markdownv2` because it shares node
    knowledge and link resolution with the existing converters.  A stateful
    service package is not needed for V1.

### Gaps that must be designed before implementation

- **Splitting:** decide deterministic block-boundary chunks, how to handle an
  oversize indivisible block, part ordering, and update/repost policy.  Never
  revive classic `TruncateContent`: it removes unmatched HTML tags and appends
  `...`, which is inapplicable to blocks
  ([utils.go:82](/home/alexes/projects2/trip2g/internal/telegram/utils.go:82)).
- **Content identity:** current storage saves one HTML string, one hash and a
  three-value post type ([schema.sql:499](/home/alexes/projects2/trip2g/db/schema.sql:499),
  [enums.go:3](/home/alexes/projects2/trip2g/internal/db/enums.go:3)).  It
  cannot describe rich JSON, chunks, asset replay data, or rich-vs-classic
  updates.  Hash canonical JSON plus relevant send options, not a Go struct’s
  incidental encoding.
- **E2E snapshots:** `cmd/tge2e` extracts note identity from `Message.Message`
  ([snapshot.go:18](/home/alexes/projects2/trip2g/cmd/tge2e/snapshot.go:18))
  and snapshots only text/entities/basic media ([snapshot.go:53](/home/alexes/projects2/trip2g/cmd/tge2e/snapshot.go:53)).
  The research note says rich `Message.text` is empty; the existing snapshot
  code will lose identity and report meaningless equality.  Add a rich-aware
  probe/dump fixture before treating snapshots as release coverage.
- **Import direction:** imports and account publishing exist, but no named code
  establishes rich inbound decoding.  The document correctly labels it
  untested.  Keep the inbox/import direction out of V1 and ensure it does not
  overwrite a rich-origin note with an empty `Message.text`.
- **Scheduled versus instant:** scheduled bot dispatch marks the note published
  immediately after enqueue, not after delivery
  ([sendtelegrampublishpost/resolve.go:99](/home/alexes/projects2/trip2g/internal/case/sendtelegrampublishpost/resolve.go:99));
  instant uses a different chat list and does not mark it
  ([sendtelegrampublishpost/resolve.go:43](/home/alexes/projects2/trip2g/internal/case/sendtelegrampublishpost/resolve.go:43)).
  Rich capability/config failures need to be surfaced before that scheduled
  mark, or have a recovery path.
- **Already-published classic posts:** retain legacy edit behaviour.  Do not
  cross-format edit; offer reset/repost only after confirmation, because it
  changes message IDs and links.

### Risk ranking

| Rank | Failure likely to bite first | Why | Adequate mitigation? |
|---|---|---|---|
| 1 | Existing update job destroys a rich post | Normal note edits automatically enqueue updates; code always sends classic edit requests. | No. This must be a launch gate. |
| 2 | One account/config failure blocks a scheduled note everywhere | `last_error` is note-wide while destinations are many. | No. Preflight or destination-scoped state is required. |
| 3 | Partial rich send/retry corrupts a split post | Current idempotency is one row per note/chat and has no part number. | No. Design storage and retry before splitting. |
| 4 | Unsupported/old client rendering | Per-chat rollout narrows blast radius. | Partly; require a manual acceptance checklist and a kill switch. |
| 5 | Silent Telegram truncation | Research reports it, but response comparison schema is unverified and likely cannot expose every rendered field. | Not yet. Enforce conservative local limits and record canonical submitted payload; add a targeted live probe. |
| 6 | Markdown quirks (`#tag`, captions, detection) | Avoided by blocks. | Yes, if blocks are used exclusively and `skip_entity_detection` is sent. |

## Part 2 — execution plan

The plan accepts the critique: bot-only V1, explicit `telegram_rich:on`,
classic-by-default `auto`, blocks not rich Markdown, no format conversion of
existing posts, no splitting until durable multi-part storage is approved.
Every phase is independently shippable.  Follow project TDD: table-driven
`testify/require` tests and generated `moq` Env mocks before implementation
([CLAUDE.md:78](/home/alexes/projects2/trip2g/CLAUDE.md:78)).

### Phase 0 — prove and freeze the wire contract (no production behaviour)

**Tests first**

- Add `internal/tgrich/types_test.go`: JSON fixtures for the smallest accepted
  heading, paragraph/runs, list/task, quote/details, code, table, link/anchor
  and photo; assert discriminator spelling, omission rules, and exactly one
  content source.
- Add `internal/tgrich/validate_test.go`: empty message, 32,768/32,769 UTF-16
  text units, 500/501 top-level blocks, 50/51 media, dangling anchors,
  unsupported custom emoji and invalid URLs.  Mark values derived from the
  research note as probe fixtures, not protocol constants, until CI/live
  validation is available.
- Add a manual `cmd/tge2e rich-probe` command plus a checked-in, redacted JSON
  expectation format.  It must send/read one representative rich message and
  record only observable response fields.  Do not alter regular snapshots yet.

**Implementation**

- Create `internal/tgrich/types.go`, `limits.go`, `validate.go`, and
  `request.go`.  Types should be closed Go structs with JSON tags, not
  `map[string]any`:

  ```go
  type Request struct { ChatID int64 `json:"chat_id"`; RichMessage InputRichMessage `json:"rich_message"`; DisableNotification bool `json:"disable_notification,omitempty"` }
  type InputRichMessage struct { Blocks []Block `json:"blocks"`; SkipEntityDetection bool `json:"skip_entity_detection"` }
  type Block struct { Type string `json:"type"`; Heading *Heading `json:"heading,omitempty"`; Paragraph *Paragraph `json:"paragraph,omitempty"`; List *List `json:"list,omitempty"`; Quote *Quote `json:"quote,omitempty"`; Code *Code `json:"code,omitempty"`; Table *Table `json:"table,omitempty"`; Media *Media `json:"media,omitempty"`; Details *Details `json:"details,omitempty"` }
  type Run struct { Text string `json:"text,omitempty"`; Bold, Italic, Underline, Strike, Marked, Code bool `json:"...,omitempty"`; Link *Link `json:"link,omitempty"` }
  ```

  Replace shorthand booleans with the exact documented schema after the probe;
  `Block.Validate` must enforce one variant and `InputRichMessage.Validate`
  must enforce blocks-only.  Do not claim a wire type is final without the raw
  contract.
- Add a capability record in the probe output only; no runtime capability probe
  or user-facing option yet.  The app transport is deliberately deferred until
  request/response decoding is known.

### Phase 1 — pure AST-to-block conversion and frontmatter parsing (no sending)

**Tests first**

- Add `internal/model/note_telegram_test.go` cases for absent/`auto`/`on`/`off`,
  booleans, invalid type/value.  Return `(mode, error)`; do not silently treat a
  typo as `off`.
- Add `internal/markdownv2/rich_converter_test.go` table cases built through
  the real loader: headings retain `id`, text/soft break, emphasis/strong,
  strike, code span, highlight, link/autolink, fenced code (tabs expanded),
  ordered/nested/task lists, quote, callout fold state, GFM table alignment,
  unsupported raw HTML, wikilink resolutions and custom emoji fallback.
- Assert typed losses (`LossUnsupportedNode`, `LossEmbeddedWikiLink`,
  `LossCustomEmoji`) rather than matching warning text; also test that `auto`
  remains classic regardless of losses in V1.

**Implementation**

- Add `NoteView.ExtractTelegramRichMode()` to
  `internal/model/note_telegram.go`; `TelegramRichMode` constants live there.
- Add `markdownv2.RichConverter` in
  `internal/markdownv2/rich_converter.go`, with `SetLinkResolver` matching the
  existing resolver API and `Process(*model.NoteView) RichConverterResult`.
  Its result has `Blocks []tgrich.Block`, `Losses []RichLoss`,
  `VisibleUTF16Length int`, and `Anchors map[string]struct{}`.
- Node mapping: `ast.Document` emits children; `ast.Heading` → `heading` with
  `Level`, node `id` and inline runs; `ast.Paragraph` → `paragraph`; `ast.Text`
  → text/newline run; `ast.Emphasis` level 1/2 → italic/bold run scope;
  `extast.Strikethrough` → strike; `highlight.HighlightAST` → marked;
  `ast.CodeSpan` → code; `ast.Link`/`AutoLink` → linked runs; `ast.FencedCodeBlock`
  → code block with language and expanded tabs; `ast.List`, `ListItem`,
  `TextBlock`, `extast.TaskCheckBox` → list block/items with ordered/start and
  checked state; `ast.Blockquote` → quote; `callout.Node` → details when
  foldable, otherwise a quote with title/body; `extast.Table`,
  `TableHeader`, `TableRow`, `TableCell` → table/cells/alignment.  Mapping must
  use the actual extension node types (task checkboxes are extracted as
  `*extast.TaskCheckBox` in [note_tasklist.go:38](/home/alexes/projects2/trip2g/internal/model/note_tasklist.go:38)).
- Treat images/enclaves, raw HTML other than a specifically supported subset,
  embedded wikilinks, thematic breaks, math, gallery, collage/slideshow, and
  all media-in-paragraph cases as typed losses in this phase.  Do not silently
  hoist `NoteView.Assets`: their map iteration has no document order
  ([convertnoteviewtotgpost/resolve.go:264](/home/alexes/projects2/trip2g/internal/case/convertnoteviewtotgpost/resolve.go:264)).

### Phase 2 — select rich per destination and send one rich message

**Tests first**

- Extend conversion/use-case tests using `moq` Env mocks: `off`/`auto` produce
  the current classic post; `on` produces a rich post only for a bot chat marked
  capable; `on` plus account destination returns a preflight error before any
  scheduled note is marked published; mixed destinations have deterministic,
  reported outcomes.
- Add `internal/tgrich/transport_test.go` with `httptest` or a narrow bot-client
  fake: method name, JSON body, `skip_entity_detection`, response `message_id`,
  non-OK Telegram error and rate-limit wrapping.
- Extend `sendtelegrammessage` tests: rich sends bypass HTML truncation and
  `tgbotapi.Chattable`; existing classic cases and retry behaviour remain
  unchanged.  Verify a rich message is inserted with its canonical payload
  hash.

**Implementation**

- **Migration decision required — ask the user before creating it**, per
  [CLAUDE.md:13](/home/alexes/projects2/trip2g/CLAUDE.md:13).  Proposed migration:
  add `rich_messages_enabled integer not null default 0` to `tg_bot_chats`; add
  `format text not null default 'classic'` and `rich_payload text not null
  default ''` to `telegram_publish_sent_messages`.  Regenerate sqlc after
  updating source query files (`make sqlc`).  The minimum is format + canonical
  payload; no media/file-ID column in the no-media V1.
- Extend `model.TelegramPost` with a tagged union (`FormatClassic` content/media
  versus `FormatRich` blocks) and extend `TelegramSendPostParams` without
  breaking jobs already serialized as classic.
- Modify `convertnoteviewtotgpost.Resolve` to read chat capability through a
  new Env method (for example `TelegramRichEnabled(ctx, chatID) (bool, error)`),
  choose explicit `on` only, and call `RichConverter`.  Keep its current
  classic path exactly intact.
- Add `SendTelegramRichMessage(ctx, dbChatID int64, req tgrich.Request)
  (tgrich.SendResult, error)` to the `sendtelegrammessage.Env`; implement it on
  `*app` in `cmd/server/telegram.go`.  That app method resolves `TgBotChat` and
  handler IO as `SendTelegramMessage` already does, performs the raw Bot API
  call and decodes it.  Add compile-time Env assertions beside the existing
  server checks.  `internal/tgrich` stays network-free.
- Modify `backjob/sendtelegrammessage.resolve` to dispatch the union.  Continue
  using `telegram.HandleRateLimit`; it matches error text rather than a concrete
  error type ([utils.go:11](/home/alexes/projects2/trip2g/internal/telegram/utils.go:11)).
  Persist canonical rich JSON and a hash of `format + payload + send options`.
- Do not call `SetTelegramPublishNoteLastError` for an unsupported account in a
  mixed rollout.  Return a preflight error before enqueue/mark, or add
  destination-scoped status in a later approved migration.

### Phase 3 — safe editing, e2e observability, and guarded rollout

**Tests first**

- Add update-job tables: classic stored format uses existing edit request;
  rich stored format uses the confirmed rich edit endpoint; cross-format edit
  returns a typed “reset/repost required” outcome; unchanged canonical hashes
  perform no request; API “not modified” is idempotent success.
- Add `cmd/tge2e` snapshot tests for rich identity and structure.  Replace the
  text-first-line identity convention with test metadata outside the rich
  message (or a mapped message ID manifest), and include rich block summary,
  anchors, details and media order.  Keep classic snapshots compatible.
- Add manual release cases: rich enabled/disabled chat pair, Android/iOS/Desktop
  client check, scheduled and instant bot posts, a link below fold, rollback by
  disabling the chat, and a regular edit to every rich fixture.

**Implementation**

- Only after an endpoint is live-probed, add `tgrich.EditRequest` and
  `SendTelegramRichEdit` app/Env methods; route
  `backjob/updatetelegrammessage.Resolve` by persisted `format`.  Rich edits
  must resend the complete canonical block tree, not translate it through HTML.
- Add an admin action/config backed by `tg_bot_chats.rich_messages_enabled` and
  an audit/log entry for each decision.  Rollout begins with one bot chat;
  disabling it affects new sends only and never rewrites old messages.
- Add a capability/version check only if the Phase-0 probe yields a stable,
  actionable signal.  A 404 fallback must be a visible operational error for
  `on`, never an automatic downgrade.

### Phase 4 — separately approved multi-part rich posts

**Tests first**

- Table-test a deterministic splitter: block-boundary packing, a single
  oversized paragraph/code/table failure, stable chunk hashes, 500-block/50-media
  boundaries, and retry after every individual chunk.
- Integration-test a database-backed part ledger: no duplicate part after a
  crash, resume at the first unsent part, ordered message IDs, and all parts
  included in update/reset decisions.

**Implementation**

- Ask for a second migration approval.  Add a child table such as
  `telegram_publish_sent_message_parts` keyed by sent-message identity and
  `part_index`, holding `message_id`, format, payload hash and canonical payload.
  Replace the current one-row existence short-circuit with per-part idempotency
  inside a transaction/outbox protocol.
- Implement splitting only after the Bot API semantics for replies/threading,
  media and edits of a chunked rich post are probed.  Do not split tables/code
  or invent continuation text in V1.

### Deliberately excluded from V1

- User-account/MTProto rich sends and edits: the research note calls them
  untested, and account tables/use cases are a separate publish path.
- Rich inbound import/inbox support: the current E2E and import assumptions are
  text-based; accepting empty text risks data loss.
- Inline/media gallery/collage/slideshow, audio/document/animation, maps,
  math, sub/superscript and custom emoji: each needs an exact wire fixture and
  a product syntax/fallback decision.
- Automatic TOC insertion and fold optimisation: existing heading IDs make it
  possible, but visual thresholds are one-client external measurements and
  changing authors’ post layout is product policy.
- Auto-selection based on classic warning strings, automatic fallback from
  `on`, and conversion of already-published classic messages: all conceal a
  format/data transition that must be explicit.
