# UTM tags on Telegram-originated site links

Date: 2026-04-10
Status: Design approved, ready for implementation plan

## Problem

When trip2g publishes a note to a Telegram channel, wikilinks inside that note
are resolved in `internal/case/convertnoteviewtotgpost/resolve.go`:

- If the target note has already been published to the same Telegram channel,
  the link becomes a `https://t.me/c/<chat>/<msg>` deep-link.
- Otherwise, the link falls back to the public site URL
  (`publicURL + linkedNV.Permalink`).
- If the target cannot be resolved at all, the link falls back to the site
  homepage (`publicURL`).

We want traffic that arrives on the website from those fallback links to be
trackable in analytics, broken down by Telegram channel and by which note was
linked. Standard mechanism: UTM query parameters.

## Goals

1. Every site-URL link rendered inside a Telegram post carries UTM tags.
2. Analytics can distinguish:
   - Traffic source is Telegram (`utm_source=telegram`).
   - Which channel the click came from (`utm_campaign=<channel>`).
   - Which specific linked note was clicked (`utm_content=note_<id>`).
3. Zero behavioral impact on `t.me/...` deep-links (no UTM — tracking
   only matters when the user lands on our own site).
4. Publish flow must never fail because of UTM plumbing. Any unexpected
   error in URL assembly falls back gracefully to the original untagged URL.

## Non-goals

- UTM tags on `t.me/...` deep-links.
- UTM tags on links rendered on the website itself.
- UTM tags on media/asset URLs.
- Frontmatter or admin-config overrides of UTM values.
- A generic URL-decorator mechanism in `markdownv2` (YAGNI).

## UTM value contract

| Parameter | Value |
|---|---|
| `utm_source` | Literal string `telegram`. Never varies. |
| `utm_campaign` | Numeric Telegram chat ID with the `-100` channel prefix stripped, rendered as decimal. |
| `utm_content` | `note_<PathID>` where `<PathID>` is `linkedNV.PathID` (internal int64). Omitted entirely when the link target could not be resolved to a specific note (the unresolved-link homepage fallback case). |

Rationale for `utm_content=note_<PathID>`: `PathID` is stable, always set on a
NoteView, and unambiguous across renames. The `note_` prefix keeps the value
self-describing in analytics reports.

Rationale for numeric-ID `utm_campaign`: channel usernames are not persisted in
our database (the `tg_bot_chats` and `telegram_publish_account_chats` tables
store `chat_title` and `telegram_id` only). The alternatives are (a) add a
schema migration and data-population path, (b) fetch live from the Telegram
API with caching, or (c) use the numeric chat ID we already have. Option (c)
is chosen: zero new infrastructure, zero new failure modes on the publish
path, and the campaign value is stable and deterministic. Analytics reports
will show opaque numeric campaigns; a small lookup table in the analytics
tool's UI can map them to friendly names once. A future change can introduce
a friendly-name layer additively without breaking existing data.

## Architecture

All changes are localized entirely to `internal/case/convertnoteviewtotgpost/`.
No new `Env` methods, no changes to `cmd/server/case_methods.go`, no
database schema changes, no GraphQL changes, no config changes, no changes
to the `markdownv2` package.

## Components

### 1. `resolveCampaign(chatID int64) string` — internal helper

Pure function in `internal/case/convertnoteviewtotgpost/utm.go`.

```go
return strconv.FormatInt(normalizeTelegramChatID(chatID), 10)
```

