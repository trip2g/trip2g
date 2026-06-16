# `GET /_system/federation/admin` Implementation Plan (trip2g)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить admin-only read-эндпоинт `GET /_system/federation/admin`, отдающий полную топологию федерации инстанса (self-identity, сабграфы, inbound/outbound секреты со scope, KB-ноты) одним JSON.

**Architecture:** Новый FastHTTP case-эндпоинт (`internal/case/system/federationtopology/`) по существующему паттерну (`Endpoint`/`Env`/`Resolve`), регистрируется через codegen роутера. Авторизация — `req.UserToken()` + `IsAdmin()` (cookie-сессия). Данные собираются из sqlc-запросов (секреты + scope батчем) и `NoteViews.MCPFederationNotes`. Ответ — easyjson.

**Tech Stack:** Go, FastHTTP, sqlc, easyjson, gencmd (router codegen).

**Spec:** `docs/dev/federation_system_endpoint.md`.

---

## File Structure

- Create: `internal/case/system/federationtopology/resolve.go` — Env, Resolve, response types.
- Create: `internal/case/system/federationtopology/endpoint.go` — Endpoint (Path/Method/Handle + admin gate).
- Create: `internal/case/system/federationtopology/resolve_test.go` — unit-тесты сборки.
- Modify: `queries.read.sql` — добавить batch-запрос scope по всем kid.
- Modify (generated): `internal/db/queries.read.sql.go` — через `make sqlc`.
- Modify (generated): `internal/router/endpoints_gen.go`, `env.go` — через `go generate ./internal/router`.
- Modify (generated): `*_easyjson.go` — через `go generate ./...` в пакете.
- Modify: `cmd/server/main.go` (app env) — реализовать недостающие методы нового `Env`, если их ещё нет.

**Reference patterns (читать перед стартом):**
- Endpoint: `internal/case/mcp/endpoint.go` (Path/Method/Handle, `env := req.Env.(Env)`).
- Admin gate: `internal/case/downloadonboardingvault/endpoint.go:21-29` (`token, _ := req.UserToken(); if token == nil || !token.IsAdmin() { 401 }`).
- Case/Env: `internal/case/admin/listfederationsecrets/resolve.go` (Env interface + Resolve).
- Existing secrets query: `queries.read.sql` `ListFederationSecrets` (JOIN + group) и `ListFederationSecretSubgraphsByKID`.
- KB-notes: `internal/model/note.go:287` (`NoteViews.MCPFederationNotes`), `internal/model/mcp_federation_note.go:9` (`MCPFederationNote{Note,URL,ID,MaxDepth}`).
- Self URL: `internal/appconfig/config.go:77` `PublicURL`, accessor `cmd/server/main.go:1638` `func (a *app) PublicURL() string`.
- easyjson directive: `internal/federation/types.go:3`.
- Endpoint codegen: `internal/router/router.go:12` (`//go:generate go run ./gencmd`), generated list `internal/router/endpoints_gen.go`.

---

## Task 1: Batch-запрос scope секретов

**Files:**
- Modify: `queries.read.sql`
- Generate: `internal/db/queries.read.sql.go` (через `make sqlc`)

- [ ] **Step 1: Добавить запрос в `queries.read.sql`**

В конце секции federation-запросов добавить (одним запросом тянем scope по всем kid, чтобы не делать N+1):

```sql
-- name: ListAllFederationSecretScopes :many
select fss.kid, s.id as subgraph_id, s.name as subgraph_name
  from federation_secret_subgraphs fss
  join subgraphs s on s.id = fss.subgraph_id
 order by fss.kid, s.name;
```

- [ ] **Step 2: Сгенерировать sqlc**

Run: `make sqlc`
Expected: в `internal/db/queries.read.sql.go` появился метод `ListAllFederationSecretScopes(ctx) ([]ListAllFederationSecretScopesRow, error)` со строкой `{Kid string; SubgraphID int64; SubgraphName string}`.

- [ ] **Step 3: Сборка**

Run: `go build ./internal/db/...`
Expected: без ошибок.

- [ ] **Step 4: Commit**

```bash
git add queries.read.sql internal/db/queries.read.sql.go
git commit -m "feat(federation): batch query for federation secret scopes"
```

---

## Task 2: Response-типы и Resolve (без авторизации)

**Files:**
- Create: `internal/case/system/federationtopology/resolve.go`
- Create: `internal/case/system/federationtopology/resolve_test.go`
- Generate: `internal/case/system/federationtopology/resolve_easyjson.go`

- [ ] **Step 1: Написать падающий тест сборки ответа**

