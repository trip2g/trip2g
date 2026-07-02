# Weekly audit — 2026-06-25 → 2026-07-02

**TL;DR.** The headline of the week is the **fleet AI agent runtime**: trip2g notes can now trigger scoped LLM agents. You write "role notes" (frontmatter = config, Jet-templated body = system prompt) in a `roles/` folder; a standalone `fleet` daemon discovers them, reconciles change/cron webhooks back to trip2g, and runs a scoped tool-loop (`search`, `read_note`, `patch_note`, `write_note`, optional off-by-default `exec`) writing results back through per-delivery scoped tokens. trip2g itself stays a dumb event source. Around it landed release-engineering (downloadable multi-platform binaries + `v0.8.0`, fleet bundled into the image), a new `trip2g lint docs` command wired into CI, a jsonnet-driven mock server for e2e, DB pool/file/WAL metrics + `_dqs` SQL hardening, and cleanup (cross-encoder reranker, in-repo kanban copy, and old k6 bench all deleted). Scale: ~509 files, +61.8k / −16k since `a528816a`.

This audit is grounded in `git log --since=2026-06-25 main`, the code under `internal/fleet/` + `internal/agentruntime/`, the fleet/dev docs, and the `## v0.8.0` changelog entry.

---

## 1. Week at a glance

