# MCP API Key Auth + Admin GraphQL Tools Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Accept API keys in MCP endpoint and add opt-in `graphql_introspection` / `graphql_request` tools gated by `enable_mcp_admin_tools` flag.

**Architecture:** Add X-API-Key detection as a third auth path in `mcp/endpoint.go` (between personal token and federation JWT). Store `mcpAPIKeyAuth` bool in context. Add two GraphQL tools to `mcp/resolve.go` visible only when `mcpAdminToolsEnabled(ctx)` is true. Execute GraphQL programmatically via gqlgen's `CreateOperationContext` + `DispatchOperation` with an admin-injected appreq.

**Tech Stack:** Go, gqlgen, sqlc, fasthttp, SQLite, quicktemplate (none new)

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `db/migrations/20260506120000_add_enable_mcp_admin_tools.sql` | Create | SQLite migration |
| `db/schema.sql` | Modify | Add column to `api_keys` table |
| `queries.write.sql` | Modify | Add `SetApiKeyMcpAdminTools` query |
| `internal/db/models.go` | Auto (sqlc) | `db.ApiKey` gains `EnableMcpAdminTools` |
| `internal/db/queries.write.sql.go` | Auto (sqlc) | `SetApiKeyMcpAdminTools` method |
| `internal/graph/schema.graphqls` | Modify | `enableMcpAdminTools` field + mutation |
| `internal/graph/model/models_gen.go` | Auto (gqlgen) | Input/payload types |
| `internal/graph/generated.go` | Auto (gqlgen) | Resolver dispatch |
| `internal/graph/schema.resolvers.go` | Modify | Implement mutation resolver |
| `internal/case/setapikeymcpadmintools/resolve.go` | Create | Use case: toggle flag |
| `internal/case/setapikeymcpadmintools/resolve_test.go` | Create | Unit tests |
| `internal/case/setapikeymcpadmintools/mocks_test.go` | Auto (moq) | Mock for Env |
| `internal/appreq/request.go` | Modify | Add `WithAdminToken` helper |
| `internal/appreq/request_test.go` | Modify | Test `WithAdminToken` |
| `internal/case/mcp/resolve.go` | Modify | Context helpers, canReadMCPNote, new tools |
| `internal/case/mcp/types.go` | Modify | Arg types for new tools |
| `internal/case/mcp/endpoint.go` | Modify | X-API-Key auth + GET text update |
| `internal/case/mcp/mocks_test.go` | Auto (moq) | Regenerate after Env change |
| `cmd/server/main.go` | Modify | `gqlServer` field, `ResolveAPIKey`, `GraphQLRequest` |
| `docs/en/user/mcp.md` | Modify | API key section + new tools |
| `docs/ru/user/mcp.md` | Modify | Same in Russian |
| `docs/en/user/agent_admin.md` | Create | Agent admin concept (EN) |
| `docs/ru/user/agent_admin.md` | Create | Agent admin concept (RU) |

---

### Task 1: DB migration and schema.sql

**Files:**
- Create: `db/migrations/20260506120000_add_enable_mcp_admin_tools.sql`
- Modify: `db/schema.sql`

- [ ] **Step 1: Create migration file**

```sql
-- db/migrations/20260506120000_add_enable_mcp_admin_tools.sql
-- migrate:up
ALTER TABLE api_keys ADD COLUMN enable_mcp_admin_tools boolean;

-- migrate:down
-- SQLite does not support DROP COLUMN before 3.35.0 – no-op
```

- [ ] **Step 2: Update db/schema.sql**

Find the `api_keys` table definition (line ~104). Change:
```sql
, skip_webhooks boolean not null default false);
```
to:
```sql
, skip_webhooks boolean not null default false
, enable_mcp_admin_tools boolean);
```

- [ ] **Step 3: Commit**

```bash
git add db/migrations/20260506120000_add_enable_mcp_admin_tools.sql db/schema.sql
git commit -m "feat(db): add enable_mcp_admin_tools to api_keys"
```

---

### Task 2: SQL write query + sqlc regeneration

**Files:**
- Modify: `queries.write.sql`
- Auto-update: `internal/db/models.go`, `internal/db/queries.write.sql.go`

- [ ] **Step 1: Add write query**

In `queries.write.sql`, after the `DisableApiKey` block, add:

```sql
-- name: SetApiKeyMcpAdminTools :exec
update api_keys set enable_mcp_admin_tools = sqlc.arg(enabled) where id = sqlc.arg(id);
```

- [ ] **Step 2: Run sqlc**

```bash
make sqlc
```

Expected: `internal/db/models.go` — `ApiKey` struct gains `EnableMcpAdminTools *bool`. `internal/db/queries.write.sql.go` gains `SetApiKeyMcpAdminTools` method.

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add queries.write.sql internal/db/
git commit -m "feat(db): add SetApiKeyMcpAdminTools sqlc query"
```

---

### Task 3: GraphQL schema update + gqlgen

**Files:**
- Modify: `internal/graph/schema.graphqls`
- Auto-update: `internal/graph/model/models_gen.go`, `internal/graph/generated.go`, `internal/graph/schema.resolvers.go`

- [ ] **Step 1: Add field to AdminApiKey type**

In `schema.graphqls`, find the `AdminApiKey` type (line ~192). Add field:

```graphql
type AdminApiKey @goModel(model: "trip2g/internal/db.ApiKey") {
  # ... existing fields ...
  enableMcpAdminTools: Boolean
}
```

- [ ] **Step 2: Add mutation input and types**

After the `# disableApiKey` block (line ~1861), add:

