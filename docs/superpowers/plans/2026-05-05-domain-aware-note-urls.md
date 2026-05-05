# Domain-Aware Note URLs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Return full domain-aware URLs for notes in both GraphQL (`NoteView.url`) and MCP search/similar results, preferring the first custom domain route if one exists.

**Architecture:** Add a reverse route index (`customDomainRoutes map[int64]struct{Host,Path string}`) to `NoteViews`, populated during `RegisterNoteRoutes`. A new `ResolveFullURL(note, publicURL)` method uses this index. MCP adds `NoteURL(*model.NoteView) string` to its `Env` interface; GraphQL resolver calls `ResolveFullURL` directly via the existing `LatestNoteViews()`.

**Tech Stack:** Go, gqlgen (GraphQL code generation), moq (mock generation), Playwright (E2E)

---

### Task 1: Add reverse route index + ResolveFullURL to NoteViews

**Files:**
- Modify: `internal/model/note.go`
- Test: `internal/model/note_test.go`

- [ ] **Step 1: Add the field to NoteViews struct**

In `internal/model/note.go`, add `"net/url"` to imports (after `"strings"`):

```go
import (
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	rl2 "trip2g/internal/russkayalatinica2"
	"unicode"

	"github.com/yuin/goldmark/ast"
)
```

Add the field to `NoteViews` struct (after the `DomainSitemaps` field, around line 312):

```go
// customDomainRoutes is a reverse index: noteID -> first custom domain route.
// Populated during RegisterNoteRoutes. Used by ResolveFullURL.
customDomainRoutes map[int64]struct{ Host, Path string }
```

- [ ] **Step 2: Initialize the field in NewNoteViews**

In `NewNoteViews()` (around line 964), add initialization:

```go
func NewNoteViews() *NoteViews {
	return &NoteViews{
		Map:                make(map[string]*NoteView),
		PathMap:            make(map[string]*NoteView),
		Subgraphs:          make(map[string]*NoteSubgraph),
		RouteMap:           make(map[string]map[string]*NoteView),
		DomainSitemaps:     make(map[string][]byte),
		customDomainRoutes: make(map[int64]struct{ Host, Path string }),
	}
}
```

- [ ] **Step 3: Populate the reverse index in RegisterNoteRoutes**

In `RegisterNoteRoutes` (around line 1248), add reverse index population inside the existing loop:

```go
func (nv *NoteViews) RegisterNoteRoutes(note *NoteView) {
	if len(note.Routes) == 0 {
		return
	}
	if nv.RouteMap == nil {
		nv.RouteMap = make(map[string]map[string]*NoteView)
	}
	for _, r := range note.Routes {
		if nv.RouteMap[r.Host] == nil {
			nv.RouteMap[r.Host] = make(map[string]*NoteView)
		}
		// Empty Path means "use note's own Permalink" (set when user writes "foo.com" without path).
		path := r.Path
		if path == "" {
			path = note.Permalink
		}
		nv.RouteMap[r.Host][path] = note
		// Reverse index: first custom domain route wins.
		if r.Host != "" {
			if _, exists := nv.customDomainRoutes[note.PathID]; !exists {
				nv.customDomainRoutes[note.PathID] = struct{ Host, Path string }{r.Host, path}
			}
		}
	}
}
```

- [ ] **Step 4: Replace the ResolveURL stub with ResolveFullURL**

Replace the stub (around line 978):

```go
// ResolveFullURL returns the full URL for a note.
// If the note has a custom domain route, the first one is used with the scheme
// derived from publicURL. Otherwise falls back to publicURL + note.Permalink.
func (nv *NoteViews) ResolveFullURL(note *NoteView, publicURL string) string {
	if r, ok := nv.customDomainRoutes[note.PathID]; ok {
		u, _ := url.Parse(publicURL)
		scheme := "https"
		if u != nil && u.Scheme != "" {
			scheme = u.Scheme
		}
		return scheme + "://" + r.Host + r.Path
	}
	return publicURL + note.Permalink
}
```

Keep the old `ResolveURL` method renamed or deleted — check for callers first:

```bash
grep -rn "ResolveURL\b" /home/alexes/projects2/trip2g --include="*.go"
```

If `ResolveURL` has no other callers, delete it.

- [ ] **Step 5: Write failing tests**

Add to `internal/model/note_test.go`:

```go
func TestResolveFullURL(t *testing.T) {
	const publicURL = "https://main.example.com"

	t.Run("no custom domain returns publicURL+permalink", func(t *testing.T) {
		nv := NewNoteViews()
		note := &NoteView{PathID: 1, Permalink: "/my-note"}
		require.Equal(t, "https://main.example.com/my-note", nv.ResolveFullURL(note, publicURL))
	})

	t.Run("first custom domain route is used", func(t *testing.T) {
		nv := NewNoteViews()
		note := &NoteView{
			PathID:    2,
			Permalink: "/my-note",
			Routes: []ParsedRoute{
				{Host: "custom.io", Path: "/custom-path"},
			},
		}
		nv.RegisterNoteRoutes(note)
		require.Equal(t, "https://custom.io/custom-path", nv.ResolveFullURL(note, publicURL))
	})

	t.Run("http scheme is preserved from publicURL", func(t *testing.T) {
		nv := NewNoteViews()
		note := &NoteView{
			PathID:    3,
			Permalink: "/note",
			Routes:    []ParsedRoute{{Host: "custom.io", Path: "/path"}},
		}
		nv.RegisterNoteRoutes(note)
		require.Equal(t, "http://custom.io/path", nv.ResolveFullURL(note, "http://localhost:8081"))
	})

	t.Run("empty route path falls back to permalink", func(t *testing.T) {
		nv := NewNoteViews()
		note := &NoteView{
			PathID:    4,
			Permalink: "/note-permalink",
			Routes:    []ParsedRoute{{Host: "custom.io", Path: ""}},
		}
		nv.RegisterNoteRoutes(note)
		require.Equal(t, "https://custom.io/note-permalink", nv.ResolveFullURL(note, publicURL))
	})

	t.Run("first of multiple custom domains wins", func(t *testing.T) {
		nv := NewNoteViews()
		note := &NoteView{
			PathID:    5,
			Permalink: "/note",
			Routes: []ParsedRoute{
				{Host: "first.io", Path: "/a"},
				{Host: "second.io", Path: "/b"},
			},
		}
		nv.RegisterNoteRoutes(note)
		require.Equal(t, "https://first.io/a", nv.ResolveFullURL(note, publicURL))
	})
}
```

- [ ] **Step 6: Run tests — expect them to fail**

```bash
cd /home/alexes/projects2/trip2g && go test ./internal/model/... -run TestResolveFullURL -v
```

