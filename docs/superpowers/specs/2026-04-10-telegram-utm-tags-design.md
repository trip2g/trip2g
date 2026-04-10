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
4. Publish flow must never fail because of UTM plumbing. Any lookup failure
   falls back gracefully.

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
| `utm_campaign` | Public channel username (without `@`) if the channel has one; otherwise the numeric chat ID normalized with `-100` prefix stripped, rendered as decimal. |
| `utm_content` | `note_<PathID>` where `<PathID>` is `linkedNV.PathID` (internal int64). Omitted entirely when the link target could not be resolved to a specific note (the unresolved-link homepage fallback case). |

Rationale for `utm_content=note_<PathID>`: `PathID` is stable, always set on a
NoteView, and unambiguous across renames. The `note_` prefix keeps the value
self-describing in analytics reports.

Rationale for `utm_campaign` fallback to numeric chat ID: private channels have
no username. Numeric ID is opaque in reports but unique and always available,
and traffic from such channels is usually small enough that the opacity is
tolerable. The fallback is deterministic and easy to document.

## Architecture

All changes are localized to `internal/case/convertnoteviewtotgpost/` plus the
corresponding Env method implementation in `cmd/server/case_methods.go` (where
Env methods for this use case are wired).

No database schema changes, no GraphQL changes, no config changes, no changes
to the `markdownv2` package.

## Components

### 1. `Env.TelegramChannelUsername(ctx, chatID int64) (string, error)` — new method

Added to the `Env` interface in `internal/case/convertnoteviewtotgpost/resolve.go`.

Contract:

- Returns the bare channel username (no leading `@`) for public channels.
- Returns `"", nil` when the channel has no username, does not exist in our
  tables, or the lookup hits a recoverable DB error.
- Returns `"", err` only for genuinely unexpected errors the caller may want
  to log; `Resolve()` will still treat any error as a fallback-to-numeric
  signal and not propagate it.

The concrete implementation lives in `cmd/server/case_methods.go` and resolves
against whichever Telegram channel / chat table holds channel metadata. If
different tables serve bot vs. account publishing flows, the implementation
picks based on which chat ID matches.

### 2. `resolveCampaign(username string, chatID int64) string` — internal helper

Pure function in `internal/case/convertnoteviewtotgpost/utm.go`.

```
if username != "" { return username }
return strconv.FormatInt(normalizeTelegramChatID(chatID), 10)
```

Reuses the existing `normalizeTelegramChatID` already in `resolve.go`.

### 3. `buildTelegramSiteURL(publicURL, permalink, campaign, noteID string) string` — pure URL builder

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

### 4. Changes in `Resolve()`

At the top of `Resolve()`, after `publicURL := env.PublicURL()`:

```go
publishChatID := source.ChatID
if publishChatID == 0 {
    publishChatID = source.TelegramChatID
}

username, _ := env.TelegramChannelUsername(ctx, publishChatID)
campaign := resolveCampaign(username, publishChatID)
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
   |-- env.TelegramChannelUsername(ctx, publishChatID) -> username or ""
   |-- campaign := resolveCampaign(username, publishChatID)
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
| `TelegramChannelUsername` returns error | Log at debug in caller-space, ignored in `Resolve`. Campaign falls back to normalized numeric chat ID. |
| `TelegramChannelUsername` returns empty string | Campaign falls back to normalized numeric chat ID. |
| `buildTelegramSiteURL` parse failure | Fall back to plain `publicURL + permalink` concatenation, no UTM tags. Log at debug (no-op for callers). |
| `publicURL == ""` | Return `""`, preserving current "public URL is not set" warning path. |

The publish pipeline must never fail because a UTM tag could not be computed.

## Testing

### New file: `internal/case/convertnoteviewtotgpost/utm_test.go`

Table-driven tests over `buildTelegramSiteURL`. Cases:

1. Plain permalink (`/notes/foo`) → `<publicURL>/notes/foo?utm_source=telegram&utm_campaign=mychan&utm_content=note_42`.
2. Permalink already carrying a query string (`/notes/foo?highlight=x`) → query params merged, neither lost nor duplicated.
3. Permalink with fragment (`/notes/foo#section-2`) → fragment preserved, UTM params appear before the fragment.
4. Permalink with both query and fragment (`/notes/foo?x=1#y`).
5. Empty `noteID` → output omits `utm_content` entirely (not empty, not present).
6. Empty `permalink` → output is publicURL-as-homepage with UTM tags attached.
7. `publicURL` with trailing slash + permalink with leading slash → exactly one slash in the joined path.
8. `publicURL` without trailing slash + permalink without leading slash → exactly one slash in the joined path.
9. Campaign containing URL-unsafe characters (e.g. a numeric chat ID is safe, but tests must pin the encoding expectation) — verify `url.Values.Encode()` handles it and we do not double-encode.
10. `publicURL == ""` → returns `""`.
11. Empty campaign (shouldn't happen in practice, but assert the helper doesn't emit a dangling `&utm_campaign=`).