```graphql
# setApiKeyMcpAdminTools
# Toggles the enable_mcp_admin_tools flag on an API key.

input SetApiKeyMcpAdminToolsInput {
  id: Int64!
  enabled: Boolean!
}

type SetApiKeyMcpAdminToolsPayload {
  apiKey: AdminApiKey!
}

union SetApiKeyMcpAdminToolsOrErrorPayload = SetApiKeyMcpAdminToolsPayload | ErrorPayload
```

- [ ] **Step 3: Register mutation**

In `schema.graphqls`, find the `AdminMutation` type (look for `createApiKey` at line ~2712). Add:

```graphql
setApiKeyMcpAdminTools(input: SetApiKeyMcpAdminToolsInput!): SetApiKeyMcpAdminToolsOrErrorPayload!
```

- [ ] **Step 4: Run gqlgen**

```bash
make gqlgen
```

Expected: `schema.resolvers.go` gains stub for `SetApiKeyMcpAdminTools`. `models_gen.go` gains new input/payload types.

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/graph/schema.graphqls internal/graph/model/ internal/graph/generated.go internal/graph/schema.resolvers.go
git commit -m "feat(graphql): add enableMcpAdminTools field and setApiKeyMcpAdminTools mutation"
```

---

### Task 4: Use case setapikeymcpadmintools

**Files:**
- Create: `internal/case/setapikeymcpadmintools/resolve.go`
- Create: `internal/case/setapikeymcpadmintools/resolve_test.go`
- Create: `internal/case/setapikeymcpadmintools/mocks_test.go` (via go generate)
- Modify: `internal/graph/schema.resolvers.go`

- [ ] **Step 1: Write failing test**

Create `internal/case/setapikeymcpadmintools/resolve_test.go`:

```go
package setapikeymcpadmintools_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"trip2g/internal/case/setapikeymcpadmintools"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

func TestResolve_AdminCanEnable(t *testing.T) {
	env := &EnvMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{Role: "admin"}, nil
		},
		SetApiKeyMcpAdminToolsFunc: func(ctx context.Context, id int64, enabled bool) error {
			require.Equal(t, int64(42), id)
			require.True(t, enabled)
			return nil
		},
		ApiKeyByIDFunc: func(ctx context.Context, id int64) (db.ApiKey, error) {
			return db.ApiKey{ID: id}, nil
		},
	}

	result, err := setapikeymcpadmintools.Resolve(context.Background(), env, setapikeymcpadmintools.Input{ID: 42, Enabled: true})
	require.NoError(t, err)
	require.Equal(t, int64(42), result.ID)
}

func TestResolve_NonAdminRejected(t *testing.T) {
	env := &EnvMock{
		CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return nil, errors.New("not admin")
		},
	}

	_, err := setapikeymcpadmintools.Resolve(context.Background(), env, setapikeymcpadmintools.Input{ID: 1, Enabled: true})
	require.Error(t, err)
}
```

- [ ] **Step 2: Generate mock**

Create `internal/case/setapikeymcpadmintools/resolve.go` with the Env interface and go:generate directive, then run:

```go
package setapikeymcpadmintools

import (
	"context"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg setapikeymcpadmintools_test . Env

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	SetApiKeyMcpAdminTools(ctx context.Context, id int64, enabled bool) error
	ApiKeyByID(ctx context.Context, id int64) (db.ApiKey, error)
}

type Input struct {
	ID      int64
	Enabled bool
}

