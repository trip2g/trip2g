# Telegram UTM tags Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tag every site URL rendered inside a Telegram post with `utm_source=telegram`, `utm_campaign=<numeric chat ID>`, and `utm_content=note_<PathID>` (the last one only when the link target resolved to a specific note).

**Architecture:** All changes live under `internal/case/convertnoteviewtotgpost/`. A new pure helper file (`utm.go`) owns `resolveCampaign` (a thin wrapper over the existing `normalizeTelegramChatID`) and `buildTelegramSiteURL` (a `net/url`-based URL assembler). `Resolve()` computes the campaign once per call and calls `buildTelegramSiteURL` at the two existing site-URL fallback sites. No `Env` method changes, no schema changes, no regeneration.

**Tech Stack:** Go 1.x, `net/url`, `strconv`, `testing`, `github.com/stretchr/testify/require`.

**Spec:** `docs/superpowers/specs/2026-04-10-telegram-utm-tags-design.md`

---

## File Structure

- **Create** `internal/case/convertnoteviewtotgpost/utm.go` — pure helpers (`resolveCampaign`, `buildTelegramSiteURL`). Package: `convertnoteviewtotgpost` (same as `resolve.go`).
- **Create** `internal/case/convertnoteviewtotgpost/utm_test.go` — internal unit tests for the helpers. Package: `convertnoteviewtotgpost` (not `_test`) so tests can reach unexported functions.
- **Modify** `internal/case/convertnoteviewtotgpost/resolve.go` — add the campaign computation near the top of `Resolve()` and replace the two inline URL-assembly sites (the unresolved homepage fallback and the unpublished external link).
- **Modify** `internal/case/convertnoteviewtotgpost/resolve_test.go` — add end-to-end tests asserting UTM tags appear in rendered content for each fallback path. Package stays `convertnoteviewtotgpost_test`.

No other files are touched. `testEnv` in `resolve_test.go` does not need updates: the `Env` interface gains no new methods.

---

## Task 1: `resolveCampaign` helper (TDD)

**Files:**
- Create: `internal/case/convertnoteviewtotgpost/utm_test.go`
- Create: `internal/case/convertnoteviewtotgpost/utm.go`

- [ ] **Step 1: Write the failing test**

Create `internal/case/convertnoteviewtotgpost/utm_test.go`:

```go
package convertnoteviewtotgpost

import "testing"

func TestResolveCampaign(t *testing.T) {
	tests := []struct {
		name   string
		chatID int64
		want   string
	}{
		{"channel with -100 prefix", -1001234567890, "1234567890"},
		{"positive chat ID", 567, "567"},
		{"non-channel negative ID", -567, "567"},
		{"zero", 0, "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCampaign(tt.chatID)
			if got != tt.want {
				t.Errorf("resolveCampaign(%d) = %q, want %q", tt.chatID, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/case/convertnoteviewtotgpost/ -run TestResolveCampaign -v
```

Expected: build failure — `undefined: resolveCampaign`.

- [ ] **Step 3: Create `utm.go` with the helper**

Create `internal/case/convertnoteviewtotgpost/utm.go`:

```go
package convertnoteviewtotgpost

import "strconv"

// resolveCampaign derives the utm_campaign value for a Telegram publish
// target. Single source of truth for the UTM campaign derivation rule.
func resolveCampaign(chatID int64) string {
	return strconv.FormatInt(normalizeTelegramChatID(chatID), 10)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/case/convertnoteviewtotgpost/ -run TestResolveCampaign -v
```

Expected: `PASS` for all four subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/case/convertnoteviewtotgpost/utm.go internal/case/convertnoteviewtotgpost/utm_test.go
git commit -m "feat(convertnoteviewtotgpost): add resolveCampaign helper"
```

---

## Task 2: `buildTelegramSiteURL` — happy path (TDD)

**Files:**
- Modify: `internal/case/convertnoteviewtotgpost/utm_test.go`
- Modify: `internal/case/convertnoteviewtotgpost/utm.go`

- [ ] **Step 1: Add the failing happy-path test**

Append to `internal/case/convertnoteviewtotgpost/utm_test.go`:

```go
func TestBuildTelegramSiteURL_PlainPermalink(t *testing.T) {
	got := buildTelegramSiteURL(
		"https://example.com",
		"/notes/foo",
		"1234567890",
		"42",
	)
	want := "https://example.com/notes/foo?utm_campaign=1234567890&utm_content=note_42&utm_source=telegram"
	if got != want {
		t.Errorf("buildTelegramSiteURL = %q, want %q", got, want)
	}
}
```

Note: the order of query parameters matches `url.Values.Encode()`, which sorts keys alphabetically: `utm_campaign` < `utm_content` < `utm_source`.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/case/convertnoteviewtotgpost/ -run TestBuildTelegramSiteURL_PlainPermalink -v
```

