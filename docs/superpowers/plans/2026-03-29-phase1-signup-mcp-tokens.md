# Phase 1: Sign-up + MCP Tokens Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable user registration and authenticated MCP access via personal tokens.

**Architecture:** Reuse existing sign-in-by-code for sign-up (backend already creates users on first sign-in). Add `mcp_tokens` table (SHA256-hashed, like api_keys). MCP endpoint resolves `?token=` param to user. New user/space tab for token CRUD. Cronjob cleans revoked tokens after 7 days.

**Tech Stack:** Go, SQLite, sqlc, gqlgen, $mol (view.tree + TypeScript), quicktemplate

---

### Task 1: Database migration — create mcp_tokens table

**Files:**
- Create: `db/migrations/20260329100000_create_mcp_tokens_table.sql`

- [ ] **Step 1: Create migration file**

```sql
-- migrate:up
create table mcp_tokens (
  id text primary key,
  user_id integer not null references users(id) on delete cascade,
  name text not null default '',
  token_hash text not null unique,
  token_prefix text not null,
  created_at datetime not null default current_timestamp,
  last_used_at datetime,
  revoked_at datetime
);

create index idx_mcp_tokens_user_id on mcp_tokens(user_id);
create index idx_mcp_tokens_token_hash on mcp_tokens(token_hash);

-- migrate:down
drop table mcp_tokens;
```

- [ ] **Step 2: Run migration**

Run: `make dbmate-up` (or `dbmate up` depending on project setup)
Expected: Migration applied, `mcp_tokens` table created.

- [ ] **Step 3: Regenerate schema.sql**

Run: `dbmate dump`
Expected: `db/schema.sql` updated with new table.

- [ ] **Step 4: Commit**

```bash
git add db/migrations/20260329100000_create_mcp_tokens_table.sql db/schema.sql
git commit -m "feat(mcp): create mcp_tokens table"
```

---

### Task 2: SQL queries for mcp_tokens

**Files:**
- Modify: `queries.write.sql`
- Modify: `queries.read.sql`

- [ ] **Step 1: Add write queries to `queries.write.sql`**

Append to the file:

```sql
-- name: InsertMCPToken :one
insert into mcp_tokens (id, user_id, name, token_hash, token_prefix)
values (?, ?, ?, ?, ?)
returning *;

-- name: RevokeMCPToken :one
update mcp_tokens
  set revoked_at = datetime('now')
where id = ? and user_id = ? and revoked_at is null
returning *;

-- name: UpdateMCPTokenLastUsedAt :exec
update mcp_tokens
  set last_used_at = datetime('now')
where id = ?;

-- name: CleanupRevokedMCPTokens :exec
delete from mcp_tokens
where revoked_at is not null
  and revoked_at < datetime('now', '-7 days');
```

- [ ] **Step 2: Add read queries to `queries.read.sql`**

Append to the file:

```sql
-- name: MCPTokenByHash :one
select * from mcp_tokens
where token_hash = ? and revoked_at is null
limit 1;

-- name: ListMCPTokensByUserID :many
select * from mcp_tokens
where user_id = ?
order by created_at desc;

-- name: CountMCPTokensByUserID :one
select count(*) from mcp_tokens
where user_id = ?;
```

- [ ] **Step 3: Generate Go code**

Run: `make sqlc`
Expected: `internal/db/queries.read.sql.go` and `internal/db/queries.write.sql.go` updated with new functions.

- [ ] **Step 4: Verify generated code compiles**

Run: `go build ./internal/db/...`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add queries.write.sql queries.read.sql internal/db/queries.read.sql.go internal/db/queries.write.sql.go
git commit -m "feat(mcp): add sqlc queries for mcp_tokens"
```

---

### Task 3: MCP token utility — generate, hash, prefix

**Files:**
- Create: `internal/mcptoken/token.go`
- Create: `internal/mcptoken/token_test.go`

- [ ] **Step 1: Write tests**

```go
package mcptoken

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	token := Generate()

	require.True(t, len(token) == 68, "token should be 68 chars: 'mcp_' prefix + 64 random")
	require.Equal(t, "mcp_", token[:4], "token should start with 'mcp_' prefix")

	// Tokens should be unique
	token2 := Generate()
	require.NotEqual(t, token, token2)
}