func Resolve(ctx context.Context, env Env, input Input) (db.ApiKey, error) {
	panic("not implemented")
}
```

```bash
cd internal/case/setapikeymcpadmintools && go generate ./...
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/case/setapikeymcpadmintools/... -v
```

Expected: FAIL (panic "not implemented")

- [ ] **Step 4: Check if ApiKeyByID query exists**

```bash
grep -n "ApiKeyByID\|api_keys where id" /home/alexes/projects2/trip2g/queries.read.sql | head -5
```

If missing, add to `queries.read.sql`:

```sql
-- name: ApiKeyByID :one
select * from api_keys where id = ? limit 1;
```

Then run `make sqlc`.

- [ ] **Step 5: Implement Resolve**

```go
func Resolve(ctx context.Context, env Env, input Input) (db.ApiKey, error) {
	if _, err := env.CurrentAdminUserToken(ctx); err != nil {
		return db.ApiKey{}, err
	}
	if err := env.SetApiKeyMcpAdminTools(ctx, input.ID, input.Enabled); err != nil {
		return db.ApiKey{}, err
	}
	return env.ApiKeyByID(ctx, input.ID)
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
go test ./internal/case/setapikeymcpadmintools/... -v
```

Expected: PASS

- [ ] **Step 7: Implement GraphQL resolver**

In `internal/graph/schema.resolvers.go`, find the generated stub for `SetApiKeyMcpAdminTools` and implement:

```go
func (r *adminMutationResolver) SetApiKeyMcpAdminTools(ctx context.Context, obj *appmodel.AdminMutation, input model.SetApiKeyMcpAdminToolsInput) (model.SetApiKeyMcpAdminToolsOrErrorPayload, error) {
	key, err := setapikeymcpadmintools.Resolve(ctx, r.env(ctx), setapikeymcpadmintools.Input{
		ID:      input.ID,
		Enabled: input.Enabled,
	})
	if err != nil {
		return model.ErrorPayload{Message: err.Error()}, nil
	}
	return model.SetApiKeyMcpAdminToolsPayload{APIKey: &key}, nil
}
```

Add import `"trip2g/internal/case/setapikeymcpadmintools"` to the file.

- [ ] **Step 8: Wire Env methods in cmd/server/main.go**

Add to the `app` struct's existing DB method delegation (search for `func (a *app) DisableApiKey` for placement):

```go
func (a *app) SetApiKeyMcpAdminTools(ctx context.Context, id int64, enabled bool) error {
	return a.db.Write().SetApiKeyMcpAdminTools(ctx, db.SetApiKeyMcpAdminToolsParams{
		ID:      id,
		Enabled: &enabled,
	})
}

func (a *app) ApiKeyByID(ctx context.Context, id int64) (db.ApiKey, error) {
	return a.db.Read().ApiKeyByID(ctx, id)
}
```

- [ ] **Step 9: Verify build**

```bash
go build ./...
```

- [ ] **Step 10: Commit**

```bash
git add internal/case/setapikeymcpadmintools/ internal/graph/schema.resolvers.go queries.read.sql internal/db/ cmd/server/main.go
git commit -m "feat(admin): setApiKeyMcpAdminTools mutation and use case"
```

---

### Task 5: appreq.WithAdminToken helper

**Files:**
- Modify: `internal/appreq/request.go`
- Modify: `internal/appreq/request_test.go`

- [ ] **Step 1: Write failing test**

In `internal/appreq/request_test.go`, add:

```go
func TestWithAdminToken_ShadowsExistingToken(t *testing.T) {
	fctx := &fasthttp.RequestCtx{}
	req := &Request{Req: fctx}
	req.SetUserToken(&usertoken.Data{Role: "user"})
	req.StoreInContext()

	ctx := appreq.WithAdminToken(fctx)

	shadowed, err := appreq.FromCtx(ctx)
	require.NoError(t, err)

	tok, err := shadowed.UserToken()
	require.NoError(t, err)
	require.True(t, tok.IsAdmin())

	// original request in fctx unchanged
	origReq, _ := appreq.FromCtx(fctx)
	origTok, _ := origReq.UserToken()
	require.False(t, origTok.IsAdmin())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/appreq/... -run TestWithAdminToken -v
```

Expected: FAIL (undefined: appreq.WithAdminToken)

- [ ] **Step 3: Implement WithAdminToken**

In `internal/appreq/request.go`, add after the `FromCtx` function:

```go
// WithAdminToken returns a context with a copy of the current appreq where
// UserToken is pre-set to an admin token. Used for internal GraphQL calls.
func WithAdminToken(ctx context.Context) context.Context {
	req, err := FromCtx(ctx)
	if err != nil {
		return ctx
	}
	adminReq := &Request{
		Req:          req.Req,
		Env:          req.Env,
		TokenManager: req.TokenManager,
	}
	adminReq.SetUserToken(&usertoken.Data{Role: "admin"})
	return context.WithValue(ctx, ctxKey, adminReq)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/appreq/... -run TestWithAdminToken -v
```

Expected: PASS

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/appreq/
git commit -m "feat(appreq): add WithAdminToken context helper"
```

---

### Task 6: MCP context helpers

**Files:**
- Modify: `internal/case/mcp/resolve.go`

- [ ] **Step 1: Write failing test**

In `internal/case/mcp/resolve_internal_test.go`, add:

```go
func TestMCPAPIKeyContext(t *testing.T) {
	ctx := context.Background()

	require.False(t, mcpAPIKeyAuthed(ctx))
	require.False(t, mcpAdminToolsEnabled(ctx))

	ctx = contextWithMCPAPIKeyAuth(ctx, false)
	require.True(t, mcpAPIKeyAuthed(ctx))
	require.False(t, mcpAdminToolsEnabled(ctx))

	ctx = contextWithMCPAPIKeyAuth(ctx, true)
	require.True(t, mcpAPIKeyAuthed(ctx))
	require.True(t, mcpAdminToolsEnabled(ctx))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/case/mcp/... -run TestMCPAPIKeyContext -v
```

Expected: FAIL (undefined)

- [ ] **Step 3: Add context helpers to resolve.go**

In `internal/case/mcp/resolve.go`, after the `federationDepthContextKey` block (around line 407), add:

```go
type mcpAPIKeyAuthContextKey struct{}

type mcpAPIKeyAuthInfo struct {
	adminTools bool
}

func contextWithMCPAPIKeyAuth(ctx context.Context, adminTools bool) context.Context {
	return context.WithValue(ctx, mcpAPIKeyAuthContextKey{}, mcpAPIKeyAuthInfo{adminTools: adminTools})
}

func mcpAPIKeyAuthed(ctx context.Context) bool {
	_, ok := ctx.Value(mcpAPIKeyAuthContextKey{}).(mcpAPIKeyAuthInfo)
	return ok
}

func mcpAdminToolsEnabled(ctx context.Context) bool {
	info, ok := ctx.Value(mcpAPIKeyAuthContextKey{}).(mcpAPIKeyAuthInfo)
	return ok && info.adminTools
}
```

- [ ] **Step 4: Update canReadMCPNote to grant admin access for API key auth**

In `resolve.go`, update `canReadMCPNote`:

```go
func canReadMCPNote(ctx context.Context, env Env, note *model.NoteView) (bool, error) {
	if mcpAPIKeyAuthed(ctx) {
		return true, nil // API key = admin, sees all notes
	}
	if auth, ok := federationAuthFromContext(ctx); ok {
		return canreadnote.ResolveWithSubgraphs(ctx, env, note, auth.AllowedSubgraphs)
	}
	return env.CanReadNote(ctx, note)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/case/mcp/... -v
```

Expected: all pass

- [ ] **Step 6: Commit**

```bash
git add internal/case/mcp/resolve.go internal/case/mcp/resolve_internal_test.go
git commit -m "feat(mcp): add API key auth context helpers and admin canRead bypass"
```

---

### Task 7: Env interface additions

**Files:**
- Modify: `internal/case/mcp/resolve.go` (Env interface)
- Modify: `internal/case/mcp/types.go`

- [ ] **Step 1: Extend Env interface**

In `resolve.go`, extend the `Env` interface (around line 39):

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
	// API key auth
	ResolveAPIKey(ctx context.Context, value, action string) (*db.ApiKey, error)
	// Admin GraphQL tools
	GraphQLRequest(ctx context.Context, query string, variables map[string]any) ([]byte, error)
}
```

- [ ] **Step 2: Add arg types for new MCP tools**

In `internal/case/mcp/types.go`, add:

```go
type GraphQLIntrospectionArguments struct {
	Pattern string `json:"pattern"`
}

type GraphQLRequestArguments struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}
```

Also add `graphql_introspection` and `graphql_request` to `reservedMCPTools` in `resolve.go`:

```go
var reservedMCPTools = map[string]bool{
	"search":                true,
	"similar":               true,
	"note_html":             true,
	"federated_search":      true,
	"federated_similar":     true,
	"federated_note_html":   true,
	"graphql_introspection": true,
	"graphql_request":       true,
	MCPMethodInitialize:     true,
}
```

- [ ] **Step 3: Verify build fails with clear error**

```bash
go build ./... 2>&1 | grep "does not implement"
```

Expected: `app` does not implement `mcp.Env` (missing `ResolveAPIKey`, `GraphQLRequest`).

- [ ] **Step 4: Commit**

```bash
git add internal/case/mcp/resolve.go internal/case/mcp/types.go
git commit -m "feat(mcp): extend Env interface with ResolveAPIKey and GraphQLRequest"
```

---

### Task 8: Implement ResolveAPIKey and GraphQLRequest in cmd/server/main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Store gqlServer as app field**

In `cmd/server/main.go`, find the `app` struct definition and add:

```go
gqlServer *handler.Server
```

Find `prepareGraphQLHandler` (line ~2246). Move the `gqlHandler` creation before the return, store it on `a`:

```go
func (a *app) prepareGraphQLHandler() func(ctx *fasthttp.RequestCtx, path string) bool {
	playgroundHandler := fasthttpadaptor.NewFastHTTPHandler(playground.Handler("GraphQL playground", "/graphql"))

	gqlMetrics := metrics.NewGraphQLMetrics()

	a.gqlServer = graph.NewHandler(a)   // <-- store on app
	a.gqlServer.Use(gqlMetrics)
	a.gqlServer.AroundOperations(gqlMetrics.Middleware())
	graphqlHandler := fasthttpadaptor.NewFastHTTPHandler(a.gqlServer)
	// ... rest unchanged
```

- [ ] **Step 2: Implement ResolveAPIKey**

Add near the existing API key DB wrappers (search for `DisableApiKey` for placement):

```go
func (a *app) ResolveAPIKey(ctx context.Context, value, action string) (*db.ApiKey, error) {
	// Try hashed value first (new keys), fall back to plain (old keys).
	hash := sha256.Sum256([]byte(value))
	hashedValue := hex.EncodeToString(hash[:])

	apiKey, err := a.db.Read().ApiKeyByValue(ctx, hashedValue)
	if err != nil && !db.IsNoFound(err) {
		return nil, fmt.Errorf("resolve api key: %w", err)
	}
	if db.IsNoFound(err) {
		apiKey, err = a.db.Read().ApiKeyByValue(ctx, value)
		if err != nil && !db.IsNoFound(err) {
			return nil, fmt.Errorf("resolve api key (plain): %w", err)
		}
		if db.IsNoFound(err) {
			return nil, errors.New("invalid API key")
		}
	}

	req, _ := appreq.FromCtx(ctx)
	ip := req.Req.RemoteIP().String()

	if err := a.db.Write().UpsertAPIKeyLogAction(ctx, action); err != nil {
		return nil, fmt.Errorf("log action: %w", err)
	}
	if err := a.db.Write().UpsertAPIKeyLogIP(ctx, ip); err != nil {
		return nil, fmt.Errorf("log ip: %w", err)
	}
	if err := a.db.Write().InsertAPIKeyLog(ctx, db.InsertAPIKeyLogParams{
		ApiKeyID: apiKey.ID,
		Action:   action,
		Ip:       ip,
	}); err != nil {
		return nil, fmt.Errorf("insert log: %w", err)
	}

	return &apiKey, nil
}
```

Add imports: `"crypto/sha256"`, `"encoding/hex"`.

- [ ] **Step 3: Implement GraphQLRequest**

```go
func (a *app) GraphQLRequest(ctx context.Context, query string, variables map[string]any) ([]byte, error) {
	// Shadow appreq with admin token so GraphQL resolvers see an admin user.
	ctx = appreq.WithAdminToken(ctx)

	params := &graphql.RawParams{
		Query:     query,
		Variables: variables,
	}

	opCtx, errList := a.gqlServer.CreateOperationContext(ctx, params)
	if errList != nil {
		return nil, fmt.Errorf("graphql operation context: %v", errList)
	}

	responseHandler, respCtx := a.gqlServer.DispatchOperation(ctx, opCtx)
	resp := responseHandler(respCtx)
	return json.Marshal(resp)
}
```

Add import: `"github.com/99designs/gqlgen/graphql"`.

- [ ] **Step 4: Verify app implements mcp.Env**

```bash
go build ./... 2>&1
```

Expected: no errors (the `var _ mcp.Env = (*app)(nil)` assertion at line ~125 will catch mismatches).

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(server): implement ResolveAPIKey and GraphQLRequest for MCP env"
```

---

### Task 9: X-API-Key auth in MCP endpoint

**Files:**
- Modify: `internal/case/mcp/endpoint.go`

- [ ] **Step 1: Write failing test**

In `internal/case/mcp/endpoint_dispatch_test.go`, add a test that verifies API key auth path succeeds and sets admin tools context when `EnableMcpAdminTools` is true:

```go
func TestDispatch_APIKeyAuth_AdminTools(t *testing.T) {
	env := buildDispatchEnv(t, true) // federation must NOT be called
	adminTools := true
	env.ResolveAPIKeyFunc = func(ctx context.Context, value, action string) (*db.ApiKey, error) {
		require.Equal(t, "test-api-key", value)
		require.Equal(t, "mcp", action)
		return &db.ApiKey{ID: 1, EnableMcpAdminTools: &adminTools}, nil
	}

	fctx := buildMCPFasthttpCtx(mcpInitBody, "")
	fctx.Request.Header.Set("X-API-Key", "test-api-key")

	req := buildMCPRequest(t, fctx, env)
	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)
	require.Equal(t, 1, len(env.ResolveAPIKeyCalls()))
}

func TestDispatch_APIKeyAuth_InvalidKey(t *testing.T) {
	env := buildDispatchEnv(t, true)
	env.ResolveAPIKeyFunc = func(ctx context.Context, value, action string) (*db.ApiKey, error) {
		return nil, errors.New("invalid API key")
	}

	fctx := buildMCPFasthttpCtx(mcpInitBody, "")
	fctx.Request.Header.Set("X-API-Key", "bad-key")

	req := buildMCPRequest(t, fctx, env)
	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err) // error is in JSON response body, not Go error

	var resp mcp.Response
	json.Unmarshal(fctx.Response.Body(), &resp)
	require.NotNil(t, resp.Error)
	require.Contains(t, resp.Error.Message, "Auth failed")
}
```

Add `buildMCPRequest` helper if not present:

```go
func buildMCPRequest(t *testing.T, fctx *fasthttp.RequestCtx, env *EnvMock) *appreq.Request {
	t.Helper()
	req := &appreq.Request{
		Req: fctx,
		Env: env,
		TokenManager: nil, // no cookie auth in tests
	}
	req.StoreInContext()
	return req
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/case/mcp/... -run TestDispatch_APIKey -v
```

Expected: FAIL (compile error — ResolveAPIKey not in mock yet; fix by regenerating mock first)

- [ ] **Step 3: Regenerate mocks**

```bash
cd internal/case/mcp && go generate ./...
```

- [ ] **Step 4: Run tests again to confirm failure**

```bash
go test ./internal/case/mcp/... -run TestDispatch_APIKey -v
```

Expected: FAIL (runtime — no X-API-Key handling in endpoint.go)

- [ ] **Step 5: Implement X-API-Key detection in endpoint.go**

In `mcp/endpoint.go`, in the `Handle` method, update the auth block. After the personal token check and before the federation JWT block:

```go
	if userToken == nil {
		// Check for API key auth.
		apiKeyValue := strings.TrimSpace(string(req.Req.Request.Header.Peek("X-API-Key")))
		if apiKeyValue != "" {
			apiKey, keyErr := env.ResolveAPIKey(req.Req, apiKeyValue, "mcp")
			if keyErr != nil {
				return writeJSONResponse(req, errorResponse(rpcReq.ID, ErrCodeInternal, "Auth failed: "+keyErr.Error()))
			}
			adminTools := apiKey.EnableMcpAdminTools != nil && *apiKey.EnableMcpAdminTools
			resolveCtx = contextWithMCPAPIKeyAuth(resolveCtx, adminTools)
		} else {
			// No API key — check for federation JWT Bearer.
			authHeader := strings.TrimSpace(string(req.Req.Request.Header.Peek("Authorization")))
			token, isBearerToken := strings.CutPrefix(authHeader, "Bearer ")
			if authHeader != "" && (!isBearerToken || strings.TrimSpace(token) == "") {
				return writeJSONResponse(req, errorResponse(rpcReq.ID, ErrCodeInternal, "Federation auth failed: malformed bearer token"))
			}
			if isBearerToken && strings.TrimSpace(token) != "" {
				kid, allowedSubgraphs, verifyErr := verifyInbound(req.Req, env, strings.TrimSpace(token))
				if verifyErr != nil {
					return writeJSONResponse(req, errorResponse(rpcReq.ID, ErrCodeInternal, "Federation auth failed: "+verifyErr.Error()))
				}
				resolveCtx = contextWithFederationAuth(resolveCtx, kid, allowedSubgraphs)
			}
		}
	}
```

Note: `env.ResolveAPIKey` signature is `(ctx context.Context, value, action string)` — pass `req.Req` as ctx (fasthttp RequestCtx implements context.Context).

- [ ] **Step 6: Update GET endpoint auth description**

In `GetEndpoint.Handle`, update the body string to include `X-API-Key`:

```go
Authentication (one of):
  Authorization: Bearer t2g_<your-token>
  ?token=t2g_<your-token>
  X-API-Key: <your-api-key>
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/case/mcp/... -v
```

Expected: all pass

- [ ] **Step 8: Commit**

```bash
git add internal/case/mcp/endpoint.go internal/case/mcp/mocks_test.go
git commit -m "feat(mcp): add X-API-Key authentication support"
```

---

### Task 10: graphql_introspection and graphql_request tools

**Files:**
- Modify: `internal/case/mcp/resolve.go`

- [ ] **Step 1: Write failing tests**

In `internal/case/mcp/resolve_test.go`, add:

```go
func TestToolsList_GraphQLToolsHiddenWithoutAdminTools(t *testing.T) {
	env := buildMinimalEnv(t)
	ctx := context.Background()

	req := Request{JSONRPC: "2.0", Method: "tools/list", ID: 1}
	resp := Resolve(ctx, env, req)

	result := resp.Result.(ListToolsResult)
	for _, tool := range result.Tools {
		require.NotEqual(t, "graphql_introspection", tool.Name)
		require.NotEqual(t, "graphql_request", tool.Name)
	}
}

func TestToolsList_GraphQLToolsVisibleWithAdminTools(t *testing.T) {
	env := buildMinimalEnv(t)
	ctx := contextWithMCPAPIKeyAuth(context.Background(), true)

	req := Request{JSONRPC: "2.0", Method: "tools/list", ID: 1}
	resp := Resolve(ctx, env, req)

	result := resp.Result.(ListToolsResult)
	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	require.Contains(t, names, "graphql_introspection")
	require.Contains(t, names, "graphql_request")
}
```

Add `buildMinimalEnv` helper in the test file if not present (minimal mock returning empty NoteViews).

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/case/mcp/... -run TestToolsList_GraphQL -v
```

Expected: FAIL

- [ ] **Step 3: Add tools to handleToolsList**

In `resolve.go`, in `handleToolsList`, after the dynamic tools loop, add:

```go
	if mcpAdminToolsEnabled(ctx) {
		tools = append(tools, Tool{
			Name:        "graphql_introspection",
			Description: "Inspect the GraphQL schema. Returns types and operations matching the pattern (regexp), plus all types they reference. Use this to discover available mutations and queries before calling graphql_request.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"pattern": {Type: "string", Description: "Regexp or substring to filter type and operation names"},
				},
				Required: []string{"pattern"},
			},
		}, Tool{
			Name:        "graphql_request",
			Description: "Execute a GraphQL query or mutation as admin. Use graphql_introspection first to find the right operation.",
			InputSchema: &InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":     {Type: "string", Description: "GraphQL query or mutation string"},
					"variables": {Type: "object", Description: "Optional variables map"},
				},
				Required: []string{"query"},
			},
		})
	}