Also a small direct test for `resolveCampaign`:

- `("foo", 123)` → `"foo"`.
- `("", -1001234567890)` → `"1234567890"`.
- `("", 567)` → `"567"`.

### Updated: `internal/case/convertnoteviewtotgpost/resolve_test.go`

New scenarios (add rows to the existing table-driven tests, do not duplicate
the harness):

1. **Published-to-TG link unchanged.** Target note is already in `sentMap`;
   the resolved URL is `https://t.me/c/.../...` with no UTM fragment anywhere
   in the rendered content.
2. **Unpublished link gets full UTM set.** Target note exists in `NoteViews`
   but is not in `sentMap`. Assert URL equals
   `<publicURL><permalink>?utm_source=telegram&utm_campaign=<campaign>&utm_content=note_<pathID>`.
3. **Unresolved link gets partial UTM.** Link target missing from
   `NoteViews.Map`. Assert URL equals
   `<publicURL>?utm_source=telegram&utm_campaign=<campaign>` with no
   `utm_content` parameter.
4. **Campaign from username.** `TelegramChannelUsername` mock returns
   `"mychan"` → `utm_campaign=mychan`.
5. **Campaign falls back to chat ID.** `TelegramChannelUsername` mock returns
   `""` → `utm_campaign=<normalized numeric chat ID as decimal>`.
6. **Campaign falls back on error.** `TelegramChannelUsername` mock returns
   `"", errors.New("db down")` → `utm_campaign=<numeric>`, and `Resolve`
   still returns successfully with no error of its own.
7. **Bot flow vs account flow chat-ID selection.** Two cases: `source.ChatID`
   set → that ID is used; `source.ChatID == 0 && source.TelegramChatID != 0`
   → `source.TelegramChatID` is used.

Mocks regenerate via `go generate ./internal/case/convertnoteviewtotgpost/...`.
The `//go:generate moq` directive on the `Env` interface picks up the new
method automatically.

## Build steps

After the change:

```
go generate ./internal/case/convertnoteviewtotgpost/...
go test ./internal/case/convertnoteviewtotgpost/...
```

No template, SQL, GraphQL, or frontend regeneration is required.

## Risks & mitigations

| Risk | Mitigation |
|---|---|
| Username lookup adds latency to every publish | One extra lookup per `Resolve()` call (not per link). Cacheable inside the Env implementation if it ever becomes measurable. |
| Campaign value differs between "imported via frontmatter" channel rows and "sent via bot" channel rows | Env implementation is the single source of truth and resolves both by the same numeric chat ID, so both flows land on the same campaign string. |
| Analytics landing URL now differs from the canonical site URL (query tail) | Expected and desired — that's the whole point. Canonicalization on the site (`<link rel=canonical>`) already exists in the default template and strips query strings, so SEO is unaffected. |
| Broken `publicURL` config silently strips UTM tags via the defensive fallback | Acceptable: the fallback is better than a broken publish. The existing "public URL is not set" warning covers the empty case. |

## Out of scope / future work

- Configurable UTM values via admin config modules.
- Per-note frontmatter overrides (`telegram_publish_utm_*`).
- UTM on links inside RSS, email digests, or other non-Telegram surfaces.
- UTM on the t.me links themselves (not useful — Telegram strips or ignores).
- A generic URL decorator in `markdownv2` for reuse outside of this case.
