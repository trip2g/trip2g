# Contributing to trip2g

Thanks for considering a contribution. This guide gets you from a fresh clone to a running dev server and a mergeable PR.

## TL;DR

```bash
git clone https://github.com/trip2g/trip2g && cd trip2g
go mod download && npm install
make db-up          # apply SQLite migrations
make air            # dev server with hot reload (starts MinIO via docker)
```

Open an issue or comment on an existing one before starting anything non-trivial, so we agree on the approach before you write code.

## Prerequisites

- Go 1.26+
- Node.js 20+
- Make
- Docker (for MinIO, the local S3-compatible asset store)

## Project layout

Go monolith + $mol frontend + SQLite + GraphQL (gqlgen).

| Path | What lives there |
|---|---|
| `cmd/server` | main binary |
| `internal/case/` | one package per GraphQL mutation/query (the use case pattern) |
| `internal/` | services: gitapi, noteloader, defaulttemplate, router, db (sqlc) |
| `assets/ui/` | $mol frontend (`.view.tree` structure + `.view.ts` behavior) |
| `docs/dev/` | developer docs; start with `app_patterns.md`, `architecture.md`, `source-tree.md` |
| `docs/en/user/`, `docs/ru/user/` | user docs, always in EN + RU pairs |

## Build and generate commands

Generated code is committed. After editing a source of generated code, run the matching generator and commit both together.

| Command | When |
|---|---|
| `make air` | dev server with hot reload |
| `make test` / `go test ./...` | run tests |
| `make lint` | golangci-lint + SQL query checks |
| `make gqlgen` | after editing `schema.graphqls` |
| `make sqlc` | after editing `queries.read.sql` / `queries.write.sql` |
| `make db-new name=...` / `make db-up` | create / apply a migration (dbmate) |
| `go generate ./internal/defaulttemplate/...` | after editing `views.html` or `langs/*.toml` (quicktemplate, not Go templates) |
| `go generate ./internal/router/...` | after adding HTTP endpoints |
| `npm run build` | TypeScript + Vite build |
| `npm run codegen:ui` | frontend GraphQL types from co-located `.graphql` files (no server needed) |
| `npm run test:e2e` | Playwright E2E tests |
| `go run -tags dev ./cmd/server lint docs` | check doc links after editing anything under `docs/` |

## Testing

- Write tests first when practical (red, green, refactor).
- Table-driven tests with `testify/require`; mocks are generated with `moq` (`//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env`).
- `make test` must pass before you open a PR.

## PR conventions

- **Language:** everything in English (code, comments, commits, PR text).
- **Commits:** short one-liners, conventional format: `feat(scope): description`, `fix(scope): ...`, `docs: ...`. No commit body, no co-author trailers.
- **Match the existing style.** Read the neighboring code before adding anything new. Comments only where the code is non-obvious, and no decorative banner comments.
- **Small PRs merge faster.** One concern per PR.
- **Generated files** (`views.html.go`, `internal/db/`, gqlgen output) belong in the same commit as their source.
- SQL migrations get extra scrutiny; call them out in the PR description.

## Where to start

- Issues labeled [`good first issue`](https://github.com/trip2g/trip2g/labels/good%20first%20issue).
- Docs fixes: the bilingual user docs under `docs/en/user/` and `docs/ru/user/` always welcome corrections (keep the EN/RU pair in sync).
- Read `docs/dev/app_patterns.md` before proposing backend changes; it explains the Env interface and service package patterns the codebase is built on.

## Questions

Open a [GitHub issue](https://github.com/trip2g/trip2g/issues) or a discussion. For security issues, please do not open a public issue; contact the maintainer directly.
