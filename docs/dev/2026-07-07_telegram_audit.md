# Telegram Subsystem Audit (main @ cd3cbaf6, 2026-07-07)

## TL;DR

- **The legacy quiz is an MBTI personality test** living almost entirely in `internal/case/handletgupdate/quiz.go` (464 lines, untouched since Aug 2025). It hijacks the bot's `/start` whenever the vault contains `_mbti/<N>.md` notes, stores answers in the `quiz_states` JSON key of `tg_user_states.data`, and ends with a WebApp button linking to `<PublicURL>/mbti/<type>/index?client=tg` — that's the "take a test, get a link" feature. **No dedicated DB tables, no GraphQL fields, no admin UI** — excision is pure code deletion, no migration, no user confirmation needed.
- **Two high-severity bugs found**, both in `internal/tgbots`: (1) the webhook URL is built from a `token` field that is **never assigned**, so in prod webhook mode every bot registers the same unauthenticated path `/_system/tg/webhook/` — multi-bot updates collide and anyone who guesses the path can inject forged updates; (2) `StartTgBot` captures a request/transaction-scoped context and Env in long-lived goroutines, so bots started from the admin UI can die or use a dead transaction after the request ends.
- **Router proposal:** the codebase already has a two-level dispatch (per-user handler mode → per-update switches). The real problem is not the switches but that the "current handler" is stored in two different places (canvas in `tg_user_current_handlers`, navigation in `tg_user_states.data.handler`). Unify that first; add small command/callback registration tables inside `handletgupdate` when the next handler lands.

---

## 1. Legacy quiz feature: map + excision plan

### What it is

An MBTI test driven by vault content. Flow:

1. `/start` in a private chat with no deep-link args calls `req.Questions(ctx)` (`internal/case/handletgupdate/resolve.go:286`). If any note matches `_mbti/(\d+)\.md` (`quiz.go:52`), the user gets a "Начать тест" menu (`quiz.go:28-43`) **instead of** the content menu.
2. Callback `start_mbti` (`resolve.go:239`) starts sending questions; each question is a 7-emoji inline keyboard whose callback data is `mbti_answer:<qID>:<score>` (`quiz.go:197-220`).
3. `mbti_answer` callbacks (`resolve.go:249-250` → `quiz.go:113-178`) store answers in `req.userState.QuizStates["mbti"]`, persisted by the deferred `updateUserState` (`resolve.go:164-171`) into the `quiz_states` key of `tg_user_states.data` JSON.
4. When all questions are answered, `sendNextQuestion` computes the MBTI type (`quiz.go:324-419`), renders bar charts, and sends a WebApp button to `fmt.Sprintf("%s/mbti/%s/index?client=tg", req.env.PublicURL(), ...)` (`quiz.go:307-313`) — the "redirect to a link" part. The target is an ordinary vault note page.

### Last touched / test status

`git log` on `quiz.go`: last real change `03a82017` "mark answers" (2025-08-07); before that "disable rate limit" and "check question ids". Tests exist (`resolve_test.go:118-330` quiz flows, `resolve_test.go:480-570` state round-trip with `quiz_states`) but nothing has exercised it live in ~11 months.

### Reachability in prod

**Conditionally reachable.** The `start_mbti` / `mbti_answer` callback routes are always registered, and the `/start` gate is purely "does the vault contain `_mbti/N.md` notes" — via `LatestNoteViews()` (i.e. including **unpublished drafts**, see finding M4). On a vault without `_mbti` notes, `/start` falls through to `sendContentMenu` and the quiz is dead weight; on a vault that still has those notes, the quiz silently takes over `/start`.

### Dependency inventory

| Artifact | Location | Quiz-only? |
|---|---|---|
| `quiz.go` (all 12 functions + 3 types) | `internal/case/handletgupdate/quiz.go` | yes — incl. helpers `trimYAMLFrontMatter`, `reverseString`, `generateHorizontalBar` used nowhere else |
| Callback cases `start_mbti`, `mbti_answer` | `resolve.go:239-250` | yes |
| `/start` gating on `Questions()` | `resolve.go:286-295` | yes |
| `request.questions` field | `resolve.go:85` | yes |
| `UserStateData.QuizStates` + init sites | `resolve.go:67, 358-363, 375-377` | yes |
| `Env.PublicURL()` | `resolve.go:41` | becomes unused in this package after excision |
| DB | `quiz_states` JSON key inside `tg_user_states.data` — **no dedicated table, no schema column** | n/a |
| GraphQL / frontend admin | **none** (repo-wide grep for mbti/quiz: only this package, its tests, an unrelated string in `tgtd/convert_test.go:329`, two seed rows) | n/a |
| Vault content | `_mbti/*.md` question notes, `/mbti/<type>/index` result pages | user content, not code |
| Seed data | `testdata/e2e_seed.sql:304-305` (`{"quiz_states":{}}`) | cosmetic |

