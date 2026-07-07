# Code & PR Analysis + One-Month Development Plan (2026-07-07 → 2026-08-07)

**TL;DR.** The last ~10 days produced ~100 merged PRs in three waves: the fleet agent runtime grew from prototype to sandboxed, tested, e2e-covered subsystem; a perf sprint added page/statement/layout caches; and a large content/SEO/landing wave prepared a public launch. Code health is good where it's new (fleet is the cleanest subsystem) and debt-heavy where it's old (note-loading/rendering pipeline, 64 untested use-case packages, a panicking GraphQL resolver). The month ahead: **week 1 — ship the launch and fix its blockers; week 2 — security/robustness hardening; week 3 — agent-platform features (fleet graph v1, updateNote, expected_hash); week 4 — perf debt + onboarding UX follow-ups.**

---

## 1. PR retrospective (PRs #46–#146, 2026-06-28 → 2026-07-06)

~100 merged PRs in 9 days. By theme:

### Fleet / agent runtime (~20 PRs) — the month's biggest investment
- Agent runtime core: fleet-as-executor, notes trigger LLM agents on edits (#53), cron-mode roles (#65), code executor (#61, #63) with multi-block pipes (#110).
- Security: Tier-0 code-exec sandbox — netns off, Landlock, rlimits, no-new-privs (#85), hardened fail-closed with PID/mount namespaces + enforcing CI (#88), scope-matcher fixes (#144).
- Ops: graceful shutdown (#70), structured logging (#72), static binary in the Docker image (#75), `make airfleet` dev loop (#113).
- Observability: agent dependency graph with JSON API + debug UI (#111).
- Testing: krisp-mock + Krisp-ingest e2e (#66, #67), jsonnet-driven mockserver (#73, #74), shared docker fleet for e2e (#105, #106, #108).

### Performance sprint (~7 PRs)
- Anonymous pre-gzipped page cache (#54) + cache-before-Resolve (#55), compiled-statement cache on the read pool (#57), Jet layout cache (#52), frontmatter-patch cache (#51), DB pool/file metrics + mattn driver drop (#69), write-pool exhaustion diagnostics (#112).

### Search (~4 PRs)
- Reranker removed as measured-worse (#78), then reintroduced in blend mode after a new benchmark showed +0.023 nDCG (#139, #142). Data-driven embedding config for custom models (#141). Token-economy correctness: fail loud on pointer misses (#109).

### Security & correctness fixes (~10 PRs)
- Jsonnet VM filesystem-import escape closed (#145), JWT token-type confusion (#140), per-subscriber ACL on noteChanges (#96), canonical `/_system/admin` routing (#146), SSE keepalive to reap dead subscribers (#114), subscription ctx.Done select (#115), Jet layout panic containment (#120), cronjob orphan-execution fix (#123) + boot-time jobs (#124).
- Kanban live-sync data-loss series finished (#46, #47, #49) + theme toggle (#48).
- Wikilink resolution: deterministic multilingual (#95), then restored Obsidian shortest-path global default (#107).

### Content / SEO / launch prep (~35 PRs)
- Landing rewrite for newcomers (#89, #100), README hero + CONTRIBUTING (#90), SEO content batches (#91, #93, #94), MCP docs-base + A/B benchmark (#99, #101), philosophy essays (universality tax, paths not taken, filming-as-testing: #131–#138), changelog nav (#129), user docs for fleet and calls-to-KB (#102, #104).
- Onboarding: launch-critical UX fixes (#116, #117), starter vault regenerated as a drift-proof build step (#133), plugin bump to 0.7.0 (#143).

### Read of the retrospective
The pace was feature-forward and launch-driven; quality was maintained by follow-up-fix chains (kanban data-loss took 4 PRs, wikilinks 3, e2e stabilization 4). That pattern — ship, then live-test finds bugs, then fix test-first — works, but it means **the debt is concentrated where no wave has passed recently**: the old note pipeline and untested payment/auth use-cases.

Unmerged right now: `feat/record-harness` (deterministic demo-recording target, 5 files, docs+scripts — low risk, mergeable after the demo video ships).

## 2. Code health

Full reports behind this summary: agent scans on 2026-07-07 (hotspots, TODO inventory, test-coverage map).

### Strengths
- **Fleet is the healthiest subsystem**: ~1:1 test-to-source ratio, zero TODOs, lint-clean, deliberate sandbox posture. Production-shaped, not prototype-shaped.
- 46 Playwright e2e specs compensate for unit-test gaps on auth/federation/MCP/fleet flows.
- `//nolint` directives are consistently annotated — suppression is heavy (229) but documented.

### Debt map (ranked)
1. **Panicking live GraphQL resolver** — `internal/graph/schema.resolvers.go:2153` (`HomeNote`) resolves via `panic("not implemented")`; two adjacent resolvers unimplemented (`:2446`, `:2452`). Also `:2343` fetches all nodes where `count(*)` would do (ties into the DoS review).
2. **Note pipeline complexity** — `mdloader`/`noteloader`/`rendernotepage` carry explicit golangci complexity exclusions; `noteloader/loader.go:465` full-copies the note-view set on a hot path. This is the documented pushNotes/render perf debt (`docs/dev/sync_perf_memory_2026-06-29.md`).
3. **Untested sensitive code** — 64 of 195 `internal/case/` packages have no unit tests; the worst combo is payments (`internal/patreon/` 2.6k LOC, `processnowpaymentsipn`) and OAuth callbacks, which rely entirely on e2e.
4. **Auth provenance stopgaps** — hardcoded `CreatedBy: 1` audit actors, `AdminActorUserID` fallback (`cmd/server/auth.go:88`), git-token admin_id not threaded (`cmd/server/notes.go:81`), OIDC id_token signature not verified against JWKS (`internal/oidcauth/client.go:1`).
5. **Silent failures** — `jobs/env.go:37` swallows resolver errors in background-job dispatch; obsidian-sync plugin logs sync failures to console only.
6. **Small irritants** — blocking sleeps in the Telegram update path (`handletgupdate/access.go:108,141`), dead Notion code (`cmd/server/main.go:446`), fleet uses global slog instead of a scoped logger.

### GraphQL DoS posture (from the 2026-07-02 review)
Nested-bomb mitigated by the complexity limit (30), but open: no HTTP rate limiter in front of `/graphql`, per-field complexity disabled (`omit_complexity: true`), persisted-query allowlist generated but never enforced, no explicit max-depth, list resolvers not audited for server-side caps.

## 3. One-month plan

Ordering logic: the launch window (Tue 07-07 / Wed 07-08 morning ET) is now, so week 1 is launch blockers + launch + triage. After launch the instance is exposed to real traffic → week 2 hardens security/robustness. Weeks 3–4 return to product: the agent platform is the strategic direction, perf debt is the biggest self-inflicted risk.

### Week 1 (Jul 7–13): Launch
Blockers, from the landing-link audit + onboarding UX review:
- [ ] README: hero GIF (still an HTML-comment placeholder), fix 3 broken links (`getting-started`/`two-way-sync` slugs), point quickstart at a working docker-compose. Add a link-check to CI.
- [ ] Landing: reroute the 7 "start →" CTAs to `/en/user/getting_started`; relabel `/simple` as "for AI agents →"; replace the fake `mcp:add` command with real per-client blocks; fill or hide the 4 dead `#` links.
- [ ] MCP initialize-note: replace the Marcus Aurelius prompt leak with a trip2g description.
- [ ] Docs: fix the false "`000000` also works" dev-code claim; lead Getting Started with the starter-vault download path.
- [ ] Launch morning: `te_check.py` green against prod, free-cloud signup tested in incognito, no deploys that day (single small VM).
- [ ] Ship the demo video / GIF from `feat/record-harness`, then merge the branch.
- [ ] Post-launch: triage incoming issues daily; keep the rest of the week unplanned on purpose.

### Week 2 (Jul 14–20): Security & robustness hardening
Now public-facing, so:
- [ ] Fix/gate the `HomeNote` panic and the two unimplemented resolvers; replace the fetch-all-nodes count with `count(*)`.
- [ ] GraphQL DoS follow-ups in priority order: HTTP rate limiter on `/graphql` → real per-field complexity → explicit max-depth → list-resolver cap audit. Persisted-query allowlist only if time permits.
- [ ] Unit tests for `processnowpaymentsipn` and the OAuth callback use-cases — best risk-to-coverage ratio in the repo.
- [ ] OIDC: verify id_token signature against `jwks_uri` (documented v1 gap).
- [ ] Audit `jobs/env.go:37` swallowed errors; surface background-job failures.
- [ ] Onboarding P1/P4: surface obsidian-sync failures (Notice + persistent error state + "last successful push" in settings) — silent sync failure is the worst first-run trap.

### Week 3 (Jul 21–27): Agent platform
The strategic lane — everything that makes agents-on-notes more capable:
- [ ] Fleet dependency graph v1 (`cmd/fleet graph`): glob-overlap derivation, `_fleet-graph.md` with Mermaid + findings, `--json`. Design is done (`2026-07-02_fleet_dependency_graph.md`); resolve the 3 open review questions first.
- [ ] `updateNote` mutation (find/replace) — the foundation for agent edits without full rewrites (design: `update_note_mutation.md`).
- [ ] `expected_hash` optimistic concurrency in pushNotes — protects agents from clobbering human edits.
- [ ] Fleet scoped logger (removes the two sloglint suppressions) before the fleet scales further.
- [ ] Telegram InstantView (issue #6, 1–2 days) — high leverage for the distribution push while launch traffic is warm; close issues #9/#11/#12 as triaged.

### Week 4 (Jul 28–Aug 7): Perf debt + polish
- [ ] Note-pipeline perf: kill the full `nvs.Copy()` on the hot path (`noteloader/loader.go:465`), then take the next pushNotes memory candidate (Jet re-parse / Jsonnet std rebuild) from `sync_perf_memory_2026-06-29.md`. Measure before/after with the existing profile.
- [ ] Onboarding second pass: remaining HIGH findings (admin login link in the anonymous block A1, trust-author dialog step A3, plugin 0.6.x → official catalog path P6, SMTP fallback doc D8).
- [ ] Philosophy/docs accuracy: retag markdown-operating-system stale rows, fix the reranker claim in search-benchmark (now "revived as blend"), add the 7 orphaned pages to nav, sync RU/EN index order.
- [ ] Cleanup: remove Notion dead code, replace `handletgupdate` sleeps with backoff, `EnableNotFoundTracking` config (default off).

### Deliberately deferred (backlog, not this month)
- pushNotes → unified write API refactor (do after `updateNote` + `expected_hash` land — they'll inform the design).
- Standard template → `templateviews.Note` refactor; `vectorMinSimilarity` config; doublestar error handling.
- Webhook agent-response assets; subprocess/tginbox external agents (the in-repo fleet covers most of this now).
- Telegram dialogs FLOOD_WAIT (limit+search+cache) — fix when someone hits it again.
- Issues #5 (scheduled-post calendar) and #26 (OKF export) — wait for demand signal.

### Success criteria for the month
1. Launched publicly with no launch-day outage; launch-blocker list empty.
2. Zero panics reachable from GraphQL; `/graphql` rate-limited; payment IPN + OAuth callbacks unit-tested.
3. Fleet graph v1, `updateNote`, and `expected_hash` merged — agents can edit safely and their topology is visible.
4. Measured reduction in pushNotes hot-path allocations (before/after profile in a dev doc).