```

- [ ] **Step 4: Add cases to handleToolsCall**

In `handleToolsCall`, add before the `default` case:

```go
	case "graphql_introspection":
		if !mcpAdminToolsEnabled(ctx) {
			return errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found: graphql_introspection")
		}
		return handleGraphQLIntrospection(ctx, env, req.ID, params.Arguments)
	case "graphql_request":
		if !mcpAdminToolsEnabled(ctx) {
			return errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found: graphql_request")
		}
		return handleGraphQLRequest(ctx, env, req.ID, params.Arguments)
```

- [ ] **Step 5: Implement handleGraphQLRequest**

```go
func handleGraphQLRequest(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[GraphQLRequestArguments](argsRaw, id, "graphql_request")
	if errResp != nil {
		return *errResp
	}
	if args.Query == "" {
		return errorResponse(id, ErrCodeInvalidParams, "query is required")
	}

	result, err := env.GraphQLRequest(ctx, args.Query, args.Variables)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "GraphQL request failed: "+err.Error())
	}

	return successResponse(id, textToolResult(string(result)))
}
```

- [ ] **Step 6: Implement handleGraphQLIntrospection**

```go
const fullIntrospectionQuery = `{
  __schema {
    queryType { name }
    mutationType { name }
    types {
      kind name description
      fields(includeDeprecated: true) {
        name description
        args { name description type { kind name ofType { kind name ofType { kind name } } } defaultValue }
        type { kind name ofType { kind name ofType { kind name } } }
      }
      inputFields {
        name description type { kind name ofType { kind name ofType { kind name } } } defaultValue
      }
      interfaces { kind name ofType { kind name } }
      enumValues(includeDeprecated: true) { name description }
      possibleTypes { kind name }
    }
  }
}`