### Excision plan

1. **Delete** `internal/case/handletgupdate/quiz.go` entirely.
2. **Edit `resolve.go`:** drop the `start_mbti` and `mbti_answer` cases from `handleCallbackQuery`; in `handleCommands` `/start`, delete the `Questions()` check so it always ends in `return req.sendContentMenu(ctx)`; remove the `questions` field from `request`; remove `QuizStates` from `UserStateData` and its two nil-init sites.
3. **Shrink `Env`:** remove `PublicURL()` (`resolve.go:41`) and regenerate `mocks_test.go` with moq.
4. **Tests:** delete the quiz cases in `resolve_test.go`; keep (adapt) the `TestUserState` case proving unknown JSON keys in `tg_user_states.data` survive round-trips — that property protects nav/canvas state and currently rides on quiz fixtures (`resolve_test.go:511-527`).
5. **Comments:** update the two "e.g. quiz_states" comments in `handletgnavigationupdate/resolve.go:40,261` (the merge-unknown-keys behavior itself must stay).
6. **DB: nothing to drop.** Stale `quiz_states` keys inside existing `tg_user_states.data` rows become inert and are dropped naturally on the next state write. No migration → the "ask before dropping tables" rule isn't triggered. Optionally clean the two seed rows in `testdata/e2e_seed.sql`.
7. **Keep:** `sendContentMenu`, all of `access.go`, the whole `UserState` load/save machinery (nav mode delegation depends on it).

Behavior change to flag: on any instance whose vault still contains `_mbti` notes, `/start` will switch from the quiz menu to the content menu. The `/mbti/...` result pages keep rendering as normal notes.

## 2. Multi-router proposal

### Current dispatch (three layers, one inconsistency)

1. **Mode layer** — `cmd/server/telegram.go:225-257`: per-user "current handler" decides which use-case package gets the update. Canvas mode is selected here via the `tg_user_current_handlers` table (`resolveHandlerValue`, `telegram.go:598-623`); everything else falls to `handletgupdate`.
2. **…except navigation**, which is selected *inside* `handletgupdate` via a different storage: the `handler` key of `tg_user_states.data` JSON (`resolve.go:159-162`). Two mechanisms for the same concept.
3. **Update layer** — inside `handletgupdate.Resolve`: a fixed switch on update kind (MyChatMember / ChatMember / CallbackQuery / Command / Message, `resolve.go:174-195`), then string switches for commands (`resolve.go:262-342`), `/start` deep-link prefixes (`group_`, `wl_`, `attach_`, `browse_`; `canvas_` handled one layer up), and callback-data prefixes split on `:` (`resolve.go:236-257`).

### Proposal (proportional: one bot lane, not a framework)

**Step 1 — unify the mode layer (do regardless of routing).** Make `tg_user_current_handlers` the single source of the current handler; when nav mode is entered/exited, write `"navigation"` / `""` there instead of the JSON `handler` key, and dispatch navigation from `resolveHandlerValue` like canvas. The mode table in `cmd/server/telegram.go` then becomes an honest registry:

```go
type modeResolver func(ctx context.Context, env *tgHandlerEnv, update tgbotapi.Update, arg string) error

var tgModes = []struct {
    prefix string // matched against handler value; "" = default, last entry
    fn     modeResolver
}{
    {"canvas:", func(ctx, env, u, arg) error { return handletgcanvasupdate.Resolve(ctx, env, handletgcanvasupdate.Input{Update: u, CanvasPath: arg}) }},
    {"navigation", func(ctx, env, u, _) error { return handletgnavigationupdate.Resolve(ctx, env, u) }},
    {"", func(ctx, env, u, _) error { return handletgupdate.Resolve(ctx, env, u) }},
}
```

New handler packages (inbox agent, import wizard, …) register by appending one entry and shipping a use-case package with its own minimal `Env` — exactly the existing app pattern; `tgHandlerEnv` (`telegram.go:208-215`) already satisfies all of them by composition.

**Step 2 — registration tables inside `handletgupdate`** (adopt when the next command/callback lands; the current switch is not yet painful):