Expected: build failure — `undefined: buildTelegramSiteURL`.

- [ ] **Step 3: Implement `buildTelegramSiteURL`**

Edit `internal/case/convertnoteviewtotgpost/utm.go`, replacing its contents with:

```go
package convertnoteviewtotgpost

import (
	"net/url"
	"strconv"
)

// resolveCampaign derives the utm_campaign value for a Telegram publish
// target. Single source of truth for the UTM campaign derivation rule.
func resolveCampaign(chatID int64) string {
	return strconv.FormatInt(normalizeTelegramChatID(chatID), 10)
}

// buildTelegramSiteURL assembles a site URL with UTM tracking parameters
// for a link rendered inside a Telegram post.
//
// Behavior:
//   - Preserves any pre-existing query parameters and fragment on the permalink.
//   - Returns "" when publicURL is empty; the caller keeps its existing
//     "public URL not set" warning path unchanged.
//   - On any parsing failure, falls back to publicURL+permalink concatenation
//     without UTM tags. Defensive — should not fire with any valid publicURL.
//   - When noteID is empty, omits utm_content entirely.
func buildTelegramSiteURL(publicURL, permalink, campaign, noteID string) string {
	if publicURL == "" {
		return ""
	}

	base, err := url.Parse(publicURL)
	if err != nil {
		return publicURL + permalink
	}

	ref, err := url.Parse(permalink)
	if err != nil {
		return publicURL + permalink
	}

	u := base.ResolveReference(ref)

	q := u.Query()
	q.Set("utm_source", "telegram")
	q.Set("utm_campaign", campaign)
	if noteID != "" {
		q.Set("utm_content", "note_"+noteID)
	}
	u.RawQuery = q.Encode()

	return u.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/case/convertnoteviewtotgpost/ -run TestBuildTelegramSiteURL_PlainPermalink -v
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add internal/case/convertnoteviewtotgpost/utm.go internal/case/convertnoteviewtotgpost/utm_test.go
git commit -m "feat(convertnoteviewtotgpost): add buildTelegramSiteURL helper"
```

---

## Task 3: `buildTelegramSiteURL` — edge cases

**Files:**
- Modify: `internal/case/convertnoteviewtotgpost/utm_test.go`

- [ ] **Step 1: Add the edge-case table test**

Append to `internal/case/convertnoteviewtotgpost/utm_test.go`:

```go
func TestBuildTelegramSiteURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		publicURL string
		permalink string
		campaign  string
		noteID    string
		want      string
	}{
		{
			name:      "existing query string",
			publicURL: "https://example.com",
			permalink: "/notes/foo?highlight=x",
			campaign:  "1234567890",
			noteID:    "42",
			want:      "https://example.com/notes/foo?highlight=x&utm_campaign=1234567890&utm_content=note_42&utm_source=telegram",
		},
		{
			name:      "fragment only",
			publicURL: "https://example.com",
			permalink: "/notes/foo#section-2",
			campaign:  "1234567890",
			noteID:    "42",
			want:      "https://example.com/notes/foo?utm_campaign=1234567890&utm_content=note_42&utm_source=telegram#section-2",
		},
		{
			name:      "query and fragment",
			publicURL: "https://example.com",
			permalink: "/notes/foo?x=1#y",
			campaign:  "1234567890",
			noteID:    "42",
			want:      "https://example.com/notes/foo?utm_campaign=1234567890&utm_content=note_42&utm_source=telegram&x=1#y",
		},
		{
			name:      "empty noteID omits utm_content",
			publicURL: "https://example.com",
			permalink: "/notes/foo",
			campaign:  "1234567890",
			noteID:    "",
			want:      "https://example.com/notes/foo?utm_campaign=1234567890&utm_source=telegram",
		},
		{
			name:      "empty permalink becomes homepage with UTM",
			publicURL: "https://example.com",
			permalink: "",
			campaign:  "1234567890",
			noteID:    "",
			want:      "https://example.com?utm_campaign=1234567890&utm_source=telegram",
		},
		{
			name:      "publicURL with trailing slash and leading-slash permalink",
			publicURL: "https://example.com/",
			permalink: "/notes/foo",
			campaign:  "1234567890",
			noteID:    "42",
			want:      "https://example.com/notes/foo?utm_campaign=1234567890&utm_content=note_42&utm_source=telegram",
		},
		{
			name:      "publicURL without slash and relative permalink",
			publicURL: "https://example.com",
			permalink: "notes/foo",
			campaign:  "1234567890",
			noteID:    "42",
			want:      "https://example.com/notes/foo?utm_campaign=1234567890&utm_content=note_42&utm_source=telegram",
		},
		{
			name:      "empty publicURL returns empty",
			publicURL: "",
			permalink: "/notes/foo",
			campaign:  "1234567890",
			noteID:    "42",
			want:      "",
		},
		{
			name:      "noteID rendered with note_ prefix",
			publicURL: "https://example.com",
			permalink: "/a",
			campaign:  "0",
			noteID:    "7",
			want:      "https://example.com/a?utm_campaign=0&utm_content=note_7&utm_source=telegram",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTelegramSiteURL(tt.publicURL, tt.permalink, tt.campaign, tt.noteID)
			if got != tt.want {
				t.Errorf("buildTelegramSiteURL = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the edge-case tests**

```bash
go test ./internal/case/convertnoteviewtotgpost/ -run TestBuildTelegramSiteURL -v
```

Expected: all subtests `PASS`. The implementation from Task 2 already handles every case — `url.ResolveReference` merges base and reference paths per RFC 3986, preserving queries and fragments.

If any subtest fails, do not guess the fix — print the actual output, compare against the expected string, and update the implementation in `utm.go` only if the behavior is genuinely wrong (not merely "expected string in plan is stale"). Parameter ordering comes from `url.Values.Encode()`, which is deterministic and alphabetical.

- [ ] **Step 3: Commit**

```bash
git add internal/case/convertnoteviewtotgpost/utm_test.go
git commit -m "test(convertnoteviewtotgpost): cover buildTelegramSiteURL edge cases"
```

---

## Task 4: Wire the UTM helpers into `Resolve()` (TDD)

**Files:**
- Modify: `internal/case/convertnoteviewtotgpost/resolve_test.go` (new test first)
- Modify: `internal/case/convertnoteviewtotgpost/resolve.go`

- [ ] **Step 1: Write a failing end-to-end test for the unpublished-link UTM output**

Append to `internal/case/convertnoteviewtotgpost/resolve_test.go`:

```go
func TestUnpublishedExternalLinkHasUTMTags(t *testing.T) {
	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{
			{
				Path: "main.md",
				Content: []byte(`---
free: true
title: "Main Note"
telegram_publish_tags: ["tag1"]
---
See [[other_note]] for details.`),
			},
			{
				Path: "other_note.md",
				Content: []byte(`---
free: true
title: "Other Note"
---
Plain site-only note.`),
			},
		},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	// Pin the linked note's identifying fields so the assertion is deterministic.
	other := nvs.Map["/other_note"]
	require.NotNil(t, other, "linked note should be resolved")
	other.Permalink = "/other_note"
	other.PathID = 99

	var mainNote *model.NoteView
	for _, nv := range nvs.List {
		if nv.Path == "main.md" {
			mainNote = nv
			break
		}
	}
	require.NotNil(t, mainNote)

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
	}

	source := model.TelegramPostSource{
		NoteView: mainNote,
		ChatID:   -1001234567890,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	require.Contains(t, post.Content,
		`href="https://example.com/other_note?utm_campaign=1234567890&amp;utm_content=note_99&amp;utm_source=telegram"`,
		"unpublished external link should carry full UTM tag set (HTML-escaped ampersands)")
}
```

Note on `&amp;`: the post body is rendered as HTML, and the markdown → HTML
converter escapes `&` in `href` attributes to `&amp;`. Assert on the escaped
form. If the rendered content in this codebase actually uses bare `&`,
adjust the expectation to match `&`; read the first failing test output
before changing the production code.

- [ ] **Step 2: Run the new test — expect it to fail**

```bash
go test ./internal/case/convertnoteviewtotgpost/ -run TestUnpublishedExternalLinkHasUTMTags -v
```

Expected: `FAIL`. Current `resolve.go` at line 187-188 concatenates `publicURL + linkedNV.Permalink` without UTM tags, so the rendered href is `https://example.com/other_note`. The substring assertion will not find the UTM-tagged form.

If the actual rendered substring uses bare `&` instead of `&amp;`, record the exact failing substring and use that form in Step 5. The point of running the failing test now is to lock in the exact rendered shape before editing production code.

