# Weekly Review (2026-08-22)

Review of the week 2026-08-16 → 2026-08-22 on `main`: 42 commits (PRs #293–#327), 446 files, +30 613 / −4 344 lines, single author. Method: every commit's full diff was read by a per-theme analyst pass, and every "suspicious code" claim was then adversarially re-verified against current HEAD — findings below survived that check; claims that turned out wrong or deliberate are listed separately.

## TL;DR

A security-heavy week. Three credential systems were built or reworked (federation shared-key rotation, sealed secrets for codellm, personal access tokens replacing the fleet's JWT-secret dependency), the webhook system became observable and loop-safe (delivery chains, run logs, open costs), and search got its worst resource problem fixed (on-disk index, 35x heap reduction). Test discipline is visibly strong — most features landed with table-driven unit tests, and two of them (the goldmark-parity guard corpus, the cross-artifact Dockerfile pins) are better than the repo's usual bar.

Two things need attention before anything else:

1. **The krisp-ingest e2e is dead at HEAD** — `ceec53ec` accidentally reverted its direct parent `9d214ae9`, leaving the fleet id disagreeing three ways across role note, compose and spec. This also means the full e2e suite was not green when the week's later commits landed.
2. **A peer's refusal to rotate a federation key is misclassified as silence**, which records a key the peer never accepted; the refusal branch built to prevent exactly that is unreachable through the real client.

## The week by theme

### Federation shared-key rotation (#325 → #327)

Design doc plus implementation of one primitive — "replace the secret, authenticated by the current one" — with three callers: the install path (the handed-over key is consumed immediately, on by default with a UI switch), a manual admin Rotate, and a future scheduler (documented, not built). Storage stays on the single `federation_secrets` row (`prev_secret_crypt`, `rotated_at`); the previous key is accepted only within a 5-minute `RotationGrace` and retired eagerly via a compare-and-swap conditional clear. The rotate tool is dispatched but never advertised in `tools/list`, takes its pairing exclusively from the verified JWT (no confused deputy), and introduced request-body binding as a side effect: a `bh` SHA-256 body-digest claim in the federation JWT, verified for every call that carries it. The outbound client gets exactly one retry, keyed on a dedicated auth error code (-32001). Concurrency is handled with conditional UPDATEs rather than transactions because both mutations are `@skipTx` (a peer HTTP call inside SQLite's write tx would deadlock against a self-pointing peer). Backend test coverage is excellent (36+ new tests across four layers); the admin UI flow has none.

### Webhooks become pipelines (#307, #308, #310, #313, #314)

The centerpiece (#308, ~5.7k lines) adds delivery chains: both delivery tables gain parent/trace/depth columns, the delivery identity rides the request context so a chain is stamped without lookups, and #307 extends that to agent-response writes by synthesizing a delivery-scoped appreq inside the job worker (fixing the real bug: applied writes now flow through `HandleLatestNotesAfterSave` — frontmatter, SSE, downstream webhooks). `tokens_used`/`steps` were replaced by an open JSON `costs` column (`{unit: amount}`, summed per-unit in Go since GraphQL can't type an open map). Idempotent writes ride along: `insertnote` returns `NoteSaveResult` and the write paths distinguish "reload the snapshot" from "raise change events", so a vault resync re-pushing thousands of identical notes no longer re-fires webhooks at full LLM cost. #314 adds agent run logs (bounded at 500 entries / 64KB, opaque `data` bag stored unread, content never logged — pinned by test), #310 makes retention configurable (90d default), #313 gates private delivery addresses behind their own flag mirroring `MCPFederationAllowPrivate`. Admin gets a full delivery-chains UI plus a krisp multi-fleet demo stand.

### Fleet write validation (#315 → #317, #318, #321)

The enforcement half of `fleet_write_validation.md`: agents can no longer author role notes. A role is defined by the marker (`fleet_id` in frontmatter), not the path; `ScopedKB.Write`/`Patch` deny content that declares one, reading the current note through the underlying KB so a role's own read scope can't defeat the guard, failing closed with a distinct "unverifiable" error. #316 is the methodological highlight: the first guard shipped with four frontmatter bypasses because it was checked against a hand-written table; the fix mirrors goldmark-meta exactly (including switching to yaml.v2, goldmark's own YAML) and adds `TestDeclaresRoleMatchesGoldmark`, which diffs the guard against the real goldmark parser over an adversarial corpus. #317 closes the check-then-patch TOCTOU by sending the verified content's hash as `expectedHash` into the already-atomic `updateNotes`. #318/#321 are ergonomics on the same trust model: the role's own frontmatter is delivered in the input bag, and a role may narrow (never widen) which env vars its code receives.

### Codellm: from stub to runtime (#304, #305, #319, #322)

#304 rebuilds the runtime image (bookworm-slim, pinned python/node/CLI toolbox placed inside the landlock-visible `/usr` subtree, declarative env in `interpreters.json`, `/tmp/node_modules` symlink for ESM). #305 ships fleetkit — twin python/node helper modules with YAML-1.1-matched rendering, API surface pinned by symmetry tests because role notes live outside the repo. #319 is the security centerpiece: `sealed:v1:` AES-GCM blobs in role frontmatter, declared via `unseal`/`unseal_env_key`; the fleet forwards declarations but never holds a key; codellm decrypts in-process (never in the sandboxed child) and injects a `secrets` bag section kept out of `frontmatter` so debug dumps don't publish credentials. Guards: startup refuses a child-exposable `SEAL_KEY`, errors are deliberately uninformative (no env oracle), opened plaintexts are redacted from 422 previews. #322 adds the auth-gated sealing form at `/_system/codellm/seal` with careful secret hygiene (POST-only form values, no query strings, no echo, `no-store`).

### Search (#299, #320, #323)

#299: on-demand `regenerateNoteEmbeddings` with a precise `force` rationale (the note-level hash doesn't cover chunking), `section_url` heading anchors in MCP search results, client-side copyable heading anchors. #320 splits reranker `enabled` (capability) from `default` (policy) with a per-request tri-state `rerank` flag threaded to a single `ShouldRerank` gate — and documents a six-week silent production failure in a post-mortem. Silently breaking: deployments relying on `enabled=true` meaning always-on must now set `default`. #323 fixes the OOM: `SEARCH_INDEX_PATH` moves bleve to an on-disk scorch index (~35x heap reduction), made correct by stored content hashes plus a first-load reconcile pass (adopt matching, delete orphans), with two documented refusals (never delete an index that fails to open; never share a directory between loaders).

### Auth & tokens (#293, #294, #297, #298, #301, #303)

One theme: separating "a way in" from "the power to provision". Admin-minted HAT links can no longer create users or grant admin (#293); dead links render a localized human answer through a new `SystemMessageError` → standalone system page pipeline instead of leaking the internal error (#298); email validation stops resolving MX records (#294). #297 adds admin-minted personal access tokens (audit-logged, 365-day cap, token never logged) with a large routed $mol user-space rework. #301/#303 move the fleet off the monolith's JWT secret entirely: a `seedownertoken` boot reconciler maintains a reserved owner token row from `OWNER_PERSONAL_TOKEN_VALUE` with careful semantics (idempotent, revoked-in-UI-stays-revoked so the admin UI remains a kill switch), and memcli seeds it so an instance is fleet-capable out of the box.

### MCP hardening (#296, #324)

#296 rune-aligns the chunker's overlap window (no more U+FFFD-leading chunks in Cyrillic corpora) and collapses three parallel argument-struct families into shared `model.MCP*Params` types — previously any field one family lacked was silently dropped at the federated hop. The params types deliberately stay on `encoding/json` (readable field-level errors for self-correcting models; ~750 lines of easyjson codegen deleted). Single-KB federated hops gain per-hop timeouts via `callPeer`. #324 fixes an ACL routing hole: `accessibleKBNotes` bypassed `canReadMCPNote`, so API-keyed callers couldn't see gated KB-notes and every federated call answered `federation_not_configured` — masquerading as a routing bug.

### Metrics (#306)

Fleet and codellm each get a loopback-only internal listener (Prometheus + pprof + probes) with a private registry, nil-safe sinks, and carefully designed seams: coderun grows a typed `ExecError` kind taxonomy and an `Observe` callback instead of importing metrics; agentruntime declares its own minimal interfaces. Metric semantics carry deliberate operational decisions (ready-after-first-attempt, freshness advances on partial but never error, attacker-controllable tool names bucketed under `unknown`), all documented in `fleet_codellm_metrics.md` with alert recipes. ~1 400 of 3 200 lines are tests.

### Housekeeping

A restored note now counts as a change (`AffectsSnapshot` vs `Versioned` — but see finding 4), $mol root no longer claims the header row (with an e2e that reproduces the CSS race by disabling the component stylesheet at runtime), double date formatting removed from admin catalogs, the paywall lock sits on the baseline, mam/mol/node pinned at SHAs for reproducible builds, obsidian-sync bumped so an unreachable store stops looking like an empty one, and a batch of e2e hygiene — one item of which went wrong (finding 2).

## Suspicious code (verified against HEAD)

Every finding below was independently re-verified: the code exists at HEAD, the claim is accurate, and no doc or comment marks it deliberate.

| # | Sev | Where | What |
|---|-----|-------|------|
| 1 | high | `internal/case/admin/rotatefederationsecret/resolve.go:96` | Peer refusal classified as silence; `ErrPeerRefused` unreachable via real client |
| 2 | high | `docker-compose.test.yml:583` | Accidental revert of `9d214ae9`; krisp-ingest e2e dead at HEAD |
| 3 | med | `cmd/codellm/internal/codellm/server.go:283` | Unseal guard ignores `ExposeEnvPrefix` — seal key can reach a child while unsealing proceeds |
| 4 | med | `internal/case/backjob/deliverchangewebhook/resolve.go:399` | Agent-apply loops missed the `cc5a146e` restore fix |
| 5 | med | `internal/case/admin/createhatlink/resolve.go:57` | Same-site redirect guard bypassable with `/\evil.com` |
| 6 | med | `internal/noteloader/search.go:135` | Startup index cleanup will delete the directory a zero-downtime predecessor is serving from |
| 7 | med | `internal/graph/schema.graphqls` (`SearchInput.rerank`) | Anonymous visitors can trigger per-request cross-encoder runs |

### 1. Federation rotation: a refusing peer takes the silence path (high)

`Propose` maps any Go error from `RotateSecret` to `ErrPeerSilent` and reserves `ErrPeerRefused` for `result.IsError` — but the real client (`internal/federation/client.go:131`) converts every JSON-RPC error response (including `-32601 Method not found` from an older instance or an adapter, and the handler's own refusals) into a Go error, and no server path answers a refused rotation as an `IsError` tool result. So an admin pressing Rotate against a peer that cannot rotate records the never-accepted key as current. The pairing then survives only on the prev-key retry; worse, `staged()` considers the proposal staged only within the 5-minute grace, so a *later* retry mints a fresh key and records it over the staged one — dropping the only key the peer actually holds and killing the link in both directions. The unit test for the refusal branch passes only because its stub fabricates `FederationResult{IsError: true}`, a shape the real transport never produces. The install path is unaffected (both outcomes store nothing there). Fix direction: classify JSON-RPC *responses* (vs transport failures) as refusals in the client or in `Propose`.

### 2. The krisp-ingest e2e is dead (high)

`ceec53ec` ("guard screenshot target…") silently reverted its direct parent `9d214ae9`'s fleet-id alignment — its hunks are the byte-exact inverse; almost certainly a stale-tree mishap. At HEAD the chain disagrees three ways: role note `fleet_id: codellm`, compose `TRIP2G_FLEET_FLEET_ID=llmcode` (with a comment still saying `e2ec`), spec polling `fleetcron:e2ec:`. Discovery filters by exact id (`cmd/fleet/internal/fleet/discovery.go:44`), so the fleet never picks up the seeded role and the spec's 30 s poll can never succeed. The spec has no skip guard, which implies the e2e suite has not been green since 2026-08-20. Re-apply `9d214ae9`.

### 3. Sealed secrets: the prefix half of the exposure guard is missing (medium)

The design doc (`unseal_codellm_secrets.md`, "same predicate, two moments") promises that a seal key exposable to code children refuses to unseal. Startup checks prefixes — but only for the default `SEAL_KEY` name. At unseal time, `unsealBag(bag, osEnv{}, s.cfg.ExposeEnv)` never receives `ExposeEnvPrefix`, and `nameIsExposed` is exact-match only (its comment "prefixes are rejected at startup" is wrong for custom names). So `unseal_env_key: KRISP_SEAL_KEY` under `CODELLM_EXPOSE_ENV_PREFIX=KRISP_` forwards the key into every child's env while unsealing proceeds — the exact combination `errSealKeyExposed` exists to refuse, and the one case the otherwise-thorough unseal tests never exercise. Fix: pass the already-computed `effectiveEnvNames` (or the prefixes) into `unsealBag`.

### 4. Agent-apply restore never fires change events (medium)

`cc5a146e` fixed the restored-note gate (`Versioned()` → `AffectsSnapshot()`) in `pushnotes` and `updatenotes` but not in the two agent-apply copies of the same loop (`deliverchangewebhook/resolve.go:399`, `delivercronwebhook/resolve.go:419`). An agent response that restores a hidden note with identical content reloads the snapshot but skips `HandleLatestNotesAfterSave` — no frontmatter materialization, no SSE, no downstream webhooks — contradicting the loop's own comment that applied notes must "trigger change webhooks like any other write". This is the cost of the four-way duplicated apply/classify loop; worth unifying rather than patching a third time.

### 5. HAT redirect guard: backslash bypass (medium)

`(!HasPrefix "/") || HasPrefix "//"` admits `/\evil.com`; browsers treat `\` as `/` in special-scheme URLs (WHATWG relative-slash state), so `Location: /\evil.com` navigates off-site — precisely what the guard's comment says it blocks, and the value is written verbatim into the Location header at exchange time. Exposure is limited (only an admin mints the value, and it rides the signed token), but the standard fix — also reject a leading `/\` — is one line. Tests cover `//` and absolute URLs, not backslashes.

### 6. On-disk index cleanup vs zero-downtime handoff (medium)

`removeStaleIndexVersions` runs unconditionally at startup and `RemoveAll`s every sibling schema directory with no lock or liveness check. The same commit's own refusal logic exists because "a zero-downtime handoff has the old instance holding the lock" — yet on the first schema bump the new binary will delete the `v1` directory the old binary is actively serving from (on Linux `RemoveAll` succeeds despite open handles; the predecessor's segment writes then fail). Latent today (only v1 exists), armed the moment `searchIndexSchemaVersion` changes. Related, lower severity: `adoptPersistedIndex` skips hash-less documents from the orphan-deletion pass with a comment claiming "it will be re-indexed" — untrue when the note was deleted.

### 7. Anonymous rerank as a CPU-burn primitive (medium)

`SearchInput.rerank` is honored for unauthenticated site search (`sitesearch.Resolve` needs no auth), and on the newly recommended `enabled=true`/`default=false` posture each request can pin the cross-encoder sidecar (~1 s/candidate on CPU per the commit's own numbers) with no rate limit or actor gating; failures degrade silently. Previously `enabled` meant everyone paid it anyway, so this is a new exposure only for opt-in instances — but nothing addresses abuse. Consider gating the flag on an authenticated actor, or rate-limiting rerank requests.

### Lower-severity, confirmed

- `internal/case/mcp/federation_rotate.go:94` — the base-side rotate handler discards the rows-affected count of its guarded UPDATE and answers "rotated" on a lost race; its admin-side twin checks `affected == 0` and surfaces `ErrRotationInFlight`.
- `assets/ui/admin/federation/catalog/catalog.view.ts:68` — rendering the federation list makes one live peer HTTP round trip per outbound row (`row_subgraph_count` → `federationPeerScope`), each retried with the prev key on error; several unreachable peers stall the catalog for 2× the federation timeout per row, to draw a number.
- `internal/case/mcp/federation_handlers.go:62` — the "bound the single-KB federated hop" fix converted five tools to `callPeer`, but `handleFederatedSearch`'s explicit-`kb_id` branch stayed on the direct path; today the client's internal 2 s timeout masks it.
- `internal/model/pid.go:42` — `note_id` as a JSON number regresses one narrow mixed-version route (multi-hop `federated_note_html` through a pre-#296 intermediate); direct hops are fine and were the thing #296 fixed.
- `internal/case/admin/createhatlink` / `internal/appconfig/config.go:744` — `OWNER_PERSONAL_TOKEN_VALUE` accepts 8 chars *including* the `t2g_` prefix, and `DisplayPrefix` is `plaintext[:8]` — a minimum-length token's "masked" prefix in every token list is the entire credential.
- `internal/case/admin/revokeusertoken` — shipped with no tests (only case package in the group without them); the `ErrNoRows` mapping and audit call are unpinned.
- `cmd/codellm/main.go:155` vs `cmd/fleet/main.go` — the "mirrored" non-loopback metric-addr warnings diverge on unparseable addresses (fleet warns, codellm stays silent); neither binary ever shuts its metrics listener down, so fleet's `/readyz` answers 200 throughout and after the webhook drain.
- `cmd/fleet/internal/fleetmetrics/metrics.go:231` — the ~30-line internal-listener mux and the status-capturing ResponseWriter are copied three times across the two binaries; the warn-helper divergence above shows the drift is already real.
- `cmd/codellm/fleetkit/fleetkit.py:25` — `__all__` omits `parse_frontmatter` while the node twin exports it and the README documents it; the symmetry tests never check `__all__`. Also `node/index.js:74`: `frontmatter()` was inserted between `note_frontmatter`'s jsdoc and its function, stranding a throws-description above a function that never throws.
- `scripts/test-codellm-sandbox.sh:221` — the `status=$?`/docker-logs diagnostics after the check runner are dead code under `set -e` (and the EXIT trap removes the container first anyway).
- `cmd/fleet/internal/fleet/handler.go:233` — `buildInputBag`'s doc comment still claims the bag "carries only non-secret trigger data (the same fields exposed to Jet templates)"; it now carries the full role frontmatter, which the sealed-secrets design expects to hold credential blobs.
- `internal/appconfig/config_test.go:63` — the webhook-allow-private flag test hides as a subtest of `TestApplyStorageEnvFallbackSecretKey` (forced by process-global flags, but invisible and deletion-coupled).

### Checked and cleared

Claims that did not survive verification, for the record: the `envDeclaration` fail-open on malformed bags is documented, tested, and consistent with `unsealBag`'s written rationale (the trust boundary is the operator allowlist, which it never widens); the 1d→90d delivery-log retention jump is explicitly priced in PR #310's body; the hub-side advertisement of the peer-decided `rerank` flag is a documented compromise in `reranker.md`; the gqlgen scaffold block committed in #314 was already removed in #320.

## Testing posture

Strong overall: most substantive commits landed tests in the repo's table-driven style, and three are worth imitating — `TestDeclaresRoleMatchesGoldmark` (diff the guard against the real parser instead of a second hand-written table), the golden-value hash pins shared between `cmd/fleet` and `updatenotes` (algorithm drift breaks a test, not production), and `TestInterpretersJSON_ShippedEnv` (pins JSON paths against what the Dockerfile actually creates). Recurring gaps: admin $mol UI ships untested (consistent repo practice, but the federation install-and-rotate flow now has real failure modes); `admin/revokeusertoken` has no tests at all; the fleet-side unseal plumbing and the federated `rerank` forwarding rest on struct tags alone; and the e2e suite is not green (finding 2).

## Suggested follow-ups, in order

1. Re-apply `9d214ae9` (un-revert the krisp fleet id) — restores the e2e suite.
2. Fix refusal-vs-silence classification in federation rotation (finding 1) and check the rows-affected result in the base-side handler.
3. Pass prefixes into the unseal exposure guard (finding 3).
4. Unify the four write-classification loops so `AffectsSnapshot` semantics exist in one place (finding 4).
5. Reject `/\` in the HAT redirect guard (one line, finding 5).
6. Decide the zero-downtime story for on-disk index cleanup before the first schema bump (finding 6), and whether anonymous `rerank` needs gating (finding 7).
