# Agent-Runtime: 3-Way Implementation Decision (ours / Codex / Opus)

## 1. TL;DR Verdict

**Yes — ours (`feat/agent-runtime`) is the right base to ship.** Across all six seams ours is either the outright winner or the most-hardened variant, and it is the *only* implementation with tests (loop, jsonnet, janitor, scope, fleet). Ours wins the two security-critical seams outright (concurrency race-safety, jsonnet redaction) and ties on the loop's security axis. The competitors are unhardened single-pass implementations — but each made 2-3 genuinely better local design choices that ours should graft in.

**Do not rewrite. Cherry-pick.** Ours' weaknesses are localized correctness bugs and DRY gaps, not architectural mistakes. The highest-value work is *fixing ours' own bugs* (some of which we'd fix even without borrowing anyone's code), then grafting Opus's cleaner abstractions and Codex's reusable helpers.

**Ranked adopt list (value / effort):**

| Rank | Adopt | From | Effort | Why it matters |
|------|-------|------|--------|----------------|
| 1 | Cron attach_notes materialization (shared `MaterializeAttachedNotes`) | Codex | high | Fixes a real **functional gap**: cron roles get zero context notes today |
| 2 | RemoteKB overlay sync on Write/Patch | Opus | low | Fixes read-after-write **staleness** in a single fleet run |
| 3 | Reject ambiguous (non-unique) `find` in patch | Opus | low | Fixes silent **wrong-card patch** in FileKB |
| 4 | Move read-scope check into `canreadnote` chokepoint | Codex | medium | Closes the **similarNotes scope leak** for free |
| 5 | Explicit `WebhookScoped` flag for read+write enforcement | Opus | medium | Fixes read **fail-open** on empty `read_patterns` |
| 6 | Centralize concurrency_mode enum in `webhookutil` | Codex | low | Kills 5x **enum duplication** |
| 7 | Named `Kind` constants + `IsPatch()` + explicit write Kind | Opus | low | Removes magic strings, symmetric write/patch |
| 8 | Patcher optional interface (split out of core KB) | Opus | medium | Correct ISP, matches the named seam, frees test KB |
| 9 | Config gaps: ceilings>0, offered-tools non-empty, healthz, api_token 400 | Opus | low/med | Daemon hardening ours lacks |
| 10 | Canonical/compact EvalJSON output | Opus | low | Cleaner signed/sent/logged bytes |

---

## 2. Seam Winner Table