Merges since 2026-06-25 (PR#, effect). Note the first few days (25–28 Jun) are the tail of the kanban/editor/perf work; the agent runtime and release engineering dominate 29 Jun–1 Jul.

### Agents / fleet runtime
| PR | Title | Effect |
|----|-------|--------|
| #53 | agent runtime — fleet-as-executor | Core landing: `internal/agentruntime` loop + `internal/fleet` daemon; `cmd/fleet --once`; delivery attribution moved to `note_version_delivery_attribution` 1:1 table |
| #57 | sqlite compiled-statement cache | Read-pool prepared-statement cache (driver upgrade to modernc v1.53.0) — perf groundwork the fleet also rides on |
| #60 | fleet e2e | fleet + llm-mock in docker-compose + `e2e/fleet.spec.js`; `RequireHTTPSUnlessDevMode` for docker-internal webhook URLs |
| #61 | code-executor | `executor: code` role kind + `exec` tool + `--allowed-programs` (off by default) + `env_passthrough` opt-in |
| #63 | code-exec refine | interpreters JSON + startup override, max-stdout config, relaxed write_patterns, pipe foundation |
| #64 | process-isolation docs | tiered sandboxing reference for the code executor |
| #65 | fleet cron mode | cron-mode roles (`/deliver/cron`) + `TRIP2G_FLEET_` env prefix |
| #66/#67 | krisp-mock + krisp-ingest e2e | synthetic Krisp API server + cron `executor:code` → transcript notes e2e |
| #68 | scoped genqlient | scoped KB lane via typed genqlient (drops hand-written GraphQL) |
| #70 | fleet graceful shutdown | drain in-flight + `--keep-webhooks-on-shutdown` |
| #72 | fleet logger | structured logging via `internal/logger` |
| #75 | fleet-in-image | static `/fleet` binary bundled into the trip2g Docker image |
| #76/#77 | krisp article (+ru humanize) | "try the prompt before the bigger model" writeup + Russian segmentation article |

Earlier fleet-hardening commits (pre-PR, on `feat/agent-runtime`): HAT admin auth (`8a27cecb`, `7937b93a`), `Role.Validate` (`e1bf8280`), `--dry-run` (`abc88708`), typed admin genqlient (`e681c004`), detached-context timeout (`fdccf62e`).

### Tooling & release
| PR | Title | Effect |
|----|-------|--------|
| #79 | `trip2g lint docs` | `internal/doclint` DB-free FS lint (broken wikilinks, cross-lang leaks, smoke-render); replaces `check-doc-lang-links.sh`; baseline-ratchet added (`e3d12426`) then dropped (`2a46398c`) |
| #62 | drop check-doc-links CI job | retires the old bash heuristic ahead of the lint command |
| #80 | release-binaries | GitHub-release archives for 5 platforms, frontend built before cross-compile, sha256 checksums, fleet bundled |
| #73/#74 | jsonnet mockserver | `cmd/mockserver` + configs; e2e mocks migrated off bespoke Go mocks |
| #69 | modernc sqlite observability | DB pool/file/WAL Prometheus metrics, `_dqs` hardening, drop `mattn/go-sqlite3` from tooling |
| #71 | intrange lint fix | metrics loop uses range-over-int |
| #81 | changelog v0.8.0 | user-facing v0.8.0 entry + `how_to_release.md`; whole changelog humanized |

### Search & cleanup
| PR | Title | Effect |
|----|-------|--------|
| #78 | release cleanup | remove cross-encoder reranker (`adf84ce7`), in-repo kanban copy (`7f118043`), old k6 render bench (`e5c25d1c`) |

### Docs
| PR | Title | Effect |
|----|-------|--------|
| #53 | agents_how_it_works (en+ru) | public explainer of the agent model |
| #64 | process_isolation.md | sandboxing design research |
| #75 | fleet_packaging.md | one-image / two-binary packaging |
| #78 | reranker.md | forensic record of the deleted reranker |
| #76/#77 | krisp thoughts articles | prompt-first / segmentation essays (bilingual) |

(The 25–28 Jun tail — kanban live-sync fixes #46–#49, theme toggle #48, memcli `--kanban` #45, editor create-file #44, preview-watch #37, mcp-search-limit #34, editor live-update #33, several perf PRs #50–#56 — predates the audit focus and is not detailed here.)

---

## 2. The agent runtime (deep section — the focus)

Two packages implement it: `internal/agentruntime` (the provider-agnostic agent loop + code executor + KB abstraction) and `internal/fleet` (the daemon: discovery, webhook reconcile, HTTP delivery, auth). `cmd/fleet/main.go` wires them.

### 2.1 The model

**Role notes.** A role is a Markdown note in the agents folder (default `roles/`, `--agents-folder`). Frontmatter is config, parsed in `role.go:46-79` into a flat key→value map: `executor` (`""`/`llm`/`code`), `model`, `tools`, `read_patterns`, `write_patterns`, `max_tokens`, `max_steps`, `timeout_seconds`, `mode` (`change`/`cron`/`both`), `trigger_include`/`trigger_exclude`/`trigger_on` (`create`/`update`/`remove`), `cron_schedule`, `attach_notes`, `max_depth`, `concurrency` (`allow_overlap`/`skip`/`queue_one`), `for_each` (`""`/`changed_files`/`attached_notes`), and (code-only) `env_passthrough`/`env_prefix`. Lists accept JSON or YAML-flow form.

`Role.Validate` (`role.go:122-213`) rejects malformed roles before they get a webhook: unknown mode; cron/both without `cron_schedule`; change/both with empty `trigger_on` or `trigger_include`; a body that mentions `change_file` without `for_each: changed_files` (renders against nil — a real footgun caught here); bad `concurrency`/`for_each` enums; negative `timeout_seconds`; tools outside the fleet's `OfferedTools`; for `executor: code`, a body without a complete fenced block or with an unsupported fence language, and an empty entry in `env_prefix` (which would match every env var). Invalid roles are logged and excluded — they never get a webhook.

**Jet-templated body.** `render.go:27-52` renders the body as a Jet template with `WithSafeWriter(nil)` (no HTML escaping — deliberate, so prompts aren't corrupted) exposing exactly five variables: `changed_files`, `change_file`, `attached_notes`, `depth`, `now`. Any other identifier is a Jet error — credential leakage into the prompt is structurally impossible. `withAttachedNotesHint` (`render.go:72-86`) appends an "Attached notes available: …" line for attached paths not already referenced.

**The daemon and discovery.** `cmd/fleet/main.go:48-127`: parse config, build an HTTP client, a HAT-authenticated admin genqlient client, and the LLM client; then `NewFleet` + `NewDiscovery` + `NewReconciler`; run one `syncOnce` (discover + reconcile) *before* the HTTP server accepts traffic; then serve `POST /deliver/` and `GET /health`, re-syncing every `--poll-seconds` (default 30s). Discovery lists role notes via the admin lane and parses/validates each.

**Webhook reconcile.** `reconcile.go` diffs desired vs. actual webhooks and only touches its own (identified by a `fleet:<id>:...` / `fleetcron:<id>:...` marker in the webhook `description`; foreign webhooks are never modified). Change webhooks (`reconcileChange`) exist for `mode: change|both`; cron webhooks (`reconcileCron`) for `mode: cron|both`. A `specVer` — a 6-byte base64url SHA-256 over trigger/scope/timeout/concurrency fields plus a manual `specSchemaVer="schema=2"` bump — is embedded in the marker so a role edit rotates the webhook. Change and cron webhooks derive **different** HMAC secrets (`secretFor`/`cronSecretFor`) so a delivery for one can't be replayed against the other. Each webhook is registered with the role's `EffectiveTimeoutSeconds()` so trip2g waits the full agent runtime before closing the connection.

**The scoped agent loop.** On delivery (`handler.go`), HMAC is verified (`handler.go:94-101`), the body is capped at 10 MiB, the endpoint shape (`/deliver/<key>` vs `/deliver/cron/<key>`) selects change vs cron, and the run dispatches into `agentruntime.Run`. The loop (`runtime.go:99-193`):

- Hard-validates LLM/KB non-nil and `MaxTokens>0`, `MaxSteps>0`.
- System prompt (`runtime.go:77-95`) states the read/write scope is "not negotiable"; first user turn is a fixed `"Begin."`.
- Each step checks the cumulative token cap *before* calling the LLM (`runtime.go:142-145` → `StatusCapped`), calls `LLM.Chat`, accumulates prompt+completion tokens, and processes tool calls. No tool calls → treat text as the answer → `StatusCompleted`. `finish` → `StatusCompleted`. Loop exhausted → `StatusMaxSteps`.
- Two enforcement layers for tools: an **advertisement** filter (`allowedToolDefs`, only the role's `Tools` + always `finish` are offered) and an **execution** `permitted` map that denies any tool the model calls anyway — jailbreak/hallucination resistant.

**Writing results back.** Tool handlers accumulate `res.Changes` (`webhookutil.AgentChange`, kinds `write`/`patch`) which the fleet applies through the per-delivery scoped KB (`remotekb.go`) — `UpdateNotesScoped` upsert for writes, patch variant for `patch_note`, with an in-memory overlay so re-reads within a run see prior writes without a round-trip.

### 2.2 Capabilities

**Tool set** (`runtime.go:21-28`): `search` (full-KB search, results filtered to read scope, returns paths), `read_note` (denied outside read scope), `write_note` (full-note upsert, denied outside write scope), `patch_note` (unique find-replace; errors if the find string is absent or ambiguous), `finish` (`{"answer":...}`), and `exec` (only advertised/reachable when `AllowedPrograms` is non-empty; registered via an extension-invoker seam at `runtime.go:35,346-353` for future MCP tools).

**`executor: code`** (`runcode.go`, `coderun.go`). A code role's body's first fenced block is extracted, the fence language resolves to an allowlisted interpreter, and `RunBlock` runs it in a throwaway `os.MkdirTemp` workdir (files `0o600`), reading the delivery bag from `$FLEET_INPUT` and emitting `{"changes":[...],"answer":"..."}` on stdout. Off by default: `--allowed-programs` / `TRIP2G_FLEET_ALLOWED_PROGRAMS` is empty, so no interpreter is permitted until an operator opts in. Interpreters ship embedded from `interpreters.json` (python/node/bash/ruby/php/perl) and can be overridden at startup via `--interpreters` (`SetInterpretersJSON`/`LoadInterpretersFile`).

**`env_passthrough` / `env_prefix`.** The child env is built minimally (`buildChildEnv`, `coderun.go:308-336`): only `PATH` + `FLEET_INPUT` by default, plus any exact-name (`env_passthrough`) or prefix-matched (`env_prefix`) vars the role explicitly forwards. JWT secrets and LLM API keys never reach the child by construction, not by blocklist.

**`for_each` fan-out** (`handler.go:239-264`): `changed_files` runs the role once per changed file (`change_file` bound per item); `attached_notes` runs once per attached note (scoped to a single-element slice). Fan-out is sequential, continue-on-error: partial success → status `partial`, all-error → 502, zero items → immediate 200 no-op (avoids retry storms).

**Budgets / ceilings.** `clampBudget(want, ceiling)` (`fleet.go:74-79`): a role's `max_tokens`/`max_steps` are clamped to the operator's non-overridable `--token-ceiling` (default 100000) and `--step-ceiling` (default 25). A role can ask for less but never more. `timeout_seconds` defaults to 300s.

**Concurrency / no-overlap.** Delegated to trip2g via the webhook's `ConcurrencyMode` — the fleet has no in-process per-role lock. `skip`/`queue_one` mean trip2g simply never sends overlapping deliveries.

**Per-delivery attribution + spend.** Each delivery carries a short-lived, write-scoped `api_token` issued by trip2g; the fleet builds `NewScopedGraphQLClient` per delivery so all writes are attributed to that delivery (in fan-out, all items reuse the same token — `handler.go:154`). `totalTokens`/`totalSteps` accumulate and are returned in the response. Attribution is persisted app-side in the new `note_version_delivery_attribution` 1:1 table (`6bd9e019`).

### 2.3 Auth & security

**Admin HAT minting** (`hatauth.go`). The fleet has no password: it mints a Hot Auth Token from the app's JWT secret (`--jwt-secret`) for `--admin-email` with `AdminEnter: true` (self-provisions the admin user if absent), POSTs it to `/_system/hat`, captures the `Set-Cookie` on the 302 (a custom `CheckRedirect` short-circuits the redirect), and caches the session cookie behind an `RWMutex`. `adminlane.go` re-authenticates and retries once on a 401.

**Scoped per-delivery tokens** (`remotekb.go` + `adminlane.go` `scopedDoer`). The KB lane for an actual agent run uses the delivery's own bearer token (write-scoped to the role's `write_patterns`), *not* the admin cookie. That token is baked into the GraphQL doer's headers only — it never appears in Jet template vars or the `$FLEET_INPUT` bag. Scope is also enforced client-side in `ScopedKB` (`scope.go`): reads/search filtered to `read_patterns`, writes/patches to `write_patterns`, with named sentinels `ErrReadDenied`/`ErrWriteDenied` and no `**`/admin escape hatch. Empty write patterns = read-only by construction.

**Code-execution isolation posture.** The shipped baseline (see `docs/dev/process_isolation.md`) is: secret-scrubbed minimal env, per-run throwaway temp workdir, `exec.CommandContext` timeout, stdout byte cap (1 MiB default), and off-by-default allowlist. It explicitly does **not** stop unbounded CPU/RAM/PID/disk within the timeout, network access, or reads of anything the uid can reach. The doc lays out an unbuilt tiered upgrade (Tier 1 rlimits + netns via `SysProcAttr`; Tier 2 nsjail/cgroups/seccomp/landlock; Tier 3 gVisor/Firecracker/wazero), noting seccomp and tight rlimits can't be applied inside the long-running multithreaded Go daemon and must live in a thin child launcher. `code_executor_content_processor.md` states plainly: "There is no network isolation or filesystem namespace (yet)."

**LLM-only base image + operator opt-in.** Per `fleet_packaging.md`, the image ships **no** language interpreters; operators enable code execution by extending the image (`apk add python3`) or copying the static `/fleet` binary into their own runtime, then setting `--allowed-programs`.

### 2.4 Packaging

`fleet_packaging.md`: one Docker image, two static (`CGO_ENABLED=0`) binaries — `/trip2g` (server, `CMD`) and `/fleet` (agent host, run as a sidecar or `command: ["/fleet"]` override). Confirmed by `9a077a9a` bundling `/fleet` into the image (#75) and the release workflow bundling `fleet` into every archive (#80). Config is entirely `TRIP2G_FLEET_*` env vars (or `cmd/fleet` flags); daemon-required values are `--callback-url`, `--jwt-secret`, `--fleet-secret`, `--llm-api-key` (falls back to `OPENAI_API_KEY`). `--admin-api-key` remains as a **deprecated/unused** field with no deprecation warning.

### 2.5 The KB-construction angle (Krisp case)

`docs/fleet/krisp/` is the worked example: source-agnostic ingest → segment → wiki-extract → `[[wikilink]]` graph.

- `transcripts/<id>.md` written by a **deterministic** `executor: code` cron ingest role (`transcript-ingest.md`, Python, hits the Krisp API, never LLM-written) →
- `transcript-segment` (triggers on `transcripts/**`, `qwen/qwen3-14b`, `write_note` only) → `segments/<id>.md` with Minto-headline topic map + `[MM:SS–MM:SS]` ranges →
- `transcript-wiki` (triggers on `segments/**`, `read_note`+`write_note`) → `qwen3-14b/wiki/<id>.md`, a knowledge note with `[[WikiLinks]]` into the vault graph. The chain terminates because nothing triggers on `wiki/**`; both use `for_each: changed_files` and `max_depth: 3`.

**Proven** (validated 2026-06-29 via `cmd/fleet --once`): segment step completed in ~7.6k tokens / 2 steps producing 3 correct segments; wiki step ~11.2k tokens / 2 steps producing a structured note with 7 `[[WikiLinks]]`; total cost fractions of a cent. **Aspirational/unbuilt (admitted):** porting the real Krisp-transcript ingest into `transcripts/*`, and the Qwen-14B vs gpt-5.4-mini quality comparison. The captured `segments/sample-call.md` and `wiki/sample-call.md` are outputs of that run over a synthetic transcript.

### 2.6 Maturity, gaps, risks in the runtime

Proven: scope enforcement, tool allowlist (both layers), token hard-cap, secret scrub, env passthrough, workdir isolation, HMAC verification, webhook reconcile lifecycle, HAT auth exchange, scoped-KB overlay — all have direct unit tests, and there is a docker-compose e2e (`e2e/fleet.spec.js`) plus the Krisp cron e2e.

Partial / risky (from the code):

- **No process sandbox.** `exec` runs in the same uid as the daemon, on the host FS, with no seccomp/namespaces/network isolation. Write-scope is the *only* barrier on what a script can change; a malicious/compromised script emits arbitrary `changes` JSON within the write scope. G204 is acknowledged in a comment as "operator-allowlisted is sufficient" — that is a policy bet, not a technical control. Tiered hardening is designed but unbuilt.
- **exec tool has no per-invocation wall-clock timeout** (`runtime.go:371-377` passes `Timeout: 0`) — bounded only by the parent context; if that has no deadline a child can run indefinitely.
- **`registry` (interpreters) is a package-level pointer with no mutex** (`coderun.go:36`); a hot-reload override concurrent with `RunBlock` races.
- **Only the first fenced block runs** ("future pipe support will run multiple blocks" — no tracking issue).
- **OpenAI LLM has no retries, no timeout config, no dollar spend cap** (`openai_llm.go`) — only token counts; a `len(resp.Choices)==0` response is silently treated as an empty completion, masking provider errors.
- **`gracefulShutdown` return is discarded** (`main.go:116` `_ = ...`) — a failed/partial drain is indistinguishable from a clean exit to a supervisor.
- **Reconcile is not atomic** (delete-then-create) and a `reconcileChange` error aborts `reconcileCron` for that tick — self-heals next poll but leaves a skew window.
- **`FileKB.Search` is a full directory walk** (`--once`/local only) — fine for local runs, not for large vaults.
- Test gaps: no direct `--once` end-to-end LLM test, no `DiscoverParsed` (`--dry-run`) integration test, no `MaxSteps`-exhaustion status assertion.

---

## 3. Everything else this week

### `trip2g lint docs` (#79, #62)
`internal/doclint` is a DB-free FS `Env` + `lint` subcommand (`cmd/server/lint.go`) that runs the real note-loader over a docs tree and reports cross-language wikilink leaks (a `ru/` note linking `en/`), layout render errors, and broken links (advisory, non-blocking). It replaces `check-doc-lang-links.sh` and is wired into CI (`.github/workflows/ci.yml`). A baseline-ratchet was added (`e3d12426`) then deliberately dropped so CI runs `trip2g lint docs` with no baseline (`2a46398c`). Fixtures live in `internal/doclint/testdata/lint/` with `TestLint_FixtureDir`.

### Downloadable release binaries (#80) + v0.8.0
`.github/workflows/release-binaries.yml` builds the frontend first (so the embed is populated), cross-compiles for **five** targets, bundles both `trip2g-server` and `fleet` into each archive, and emits `.sha256` files. Release `v0.8.0` (2026-07-01) assets confirm sizes: linux amd64 tar.gz **36.2 MB**, linux arm64 33.0 MB, darwin amd64 37.0 MB, darwin arm64 34.7 MB, windows amd64 zip 37.0 MB (compressed; the uncompressed two-binary payload is the ~90 MB-class figure). macOS binaries are unsigned (Gatekeeper warns).

### Search / reranker removal (#78)
`adf84ce7` removed the optional cross-encoder reranker (`internal/reranker`, the `reranker-server/` Python sidecar, `VectorSearchConfig.Reranker`). Two benchmark rounds showed it strictly hurt: nDCG 0.9221 → 0.8881 (512-char passages) → ~0.39 (full-note passages), because the cross-encoder over-weighted surface term overlap and truncated long passages. Forensic record kept in `docs/dev/reranker.md` (which notes an untried "blend rerank score with RRF rank" idea). Same PR removed the in-repo kanban_template copy (`7f118043` — source lives in `trip2g/kanban_template`) and the old k6 render benchmark (`e5c25d1c`).

### Mockserver (#73, #74)
`cmd/mockserver` is a jsonnet-driven mock server with `configs/` (krisp + llm configs); e2e mocks migrated off bespoke Go mocks to run via it (`b591c27d`). `cmd/krispmock` (`43f42380`) is a synthetic Krisp API used by the Krisp-ingest e2e.

### SQLite / metrics (#69)
`f81f44f4` exports DB connection-pool, file-size, and WAL-size Prometheus metrics on the existing internal metrics endpoint (WAL size read from the filesystem, not `PRAGMA wal_checkpoint`, which has side effects). `879c80f0` adds `_dqs=false` so double-quoted string literals become startup parse errors. `de4cb70c` drops `mattn/go-sqlite3` from test tooling (app already runs pure-Go modernc). Earlier in the week `9cc784ab`/#57 added a compiled-statement cache on the read pool (driver upgrade to modernc v1.53.0). Details in `docs/dev/modernc_sqlite.md`, which flags one unenforced soft invariant: `modernc.org/libc` must match the version `modernc.org/sqlite` pins — a CI assertion is recommended but not yet added.

### Changelog (#81)
`docs/en/changelog.md` `## v0.8.0 (2026-07-01)` frames the release around five items: fleet agent runtime, downloadable binaries with checksums, `trip2g lint docs`, DB metrics + stricter SQL, and reranker removal. The Russian twin `docs/ru/changelog.md` exists. `how_to_release.md` was added.

---

## 4. Observations & risks

- **Code-execution sandboxing is the biggest open risk.** The `exec` path has no OS-level isolation today — write-scope is the only barrier, network is open, and the child shares the daemon's uid/filesystem. It's off by default and the tiered design exists, but any operator who turns on `--allowed-programs` is running arbitrary role-authored code with only a policy fence. Prioritize at least Tier-1 (rlimits + netns via a thin child launcher) before recommending code roles to untrusted authors.
- **LLM client is thin.** No retries/backoff on 429/5xx, no request timeout config beyond context, no dollar-denominated spend cap (only token count), and a silent empty-completion path. For an unattended daemon calling a paid API in a loop, add retry + a spend guardrail and treat `len(Choices)==0` as an error.
- **Binary size.** Archives are ~33–37 MB compressed for five platforms; the uncompressed two-binary payload is large (~90 MB class). Acceptable for a single-binary story but worth a `-ldflags "-s -w"` / UPX pass if it grows.
- **Small correctness/observability debts:** discarded `gracefulShutdown` error (`main.go:116`), non-atomic reconcile with a change→cron abort skew window, unsynchronized interpreter `registry` pointer, only-first-fenced-block execution, and `env_prefix` accepting single-char prefixes (only empty is rejected).
- **Test coverage of the agent paths is good at the unit level** (scope, allowlist, budgets, secret scrub, HMAC, reconcile, auth, remote KB) with a docker-compose + Krisp cron e2e, but has gaps: no `--once` end-to-end LLM test, no `--dry-run`/`DiscoverParsed` integration test, no `MaxSteps`-exhaustion assertion.
- **Unenforced dependency invariant:** the `modernc.org/libc` ↔ `modernc.org/sqlite` version pin is satisfied but not guarded in CI; a `go get -u` could introduce a subtle mismatch. Cheap to add the recommended assertion.
- **Follow-ups worth doing:** deprecation warning (or removal) for `--admin-api-key`; the `FileControlDataVersion` cache-invalidation primitive noted in `modernc_sqlite.md` (addresses the read-replica freshness hole); tracking issue for multi-block pipe execution.
