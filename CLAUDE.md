# trip2g

Publishing platform: Obsidian vaults → websites with subscriptions & Telegram integration.
Go monolith + $mol frontend + SQLite + GraphQL (gqlgen).

## Rules

- **Language**: all communication, code, comments, commits in English
- **Commits**: short one-liner, no Co-Authored-By, no body. Conventional: `feat(scope): desc`
- **Consistency**: match existing patterns and style before adding anything new
- **Error checks**: for long calls or contextual error wrapping, assign `err` on a separate line before checking it; avoid forcing `if err := ...` when it harms readability
- **Comments**: short, and only where non-obvious. Less is better than more. NO decorative/banner dividers (`// ── section ──`, `// ===`, `// ----`) — they go stale and clutter. Don't restate what the code says.
- **SQL migrations**: always ask for confirmation before creating
- **Proactive suggestions**: short remarks only, don't elaborate unless asked
- **TDD**: write tests first, then implementation. Red → Green → Refactor

## Build Commands

| Command | When |
|---------|------|
| `go generate ./internal/defaulttemplate/...` | After editing `views.html` or `langs/*.toml` |
| `npm run defaulttemplate-css` | After editing `assets/defaulttemplate/src/index.scss` |
| `go generate ./internal/router/...` | After adding HTTP endpoints |
| `make gqlgen` | After editing `schema.graphqls` |
| `make sqlc` | After editing `queries.read.sql` / `queries.write.sql` (repo root; generated into `internal/db/`) |
| `npm run graphqlgen` | After GraphQL schema changes (frontend types) |
| `npm run toc` | After editing `assets/toc/src/index.ts` |
| `npm run build` | TypeScript + Vite build |
| `npm run test:e2e` | E2E tests (Playwright) |
| `make air` | Dev server with hot reload |
| `cd docs && node ../obsidian-sync/dist/trip2g-sync.mjs --folder .` | Sync the `docs/` vault (e.g. `docs/demo`) to the local instance after editing demo notes/assets. Auto-discovers the API key from `.obsidian/plugins/trip2g/data.json`, so it must run **from the vault root** — no key needed. Add `--dry-run` to preview. Requires `dist/` to be built (`npm run build` inside `obsidian-sync/`). |

## Critical: Default Template Pipeline

`views.html` uses **quicktemplate** syntax (`{%s %}`, `{%= %}`), NOT standard Go templates (`{{.}}`).

After any change to `internal/defaulttemplate/views.html`:
1. Run `go generate ./internal/defaulttemplate/...` to regenerate `views.html.go`
2. The generated file must be committed together with the template

i18n: `internal/defaulttemplate/langs/en.toml` + `ru.toml`, used as `{%s ctx.T("key") %}`

CSS: `assets/defaulttemplate/src/index.scss` → compile with `npm run defaulttemplate-css`

## Key Patterns

### Use Case Pattern (backend)
Each mutation/query is a package under `internal/case/`:
```
internal/case/{name}/
  resolve.go    — Env interface + Resolve(ctx, env, input) function
  endpoint.go   — HTTP endpoint (if needed)
```
`Env` interface declares only the dependencies that specific use case needs.

### Service Package Pattern (stateful orchestration)
Extract stateful, multi-step logic (caches, queues, background loops) into a package under `internal/<domain>/`, gitapi-style (see `docs/dev/app_patterns.md`, `internal/gitapi`, `internal/chartdata`). Rules:
- The package declares its own minimal **`Env interface`** (the primitives it needs). The `app` implements it. Add a compile-time check: `var _ <pkg>.Env = (*app)(nil)`.
- Name the type after the **domain**, not `Service` (e.g. `chartdata.ChartData`, not `chartdata.Service`).
- **Embed the type anonymously in `app`** (`*chartdata.ChartData`) so its methods promote onto `app` — never write proxy methods like `func (a *app) Save() { return a.x.Save() }`.
- A promoted method can't share the embedded field's name; name methods to avoid the clash (e.g. type `ChartData` → provider method `ChartRows`, not `ChartData`).
- In the `Env` impl, **reuse existing `app` methods — don't duplicate them.** A reload helper should call the real `PrepareLatestNotes`/`PrepareLiveNotes` (by mode), not re-implement them.
- State lives on `app` (or the service), never as package-level `var`.