func TestHash(t *testing.T) {
	token := "mcp_abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678"
	hash := Hash(token)

	require.Len(t, hash, 64, "SHA256 hex should be 64 chars")

	// Same input = same hash
	require.Equal(t, hash, Hash(token))

	// Different input = different hash
	require.NotEqual(t, hash, Hash("mcp_different"))
}

func TestPrefix(t *testing.T) {
	token := "mcp_abcdef1234567890"
	prefix := Prefix(token)

	require.Equal(t, "mcp_abcd", prefix, "prefix should be first 8 chars")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcptoken/...`
Expected: FAIL — package doesn't exist yet.

- [ ] **Step 3: Write implementation**

```go
package mcptoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

const (
	prefix   = "mcp_"
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length   = 64
)

// Generate creates a new MCP token with "mcp_" prefix + 64 random alphanumeric chars.
func Generate() string {
	result := make([]byte, length)
	for i := range length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			panic(err)
		}
		result[i] = alphabet[n.Int64()]
	}
	return prefix + string(result)
}

// Hash returns the SHA256 hex digest of the token.
func Hash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// Prefix returns the first 8 characters of the token for display.
func Prefix(token string) string {
	if len(token) < 8 {
		return token
	}
	return token[:8]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcptoken/...`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/mcptoken/
git commit -m "feat(mcp): add mcptoken utility (generate, hash, prefix)"
```

---

### Task 4: Create MCP token use case

**Files:**
- Create: `internal/case/createmcptoken/resolve.go`
- Create: `internal/case/createmcptoken/resolve_test.go`

- [ ] **Step 1: Write tests**

```go
package createmcptoken_test

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg createmcptoken_test . Env

import (
	"context"
	"testing"

	"trip2g/internal/case/createmcptoken"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

func TestResolve_Success(t *testing.T) {
	env := &EnvMock{
		CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 42}, nil
		},
		CountMCPTokensByUserIDFunc: func(ctx context.Context, userID int64) (int64, error) {
			return 3, nil
		},
		GenerateUniqIDFunc: func() string {
			return "test-id"
		},
		InsertMCPTokenFunc: func(ctx context.Context, params db.InsertMCPTokenParams) (db.McpToken, error) {
			return db.McpToken{
				ID:          params.ID,
				UserID:      params.UserID,
				Name:        params.Name,
				TokenHash:   params.TokenHash,
				TokenPrefix: params.TokenPrefix,
			}, nil
		},
	}

	result, err := createmcptoken.Resolve(context.Background(), env, createmcptoken.Input{Name: "My Claude"})
	require.NoError(t, err)

	payload, ok := result.(*createmcptoken.SuccessPayload)
	require.True(t, ok)
	require.NotEmpty(t, payload.PlaintextToken)
	require.Equal(t, "mcp_", payload.PlaintextToken[:4])
	require.Equal(t, "test-id", payload.Token.ID)
}

func TestResolve_LimitExceeded(t *testing.T) {
	env := &EnvMock{
		CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
			return &usertoken.Data{ID: 42}, nil
		},
		CountMCPTokensByUserIDFunc: func(ctx context.Context, userID int64) (int64, error) {
			return 10, nil
		},
	}

	result, err := createmcptoken.Resolve(context.Background(), env, createmcptoken.Input{Name: "Too many"})
	require.NoError(t, err)

	_, ok := result.(*createmcptoken.ErrorPayload)
	require.True(t, ok, "should return error payload when limit exceeded")
}
```

- [ ] **Step 2: Generate mocks**

Run: `go generate ./internal/case/createmcptoken/...`

- [ ] **Step 3: Write implementation**

```go
package createmcptoken

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/mcptoken"
	"trip2g/internal/usertoken"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg createmcptoken_test . Env

const MaxTokensPerUser = 10

type Env interface {
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	CountMCPTokensByUserID(ctx context.Context, userID int64) (int64, error)
	GenerateUniqID() string
	InsertMCPToken(ctx context.Context, params db.InsertMCPTokenParams) (db.McpToken, error)
}

type Input struct {
	Name string
}

type SuccessPayload struct {
	PlaintextToken string
	Token          db.McpToken
}

type ErrorPayload struct {
	Message string
}

type Payload interface{}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	token, err := env.CurrentUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}

	userID := int64(token.ID)

	count, err := env.CountMCPTokensByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count mcp tokens: %w", err)
	}

	if count >= MaxTokensPerUser {
		return &ErrorPayload{Message: "Maximum number of MCP tokens reached (10). Revoke an existing token first."}, nil
	}

	plaintext := mcptoken.Generate()
	hash := mcptoken.Hash(plaintext)
	prefix := mcptoken.Prefix(plaintext)

	params := db.InsertMCPTokenParams{
		ID:          env.GenerateUniqID(),
		UserID:      userID,
		Name:        input.Name,
		TokenHash:   hash,
		TokenPrefix: prefix,
	}

	created, err := env.InsertMCPToken(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to insert mcp token: %w", err)
	}

	return &SuccessPayload{
		PlaintextToken: plaintext,
		Token:          created,
	}, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/case/createmcptoken/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/case/createmcptoken/
git commit -m "feat(mcp): add create mcp token use case"
```

---

### Task 5: Revoke MCP token use case

**Files:**
- Create: `internal/case/revokemcptoken/resolve.go`

- [ ] **Step 1: Write implementation**

```go
package revokemcptoken

import (
	"context"
	"fmt"

	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	RevokeMCPToken(ctx context.Context, arg db.RevokeMCPTokenParams) (db.McpToken, error)
}

type Input struct {
	ID string
}

func Resolve(ctx context.Context, env Env, input Input) (db.McpToken, error) {
	token, err := env.CurrentUserToken(ctx)
	if err != nil {
		return db.McpToken{}, fmt.Errorf("failed to get current user token: %w", err)
	}

	revoked, err := env.RevokeMCPToken(ctx, db.RevokeMCPTokenParams{
		ID:     input.ID,
		UserID: int64(token.ID),
	})
	if err != nil {
		if db.IsNoFound(err) {
			return db.McpToken{}, model.NewFieldError("id", "Token not found or already revoked")
		}
		return db.McpToken{}, fmt.Errorf("failed to revoke mcp token: %w", err)
	}

	return revoked, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/case/revokemcptoken/...`
Expected: No errors. (May need to check the exact `db.IsNoFound` helper — adjust if the project uses a different pattern.)

- [ ] **Step 3: Commit**

```bash
git add internal/case/revokemcptoken/
git commit -m "feat(mcp): add revoke mcp token use case"
```

---

### Task 6: MCP endpoint auth — resolve token from URL param

**Files:**
- Modify: `internal/case/mcp/endpoint.go`
- Modify: `internal/case/mcp/resolve.go`

- [ ] **Step 1: Add MCPUser type and extend Env interface in resolve.go**

Add to the top of `internal/case/mcp/resolve.go`, after the existing imports:

```go
// MCPUser represents the authenticated user from an MCP token.
type MCPUser struct {
	UserID int64
	Role   string // "admin" or "user"
}
```

Extend the `Env` interface:

```go
type Env interface {
	similarnotes.Env
	SearchLatestNotes(query string) ([]model.SearchResult, error)
	OpenAI() *openai.Client
	PublicURL() string
	Logger() logger.Logger
	MCPTokenByHash(ctx context.Context, hash string) (db.McpToken, error)
	UpdateMCPTokenLastUsedAt(ctx context.Context, id string) error
	UserRoleByID(ctx context.Context, userID int64) (string, error)
}
```

- [ ] **Step 2: Update Resolve signature to accept MCPUser**

Change `Resolve` to accept an optional user:

```go
func Resolve(ctx context.Context, env Env, req Request, user *MCPUser) Response {
	switch req.Method {
	case MCPMethodInitialize:
		return handleInitialize(env, req.ID)
	case "notifications/initialized":
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return handleToolsList(env, req.ID)
	case "tools/call":
		return handleToolsCall(ctx, env, req)
	default:
		return errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found: "+req.Method)
	}
}
```

The `user` param is stored for future use in Phase 2 (admin tools, rate limiting). For now, it's just resolved and passed through.

- [ ] **Step 3: Add token resolution in endpoint.go**

Replace the `Handle` method:

```go
func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)

	// Resolve MCP token from ?token= param
	var user *MCPUser
	tokenParam := string(req.Req.QueryArgs().Peek("token"))
	if tokenParam != "" {
		resolved, err := resolveMCPToken(req.Req, env, tokenParam)
		if err != nil {
			resp := errorResponse(nil, ErrCodeInvalidParams, "Invalid token: "+err.Error())
			return writeJSONResponse(req, resp)
		}
		user = resolved
	}

	// Parse JSON-RPC request
	var rpcReq Request
	err := json.Unmarshal(req.Req.PostBody(), &rpcReq)
	if err != nil {
		resp := errorResponse(nil, ErrCodeParseError, "Parse error: "+err.Error())
		return writeJSONResponse(req, resp)
	}

	// Validate JSON-RPC version
	if rpcReq.JSONRPC != "2.0" {
		resp := errorResponse(rpcReq.ID, ErrCodeInvalidRequest, "Invalid JSON-RPC version")
		return writeJSONResponse(req, resp)
	}

	// Handle request
	resp := Resolve(req.Req, env, rpcReq, user) // req.Req is fasthttp.RequestCtx which implements context.Context
	return writeJSONResponse(req, resp)
}
```

- [ ] **Step 4: Add resolveMCPToken helper in endpoint.go**

```go
func resolveMCPToken(ctx context.Context, env Env, plainToken string) (*MCPUser, error) {
	hash := mcptoken.Hash(plainToken)

	token, err := env.MCPTokenByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("token not found")
	}

	// Update last_used_at in background (best effort)
	go func() {
		_ = env.UpdateMCPTokenLastUsedAt(context.Background(), token.ID)
	}()

	role, err := env.UserRoleByID(ctx, token.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user role")
	}

	return &MCPUser{
		UserID: token.UserID,
		Role:   role,
	}, nil
}
```

Add the import for `"trip2g/internal/mcptoken"`.

- [ ] **Step 5: Add UserRoleByID query**

Add to `queries.read.sql`:

```sql
-- name: UserRoleByID :one
select case when a.user_id is not null then 'admin' else 'user' end as role
from users u
left join admins a on u.id = a.user_id
where u.id = ?;
```

Run: `make sqlc`

Note: `UserRoleByID` may need to return a custom type. If sqlc generates it as a string, it works directly. Check the generated code and adapt if needed.

- [ ] **Step 6: Update existing tests**

Update `internal/case/mcp/resolve_test.go` — add the new `Env` methods to the mock and update `Resolve` calls to pass `nil` for the user param.

- [ ] **Step 7: Verify all tests pass**

Run: `go test ./internal/case/mcp/...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/case/mcp/ internal/mcptoken/ queries.read.sql internal/db/queries.read.sql.go
git commit -m "feat(mcp): add token auth resolution to MCP endpoint"
```

---

### Task 7: Cronjob — cleanup revoked MCP tokens

**Files:**
- Create: `internal/case/cronjob/cleanuprevokedmcptokens/job.go`
- Create: `internal/case/cronjob/cleanuprevokedmcptokens/resolve.go`
- Modify: `cmd/server/cronjobs.go`

- [ ] **Step 1: Write job.go**

```go
package cleanuprevokedmcptokens

import (
	"context"
)

type Job struct{}

func (j *Job) Name() string {
	return "cleanup_revoked_mcp_tokens"
}

// Schedule runs daily at 4:00 AM.
func (j *Job) Schedule() string {
	return "0 0 4 * * *"
}

func (j *Job) ExecuteAfterStart() bool {
	return false
}

func (j *Job) Execute(ctx context.Context, env any) (any, error) {
	return Resolve(ctx, env.(Env))
}
```

- [ ] **Step 2: Write resolve.go**

```go
package cleanuprevokedmcptokens

import (
	"context"
	"fmt"

	"trip2g/internal/logger"
)

type Env interface {
	CleanupRevokedMCPTokens(ctx context.Context) error
	Logger() logger.Logger
}

type Result struct {
	Cleaned bool
}

func Resolve(ctx context.Context, env Env) (*Result, error) {
	err := env.CleanupRevokedMCPTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup revoked mcp tokens: %w", err)
	}

	return &Result{Cleaned: true}, nil
}
```

- [ ] **Step 3: Register cronjob in `cmd/server/cronjobs.go`**

Add import:
```go
"trip2g/internal/case/cronjob/cleanuprevokedmcptokens"
```

Add compile-time check:
```go
_ cleanuprevokedmcptokens.Env = app
```

Add to jobs slice:
```go
&cleanuprevokedmcptokens.Job{},
```

- [ ] **Step 4: Implement Env method on app**

In `cmd/server/main.go`, add:

```go
func (a *app) CleanupRevokedMCPTokens(ctx context.Context) error {
	return a.writeDB.CleanupRevokedMCPTokens(ctx)
}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./cmd/server/...`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add internal/case/cronjob/cleanuprevokedmcptokens/ cmd/server/cronjobs.go cmd/server/main.go
git commit -m "feat(mcp): add cronjob to cleanup revoked mcp tokens after 7 days"
```

---

### Task 8: GraphQL schema + resolvers

**Files:**
- Modify: `internal/graph/schema.graphqls`
- Modify: `internal/graph/schema.resolvers.go` (after gqlgen)
- Modify: `internal/graph/resolver.go` (Env interface)

- [ ] **Step 1: Add GraphQL types and mutations to schema.graphqls**

Add types:

```graphql
type McpToken {
  id: ID!
  name: String!
  tokenPrefix: String!
  createdAt: Time!
  lastUsedAt: Time
  revokedAt: Time
}

type CreateMcpTokenPayload {
  plaintextToken: String!
  token: McpToken!
}

type RevokeMcpTokenPayload {
  token: McpToken!
}

input CreateMcpTokenInput {
  name: String!
}

input RevokeMcpTokenInput {
  id: ID!
}
```

Add to `type User` fields:

```graphql
  mcpTokens: [McpToken!]!
```

Add mutations (inside `type Mutation` or via viewer pattern — follow existing project convention):

```graphql
  createMcpToken(input: CreateMcpTokenInput!): CreateMcpTokenPayload!
  revokeMcpToken(input: RevokeMcpTokenInput!): RevokeMcpTokenPayload!
```

- [ ] **Step 2: Run gqlgen**

Run: `make gqlgen`
Expected: Generates resolver stubs in `schema.resolvers.go`.

- [ ] **Step 3: Implement resolvers**

In the generated resolver stubs, wire to use cases:

For `CreateMcpToken`:
```go
func (r *mutationResolver) CreateMcpToken(ctx context.Context, input model.CreateMcpTokenInput) (*model.CreateMcpTokenPayload, error) {
	result, err := createmcptoken.Resolve(ctx, r.env(ctx), createmcptoken.Input{Name: input.Name})
	if err != nil {
		return nil, err
	}

	switch p := result.(type) {
	case *createmcptoken.SuccessPayload:
		return &model.CreateMcpTokenPayload{
			PlaintextToken: p.PlaintextToken,
			Token: &model.McpToken{
				ID:          p.Token.ID,
				Name:        p.Token.Name,
				TokenPrefix: p.Token.TokenPrefix,
				CreatedAt:   p.Token.CreatedAt,
				LastUsedAt:  p.Token.LastUsedAt,
				RevokedAt:   p.Token.RevokedAt,
			},
		}, nil
	case *createmcptoken.ErrorPayload:
		return nil, fmt.Errorf(p.Message)
	default:
		return nil, fmt.Errorf("unexpected payload type")
	}
}
```

For `RevokeMcpToken`:
```go
func (r *mutationResolver) RevokeMcpToken(ctx context.Context, input model.RevokeMcpTokenInput) (*model.RevokeMcpTokenPayload, error) {
	revoked, err := revokemcptoken.Resolve(ctx, r.env(ctx), revokemcptoken.Input{ID: input.ID})
	if err != nil {
		return nil, err
	}

	return &model.RevokeMcpTokenPayload{
		Token: &model.McpToken{
			ID:          revoked.ID,
			Name:        revoked.Name,
			TokenPrefix: revoked.TokenPrefix,
			CreatedAt:   revoked.CreatedAt,
			LastUsedAt:  revoked.LastUsedAt,
			RevokedAt:   revoked.RevokedAt,
		},
	}, nil
}
```

For `User.McpTokens` field resolver:
```go
func (r *userResolver) McpTokens(ctx context.Context, obj *model.User) ([]*model.McpToken, error) {
	token, err := r.env(ctx).CurrentUserToken(ctx)
	if err != nil {
		return nil, err
	}

	tokens, err := r.env(ctx).ListMCPTokensByUserID(ctx, int64(token.ID))
	if err != nil {
		return nil, err
	}

	result := make([]*model.McpToken, len(tokens))
	for i, t := range tokens {
		result[i] = &model.McpToken{
			ID:          t.ID,
			Name:        t.Name,
			TokenPrefix: t.TokenPrefix,
			CreatedAt:   t.CreatedAt,
			LastUsedAt:  t.LastUsedAt,
			RevokedAt:   t.RevokedAt,
		}
	}

	return result, nil
}
```

Note: Adapt the resolver code to the exact types gqlgen generates — field names may differ slightly. Check the generated `model` package.

- [ ] **Step 4: Add Env methods to resolver Env interface and implement on app**

In `internal/graph/resolver.go`, add to Env interface:

```go
createmcptoken.Env
revokemcptoken.Env
ListMCPTokensByUserID(ctx context.Context, userID int64) ([]db.McpToken, error)
```

In `cmd/server/main.go`, implement the methods that delegate to the DB:

```go
func (a *app) InsertMCPToken(ctx context.Context, params db.InsertMCPTokenParams) (db.McpToken, error) {
	return a.writeDB.InsertMCPToken(ctx, params)
}

func (a *app) CountMCPTokensByUserID(ctx context.Context, userID int64) (int64, error) {
	return a.readDB.CountMCPTokensByUserID(ctx, userID)
}

func (a *app) RevokeMCPToken(ctx context.Context, arg db.RevokeMCPTokenParams) (db.McpToken, error) {
	return a.writeDB.RevokeMCPToken(ctx, arg)
}

func (a *app) ListMCPTokensByUserID(ctx context.Context, userID int64) ([]db.McpToken, error) {
	return a.readDB.ListMCPTokensByUserID(ctx, userID)
}

func (a *app) MCPTokenByHash(ctx context.Context, hash string) (db.McpToken, error) {
	return a.readDB.MCPTokenByHash(ctx, hash)
}

func (a *app) UpdateMCPTokenLastUsedAt(ctx context.Context, id string) error {
	return a.writeDB.UpdateMCPTokenLastUsedAt(ctx, id)
}

func (a *app) UserRoleByID(ctx context.Context, userID int64) (string, error) {
	return a.readDB.UserRoleByID(ctx, userID)
}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 6: Regenerate frontend GraphQL types**

Run: `npm run graphqlgen`
Expected: Frontend TypeScript types updated with new mutations and McpToken type.

- [ ] **Step 7: Commit**

```bash
git add internal/graph/ cmd/server/main.go
git commit -m "feat(mcp): add GraphQL schema and resolvers for mcp tokens"
```

---

### Task 9: Frontend — MCP tokens tab in user/space widget

**Files:**
- Create: `assets/ui/user/space/mcp_tokens/mcp_tokens.view.tree`
- Create: `assets/ui/user/space/mcp_tokens/mcp_tokens.view.ts`
- Create: `assets/ui/user/space/mcp_tokens/mcp_tokens.view.css.ts`
- Modify: `assets/ui/user/space/space.view.tree`

- [ ] **Step 1: Add MCP tab to space widget**

In `assets/ui/user/space/space.view.tree`, add the new tab in `spreads`:

```
	spreads *
		home <= Home $trip2g_user_space_home
		subscriptions <= Subscriptions $trip2g_user_space_subscriptions
		mcp <= Mcp $trip2g_user_space_mcp_tokens
```

- [ ] **Step 2: Create mcp_tokens.view.tree**

```
$trip2g_user_space_mcp_tokens $mol_view
	sub /
		<= Page $mol_page
			title @ \MCP Tokens
			tools /
				<= Add_button $mol_button_minor
					title @ \+ Add
					click? <=> add_token? null
			body /
				<= Created_alert $mol_view
					sub / <= created_alert_content /
				<= List $mol_list
					rows <= rows /
						<= Row*0 $mol_row
							sub <= row_content* /
								<= Name_cell* $mol_labeler
									title @ \Name
									Content <= Name_text* $mol_view
										sub / <= row_name* \
								<= Prefix_cell* $mol_labeler
									title @ \Token
									Content <= Prefix_text* $mol_view
										sub / <= row_prefix* \
								<= Created_cell* $mol_labeler
									title @ \Created
									Content <= Created_time* $trip2g_table_cell_time
										value <= row_created_at* null
								<= Used_cell* $mol_labeler
									title @ \Last used
									Content <= Used_time* $trip2g_table_cell_time
										value <= row_last_used_at* null
								<= Url_cell* $mol_labeler
									title @ \MCP URL
									Content <= Url_copy* $mol_button_copy
										text <= row_mcp_url* \
								<= Revoke_cell* $mol_view
									sub /
										<= Revoke_button* $mol_button_minor
											title @ \Revoke
											click? <=> revoke_token*? null
```

- [ ] **Step 3: Create mcp_tokens.view.ts**

```typescript
namespace $.$$ {

	const listQuery = $trip2g_graphql_request(/* GraphQL */ `
		query UserMcpTokens {
			viewer {
				user {
					mcpTokens {
						id
						name
						tokenPrefix
						createdAt
						lastUsedAt
						revokedAt
					}
				}
			}
		}
	`)

	const createMutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation CreateMcpToken($input: CreateMcpTokenInput!) {
			createMcpToken(input: $input) {
				plaintextToken
				token {
					id
					name
					tokenPrefix
					createdAt
				}
			}
		}
	`)

	const revokeMutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation RevokeMcpToken($input: RevokeMcpTokenInput!) {
			revokeMcpToken(input: $input) {
				token {
					id
					revokedAt
				}
			}
		}
	`)

	export class $trip2g_user_space_mcp_tokens extends $.$trip2g_user_space_mcp_tokens {

		@$mol_mem
		created_token( next?: string | null ): string | null {
			return next ?? null
		}

		override created_alert_content() {
			const token = this.created_token()
			if( !token ) return []
			return [ token ]
		}

		@$mol_mem
		data( reset?: null ) {
			const res = listQuery()
			if( !res.viewer.user ) throw new Error( 'User not found' )
			return $trip2g_graphql_make_map(
				res.viewer.user.mcpTokens.filter( ( t: any ) => !t.revokedAt )
			)
		}

		override rows() {
			return this.data().map( key => this.Row( key ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_name( id: any ): string {
			return this.row( id ).name || '(unnamed)'
		}

		override row_prefix( id: any ): string {
			return this.row( id ).tokenPrefix + '...'
		}

		override row_created_at( id: any ) {
			return this.row( id ).createdAt
		}

		override row_last_used_at( id: any ) {
			return this.row( id ).lastUsedAt
		}

		override row_mcp_url( id: any ): string {
			// This is a placeholder — plaintext token is only shown at creation
			return location.origin + '/_system/mcp?token=' + this.row( id ).tokenPrefix + '...'
		}

		override add_token() {
			const name = prompt( 'Token name:' )
			if( !name ) return

			const res = createMutation( { input: { name } } )
			const url = location.origin + '/_system/mcp?token=' + res.createMcpToken.plaintextToken
			this.created_token( url )
			this.data( null )
		}

		override revoke_token( id: any ) {
			if( !confirm( 'Revoke this token?' ) ) return
			revokeMutation( { input: { id } } )
			this.data( null )
		}
	}
}
```