Thin wrapper over the existing `normalizeTelegramChatID` already in
`resolve.go`. It exists as a named function so the intent ("this is the UTM
campaign derivation") is clear at call sites and the behavior has one place
to evolve if we later add a friendly-name layer.

### 2. `buildTelegramSiteURL(publicURL, permalink, campaign, noteID string) string` — pure URL builder

Pure function in `internal/case/convertnoteviewtotgpost/utm.go`.

Responsibilities:

- Parse `publicURL` + `permalink` using `net/url`. Preserve any pre-existing
  query parameters and fragment on the permalink.
- Append `utm_source=telegram`, `utm_campaign=<campaign>`, and
  `utm_content=note_<noteID>` — the last one only when `noteID != ""`.
- Concatenate `publicURL` and `permalink` correctly: normalize the single
  slash at the join, avoiding both missing-slash and double-slash bugs.
- If `publicURL == ""`, return `""`. The caller preserves the existing
  "public URL not set" warning path unchanged.
- If URL parsing fails for any reason (should not occur with validated
  config), fall back to plain string concatenation of `publicURL + permalink`
  without UTM tags. Defensive belt-and-braces — we never want to drop a link
  over a tagging failure.

### 3. Changes in `Resolve()`

At the top of `Resolve()`, after `publicURL := env.PublicURL()`:

```go
publishChatID := source.ChatID
if publishChatID == 0 {
    publishChatID = source.TelegramChatID
}
campaign := resolveCampaign(publishChatID)
```

Inside the `SetLinkResolver` closure, two substitutions:

| Case | File location | Before | After |
|---|---|---|---|
| Link target not found at all | line 141 (current) | `&markdownv2.LinkResolverResult{URL: publicURL}` | `&markdownv2.LinkResolverResult{URL: buildTelegramSiteURL(publicURL, "", campaign, "")}` |
| Linked note found but not yet published | line 187-188 (current) | `externalURL := publicURL + linkedNV.Permalink` | `externalURL := buildTelegramSiteURL(publicURL, linkedNV.Permalink, campaign, strconv.FormatInt(linkedNV.PathID, 10))` |

The "linked note is already published to this channel" branch at line 152
that returns a `https://t.me/c/...` URL is **not** touched — t.me links
never get UTM tags.

## Data flow

```
Caller (sendtelegrampublishpost / sendtelegramaccountpublishpost / backjobs)
   |
   v
convertnoteviewtotgpost.Resolve(ctx, env, source)
   |
   |-- publishChatID := source.ChatID or source.TelegramChatID
   |-- campaign := resolveCampaign(publishChatID)   // numeric string
   |
   |-- for each wikilink in note:
   |      |-- already published on this channel -> t.me/c/... (unchanged)
   |      |-- found but unpublished            -> buildTelegramSiteURL(..., noteID=<PathID>)
   |      |-- not found                        -> buildTelegramSiteURL(..., noteID="")
   |
   v
model.TelegramPost with UTM-tagged site links in content
```

## Error handling

| Condition | Behavior |
|---|---|
| `buildTelegramSiteURL` parse failure | Fall back to plain `publicURL + permalink` concatenation, no UTM tags. Defensive only — should not happen with any valid `publicURL` config. |
| `publicURL == ""` | `buildTelegramSiteURL` returns `""`. The existing caller logic that logs "public URL is not set, cannot generate external link" is preserved unchanged. |
| `publishChatID == 0` (neither `source.ChatID` nor `source.TelegramChatID` set) | `campaign == "0"`. This should not occur in practice; no special handling. |

The publish pipeline must never fail because a UTM tag could not be computed.

## Testing

### New file: `internal/case/convertnoteviewtotgpost/utm_test.go`

Table-driven tests over `buildTelegramSiteURL`. Cases:

1. Plain permalink (`/notes/foo`) → `https://example.com/notes/foo?utm_source=telegram&utm_campaign=1234567890&utm_content=note_42`.
2. Permalink already carrying a query string (`/notes/foo?highlight=x`) → existing query preserved, UTM params merged, nothing lost or duplicated.
3. Permalink with fragment (`/notes/foo#section-2`) → fragment preserved on the tail of the URL, UTM params appear before the fragment.
4. Permalink with both query and fragment (`/notes/foo?x=1#y`).
5. Empty `noteID` → output omits `utm_content` entirely (not `utm_content=`, not present at all).
6. Empty `permalink` → output is the publicURL-as-homepage with UTM tags attached (e.g. `https://example.com/?utm_source=telegram&utm_campaign=1234567890`).
7. `publicURL` with trailing slash (`https://example.com/`) + permalink with leading slash (`/notes/foo`) → exactly one slash between host and path.
8. `publicURL` without trailing slash + permalink without leading slash → exactly one slash between host and path.
9. `publicURL == ""` → returns `""`.
10. Non-zero `noteID` is rendered as `note_<id>` (e.g. `note_42`), not `42` alone.

Also a small direct test for `resolveCampaign`:

- `resolveCampaign(-1001234567890)` → `"1234567890"` (channel, `-100` prefix stripped).
- `resolveCampaign(567)` → `"567"` (positive chat ID passed through).
- `resolveCampaign(-567)` → `"567"` (non-channel negative ID, absolute value).
- `resolveCampaign(0)` → `"0"`.

### Updated: `internal/case/convertnoteviewtotgpost/resolve_test.go`

The test env (`testEnv` in `resolve_test.go`) is hand-written and does not
need regeneration. No new methods are added to the `Env` interface, so
`testEnv` stays as-is.

New scenarios (add cases or new test functions alongside the existing ones):

1. **Published-to-TG link unchanged.** Target note is already in `sentMap`
   (or referenced via frontmatter of another note with matching channel);
   the resolved URL is `https://t.me/c/.../...` with no `utm_` substring
   anywhere in the rendered content.
2. **Unpublished link gets full UTM set.** Target note exists in `NoteViews`
   but not in `sentMap`. Assert rendered URL equals
   `<publicURL><permalink>?utm_source=telegram&utm_campaign=<numericChatID>&utm_content=note_<pathID>`
   with `<numericChatID>` being the normalized chat ID from the source.
3. **Unresolved link gets partial UTM.** Link target missing from
   `NoteViews.Map`. Assert rendered URL equals
   `<publicURL>?utm_source=telegram&utm_campaign=<numericChatID>`, no
   `utm_content` parameter present.
4. **Bot flow chat-ID selection.** `source.ChatID` set, `source.TelegramChatID` zero → campaign derives from `source.ChatID`.
5. **Account flow chat-ID selection.** `source.ChatID == 0`, `source.TelegramChatID` set → campaign derives from `source.TelegramChatID`.

## Build steps

After the change:

```
go test ./internal/case/convertnoteviewtotgpost/...
```

No `go generate`, no template, SQL, GraphQL, or frontend regeneration is
required.

## Risks & mitigations

| Risk | Mitigation |
|---|---|
| Analytics landing URL now differs from the canonical site URL (query tail) | Expected and desired — that's the whole point. Canonicalization on the site (`<link rel=canonical>`) already exists in the default template and strips query strings, so SEO is unaffected. |
| Broken `publicURL` config silently strips UTM tags via the defensive fallback | Acceptable: the fallback is better than a broken publish. The existing "public URL is not set" warning covers the empty case. |
| Opaque numeric `utm_campaign` in analytics | Documented trade-off (see rationale in "UTM value contract"). Analytics-side mapping table handles the one-off translation; a friendly-name layer is kept as additive future work. |

## Out of scope / future work

- Configurable UTM values via admin config modules.
- Per-note frontmatter overrides (`telegram_publish_utm_*`).
- UTM on links inside RSS, email digests, or other non-Telegram surfaces.
- UTM on the t.me links themselves (not useful — Telegram strips or ignores).
- A generic URL decorator in `markdownv2` for reuse outside of this case.