`resolve.go` ещё нет — тест не скомпилируется (ожидаемо).

```go
package federationtopology_test

import (
	"context"
	"testing"
	"time"

	"trip2g/internal/case/system/federationtopology"
	"trip2g/internal/db"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestBuildTopology(t *testing.T) {
	kbURL := "https://bob.team.io/_system/mcp"
	env := &EnvMock{
		PublicURLFunc: func() string { return "https://alice.team.io" },
		ListAllSubgraphsFunc: func(ctx context.Context) ([]db.Subgraph, error) {
			return []db.Subgraph{{ID: 1, Name: "internal"}}, nil
		},
		ListFederationSecretsFunc: func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
			return []db.ListFederationSecretsRow{
				{ID: 12, Kid: "bob-2026", KbUrl: &kbURL},          // outbound (есть kb_url)
				{ID: 7, Kid: "carol-2026"},                        // inbound (нет kb_url)
			}, nil
		},
		ListAllFederationSecretScopesFunc: func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
			return []db.ListAllFederationSecretScopesRow{
				{Kid: "bob-2026", SubgraphID: 1, SubgraphName: "internal"},
			}, nil
		},
		LatestNoteViewsFunc: func(ctx context.Context) (*model.NoteViews, error) {
			return &model.NoteViews{MCPFederationNotes: []*model.MCPFederationNote{
				{URL: kbURL, ID: "bob"},
			}}, nil
		},
	}

	out, err := federationtopology.Resolve(context.Background(), env)
	require.NoError(t, err)

	require.Equal(t, "https://alice.team.io/_system/mcp", out.Self.MCPURL)
	require.Equal(t, "alice.team.io", out.Self.KBID)
	require.Len(t, out.Self.Subgraphs, 1)

	require.Len(t, out.Outbound, 1)
	require.Equal(t, kbURL, *out.Outbound[0].KBURL)
	require.Equal(t, []string{"internal"}, namesOf(out.Outbound[0].Subgraphs))

	require.Len(t, out.Inbound, 1)
	require.Nil(t, out.Inbound[0].KBURL)

	require.Len(t, out.KBNotes, 1)
	require.Equal(t, kbURL, out.KBNotes[0].KBURL)

	_ = time.Now
}

func namesOf(ss []federationtopology.Subgraph) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, s.Name)
	}
	return out
}
```

- [ ] **Step 2: Реализовать `resolve.go`**

```go
package federationtopology

//go:generate go tool github.com/mailru/easyjson/easyjson -snake_case -all -no_std_marshalers ./resolve.go
//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg federationtopology_test . Env

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/model"
)

type Env interface {
	PublicURL() string
	ListAllSubgraphs(ctx context.Context) ([]db.Subgraph, error)
	ListFederationSecrets(ctx context.Context) ([]db.ListFederationSecretsRow, error)
	ListAllFederationSecretScopes(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error)
	LatestNoteViews(ctx context.Context) (*model.NoteViews, error)
}

type Subgraph struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Self struct {
	Name      string     `json:"name"`
	KBID      string     `json:"kb_id"`
	MCPURL    string     `json:"mcp_url"`
	Subgraphs []Subgraph `json:"subgraphs"`
}

type Secret struct {
	ID        int64      `json:"id"`
	KID       string     `json:"kid"`
	KBURL     *string    `json:"kb_url,omitempty"`
	RevokedAt *string    `json:"revoked_at"`
	Subgraphs []Subgraph `json:"subgraphs"`
}

type KBNote struct {
	KBID        string `json:"kb_id"`
	KBURL       string `json:"kb_url"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type Response struct {
	Self     Self     `json:"self"`
	Outbound []Secret `json:"outbound"`
	Inbound  []Secret `json:"inbound"`
	KBNotes  []KBNote `json:"kb_notes"`
}