| Seam | Winner | What to adopt (from whom) | Effort |
|------|--------|---------------------------|--------|
| core-loop | mixed (ours' enforcement + Opus's KB design) | Patcher interface, uniqueness guard, Kind constants (Opus); defensive Kind validation (Codex) | low-med |
| jsonnet-transform | **ours** | Canonical compact output (Opus); empty-src fast path (Codex) | low |
| concurrency | **ours** | webhookutil concurrency centralization (Codex) | low |
| scope-enforcement | mixed (ours' coverage + Codex chokepoint + Opus flag) | canreadnote chokepoint (Codex); `WebhookScoped` flag + resolved-path match (Opus) | medium |
| fleet | **ours** | overlay sync, Config.Validate, healthz, transform passthrough (Opus) | low-high |
| attach-attrib-migrations | mixed (ours' concurrency + Codex attach helpers) | shared MaterializeAttachedNotes for both kinds (Codex); richer Meta allowlist (Opus); coalesce spend (Codex) | low-high |

---

## 3. Prioritized Adopt List

### Clear wins (pull into `feat/agent-runtime`)

1. **Cron attach_notes materialization** — *from Codex.* Extract `internal/webhookutil/attached_notes.go` with a single `MaterializeAttachedNotes(patterns, nvs)` + `AttachGateSatisfied`, and call the *same* function from both `deliverchangewebhook` and `delivercronwebhook`. Beats ours because ours materializes for change webhooks only — cron roles with `attach_notes` configured receive nothing despite the column/gate existing. Also de-duplicates ours' inlined materialize+apply logic. **Effort: high.**

2. **RemoteKB overlay sync on Write/Patch** — *from Opus* (`remotekb.go:107,120`; also Codex `remotekb.go:60,74`). Set `overlay[path]=content` on write; apply the single find→replace to `overlay[path]` on patch. Beats ours because ours' `RemoteKB.Write/Patch` never update the overlay, so a read after a same-run edit returns stale attached content. **Effort: low.**

3. **Reject ambiguous `find` in the patch path** — *from Opus* (`scope.go:103-104`; Codex `filekb.go:115-117`). Error when `find` occurs zero or >1 times; instruct the model in the system prompt to add surrounding context for uniqueness (Opus `runtime.go:82`). Beats ours because `FileKB.Patch` uses `strings.Replace(..., 1)` and silently patches the *first* of N matches — a wrong-card hazard for surgical kanban edits. **Effort: low.**

4. **canreadnote read-scope chokepoint** — *from Codex* (`case/canreadnote/resolve.go:24`). Inject the `read_patterns` check at the top of `canreadnote.Resolve`. Beats ours because ours duplicates the read check in 3 places (note resolver, notePaths filter, sitesearch) and *missed* `similarNotes`, letting a scoped agent read foreign note content. Keep ours' notePaths filter (it bypasses canreadnote) and ours' global `AroundOperations` stamp. **Effort: medium.**

5. **Explicit `WebhookScoped` flag** — *from Opus* (`appreq/request.go:355`). Add `req.WebhookScoped bool` + `Scoped()/ReadPatterns()/WritePatterns()` helpers set in every shortapitoken parse path; drive both read and write enforcement off `Scoped()`. Beats ours because ours keys reads on `len(rp)>0` (fail-**open** on empty patterns) and writes on `DeliveryKind!=""` — two inconsistent signals; the empty-`read_patterns` case currently reads everything. **Effort: medium.**

6. **Centralize concurrency_mode enum** — *from Codex* (`webhookutil/concurrency.go`). Constants + `NormalizeConcurrencyMode` + `ValidateConcurrencyMode`, used in the dispatch switch and all four CRUD resolvers. Beats ours' 5 independent copies of `{allow_overlap,skip,queue_one}` (drift risk). No behavior change to ours' superior atomic SQL guard. **Effort: low.**

7. **Named Kind constants + explicit write Kind** — *from Opus* (`agentresponse.go:13-14,40-42`). Replace bare `"patch"` magic strings with `AgentChangeKindWrite/Patch` + `IsPatch()`, and stamp an explicit Kind on `write_note`. Beats ours' empty-Kind-on-write asymmetry that leans on `empty==write` backward-compat. **Effort: low.**

8. **Coalesce delivery spend** — *from Codex.* `UpdateWebhookDeliveryResult` should set `tokens_used/steps` via `coalesce(sqlc.narg(...), tokens_used)` instead of plain assignment. Beats ours' NULL-clobber risk (safe only because spend is single-write today). **Effort: low.**

### Optional (quality, no behavior break)

9. **Patcher optional interface** — *from Opus* (`kb.go:36-38` + `scope.go:88-108`). Split `Patch` out of the core KB interface; `ScopedKB.Patch` delegates to `Patcher` when implemented, else a single-unique-occurrence RMW fallback. Correct ISP, literal match for the seam's named "KB/Patcher interface", and lets the test memKB drop its Patch boilerplate. Pairs naturally with adopt #3. Opus's `remotekb.go` already proves the payoff with a compile-time `_ agentruntime.Patcher` assertion. **Effort: medium.**

10. **Config.Validate() method** — *from Opus* (`config.go:40-77`). Move validation+normalization onto `fleet.Config`, adding `TokenCeiling/StepCeiling>0` and non-empty `OfferedTools` guards (ours' `main.go` validateConfig omits these — `-token-ceiling 0` yields `MaxTokens 0`). **Effort: medium.**

11. **healthz + api_token 400 guard** — *from Opus* (`fleet.go:83,119`; Codex `handler.go:68`). Standard daemon ergonomics; fail fast instead of an eventual 502. **Effort: low.**

12. **TransformJsonnet on Role** — *from Opus.* Add the field and thread it into `reconcile.create` instead of hardcoding `""`. Closes a real feature gap (roles can't configure an outbound transform). **Effort: medium.**

13. **Richer Meta allowlist** — *from Opus* (`AllowlistMeta`+`MetaTags`). Expose status/route/routes/lang (not just tags+layout), as `map[string]any` so non-string frontmatter survives. **Effort: medium.**

14. **Canonical compact EvalJSON output** — *from Opus* (`jsonneteval.go:45-56`). Round-trip through `json.Unmarshal`→`json.Marshal`. Beats ours' raw go-jsonnet output (indented + trailing newline) for the triple-duty signed/sent/logged body. **Effort: low.**

15. **Validate empty-src fast path** — *from Codex* (`jsonneteval.go:42-45`). Make `jsonneteval.Validate` foolproof regardless of caller guards. **Effort: low.**

16. **Resolved-path Note match** — *from Opus* (`schema.resolvers.go:3169`). Match the Note read check on `response.Note.Path` rather than ours' permalink→fsPath indirection that can return `""` and fall back to a URL-namespace path. **Effort: low.**

17. **Defensive Kind validation** — *from Codex* (`agentresponse.go:44`). `ozzo.In("write","patch")` in `AgentChange.Validate` to fail fast on garbage Kind. **Effort: low.**

### Skip

- **Opus's in-process `sync.Mutex` concurrency guard** — strictly worse than ours' atomic SQL conditional insert; breaks under multi-instance (LiteFS leader+replica), which the repo's read-replica branch makes a real concern.
- **Codex's constant marker version (`#v1`)** — security regression: the per-role HMAC secret never rotates on spec change. Ours' specVer-in-marker rotation is correct.
- **Codex's flat `-1 hour` stale window** — correctness regression vs ours' per-webhook `timeout_seconds+30s`: reaps long agent runs and blocks crashed fast webhooks for an hour.
- **Opus's denylist ExtVar redaction** — fail-open; keep ours' fail-closed allowlist. Same for Codex's denylist + hardcoded `meta:"{}"`.
- **Opus's whole-body `payload` ExtVar / `transformExtVars` dead error branch** — keep ours' scoped change/attached_notes/meta vars.

---

## 4. Bugs / Weaknesses in OURS (fix regardless of adoption)

Highest priority — these are correctness/security defects the comparison surfaced. Several should be fixed even if we borrow none of their code.

1. **[functional] Cron attach_notes silently dropped.** `cronWebhookPayload` (`delivercronwebhook/resolve.go:57-63`) has no `AttachedNotes` field and the cron deliver job never materializes — a cron role with `attach_notes` configured receives zero context notes though column/gate/CRUD all exist. *(Adopt #1)*

2. **[security] similarNotes is unscoped.** `case/similarnotes/resolve.go:137` calls `env.CanReadNote`, but ours never added scope to `canreadnote` and the read check lives only in the note resolver + sitesearch — a scoped agent can read content/similarity of notes outside its `read_patterns`. *(Adopt #4)*

3. **[security] Read enforcement fails OPEN on empty scope.** `schema.resolvers.go:3154` / `helpers.go:69` gate on `len(rp)>0`, so a scoped token with empty `read_patterns` reads everything — inconsistent with the fail-closed write path. *(Adopt #5)*

4. **[correctness] FileKB.Patch first-match replacement.** `filekb.go:111` uses `strings.Replace(content, find, replace, 1)` with no uniqueness guard — an ambiguous `find` patches the wrong (first) location with no signal to the model. *(Adopt #3)*

5. **[correctness] RemoteKB read-after-write staleness.** `Write/Patch` (`remotekb.go:105-115`) never update the `attach_notes` overlay; a read after a same-run edit returns stale content. *(Adopt #2)*

6. **[config] Missing ceiling/offered-tools guards.** `validateConfig` omits `TokenCeiling/StepCeiling>0` (`-token-ceiling 0` → `MaxTokens 0`) and non-empty `OfferedTools` (empty set silently disables all role tools). *(Adopt #10)*

7. **[robustness] Spend NULL-clobber.** `UpdateWebhookDeliveryResult` uses plain `tokens_used = ?`; a later status-only update would wipe recorded spend. Safe only because spend is single-write today. *(Adopt #8)*

8. **[leak, shared] resolveWikilinks unscoped.** Returns `target.Path`+URL with no `read_patterns` check — leaks existence/paths of out-of-scope notes. All three share this; ours should still fix it.

9. **[DRY] concurrency_mode enum in 5 places** (4 CRUD validators + inline dispatch literals). *(Adopt #6)*

10. **[design smell] Patch on the core KB interface** forces every backend (incl. trivial test memKB) into RMW boilerplate; spec names a "KB/Patcher interface". *(Adopt #9)*

11. **[design smell] Asymmetric / magic Kind.** `write_note` emits empty Kind while patch sets `"patch"`; bare magic string, no constant. *(Adopt #7)*

12. **[non-canonical] EvalJSON ships indented + trailing-newline JSON** as the signed/sent/logged body. *(Adopt #14)*

13. **[cosmetic] Dead defensive guards.** `where sqlc.arg(stale_window) is not null` (always true); `heartbeat_at` in the skip-guard coalesce but never written by `MarkWebhookDeliveryRunning`; `ExpireStaleWebhookDeliveries` vs `ExpireStaleCronWebhookDeliveries` name asymmetry. Either wire a real heartbeat or drop the column from the coalesce.

---

## 5. Honest Closing

**Ship ours as the base — it is already the strongest and the only tested/hardened tree.** Ours wins the two seams where a mistake is expensive and hard to reverse: concurrency (atomic SQL conditional insert is race-safe across goroutines *and* processes; Codex's flat stale window and Opus's process-local mutex both regress) and jsonnet redaction (fail-closed allowlist + F9 transform/pass_api_key/https guards the others entirely lack). On the core loop it ties Opus on security (execution-time allowlist) while keeping richer `Denials` observability. That foundation is not on offer from either competitor.

**How much better can cherry-picking make it?** Materially, but bounded — the gains are localized, not architectural:

- **Two fixes are non-negotiable before shipping**: cron attach_notes materialization (#1, a silent feature gap) and the similarNotes scope leak (#2/#3, a real read-authorization hole). Both are ours' own bugs, both have clean reference implementations to copy.
- **Three more are cheap, high-value correctness fixes**: overlay sync (#2-list), patch uniqueness guard (#3-list), spend coalesce (#8-list).
- **The rest (Patcher interface, Kind constants, enum centralization, Config.Validate, healthz, canonical output)** are quality/ergonomics that raise the floor and reduce drift, but none block shipping.

Net: cherry-picking the "clear wins" closes every correctness and security gap the comparison revealed and absorbs the competitors' best abstractions — while ours keeps the tests, multi-instance race-safety, fail-closed redaction, and observability the others never built. The right move is **ours as base + ~6 grafts (Opus's KB/overlay/config polish, Codex's attach+concurrency helpers)**, not a rebuild. Sequence the two security/functional fixes first; treat the abstraction cleanups as fast-follows.