func handleGraphQLIntrospection(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[GraphQLIntrospectionArguments](argsRaw, id, "graphql_introspection")
	if errResp != nil {
		return *errResp
	}
	if args.Pattern == "" {
		return errorResponse(id, ErrCodeInvalidParams, "pattern is required")
	}

	raw, err := env.GraphQLRequest(ctx, fullIntrospectionQuery, nil)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "Introspection failed: "+err.Error())
	}

	filtered, err := filterIntrospection(raw, args.Pattern)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "Filter failed: "+err.Error())
	}

	return successResponse(id, textToolResult(string(filtered)))
}
```

- [ ] **Step 7: Implement filterIntrospection**

Add to `resolve.go` (or a new `graphql_tools.go` file in the same package):

```go
type introspectionTypeRef struct {
	Kind   string               `json:"kind"`
	Name   string               `json:"name"`
	OfType *introspectionTypeRef `json:"ofType,omitempty"`
}

type introspectionField struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Args        []introspectionArg   `json:"args,omitempty"`
	Type        introspectionTypeRef `json:"type"`
}

type introspectionArg struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Type        introspectionTypeRef `json:"type"`
	DefaultValue *string             `json:"defaultValue,omitempty"`
}

type introspectionType struct {
	Kind        string               `json:"kind"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Fields      []introspectionField `json:"fields,omitempty"`
	InputFields []introspectionArg   `json:"inputFields,omitempty"`
	Interfaces  []introspectionTypeRef `json:"interfaces,omitempty"`
	EnumValues  []struct {
		Name string `json:"name"`
	} `json:"enumValues,omitempty"`
	PossibleTypes []introspectionTypeRef `json:"possibleTypes,omitempty"`
}

func typeRefName(ref *introspectionTypeRef) string {
	if ref == nil {
		return ""
	}
	if ref.Name != "" {
		return ref.Name
	}
	return typeRefName(ref.OfType)
}

func collectReferencedNames(t introspectionType, out map[string]bool) {
	for _, f := range t.Fields {
		out[typeRefName(&f.Type)] = true
		for _, a := range f.Args {
			out[typeRefName(&a.Type)] = true
		}
	}
	for _, f := range t.InputFields {
		out[typeRefName(&f.Type)] = true
	}
	for _, i := range t.Interfaces {
		out[typeRefName(&i)] = true
	}
	for _, p := range t.PossibleTypes {
		out[typeRefName(&p)] = true
	}
}

func filterIntrospection(data []byte, pattern string) ([]byte, error) {
	var wrapper struct {
		Data struct {
			Schema struct {
				QueryType    *struct{ Name string } `json:"queryType"`
				MutationType *struct{ Name string } `json:"mutationType"`
				Types        []introspectionType    `json:"types"`
			} `json:"__schema"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		re = regexp.MustCompile("(?i)" + regexp.QuoteMeta(pattern))
	}

	allTypes := wrapper.Data.Schema.Types
	typeByName := make(map[string]introspectionType, len(allTypes))
	for _, t := range allTypes {
		typeByName[t.Name] = t
	}

	// Seed: types whose name matches.
	needed := map[string]bool{}
	for _, t := range allTypes {
		if strings.HasPrefix(t.Name, "__") {
			continue
		}
		if re.MatchString(t.Name) {
			needed[t.Name] = true
		}
		// Also match field names within Query/Mutation root types.
		for _, f := range t.Fields {
			if re.MatchString(f.Name) {
				needed[t.Name] = true
			}
		}
	}

	// Expand: include all transitively referenced types.
	for changed := true; changed; {
		changed = false
		for name := range needed {
			t, ok := typeByName[name]
			if !ok {
				continue
			}
			refs := map[string]bool{}
			collectReferencedNames(t, refs)
			for ref := range refs {
				if ref == "" || strings.HasPrefix(ref, "__") {
					continue
				}
				if !needed[ref] {
					needed[ref] = true
					changed = true
				}
			}
		}
	}

	// Filter.
	filtered := make([]introspectionType, 0, len(needed))
	for _, t := range allTypes {
		if needed[t.Name] {
			filtered = append(filtered, t)
		}
	}
	wrapper.Data.Schema.Types = filtered

	return json.Marshal(wrapper)
}
```

Add import `"regexp"` to the file (or to the new file).

- [ ] **Step 8: Run all MCP tests**

```bash
go test ./internal/case/mcp/... -v
```

Expected: all pass

- [ ] **Step 9: Verify build**

```bash
go build ./...
```

- [ ] **Step 10: Commit**

```bash
git add internal/case/mcp/
git commit -m "feat(mcp): add graphql_introspection and graphql_request tools"
```

---

### Task 11: Docs

**Files:**
- Modify: `docs/en/user/mcp.md`
- Modify: `docs/ru/user/mcp.md`
- Create: `docs/en/user/agent_admin.md`
- Create: `docs/ru/user/agent_admin.md`

- [ ] **Step 1: Update docs/en/user/mcp.md — Methods table**

Add two rows to the Methods table:

```markdown
| `graphql_introspection(pattern)` | Inspect the GraphQL schema — returns types and operations matching the pattern, plus referenced types. Requires admin tools enabled. |
| `graphql_request(query, variables?)` | Execute any GraphQL query or mutation as admin. Requires admin tools enabled. See [[en/user/agent_admin]] for usage. |
```

- [ ] **Step 2: Add API key auth section to docs/en/user/mcp.md**

After the "Personal access tokens" section, add:

```markdown
### API key authentication

