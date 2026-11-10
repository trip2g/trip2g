# Repository bug hunt — 2026-11-10

Reviewed revision: `0787a296`. Six parallel read-only reviews by area (note access, auth, sync and note loading, queues/Telegram/webhooks, fleet/codellm, payments), plus `go vet` and `go test -race`. This is a review, not a patch: no code was changed.

**Evidence levels.** *Reproduced*: a test or run showed the failure. *Checked*: a second read of the code confirmed the reviewer's claim. *Reported*: one reviewer's reading only, not re-checked.

## Tooling

- `go vet ./...` is clean.
- `go test -race ./internal/... ./cmd/...` fails in one place: `TestFederatedAssetAcrossTwoStores` (`cmd/server`). `fasthttp.Server.Shutdown` writes a field that `RequestCtx.Done()` reads from a `database/sql` goroutine, because the `RequestCtx` is passed to queries as their context. It shows at server shutdown. Neither the Makefile nor CI runs `-race`, so it went unseen.

## Findings by priority

| # | Problem | Evidence | Decision (2026-11-10) |
|---|---------|----------|-----------------------|
| 1 | Email sign-in code can be brute-forced | Checked | Fix; approach open |
| 2 | Magazine cards print the first paragraph of paid notes | Checked | Proposed: fix |
| 3 | `![[Paid note]]` in a free note serves the paid body | Checked | By design for now |
| 4 | Two concurrent note reloads can crash the process | Reproduced | Fix |
| 5 | Datachart `url` source bypasses `ssrfsafe` | Checked | Open: likely drop the `url` source |
| 6 | A removed admin stays admin until the JWT expires | Checked | Fix; approach open |
| 7 | Patreon and Boosty never revoke access | Checked | Issue filed; no fix while unused |
| 8 | A two-block codellm pipeline hangs forever | Reproduced | Fix |
| 9 | Fleet: a failed apply leaves earlier writes, then the hub retries | Reported | Open |

### 1. Email sign-in code can be brute-forced

`internal/case/signinbyemail/resolve.go:71`, `queries.read.sql:121` (`VerifySignInCode`).

Nothing counts wrong guesses. Turnstile guards only `requestEmailSignInCode`. Up to three codes stay valid for five minutes, and GraphQL aliases fit about 15 guesses into one request under the complexity limit. The reviewer's estimate is about 20k requests in five minutes for a ~63% chance against any account, admins included; the arithmetic was not re-run. `sign_in_codes` has no attempt column, and `internal/throttle` is a debounce helper, not a rate limiter, so there is nothing to reuse.

### 2. Magazine cards print the first paragraph of paid notes

`internal/defaulttemplate/magazine.go:26`, rendered at `views.html:404`.

`MagazineItems` lists every non-system note with no read check. A paid note without `description:` gets `IntroText()` as its card text. An anonymous visitor sees it on `/` of any vault with no index note, and on any free note with `content: magazine`; the page also enters the anonymous page cache. `rendernotepage/meta_description.go:25` already refuses this case for meta descriptions.

### 3. An embedded paid note is served in full inside a free note

`internal/mdloader/link_renderer.go:233` writes `note.HTML` unconditionally at load time; `views.html:188`, `:272`, `:277` do the same for `content: [[paid]]`.

The paid body then reaches anonymous renders, the page cache, MCP `note_html`/`expand`, RSS and search snippets. `2026-09-28_report.md` records this as an open policy question. **Treated as by design for now.**

### 4. Two concurrent note reloads can crash the process

`internal/noteloader/loader.go:218-436`, `internal/noteloader/search.go:186-260`.

`Load` holds no lock while it builds the next note set and search index. `noteWriteMu` (`cmd/server/notes.go:31`) is taken only by `pushNotes` and git. Webhook jobs (`deliverchangewebhook/resolve.go:413`, `delivercronwebhook/resolve.go:434`) and `ReloadChartDataNotes` (`cmd/server/chartdata.go:59`) reload without it.

- Unlocked writes to the `contentHashes` and `indexedPermalinks` maps: Go aborts the process on concurrent map writes.
- The reload that finishes last publishes, even if it read older data, so a fresh write can vanish until the next reload.

Reproduced with a `-race -overlay` test running two `Load` calls at once: races at `loader.go:237/238` and `search.go:200/203/240/241/300/335`.