```go
// internal/case/handletgupdate/router.go
type cmdFn func(ctx context.Context, req *request, args string) error

var commands = map[string]cmdFn{
    "start": (*request).cmdStart, // runs startDeepLinks table, then content menu
    "browse": ..., "content": ..., "chats": ..., "user": ..., "id": ...,
}
var startDeepLinks = []struct{ prefix string; fn cmdFn }{
    {"group_", (*request).handleGroupAccess},
    {"wl_", (*request).handleWaitListRequest},
    {"attach_", (*request).handleAttachCode},
    {"browse_", ...},
}
// callback data convention already uniform: "name:arg1:arg2"
var callbacks = map[string]func(ctx context.Context, req *request, parts []string) error{
    "join_chat": (*request).handleJoinChat,
}
```

Constant function tables, not mutable state — consistent with the repo's route-registration style. The update-kind switch (`resolve.go:174-195`) should **stay a switch**: it enumerates five fixed Telegram concepts, not a growth axis, and the cross-cutting concerns around it (profile upsert, user-state load, deferred save) must keep wrapping every branch.

**Mapping:** quiz excision under this design = deleting two `callbacks` entries plus one `/start` gate — features become additive entries instead of edits interleaved in a 100-line switch. Trade-off: tables add one indirection and are slightly less greppable than a `case` literal; at today's size (6 commands, 3 callbacks) the benefit is marginal, which is why step 2 is deferred to the next feature while step 1 fixes an actual inconsistency now.

## 3. Telegram health findings (ranked)

### HIGH

**H1. Webhook token never set → shared, unauthenticated webhook path + multi-bot collision.**
`internal/tgbots/handler_io.go:19` declares `token string`; the only constructor `HandlerIO{bot: bot, dbBotID: id, logger: io.logger}` (`bots.go:138`) never sets it; `registerWebhook` builds the path as `fmt.Sprintf("%s/%s", io.webhookURL.Path, handlerIO.token)` (`bots.go:274`). In prod (https `PublicURL` → webhook mode, `bots.go:67-75`), every bot therefore registers `https://<host>/_system/tg/webhook/` with Telegram. Consequences: (a) with 2+ bots, `webhookMap[webhookPath]` (`bots.go:291`) is overwritten — all bots' updates are processed with the **last-registered bot's** `HandlerIO`/`BotID`, corrupting per-bot state and replying from the wrong bot; (b) the path is guessable with no `secret_token` verification, so anyone can POST forged updates through `ProcessWebhookRequest` (`bots.go:197-225`, wired at `cmd/server/routing.go:256`) — forged `chat_member` events insert membership rows (`access.go:238`), forged `my_chat_member` marks chats removed (`access.go:188`), forged messages probe `attach_` codes. Fix: assign `token` at construction, and preferably switch to `setWebhook` with `secret_token` + check `X-Telegram-Bot-Api-Secret-Token`.

**H2. `StartTgBot` leaks request-scoped context and tx-scoped Env into long-lived goroutines.**
`bots.go:114-146`: when called from the admin GraphQL mutations (`internal/case/admin/createtgbot/resolve.go:64`, `updatetgbot/resolve.go:54-56`), it swaps in the transaction Env from `appreq.FromCtx` (`bots.go:118-124`) and then launches `go io.checkBotPermissionsInAllChats(ctx, &handlerIO, env)` (`bots.go:141`) — that goroutine keeps issuing DB reads/writes through the **transaction-scoped** Env after the mutation's tx has closed. Worse, in polling mode the bot's whole lifetime context derives from the request context (`bots.go:165`), so a bot enabled via the admin UI stops polling when the HTTP request finishes — or races with fasthttp `RequestCtx` recycling, the exact hazard `tgtd.safeCtx` documents (`internal/tgtd/client.go:184-207`). Webhook mode returns before the polling goroutine (`bots.go:148-151`), so prod impact is limited to the permissions goroutine; dev/self-hosted http deployments hit the full failure. Fix: derive both goroutines from a server-lifetime context and always use the app-level Env for the background check.

### MEDIUM

**M1. Nil-pointer panic on `CallbackQuery.Message` crashes the process.** `resolve.go:138`, `quiz.go:185`, and `handletgnavigationupdate/resolve.go:311` dereference `CallbackQuery.Message`, which is `*Message` (tgbotapi v5 types.go:48) and nil for callbacks on inaccessible/inline messages. The polling loop calls the handler with no `recover` (`bots.go:183-192`), so one such update kills the whole monolith. Low likelihood today (no inline mode), total blast radius; add a nil guard + `recover` in the update loop.

**M2. Webhook updates processed under a 5-second budget with errors swallowed.** `ProcessWebhookRequest` wraps the handler in `context.WithTimeout(Background, 5s)` (`bots.go:214`) and returns 200 regardless. Flows with several Telegram round-trips — `getBotCanInviteWithRetry` alone can sleep 1.5s across three API calls (`access.go:34-54`), `sendContentMenu` generates an auth URL per subgraph — can blow the budget; the update is then half-applied, logged, and never redelivered by Telegram (we returned 200).