API keys (the same keys used by the Obsidian sync plugin) are accepted by the MCP endpoint. This lets an agent that already has an API key from a pre-configured vault access MCP without any extra setup.

#### Use the key

```bash
curl https://yoursite.com/_system/mcp \
  -H "X-API-Key: <your-api-key>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

#### Enable admin tools

By default, API key auth gives access to content (search, notes) at admin level — the same notes an admin user sees.

To also enable `graphql_introspection` and `graphql_request`:

1. Go to **Admin → API Keys**
2. Find the key
3. Enable **MCP admin tools**

Once enabled, the agent can inspect the GraphQL schema and call mutations directly. See [[en/user/agent_admin]] for the full workflow.
```

- [ ] **Step 3: Create docs/en/user/agent_admin.md**

```markdown
---
title: Agent Admin Access
free: true
lang_redirect: "[[ru/user/agent_admin]]"
---

trip2g can be fully controlled by an AI agent using nothing but a single API key. The same key the Obsidian sync plugin uses to push notes can also authenticate MCP and call admin GraphQL operations — no browser login needed.

### How it works

```
Obsidian vault zip          API key (already inside)
      │                              │
      ▼                              ▼
Agent opens vault     ──────▶   MCP endpoint
                                     │
                               Content access
                               (search, notes)
                                     │
                         enable_mcp_admin_tools?
                                     │
                                     ▼
                           graphql_introspection
                           graphql_request
                                     │
                              Admin mutations
                           (webhooks, patches, etc.)