- [ ] **Step 4: Create mcp_tokens.view.css.ts**

```typescript
namespace $ {
	$mol_style_define( $trip2g_user_space_mcp_tokens, {
	})
}
```

- [ ] **Step 5: Verify build**

Run: `npm run build`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add assets/ui/user/space/
git commit -m "feat(mcp): add MCP tokens tab to user/space widget"
```

---

### Task 10: Wire everything together and integration test

**Files:**
- Modify: `cmd/server/main.go` (Env method implementations)
- Modify: `internal/graph/resolver.go` (Env interface)

- [ ] **Step 1: Verify all Env methods are implemented on app**

Run: `go build ./cmd/server/...`

If there are missing method errors, implement them following the pattern in Task 8 Step 4 — delegate to `a.readDB` or `a.writeDB`.

- [ ] **Step 2: Run full test suite**

Run: `go test ./...`
Expected: All tests pass.

- [ ] **Step 3: Manual smoke test**

Start dev server:
```bash
make air
```

Test flow:
1. Open site, navigate to user/space widget
2. See "MCP" tab
3. Click "Add", enter name, get token URL
4. Copy URL, test with curl:
```bash
curl -X POST "http://localhost:8081/_system/mcp?token=<your_token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize"}'
```
Expected: JSON-RPC response with protocolVersion and instructions.

5. Test without token (guest):
```bash
curl -X POST "http://localhost:8081/_system/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```
Expected: Tools list (guest access).

6. Test with invalid token:
```bash
curl -X POST "http://localhost:8081/_system/mcp?token=invalid" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize"}'
```
Expected: Error response "Invalid token".

- [ ] **Step 4: Commit final wiring**

```bash
git add cmd/server/ internal/graph/
git commit -m "feat(mcp): wire up mcp tokens end-to-end"
```