### 5. Datachart `url` source bypasses `ssrfsafe`

`internal/case/backjob/refreshchartdata/resolve.go:78`.

The fetch uses `http.DefaultClient`: no `ssrfsafe`, redirects followed. Up to 5 MiB of JSON is cached and rendered into the page (`mdloader/chart_renderer.go:104`). Anyone who can write a note can point it at `169.254.169.254` or a loopback listener: a sync key, a fleet agent, a change-webhook response. Admin-defined webhooks do go through `ssrfsafe`.

### 6. A removed admin stays admin until the JWT expires

`internal/graph/helpers.go:40` (`checkAdmin`), `cmd/server/auth.go:80` (`CurrentAdminUserToken`).

Both trust the `r:"admin"` claim; neither reads the `admins` table. `deleteadmin` removes the row and touches no session. The token lives 30 days. `signOut` only clears the cookie, so the same JWT keeps working as a `Bearer`.

Related, *reported*:

- Personal tokens (`t2g_`) skip the ban check: `internal/personaltoken/resolver.go:40` has no ban lookup, and `banuser` revokes no tokens.
- MCP admin tools keep working after the API key's creator stops being an admin: `internal/case/mcp/endpoint.go:382` plus `appreq.WithAdminToken` set the role without the DB check at `auth.go:91`.

### 7. Patreon and Boosty never revoke access

- Patreon sync reads only the first page of members: `internal/case/refreshpatreondata/resolve.go:249` never follows `Links.Next`.
- Marking missed Patreon members is a TODO (`:278`); the upsert never clears `current_tier_id` (`queries.write.sql:464`).
- Boosty sets `missed_at`, but `ListActiveBoostySubgraphNamesByUserID` (`queries.read.sql:170`) never reads it.
- `MarkBoostyMembersAsMissed` (`queries.write.sql:563`) has no `credentials_id` filter, so syncing one blog marks every other blog's members.
- Soft-deleted credentials do not remove access (*reported*).
- Member emails are matched case-sensitively against lowercased `users.email` (`queries.read.sql:163,173`), so `John@x.com` pays and gets nothing (*reported*).

### 8. A two-block codellm pipeline hangs forever

`cmd/codellm/internal/coderun/runcode.go:418-510`.

Block 1 prints a lot; block 2 exits without reading stdin. Block 1's stdout copy blocks in `pw.Write` on the `io.Pipe`. `pipeCtx` is never used, so `pipeCancel()` kills nothing, and `cmd.Wait()` has no `WaitDelay`. Reproduced: with a 3 s block timeout `ExecCode` had not returned 12 s later, sandbox on or off. Each hub retry leaks another set of goroutines and a `fleet-coderun-*` workdir.

Related, *reproduced*: with the sandbox off, a backgrounded grandchild outlives the timeout (`coderun.go:276,296`, no `Setpgid`).

### 9. Fleet: a failed apply leaves earlier writes, then the hub retries

`cmd/fleet/internal/fleet/handler.go:438` sets `HardFailApply: true` for every role; the comment at `agentruntime/runtime.go:124` says it is left false for real-LLM roles.

`runToolCalls` applies each call as it reaches it, so `write A` lands before `patch B` fails. The run errors, fleet answers 502, and the hub re-delivers up to `retries` times. The user doc (`docs/en/user/fleet.md`, "Retries") already tells authors to set `retries: 0` on non-idempotent roles, so part of this is documented behaviour; what is not is that one mistyped `find` from a real model is fatal instead of self-corrected.

Related, *reported*:

- A failed run returns no log and no costs (`runtime.go:204`, `handler.go:176,195,348`), so the runs that need debugging leave nothing.
- `write_note` can overwrite an existing role note, which `patch_note` and `set_note_meta` refuse (`agentruntime/scope.go:160`).

## Medium