```

### Setup

1. Create an API key in **Admin → API Keys**
2. Distribute the vault zip with the key pre-configured in the sync plugin
3. Enable **MCP admin tools** on that key if the agent needs to call mutations

### Available admin tools

Once `enable_mcp_admin_tools` is on, two additional MCP tools appear:

| Tool | What it does |
|------|-------------|
| `graphql_introspection(pattern)` | Find GraphQL operations matching a keyword. Returns matched types + all types they reference — like `grep -A -B`. |
| `graphql_request(query, variables?)` | Execute any query or mutation as admin. Full access. |

### Example: apply frontmatter patches

```
Agent: graphql_introspection("frontmatter")
→ Returns FrontmatterPatch types and createFrontmatterPatch mutation

Agent: graphql_request(
  query: "mutation { adminMutation { createFrontmatterPatch(input: {...}) { ... } } }",
  variables: { ... }
)
→ Patch applied
```

### Example: configure a webhook

```
Agent: graphql_introspection("webhook")
→ Returns ChangeWebhook, createWebhook mutation with all input fields

Agent: graphql_request(
  query: "mutation { adminMutation { createWebhook(input: {...}) { ... } } }"
)
→ Webhook created
```

### Security

- API key auth gives admin-level content access (all notes and subgraphs)
- `graphql_request` can execute any mutation — treat the key like a root password
- Revoke keys in **Admin → API Keys** if compromised
```