- [ ] **Step 3: Modify `Resolve()` — compute `campaign` once per call**

Edit `internal/case/convertnoteviewtotgpost/resolve.go`. Find the block right after `publicURL := env.PublicURL()` (around line 125) and insert the campaign derivation:

Before:

```go
	publicURL := env.PublicURL()
	post := model.TelegramPost{}
```

After:

```go
	publicURL := env.PublicURL()

	publishChatID := source.ChatID
	if publishChatID == 0 {
		publishChatID = source.TelegramChatID
	}
	campaign := resolveCampaign(publishChatID)

	post := model.TelegramPost{}
```

- [ ] **Step 4: Modify the unresolved-link fallback**

Find the current line 141 inside the link resolver closure:

```go
			post.UnresolvedLinkCount++
			return &markdownv2.LinkResolverResult{URL: publicURL}, nil
```

Replace with:

```go
			post.UnresolvedLinkCount++
			return &markdownv2.LinkResolverResult{
				URL: buildTelegramSiteURL(publicURL, "", campaign, ""),
			}, nil
```

- [ ] **Step 5: Modify the unpublished-link external URL generation**

Find the current lines 185-188 inside the `if allowExternalLinks` block:

```go
			post.ExternalLinkCount++

			externalURL := publicURL + linkedNV.Permalink
			return &markdownv2.LinkResolverResult{URL: externalURL}, nil
```

Replace with:

```go
			post.ExternalLinkCount++

			externalURL := buildTelegramSiteURL(
				publicURL,
				linkedNV.Permalink,
				campaign,
				strconv.FormatInt(linkedNV.PathID, 10),
			)
			return &markdownv2.LinkResolverResult{URL: externalURL}, nil
```

`strconv` is already imported in `resolve.go` (used by `normalizeTelegramChatID`) — no import change needed.

- [ ] **Step 6: Run the new test — expect it to pass**

```bash
go test ./internal/case/convertnoteviewtotgpost/ -run TestUnpublishedExternalLinkHasUTMTags -v
```

Expected: `PASS`.

If the rendered href uses bare `&` (not `&amp;`), the assertion will fail with an actual vs expected diff. In that case, update the assertion string in Step 1 to match the rendered form, re-run, and verify PASS. Do not change `buildTelegramSiteURL` output — the helper's own unit tests already pin its output shape; any escaping difference comes from the HTML renderer above it.

- [ ] **Step 7: Run the entire package test suite to verify no regressions**

```bash
go test ./internal/case/convertnoteviewtotgpost/...
```

Expected: all existing tests still pass. The pre-existing `TestPublishedLinkPreservesWikilinkLabel` already asserts that published-to-TG links produce a `https://t.me/c/...` href; it must continue passing (those URLs never get UTM tags).

- [ ] **Step 8: Commit**

```bash
git add internal/case/convertnoteviewtotgpost/resolve.go internal/case/convertnoteviewtotgpost/resolve_test.go
git commit -m "feat(convertnoteviewtotgpost): tag Telegram-originated site links with UTM"
```

---

## Task 5: Integration tests for unresolved link and chat-ID selection

**Files:**
- Modify: `internal/case/convertnoteviewtotgpost/resolve_test.go`

- [ ] **Step 1: Add a test for the unresolved-link homepage fallback**

Append to `internal/case/convertnoteviewtotgpost/resolve_test.go`:

```go
func TestUnresolvedLinkGetsHomepageUTM(t *testing.T) {
	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path: "main.md",
			Content: []byte(`---
free: true
title: "Main Note"
telegram_publish_tags: ["tag1"]
---
See [[nonexistent_target]] for details.`),
		}},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
	}

	source := model.TelegramPostSource{
		NoteView: nvs.List[0],
		ChatID:   -1001234567890,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	// Homepage fallback carries source + campaign only (no utm_content).
	require.Contains(t, post.Content,
		`href="https://example.com?utm_campaign=1234567890&amp;utm_source=telegram"`)
	require.NotContains(t, post.Content, "utm_content",
		"unresolved-link fallback must not emit utm_content")
	require.Greater(t, post.UnresolvedLinkCount, 0,
		"unresolved link should be counted")
}
```

(If the rendered form uses bare `&`, adjust the expected substring to match — same reasoning as Task 4 Step 6.)

- [ ] **Step 2: Add a test for the account-flow chat-ID fallback**

Append to `internal/case/convertnoteviewtotgpost/resolve_test.go`:

```go
func TestCampaignFromTelegramChatIDWhenChatIDZero(t *testing.T) {
	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{{
			Path: "main.md",
			Content: []byte(`---