| Problem | Where | Evidence |
|---------|-------|----------|
| An `updateNotes` batch commits partially: `ErrorPayload` does not roll back, so change A is saved with no reload and no webhooks while the client is told the batch failed | `internal/graph/handler.go:223`, `updatenotes/resolve.go:145` | Checked |
| A rolled-back mutation stays live in memory: the reload reads through the open transaction, and a later find/replace can write the phantom text back | `updatenotes/resolve.go:170`, `cmd/server/note_loader_envs.go:250` | Reported |
| `ExpectedHash` and the find/replace base come from the in-memory snapshot, not the database | `updatenotes/resolve.go:216,252,266,329` | Reported |
| Hidden notes stay public with `show_draft_versions=false`: `AllLiveNotes` lacks `hidden_by is null` | `queries.read.sql:413` | Checked |
| `federated_instructions` serves its cache before the access check; the cache key is `kbID` only | `internal/case/mcp/federation_handlers.go:160` | Checked |
| Readers that ignore releases and use latest versions (known, issue #438) | `runnotequery/resolve.go:99`, `mcp/resolve.go:669`, `similarnotes/resolve.go:38` | Reported |
| Access expiry is compared in the site timezone against a UTC string | `cmd/server/config.go:134`, `queries.read.sql:138,341,350` | Reported |
| The read replica misses `makeReleaseLive`, a `require_signin` flip and frontmatter patches | `queries.read.sql:1682` | Reported |
| NOWPayments never revokes on `refunded`/`failed` after a successful status | `processnowpaymentsipn/resolve.go:52` | Checked |
| A Telegram post can be sent twice when the job stops between send and record | `sendtelegrammessage/resolve.go:163` | Reported |
| A 2xx webhook response over 1 MB counts as failed and is re-delivered with no delay | `internal/webhookutil/httpclient.go:76` | Reported |
| `ssrfsafe` does not block `::` | `internal/ssrfsafe/dialer.go:19` | Checked |
| `111111` is gated by the runtime `DEV` flag, not a build tag | `cmd/server/auth.go:236` | Reported |

The NOWPayments underpayment path from the review (`confirmed` followed by `partially_paid`) depends on an IPN order nobody verified; only the missing revoke is established.

## Low

- `createpaymentlink/resolve.go:82` panics for a signed-in account with no email.
- `signinbypurchasetoken` never succeeds: nothing sets `purchases.user_id`.
- A failed backup upload leaks a goroutine and keeps a database-sized deleted file open (`internal/simplebackup/backup.go:79`).
- Restore replaces the main DB file but leaves a stale `-wal`/`-shm` (`internal/simplebackup/restore.go:127`).
- Embedding calls have no timeout and can hold a global queue slot (`internal/openai/client.go:43`).
- Truncating a long Telegram post inside a tag drops everything after the tag opens (`internal/telegram/utils.go:174`).
- Google sign-in does not check `verified_email` (`handlegooglecallback/endpoint.go:106`).
- `LoadNoteViewByVersionID` takes `List[0]` without a length check (`cmd/server/notes.go:198`).
- A vault frontmatter-patch edit mutates the published snapshot's pre-patch frontmatter (`internal/mdloader/loader.go:669`).
- Duplicate IPNs can insert two access rows: count-then-insert with no unique index.
- Admin access grants write no audit log.
- `codellm/server.go:313` reads the request body with no size cap (behind auth).

## Checked and clean

- Page cache: anonymous only, keyed on path, host, version, config epoch and UI language; cleared on every reload.
- Asset serving, `sitesearch` permission filter, sitemap, RSS `Public()` filter, GraphQL `note`/`inLinks`/`noteChanges`.
- MCP: every fuzzy path match goes through `canReadMCPNote`; `size_tokens`/`sections` cannot panic on an empty note.
- JWT algorithm and token-type confusion, OAuth redirect handling, IDOR on user tokens, scoped-token write patterns, cookie flags, CORS.
- NOWPayments and Patreon webhook signatures; bot webhook secret.
- `GLOBAL_QUEUE_LIMIT` is validated `Min(1)`; cron jobs cannot overlap themselves.
- Search: the "half the query's terms" threshold, an empty query after stop-word removal, the vector timeout path.
- Fleet scope normalisation, metrics listeners and labels, LLM retry idempotency, token budget.
- The same-name image/note bug from `mdloader_same_name_bug.md` is fixed.

## Not covered

`movebus`, `frontmatteredit`, asset upload, path normalisation (NFC/NFD, case, trailing dots), `jsonneteval` resource bounds, `layoutloader`, route collisions with `/_system` and `/admin`, multilang, cyclic and deep embeds, `tgtd` and UTF-16 entity offsets, the Telegram response cache, `miniostorage`, `internal/image`, `internal/fleetinput`, fleet reconcile, each of the ~100 admin resolvers individually, `govulncheck`, a secrets scan of git history, and the $mol frontend.