### Error Handling
- Validation/business errors → `ErrorPayload, nil` (user sees message)
- System errors → `nil, error` (user sees "Internal Error")
- Use `model.NewOzzoError()` and `model.NewFieldError()` helpers

### Frontend ($mol)
- Components in `assets/ui/` with `.view.tree` (structure) + `.view.ts` (behavior)
- Symlink: `assets/ui/` ← `../mam/trip2g/`
- `$mol_wire_sync` for async→sync integration
- `$mol_import.script` for async script loading
- Localization: `@` marker in view.tree, locale files `*.view.tree.locale=ru.json`
- GraphQL: `$trip2g_graphql_request()` for queries/mutations

### Testing
- Table-driven tests with `moq` mocks, `testify/require`, `pretty.Diff`
- Mock generation: `//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env`

### Admin GraphQL from a script (dev)
Endpoint `/_system/graphql`. Sign in, then send the returned token as the `trip2g_token` cookie:

```bash
G=http://localhost:8081/_system/graphql
curl -sS -X POST "$G" -H 'Content-Type: application/json' \
  -d '{"query":"mutation{requestEmailSignInCode(input:{email:\"hello@example.com\"}){__typename}}"}'
S=$(curl -sS -X POST "$G" -H 'Content-Type: application/json' \
  -d '{"query":"mutation{signInByEmail(input:{email:\"hello@example.com\",code:\"111111\"}){__typename ... on SignInPayload{token}}}"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['signInByEmail']['token'])")
curl -sS -X POST "$G" -H 'Content-Type: application/json' -b "trip2g_token=$S" -d '{"query":"..."}'
```

- `111111` is `DevSignInCode` (`cmd/server/auth.go`) — it only works in dev builds.
- **Admin queries and mutations live under the `admin` field**, not on `Query`/`Mutation` directly: `{admin{tgBot(id:1){...}}}`, `mutation{admin{createTgBot(input:{...}){...}}}`. Asking for them at the top level fails with `Cannot query field ... on type "Mutation"`.
- The admin UI is hash-routed: `/admin#!nav=system/system_nav=sync`. Plain paths like `/admin/system` return 404.

### Bilingual Docs
User docs are always created in pairs: `docs/en/user/*.md` + `docs/ru/user/*.md`

## Docs Map (docs/dev/)

Read the relevant doc before working in that area. Docs are partially outdated (generated Nov 2025) — verify against current code.

### Start Here
| Doc | When to read |
|-----|-------------|
| `app_patterns.md` | **MUST READ before implementation plans.** Env interface, external service client (gitapi pattern), IO on app layer, error handling, cronjobs, config, GraphQL resolvers, $mol frontend |
| `index.md` | Project overview, tech stack, statistics |
| `architecture.md` | System design, data flows, component interactions |
| `source-tree.md` | Directory structure, file organization |
| `principles.md` | Architectural decisions: Env pattern, error handling, SQLite R/W split, commit convention |
| `development-operations.md` | Setup, build commands, deployment |
| `aicontext.md` | Conventions for extending backend and frontend |

### Backend
| Doc | When to read |
|-----|-------------|
| `instructions.md` | GraphQL mutation step-by-step, testing patterns, validation |
| `graphql.md` | Why GraphQL, gqlgen usage |
| `gqlgen.md` | gqlgen features and config |
| `gqlgen_fasthttp.md` | GraphQL subscriptions over SSE with fasthttp |
| `sqlite.md` | SQLite config, WAL mode, R/W separation |
| `queues.md` | Background jobs: goqite + backlite |
| `goqite.md` | Job queue details, long vs short jobs |
| `obsidian_sync.md` | Obsidian plugin sync protocol |
| `obsidian_links.md` | Wikilink resolution algorithm |
| `note_rendering.md` | Markdown → HTML pipeline |
| `routes.md` | URL routing with `route`/`routes` frontmatter |
| `multilang.md` | Multi-language support (`lang`/`lang_redirect`) |
| `features.md` | Feature flags system |
| `cache.md` | Telegram API response caching |
| `hat.md` | Hot Auth Token — fast auth from external systems |
| `webhooks.md` | Webhook architecture |
| `data-models.md` | Database schema (50+ tables) — generated |
| `api-contracts.md` | GraphQL API reference (100+ operations) — generated |