- [ ] **Step 4: Create docs/ru/user/agent_admin.md**

```markdown
---
title: Агентный доступ администратора
free: true
lang_redirect: "[[en/user/agent_admin]]"
---

trip2g можно полностью контролировать через AI-агента, используя только один API-ключ. Тот же ключ, который плагин синхронизации Obsidian использует для пуша заметок, авторизует MCP и даёт доступ к мутациям GraphQL — без входа через браузер.

### Как это работает

```
Архив vault Obsidian      API-ключ (уже внутри)
       │                          │
       ▼                          ▼
Агент открывает vault  ──▶  MCP-сервер
                                  │
                          Доступ к контенту
                          (поиск, заметки)
                                  │
                    enable_mcp_admin_tools?
                                  │
                                  ▼
                      graphql_introspection
                      graphql_request
                                  │
                          Мутации администратора
                      (вебхуки, патчи и т.д.)
```

### Настройка

1. Создайте API-ключ в **Администрирование → API-ключи**
2. Передайте архив vault с уже прописанным ключом в плагине синхронизации
3. Включите **MCP admin tools** на ключе, если агенту нужен доступ к мутациям

### Доступные инструменты

Когда `enable_mcp_admin_tools` включён, в MCP появляются два дополнительных инструмента:

| Инструмент | Что делает |
|------------|-----------|
| `graphql_introspection(pattern)` | Находит операции GraphQL по ключевому слову. Возвращает совпавшие типы и все типы, на которые они ссылаются — как `grep -A -B`. |
| `graphql_request(query, variables?)` | Выполняет любой запрос или мутацию с правами администратора. |

### Пример: применить frontmatter-патч

```
Агент: graphql_introspection("frontmatter")
→ Возвращает типы FrontmatterPatch и мутацию createFrontmatterPatch

Агент: graphql_request(
  query: "mutation { adminMutation { createFrontmatterPatch(input: {...}) { ... } } }",
  variables: { ... }
)
→ Патч применён
```

### Безопасность

- API-ключ даёт права администратора на весь контент (все заметки и подграфы)
- `graphql_request` может выполнить любую мутацию — обращайтесь с ключом как с root-паролем
- Отзовите ключ в **Администрирование → API-ключи** при компрометации
```

- [ ] **Step 5: Update docs/ru/user/mcp.md**

Add the same API key section and two tool rows to the Russian mcp.md (mirror of the EN changes, in Russian).

- [ ] **Step 6: Commit**

```bash
git add docs/en/user/mcp.md docs/ru/user/mcp.md docs/en/user/agent_admin.md docs/ru/user/agent_admin.md
git commit -m "docs(mcp): add API key auth section and agent_admin guide"
```

---

## Self-Review

**Spec coverage check:**
- ✅ API key auth (X-API-Key) in MCP endpoint — Task 9
- ✅ `contextWithMCPAPIKeyAuth` / `mcpAdminToolsEnabled` — Task 6
- ✅ `ResolveAPIKey` single method wrapping logging — Task 8
- ✅ `enable_mcp_admin_tools` migration — Task 1
- ✅ Admin UI (GraphQL mutation) — Task 4
- ✅ `graphql_introspection(pattern)` with grep-like filtering — Task 10
- ✅ `graphql_request` pass-through — Task 10
- ✅ Tools hidden without flag — Task 10
- ✅ `appreq.WithAdminToken` for internal GraphQL execution — Task 5
- ✅ `docs/{en,ru}/user/mcp.md` updated — Task 11
- ✅ `docs/{en,ru}/user/agent_admin.md` created — Task 11

**Placeholder scan:** None found.

**Type consistency:**
- `contextWithMCPAPIKeyAuth(ctx, bool)` used in Task 6 and Task 9 ✅
- `mcpAdminToolsEnabled(ctx)` used in Task 7 and Task 10 ✅
- `mcp.Env.ResolveAPIKey(ctx, value, action string) (*db.ApiKey, error)` defined in Task 7, implemented in Task 8, called in Task 9 ✅
- `mcp.Env.GraphQLRequest(ctx, query string, variables map[string]any) ([]byte, error)` defined in Task 7, implemented in Task 8, called in Task 10 ✅
- `GraphQLIntrospectionArguments` and `GraphQLRequestArguments` defined in Task 7, used in Task 10 ✅
- `db.ApiKey.EnableMcpAdminTools *bool` from sqlc — used in Task 9 with nil check ✅