func Resolve(ctx context.Context, env Env) (*Response, error) {
	publicURL := strings.TrimRight(env.PublicURL(), "/")
	mcpURL := publicURL + "/_system/mcp"

	host := publicURL
	if u, err := url.Parse(publicURL); err == nil && u.Host != "" {
		host = u.Host
	}

	subs, err := env.ListAllSubgraphs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list subgraphs: %w", err)
	}
	selfSubs := make([]Subgraph, 0, len(subs))
	for _, s := range subs {
		selfSubs = append(selfSubs, Subgraph{ID: s.ID, Name: s.Name})
	}

	scopes, err := env.ListAllFederationSecretScopes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scopes: %w", err)
	}
	scopeByKID := map[string][]Subgraph{}
	for _, sc := range scopes {
		scopeByKID[sc.Kid] = append(scopeByKID[sc.Kid], Subgraph{ID: sc.SubgraphID, Name: sc.SubgraphName})
	}

	rows, err := env.ListFederationSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	resp := &Response{
		Self:     Self{Name: host, KBID: host, MCPURL: mcpURL, Subgraphs: selfSubs},
		Outbound: []Secret{},
		Inbound:  []Secret{},
		KBNotes:  []KBNote{},
	}
	for _, r := range rows {
		sec := Secret{ID: r.ID, KID: r.Kid, KBURL: r.KbUrl, Subgraphs: scopeByKID[r.Kid]}
		if sec.Subgraphs == nil {
			sec.Subgraphs = []Subgraph{}
		}
		if r.RevokedAt != nil {
			s := r.RevokedAt.UTC().Format("2006-01-02T15:04:05Z")
			sec.RevokedAt = &s
		}
		if r.KbUrl != nil {
			resp.Outbound = append(resp.Outbound, sec)
		} else {
			resp.Inbound = append(resp.Inbound, sec)
		}
	}

	views, err := env.LatestNoteViews(ctx)
	if err != nil {
		return nil, fmt.Errorf("note views: %w", err)
	}
	for _, n := range views.MCPFederationNotes {
		note := KBNote{KBID: n.ID, KBURL: n.URL, Path: notePath(n), Description: noteDesc(n)}
		resp.KBNotes = append(resp.KBNotes, note)
	}

	return resp, nil
}

// notePath / noteDesc извлекают путь и краткое описание из NoteView.
// ВНИМАНИЕ: подтвердить точные поля NoteView (internal/model/note.go) —
// path-поле и body. Если у MCPFederationNote.Note nil, вернуть пустые строки.
func notePath(n *model.MCPFederationNote) string {
	if n.Note == nil {
		return ""
	}
	return n.Note.Path // подтвердить имя поля по internal/model/note.go
}

func noteDesc(n *model.MCPFederationNote) string {
	if n.Note == nil {
		return ""
	}
	return "" // заполнить из тела ноты; см. NoteView (поле с markdown/plain телом)
}
```

> **Перед реализацией** подтвердить точные поля `model.NoteView` (path и тело) по `internal/model/note.go` и точное имя/тип `db.Subgraph` (поля `ID`, `Name`) по sqlc-моделям. Если `db.Subgraph` называет поля иначе — поправить маппинг. Это единственные неуточнённые сигнатуры; всё остальное скопировано из реальных типов.

- [ ] **Step 3: Сгенерировать easyjson + моки**

Run: `go generate ./internal/case/system/federationtopology/`
Expected: созданы `resolve_easyjson.go` и `mocks_test.go`.

- [ ] **Step 4: Прогнать тест**

Run: `go test ./internal/case/system/federationtopology/ -run TestBuildTopology -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w ./internal/case/system/federationtopology/
git add internal/case/system/federationtopology/
git commit -m "feat(federation): topology response builder for /_system/federation/admin"
```

---

## Task 3: Endpoint с admin-гейтом

**Files:**
- Create: `internal/case/system/federationtopology/endpoint.go`

- [ ] **Step 1: Реализовать Endpoint**

```go
package federationtopology