free: true
title: "Main Note"
telegram_publish_tags: ["tag1"]
---
See [[nonexistent_target]].`),
		}},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)

	env := &testEnv{
		nvs:       nvs,
		logger:    &logger.TestLogger{},
		sentMsgs:  []db.ListTelegramPublishSentMessagesByChatIDRow{},
		publicURL: "https://example.com",
	}

	// ChatID is zero; TelegramChatID is set (account-flow shape).
	source := model.TelegramPostSource{
		NoteView:       nvs.List[0],
		ChatID:         0,
		TelegramChatID: -1002222222222,
	}

	post, err := convertnoteviewtotgpost.Resolve(context.Background(), env, source)
	require.NoError(t, err)

	require.Contains(t, post.Content, "utm_campaign=2222222222",
		"campaign should derive from source.TelegramChatID when source.ChatID is 0")
}
```

- [ ] **Step 3: Run the full package test suite**

```bash
go test ./internal/case/convertnoteviewtotgpost/... -v
```

Expected: all tests `PASS`, including the three new ones (`TestUnpublishedExternalLinkHasUTMTags`, `TestUnresolvedLinkGetsHomepageUTM`, `TestCampaignFromTelegramChatIDWhenChatIDZero`) plus every pre-existing test.

- [ ] **Step 4: Commit**

```bash
git add internal/case/convertnoteviewtotgpost/resolve_test.go
git commit -m "test(convertnoteviewtotgpost): cover UTM fallback and chat-ID selection"
```

---

## Task 6: Final verification

**Files:** none.

- [ ] **Step 1: Build the whole project**

```bash
go build ./...
```

Expected: exit 0, no errors. This catches any unexpected import or type regressions outside the immediate package.

- [ ] **Step 2: Run the whole test suite for the surrounding area**

```bash
go test ./internal/case/convertnoteviewtotgpost/... ./internal/case/sendtelegrampublishpost/... ./internal/case/sendtelegramaccountpublishpost/... ./internal/case/backjob/sendtelegrammessage/... ./internal/case/backjob/sendtelegrampost/...
```

Expected: `PASS` across all listed packages. These are the direct callers of `convertnoteviewtotgpost.Resolve`; they use mocks that should be untouched by this change.

- [ ] **Step 3: Confirm no stray generator artifacts**

```bash
git status
```

Expected: clean working tree. If any generated file changed unexpectedly, investigate before finishing.

- [ ] **Step 4: Smoke-check the helper with a real rendered post (optional, no commit)**

This step is only a mental smoke test — no code change. Re-read `internal/case/convertnoteviewtotgpost/resolve.go` and confirm:

1. `campaign` is derived exactly once, before the link resolver closure is built.
2. The published-to-TG branch (current line 152) still returns a `https://t.me/c/...` URL and does not invoke `buildTelegramSiteURL`.
3. The unresolved branch (current line 141) uses `buildTelegramSiteURL` with an empty permalink and empty noteID.
4. The unpublished external branch (current line 187-188) passes `linkedNV.Permalink` and `strconv.FormatInt(linkedNV.PathID, 10)`.

If any of these is wrong, fix before declaring done.

---

## Coverage check against spec

| Spec section | Plan task(s) |
|---|---|
| UTM value contract (source/campaign/content) | Task 1 (campaign derivation), Task 2–3 (helper builds the string) |
| Component 1: `resolveCampaign` | Task 1 |
| Component 2: `buildTelegramSiteURL` | Task 2 (happy path) + Task 3 (edge cases) |
| Component 3: changes in `Resolve()` | Task 4 |
| Data flow (campaign once per call, three link branches) | Task 4 Step 3 + Task 6 Step 4 |
| Error handling: empty `publicURL`, parse failure | Task 3 (edge cases include empty publicURL); parse failure fallback is in the implementation but not exercised by tests — it is defensive and unreachable from any configured publicURL |
| Testing: `utm_test.go` | Task 1 + Task 2 + Task 3 |
| Testing: `resolve_test.go` updates (published unchanged, unpublished + full UTM, unresolved + partial UTM, bot vs account flow) | Task 4 (unpublished + published untouched) + Task 5 (unresolved + account chat-ID fallback) |
| Build steps: `go test ./internal/case/convertnoteviewtotgpost/...` | Task 5 Step 3 + Task 6 Step 2 |
| Goal: publish flow never fails over UTM plumbing | Guaranteed by defensive branches in `buildTelegramSiteURL` and by never returning an error from the UTM code path |