### Frontend
| Doc | When to read |
|-----|-------------|
| `frontend.md` | Admin CRUD patterns, mol essentials, GraphQL requests, localization |
| `frontend_crud.md` | Step-by-step admin CRUD guide |
| `mol.md` | $mol view.tree syntax, properties, bindings, localization tricks |
| `ui-components.md` | Component catalog (131 components) — generated |
| `bem.md` | BEM naming reference |
| `browser_sync.md` | Browser sync module |
| `editor.md` | File editor component |

### Default Template
| Doc | When to read |
|-----|-------------|
| `default_template.md` | Template architecture, Ctx fields, widgets, magazine, CSS, i18n |
| `mermaid.md` | Mermaid diagrams + conditional per-note widget script loading (backend-decided) |
| `layouts.md` | Layouts API for templates |
| `template_processors.md` | Template processor system |
| `template_content_type.md` | Content-Type from template |
| `json_layouts.md` | JSON layouts format |
| `design.md` | Design tasks and decisions |

### Telegram
| Doc | When to read |
|-----|-------------|
| `telegram.md` | Publishing architecture, message formatting |
| `telegram_e2e.md` | Telegram E2E testing |
| `telegram_bot_vs_userbot.md` | Bot vs userbot: emoji and media limits |
| `telegram_custom_emojies.md` | Custom emoji in Obsidian |
| `telegram_import.md` | Channel import plan |
| `telegram_inbox_agent.md` | Inbox bot: Telegram → trip2g |
| `telegram_publish_through_accounts.md` | Publishing through user accounts |

### Infrastructure & Config
| Doc | When to read |
|-----|-------------|
| `admin_config_modules.md` | Self-hosted config philosophy |
| `config_refactoring.md` | Config refactoring plan |
| `multidomain.md` | Multi-domain routing |
| `simplebackup.md` | Simple backup system |
| `fleet_codellm_metrics.md` | fleet/codellm Prometheus metrics: second listener, metric catalog, what to alert on |
| `sitemap.md` | Sitemap.xml generation |
| `rss.md` | RSS feed |
| `vector_search.md` | Vector search (OpenAI embeddings) |
| `google_github_auth.md` | OAuth implementation |

### Design Docs & Plans
| Doc | When to read |
|-----|-------------|
| `onboarding.md` | Onboarding page design |
| `frontmatter_patches.md` | Frontmatter patch system |
| `change_webhooks.md` | Change webhooks design |
| `cron_webhooks.md` | Cron webhooks design |
| `shared_webhooks.md` | Shared webhook infrastructure |
| `webhook_bots.md` | Webhook bot patterns |
| `job_statuses.md` | Job status tracking |
| `task_loop.md` | Task loop workflow |
| `subprocess_agent.md` | Subprocess agent (subot) |
| `update_note_mutation.md` | Atomic note updates via find/replace |
| `obsidian_sync_refactoring.md` | Sync plugin refactoring plan |
| `obsidian_sse_pulls.md` | Live note change subscriptions |
| `canvas.md` | Rendering Obsidian Canvas (.canvas) — design, not yet built |
| `bases.md` | Obsidian Bases (.base) data-views — design, not yet built |
| `excalidraw.md` | Rendering Excalidraw drawings — design, not yet built |
| `grid_layouts.md` | Flexible grid/flex page-layout DSL — subsumes magazine/wide/sidebars; design, not yet built |

### Testing & Ops
| Doc | When to read |
|-----|-------------|
| `TESTING.md` | E2E testing with Playwright |
| `mol_testing.md` | Frontend unit tests ($mol_test): how to write/run, what's testable |
| `e2e_seed.md` | E2E test database seeding |
| `agents.md` | AI agents documentation |

### History
| Doc | When to read |
|-----|-------------|
| `../en/changelog.md`, `../ru/changelog.md` | User-facing feature changelog (versioned, What/Why/How) — the real one, not under docs/dev |
| `current_tasks.md` | Current task list |
| `refactor.md` | Refactoring roadmap |
| `aiderlog.md` | Historical aider session log |
| `null_string_refactoring.md` | sql.Null* → pointers migration |
| `mdloader_same_name_bug.md` | Known bug: image with same name as note |