import (
	"context"
	"net/http"

	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Path() string   { return "/_system/federation/admin" }
func (*Endpoint) Method() string { return http.MethodGet }

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}
	if token == nil || !token.IsAdmin() {
		req.Req.SetStatusCode(http.StatusUnauthorized)
		return nil, nil
	}

	env := req.Env.(Env)
	resp, err := Resolve(context.Context(req.Req), env)
	if err != nil {
		return nil, err
	}
	return resp, nil // *Response реализует easyjson.Marshaler → роутер сам сериализует (router.go:125)
}
```

> `context.Context(req.Req)` — как в `internal/case/mcp/endpoint.go`. Если там используется иной способ получить ctx — повторить его.

- [ ] **Step 2: Сборка пакета**

Run: `go build ./internal/case/system/federationtopology/`
Expected: без ошибок.

- [ ] **Step 3: Commit**

```bash
gofmt -w ./internal/case/system/federationtopology/
git add internal/case/system/federationtopology/endpoint.go
git commit -m "feat(federation): admin-gated GET /_system/federation/admin endpoint"
```

---

## Task 4: Регистрация роута + реализация Env в app

**Files:**
- Generate: `internal/router/endpoints_gen.go`, `internal/router/env.go`
- Modify: `cmd/server/main.go` (методы app, которых ещё нет)

- [ ] **Step 1: Регенерировать список эндпоинтов**

Run: `go generate ./internal/router`
Expected: в `endpoints_gen.go` добавился `&federationtopology....Endpoint{}` и его `Env` в `RoutesEnv`.

- [ ] **Step 2: Собрать весь проект — найти недостающие методы Env**

Run: `go build ./...`
Expected: ошибки компиляции вида «app does not implement RoutesEnv (missing method ListAllFederationSecretScopes / LatestNoteViews)». Это список того, что надо добавить на app.

- [ ] **Step 3: Реализовать недостающие методы app**

В `cmd/server/main.go` (где определены `func (a *app) ...`) добавить тонкие обёртки над уже существующими зависимостями. Пример для нового scope-запроса:

```go
func (a *app) ListAllFederationSecretScopes(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) {
	return a.queries.ListAllFederationSecretScopes(ctx)
}
```

Для `ListAllSubgraphs`, `ListFederationSecrets`, `LatestNoteViews`, `PublicURL` — проверить, реализованы ли уже (скорее всего да: `PublicURL` есть на `cmd/server/main.go:1638`, `ListFederationSecrets` используется в GraphQL). Добавить только реально отсутствующие. Имена методов брать из ошибок компиляции Step 2.

> Если `LatestNoteViews` на app называется иначе (например, отдаёт views синхронно без ctx) — поправить сигнатуру `Env` в `resolve.go` под фактическую и перегенерировать.

- [ ] **Step 4: Сборка**

Run: `go build ./...`
Expected: без ошибок.

- [ ] **Step 5: Commit**

```bash
gofmt -w .
git add internal/router/ cmd/server/main.go
git commit -m "feat(federation): register /_system/federation/admin route and wire env"
```

---

## Task 5: Тест авторизации + smoke

**Files:**
- Modify: `internal/case/system/federationtopology/resolve_test.go` (добавить кейсы)

- [ ] **Step 1: Тест — revoked секрет отдаётся с revoked_at, пустой scope = пустой массив**

Добавить кейс: секрет с `RevokedAt != nil` и без scope → `Outbound[i].RevokedAt != nil`, `Outbound[i].Subgraphs == []` (не nil, сериализуется как `[]`).

```go
func TestRevokedAndEmptyScope(t *testing.T) {
	now := time.Now()
	kbURL := "https://x/_system/mcp"
	env := &EnvMock{
		PublicURLFunc:        func() string { return "https://me.io" },
		ListAllSubgraphsFunc: func(ctx context.Context) ([]db.Subgraph, error) { return nil, nil },
		ListFederationSecretsFunc: func(ctx context.Context) ([]db.ListFederationSecretsRow, error) {
			return []db.ListFederationSecretsRow{{ID: 1, Kid: "k", KbUrl: &kbURL, RevokedAt: &now}}, nil
		},
		ListAllFederationSecretScopesFunc: func(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error) { return nil, nil },
		LatestNoteViewsFunc:               func(ctx context.Context) (*model.NoteViews, error) { return &model.NoteViews{}, nil },
	}
	out, err := federationtopology.Resolve(context.Background(), env)
	require.NoError(t, err)
	require.NotNil(t, out.Outbound[0].RevokedAt)
	require.NotNil(t, out.Outbound[0].Subgraphs)
	require.Len(t, out.Outbound[0].Subgraphs, 0)
}
```

- [ ] **Step 2: Прогнать пакет**

Run: `go test ./internal/case/system/federationtopology/ -v`
Expected: PASS.

- [ ] **Step 3: Полный тест-сьют**

Run: `go test ./...`
Expected: PASS (или без новых падений).

- [ ] **Step 4: Commit**

```bash
gofmt -w .
git add internal/case/system/federationtopology/
git commit -m "test(federation): revoked and empty-scope cases for topology endpoint"
```

---

## Self-Review checklist

- Покрытие спека: self-identity ✓ (Task2), subgraphs ✓, outbound/inbound со scope ✓ (Task1+2), kb_notes ✓ (Task2), admin-only ✓ (Task3), revoked/broad ✓ (Task5). KB-note `description` — подтвердить источник тела ноты при реализации.
- Плейсхолдеры: два явно помеченных места требуют сверки полей (`NoteView.Path`/тело, `db.Subgraph` поля) — это сверка реальных типов, не выдуманные имена; альтернатива хуже (угадать имя). Реализатор сверяет по указанным file:line.
- Типы согласованы: `Response`/`Secret`/`Subgraph` используются одинаково в resolve.go, endpoint.go и тестах.