**M3. `io.handler` data race + startup update loss.** `tgbots.New` starts polling goroutines and registers webhooks (`bots.go:87-89`) before `SetHandler` is called (`cmd/server/telegram.go:225`); the goroutines read `io.handler` unsynchronized (`bots.go:184,217`) against the later write (`bots.go:95`). Updates arriving in that window hit the no-op debug handler and are dropped. Pass the handler into `New`, or guard the field.

**M4. Bot serves unpublished drafts.** `Env.LatestNoteViews()` carries its own TODO — "read LiveNoteViews for production users" (`resolve.go:42`). The content menu (`access.go:352`), quiz questions, and the navigation browser (`handletgnavigationupdate/resolve.go:28`) all read the latest (draft) vault view, exposing unpublished content to any Telegram user of the bot.

**M5. FLOOD_WAIT root cause still live.** The known dialogs issue (docs/dev/current_tasks.md bottom) has two remaining code paths: `ResolveTelegramChatUsernameViaAccount` calls `ListTelegramAccountDialogs(ctx, account.ID, 0)` — limit 0 = paginate **all** dialogs per candidate account (`cmd/server/telegram.go:521`); and `resolvePeerFromDialogs` does the same full pagination on every access-hash cache miss (`internal/tgtd/client.go:1075-1114`).

**M6. `uploadFromURL` uses `http.DefaultClient` with no timeout** (`client.go:695`). Outside fasthttp contexts `safeCtx` imposes no deadline (`client.go:206`), and the telegram queues run with `Limit: 1` (`cmd/server/telegram.go:30-43`) — one hung media download stalls the entire publishing lane.

**M7. `verifyOngoingGroupAccess` is security theater.** It unconditionally returns nil with a "placeholder, replace with actual verification" comment (`access.go:312-326`), yet `sendContentMenu` loops over subgraphs calling it as if it filtered (`access.go:339-350`). Users who left paid groups keep menu access. Either implement or delete the loop and the false comfort.

### LOW

- **L1. Lost updates on `tg_user_states`:** `UpdateCount` incremented but never used as an optimistic-lock predicate (`resolve.go:387-411`; same read-then-upsert in nav `saveState`, `handletgnavigationupdate/resolve.go:287-304`). Concurrent webhook deliveries silently clobber each other.
- **L2. `retryOnFloodWait` discards the final error**, returning a bare "max retries exceeded" (`client.go:101`) — diagnosis-hostile.
- **L3. Dead code / no rate limiting:** the entire `RateLimiter` (`ratelimit.go`, invocation commented out at `resolve.go:25-28, 90-109` since commit `3aa2b534` "disable rate limit") — the bot currently has **no** per-user rate limiting at all; `PendingAuth.err` is write-only (`auth.go:121`), so auth-client crashes are silently ignored.
- **L4. Minor race in `AuthManager.CompleteAuth`:** `pending.State`/`PasswordHint` mutated outside the mutex (`auth.go:201-202`) while `GetPendingAuth` returns the shared pointer under lock (`auth.go:267-271`).
- **L5. `resolveHandlerValue` swallows all DB errors** via `err == nil &&` chains (`telegram.go:604-620`) — a DB outage silently degrades everyone to the legacy handler.

Healthy areas for the record: `access.go` and the new backoff (#148) are well-tested; `tgtd/convert.go` (Telegram→Markdown import) is careful with UTF-16 offsets and has a substantial test file; the canvas handler follows the Env pattern cleanly and is the best-tested handler package.

## Suggested follow-up work items

1. **P0:** Fix webhook token/secret (H1) — small change, real security exposure; verify with two bots on a staging https instance.
2. **P0:** Fix `StartTgBot` context/Env capture (H2).
3. **P1:** Quiz excision per section 1 (pure deletion; one behavior change to confirm with the user).
4. **P1:** Nil-guard `CallbackQuery.Message` + `recover` in update loop (M1); raise/rework the 5s webhook budget and return non-200 on failure (M2); handler-into-`New` (M3).
5. **P1:** Ship the already-planned dialogs limit/search/cache task (M5) — include the `telegram.go:521` call site in its scope.
6. **P2:** Unify current-handler storage on `tg_user_current_handlers` (router step 1); adopt command/callback tables with the next new handler (step 2).
7. **P2:** Decide the fate of `verifyOngoingGroupAccess` (M7) and of rate limiting (L3 — re-enable with a per-bot limiter or delete `ratelimit.go`).
8. **P2:** `LatestNoteViews` → live views for end-user-facing bot surfaces (M4) — needs a product decision for dev instances.