Expected: FAIL (method doesn't exist yet — you wrote it in step 4, so actually this should pass now).

- [ ] **Step 7: Run tests — verify they pass**

```bash
cd /home/alexes/projects2/trip2g && go test ./internal/model/... -v
```

Expected: all model tests PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/model/note.go internal/model/note_test.go
git commit -m "feat(model): add reverse route index and ResolveFullURL to NoteViews"
```

---

### Task 2: Add NoteURL to MCP Env + update URL building

**Files:**
- Modify: `internal/case/mcp/resolve.go`
- Regenerate: `internal/case/mcp/mocks_test.go`
- Modify: `internal/case/mcp/resolve_test.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add NoteURL to the MCP Env interface**

In `internal/case/mcp/resolve.go`, add `NoteURL` to the `Env` interface (after `PublicURL()`):

```go
type Env interface {
	similarnotes.Env
	model.FederationClientFactory
	SearchLatestNotes(query string) ([]model.SearchResult, error)
	LatestNoteChunks() []model.NoteChunk
	OpenAI() *openai.Client
	PublicURL() string
	NoteURL(note *model.NoteView) string
	Logger() logger.Logger
	FederationSecretByKBURL(ctx context.Context, kbURL string) (db.FederationSecret, bool, error)
	FederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, bool, error)
	ListFederationSecretSubgraphsByKID(ctx context.Context, kid string) ([]string, error)
	DecryptData([]byte) ([]byte, error)
	FederationMaxDepth() int
}
```

- [ ] **Step 2: Update searchResultItemFromNote signature**

Change `searchResultItemFromNote` to accept a URL resolver func instead of a plain string:

```go
func searchResultItemFromNote(note *model.NoteView, score float64, noteURL func(*model.NoteView) string) SearchResultItem {
	item := SearchResultItem{
		Title:    note.Title,
		NoteID:   note.PathID,
		NotePath: note.Path,
		Href:     note.Permalink,
		URL:      noteURL(note),
		Kind:     noteKind(note),
		Score:    score,
		TOC:      buildNoteTOC(note.Headings),
	}
	// ... rest of the function unchanged
```

- [ ] **Step 3: Update buildSearchPayload signature and call site**

Change `buildSearchPayload`:

```go
func buildSearchPayload(query string, results []model.SearchResult, noteURL func(*model.NoteView) string, chunks []model.NoteChunk) SearchResultPayload {
	payload := SearchResultPayload{Query: query}
	for _, r := range results {
		if r.NoteView == nil {
			continue
		}
		item := searchResultItemFromNote(r.NoteView, r.Score, noteURL)
		// ... rest unchanged
```

Update the call site (around line 339):

```go
payload := buildSearchPayload(args.Query, results, env.NoteURL, env.LatestNoteChunks())
```

- [ ] **Step 4: Update buildSimilarPayload and its call sites**

Change `buildSimilarPayload`:

```go
func buildSimilarPayload(sourceNote *model.NoteView, results []graphmodel.SimilarNote, noteURL func(*model.NoteView) string) SimilarResultPayload {
	payload := SimilarResultPayload{
		Source: searchResultItemFromNote(sourceNote, 1, noteURL),
	}
	for _, r := range results {
		if r.Note == nil || r.Note.NoteView == nil {
			continue
		}
		payload.Results = append(payload.Results, searchResultItemFromNote(r.Note.NoteView, r.Score, noteURL))
	}
	return payload
}
```

Update the text output line in `resolveSimilar` (around line 622):

```go
sb.WriteString(fmt.Sprintf("%d. %s (%.2f)\n   %s\n   %s\n\n", i+1, note.Title, r.Score, note.Path, env.NoteURL(note)))
```

Update the `buildSimilarPayload` call (around line 628):

```go
return successResponse(id, structuredToolResult(sb.String(), buildSimilarPayload(sourceNote, results, env.NoteURL)))
```

- [ ] **Step 5: Implement NoteURL on app**

In `cmd/server/main.go`, add after the `PublicURL()` method (around line 1501):

```go
func (a *app) NoteURL(note *model.NoteView) string {
	return a.LatestNoteViews().ResolveFullURL(note, a.config.PublicURL)
}
```

- [ ] **Step 6: Verify the build compiles**

```bash
cd /home/alexes/projects2/trip2g && go build ./...
```

Expected: compile errors in `mocks_test.go` because `NoteURL` is not yet in the mock. That's fine — the mock regeneration fixes it.

- [ ] **Step 7: Regenerate the MCP mock**

```bash
cd /home/alexes/projects2/trip2g/internal/case/mcp && go generate .
```

This regenerates `mocks_test.go` adding `NoteURLFunc func(*model.NoteView) string` and the `NoteURL` method.

- [ ] **Step 8: Update existing tests to add NoteURLFunc**

In `internal/case/mcp/resolve_test.go`, every `EnvMock` that tests `search` or `similar` results needs `NoteURLFunc`. There are 8 occurrences of `PublicURLFunc`.

For each EnvMock setup that had:
```go
PublicURLFunc: func() string {
    return "https://markavrelii.2pub.me"
},
```

Add alongside it:
```go
NoteURLFunc: func(note *appmodel.NoteView) string {
    return "https://markavrelii.2pub.me" + note.Permalink
},
```

(Use whichever base URL that test's `PublicURLFunc` returns.)

For test setups that use `PublicURLFunc: func() string { return "https://bob.team.io/..." }` or other URLs, use that same base URL in the `NoteURLFunc`.

- [ ] **Step 9: Run MCP unit tests**

```bash
cd /home/alexes/projects2/trip2g && go test ./internal/case/mcp/... -v
```

Expected: all PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/case/mcp/resolve.go internal/case/mcp/mocks_test.go internal/case/mcp/resolve_test.go cmd/server/main.go
git commit -m "feat(mcp): use domain-aware NoteURL in search and similar results"
```

---

### Task 3: Add url field to GraphQL NoteView

**Files:**
- Modify: `internal/graph/schema.graphqls`
- Modify: `internal/graph/schema.resolvers.go` (after gqlgen)

- [ ] **Step 1: Add the field to the schema**

In `internal/graph/schema.graphqls`, add `url` to the `NoteView` type (after `permalink`):

```graphql
type NoteView {
  id: String!
  path: String!
  title: String!
  content: String!
  html: String!
  permalink: String!
  url: String! @goField(forceResolver: true)
  free: Boolean!
  # ... rest unchanged
```

- [ ] **Step 2: Regenerate GraphQL code**

```bash
cd /home/alexes/projects2/trip2g && make gqlgen
```

Expected: generates resolver stub `func (r *noteViewResolver) URL(...)` in `schema.resolvers.go`.

- [ ] **Step 3: Implement the resolver**

In `internal/graph/schema.resolvers.go`, find the generated stub and implement it:

```go
// URL is the resolver for the url field.
func (r *noteViewResolver) URL(ctx context.Context, obj *appmodel.NoteView) (string, error) {
	return r.env(ctx).LatestNoteViews().ResolveFullURL(obj, r.env(ctx).PublicURL()), nil
}
```

- [ ] **Step 4: Build to verify**

```bash
cd /home/alexes/projects2/trip2g && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 5: Commit**

```bash
git add internal/graph/schema.graphqls internal/graph/schema.resolvers.go internal/graph/generated.go
git commit -m "feat(graphql): add url field to NoteView with domain-aware resolution"
```

---

### Task 4: Add seedvault note with custom domain route

**Files:**
- Create: `testdata/seedvault/mcp-url-test.md`

- [ ] **Step 1: Create the seed note**

Create `testdata/seedvault/mcp-url-test.md`:

```markdown
---
title: MCP URL Test Note
publish: true
free: true
route: customdomain.test/mcp-url-test
---

# MCP URL Test Note

This note is served on a custom domain for E2E URL resolution testing.
Unique phrase: xyzzy-mcp-url-test-sentinel.
```

The `route: customdomain.test/mcp-url-test` frontmatter means this note's URL should resolve to `http://customdomain.test/mcp-url-test` (http because the dev server uses http).

- [ ] **Step 2: Commit**

```bash
git add testdata/seedvault/mcp-url-test.md
git commit -m "test(seed): add note with custom domain route for URL E2E tests"
```

---

### Task 5: E2E test — GraphQL NoteView.url

**Files:**
- Create: `e2e/note-url.spec.js`

- [ ] **Step 1: Write the failing test**

Create `e2e/note-url.spec.js`:

```js
// @ts-check
/**
 * E2E tests for NoteView.url field — verifies domain-aware URL resolution
 * in GraphQL. A note with `route: customdomain.test/mcp-url-test` must return
 * a url starting with http://customdomain.test/, not APP_URL.
 */
import { test, expect } from '@playwright/test';
import { graphqlSignIn, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:8081';

const NOTE_URL_QUERY = `
  query {
    notePaths(filter: { like: "mcp-url-test.md" }) {
      latestNoteView {
        path
        permalink
        url
      }
    }
  }
`;

const REGULAR_NOTE_QUERY = `
  query {
    notePaths(filter: { like: "team-status.md" }) {
      latestNoteView {
        path
        permalink
        url
      }
    }
  }
`;

test.describe('NoteView.url field', () => {
  let cookie;

  test.beforeAll(async ({ request }) => {
    const jwt = await graphqlSignIn(request);
    cookie = `${USER_TOKEN_COOKIE_NAME}=${jwt}`;
  });

  test('custom domain note url uses the custom domain', async ({ request }) => {
    const res = await request.post(`${APP_URL}/graphql`, {
      headers: { 'Content-Type': 'application/json', Cookie: cookie },
      data: { query: NOTE_URL_QUERY },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.errors).toBeUndefined();

    const notePaths = body.data.notePaths;
    expect(notePaths.length).toBeGreaterThan(0);

    const noteView = notePaths[0].latestNoteView;
    expect(noteView).not.toBeNull();
    expect(noteView.url).toMatch(/^http:\/\/customdomain\.test\//);
    expect(noteView.url).toBe('http://customdomain.test/mcp-url-test');
  });

  test('regular note url uses APP_URL', async ({ request }) => {
    const res = await request.post(`${APP_URL}/graphql`, {
      headers: { 'Content-Type': 'application/json', Cookie: cookie },
      data: { query: REGULAR_NOTE_QUERY },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.errors).toBeUndefined();

    const notePaths = body.data.notePaths;
    expect(notePaths.length).toBeGreaterThan(0);

    const noteView = notePaths[0].latestNoteView;
    expect(noteView).not.toBeNull();
    expect(noteView.url).toMatch(new RegExp(`^${APP_URL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}/`));
  });
});
```

- [ ] **Step 2: Run E2E tests (expect pass after server restart with new seed)**

```bash
cd /home/alexes/projects2/trip2g && npm run test:e2e -- --grep "NoteView.url"
```

Expected: both tests PASS (requires the dev server to be running with the seedvault loaded).

- [ ] **Step 3: Commit**

```bash
git add e2e/note-url.spec.js
git commit -m "test(e2e): verify NoteView.url uses custom domain when route is set"
```

---

### Task 6: E2E test — MCP search domain-aware URLs

**Files:**
- Create: `e2e/mcp-url.spec.js`

- [ ] **Step 1: Write the failing test**

Create `e2e/mcp-url.spec.js`:

```js
// @ts-check
/**
 * E2E tests for MCP search URL resolution.
 * A note with `route: customdomain.test/mcp-url-test` must appear in
 * search results with url = http://customdomain.test/mcp-url-test.
 * Regular notes must have urls starting with APP_URL.
 *
 * JSON field names (from MCP types.go):
 *   SearchResultPayload: { query, results[] }
 *   SearchResultItem:    { note_path, url, title, href, note_id, score }
 *   Response:            { result: { content[], structuredContent } }
 */
import { test, expect } from '@playwright/test';
import { graphqlSignIn, createPersonalToken, revokePersonalToken, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

const APP_URL = process.env.APP_URL || 'http://localhost:8081';
const MCP_URL = `${APP_URL}/_system/mcp`;

async function mcpCall(request, method, params = {}, token) {
  const res = await request.post(MCP_URL, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    data: { jsonrpc: '2.0', id: 1, method, params },
  });
  expect(res.ok()).toBeTruthy();
  return res.json();
}

// Returns SearchResultPayload: { query, results: [{ note_path, url, ... }] }
async function mcpSearch(request, query, token) {
  const resp = await mcpCall(request, 'tools/call', {
    name: 'search',
    arguments: { query },
  }, token);
  expect(resp.error).toBeUndefined();
  return resp.result?.structuredContent;
}

test.describe.serial('MCP search URL resolution', () => {
  let adminRequest;
  let adminCookie;
  let token;
  let tokenId;

  test.beforeAll(async ({ playwright }) => {
    adminRequest = await playwright.request.newContext({ baseURL: APP_URL });
    const jwt = await graphqlSignIn(adminRequest);
    adminCookie = `${USER_TOKEN_COOKIE_NAME}=${jwt}`;

    const result = await createPersonalToken(adminRequest, APP_URL, adminCookie, {
      name: 'e2e-mcp-url',
      expiresInDays: 1,
    });
    token = result.plaintextToken;
    tokenId = result.id;

    // Initialize MCP session
    await mcpCall(adminRequest, 'initialize', {
      protocolVersion: '2024-11-05',
      capabilities: {},
      clientInfo: { name: 'e2e-test', version: '1' },
    }, token);
  });

  test.afterAll(async () => {
    if (tokenId) {
      await revokePersonalToken(adminRequest, APP_URL, adminCookie, tokenId).catch(() => {});
    }
    await adminRequest?.dispose();
  });

  test('custom domain note appears with custom domain URL', async () => {
    const payload = await mcpSearch(adminRequest, 'xyzzy-mcp-url-test-sentinel', token);
    expect(payload).not.toBeNull();
    expect(payload.results?.length).toBeGreaterThan(0);

    const customNote = payload.results.find(r => r.note_path === 'mcp-url-test.md');
    expect(customNote).toBeDefined();
    expect(customNote.url).toBe('http://customdomain.test/mcp-url-test');
  });

  test('regular note appears with main domain URL', async () => {
    const payload = await mcpSearch(adminRequest, 'Weekly team status', token);
    expect(payload).not.toBeNull();
    expect(payload.results?.length).toBeGreaterThan(0);

    const regularNote = payload.results.find(r => r.note_path === 'team-status.md');
    expect(regularNote).toBeDefined();
    expect(regularNote.url).toMatch(new RegExp(`^${APP_URL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}/`));
  });
});
```

- [ ] **Step 2: Run E2E tests**

```bash
cd /home/alexes/projects2/trip2g && npm run test:e2e -- --grep "MCP search URL"
```

Expected: both tests PASS.

- [ ] **Step 3: Commit**

```bash
git add e2e/mcp-url.spec.js
git commit -m "test(e2e): verify MCP search returns custom domain URLs"
```
