# View the site as a user

**Status: pending approval. Nothing here is built.**

- **Who builds:** a separate agent, after the owner has read and approved this
  plan. **Who reviews:** a third.
- **Read from source:** `trip2g/trip2g` `main` at `e0e339c9`. Every `file:line`
  below is from that commit.
- **Not measured live:** nothing. No code was run for this plan. Section 9
  lists what is unknown and the experiment that settles each point.
- **Decided by the owner on 2026-11-18:** a separate PR from `main`, read-only,
  an ordinary user as the target, a second cookie beside the admin's own
  session, a banner "you are viewing as X" with a button that returns the
  admin's rights. The banner is drawn by the default template only; a custom
  layout shows it by calling a method. The plan's author had recommended
  splicing it into every page instead, as the only exit a person can see.

## 1. Why

An admin cannot see what a reader sees. The padlock on a note link means "this
note is not free"; it says nothing about who can open it. A note with no
subgraph looks as locked as a note in a private subgraph, and it is readable by
every reader with any active subscription
(`internal/case/canreadnote/resolve.go:80-88`). The only way to check today is to
create a test user, grant it the same access and sign in as it.

The feature answers one question: **what does this particular reader get on
this site right now**, through the same code that serves the reader.

## 2. What exists and why it is not enough

The admin user page has a "Sign-in link" action (`admin.createHatLink`,
`internal/case/admin/createhatlink/resolve.go`). It mints a link that signs its
holder in as that email for 1 to 60 minutes. An admin can open it themselves.
It is a way to let a person in, and it makes a poor inspection tool:

- **It is a full session.** Purchases, form posts and personal tokens all work
  under it.
- **Views are recorded under the user**, so the inspection changes the reader's
  history and the analytics.
- **It replaces the admin's session** in that browser. Coming back means
  signing in again or keeping a second browser profile.
- **Nothing on the page says whose session it is.**
- **The audit log has the minting only.** What was done under the session
  cannot be told apart from the user's own actions.

## 3. Behaviour

1. On the admin page of a user, the admin presses **View as this user**.
2. The browser opens the site's front page. A bar across the top reads
   "You are viewing this site as maria@example.com. Read-only." with a
   **Back to admin** button.
3. Every page, search, attachment and note query answers as that user would
   get it. Nothing can be changed: every GraphQL mutation and every
   state-changing HTTP request is refused.
4. No page view is recorded and the reader does not appear to move on the
   admin graph.
5. **Back to admin** removes the second cookie and returns to the user's admin
   page. The admin's own session was never touched.
6. After an hour the second cookie expires and the admin is themselves again
   on the next request.

## 4. Design

### 4.1 The override cookie

- **Name:** the session cookie name plus `_as` (`trip2g_token_as` by default;
  the session name comes from `--user-token-cookie-name`).
- **Attributes:** the same as the session cookie (`internal/usertoken/token.go:172-209`):
  `HttpOnly`, `Path=/`, `SameSite=Lax`, `Secure` unless `--user-token-insecure`.
- **Value:** a JWT signed with the same secret as sessions, HS256, carrying
  `as` (target user id), `by` (admin user id), `use: "view-as"` and `exp`
  (one hour). Nothing secret is in the payload: two ids and a purpose.
- **`use` keeps the two token kinds apart:** the session parser refuses a token
  with a `use`, the override parser refuses one without it. Without this an
  override JWT pasted into the session cookie would be a session for user `i`
  with an empty role.

### 4.2 Where the override is applied

`appreq.Request.UserToken()` (`internal/appreq/request.go:203-272`) is where a
request gets its identity, and it caches the result for the request. The
override is applied there, on the cookie branch only (`:216`), before the
result is cached. It is not applied to `Authorization: Bearer`, personal
tokens or `?token=`: those are machine credentials and nobody presses a
button under them.

The override is honoured only when all of these hold:

- the session cookie resolved to an admin **after** the validators ran, so an
  admin whose role was removed loses the override at once
  (`cmd/server/auth.go:26-52` re-reads the admin row on every request);
- `by` equals that admin's id;
- the token is unexpired and carries `use: "view-as"`;
- the target user exists, is not an admin and is not banned.

Anything else: the override cookie is deleted and the request is served as the
admin. A stolen override cookie is worthless alone, and with the admin's
session cookie the thief already is the admin.

`appreq` has no database. It gets a small resolver the way it already gets
`PersonalTokenResolver` (`cmd/server/server.go:68-75`): an interface on the
request, implemented by the app, that turns the two cookies into the effective
`usertoken.Data`.

The effective identity is a **new** `usertoken.Data{ID: target, Role: "user"}`
with one unexported-from-JSON field, `ViewAsBy int`, and a method `IsViewAs()`.
It is never the admin's parsed struct with a field changed: the validator
mutates that struct in place (`cmd/server/auth.go:42`).

### 4.3 Read-only

One predicate, `appreq.DenyViewAs(ctx)`, asked in two places:

- **GraphQL.** In `makeAroundOperations` (`internal/graph/handler.go:173-247`),
  before the transaction gate at `:203`, so it also covers `@skipTx`
  mutations. A mutation is refused with "view-as is read-only". One exception:
  a mutation that is only `signOut` ends the view (it deletes the override
  cookie and leaves the admin's session), because the user-space widget shows
  a sign-out button and it has to do something sensible.
- **Everything else.** Non-GraphQL requests have no single dispatch point: they
  pass through a middleware list (`cmd/server/routing.go:191-244`), then
  `?hat=` handling, then the router. The check therefore sits in the request
  handler before the middlewares (`cmd/server/server.go:78`): a method other
  than GET, HEAD or OPTIONS from a view-as request answers 403, except the
  GraphQL prefixes (their own check above) and the stop endpoint.

Sign-in paths that work over GET stay open on purpose: `?hat=`, the HAT GET
link, OAuth and OIDC callbacks, the Telegram link, the purchase-token cookie.
Each replaces the session cookie with a real account. The next request then
fails the `by` check and the override is dropped. `SetupUserToken`
(`cmd/server/auth.go:162-192`) and `TokenManager.Delete` also delete the
override cookie, so a real sign-in or sign-out never leaves one behind.

### 4.4 No trace in the reader's history

`handleUserToken` (`internal/case/rendernotepage/resolve.go:361-421`) does two
things for a signed-in reader on every page: it records the view on a
background context (`:389-393`) and publishes the reader's move to the admin
graph (`:398-404`). Both are skipped for a view-as identity. The reads in the
same function (active subgraphs, last view) stay, because the page needs them
to render what the reader would see.

### 4.5 The page cache

Nothing to do. The cache stores and serves pages only for a nil token
(`internal/case/rendernotepage/pagecache.go:67-69`, `:126-128`); a view-as
identity is not nil.

### 4.6 The banner

Rendered by the server, with no JavaScript, as a form that posts to the stop
endpoint. The default template has one `<body>` for every mode
(`internal/defaulttemplate/views.html:84`), so a block right after it shows on
note pages, the paywall, the sign-in wall and 404.

Custom Jet layouts render their own HTML and do not get the banner by
themselves. A layout shows it by calling one method on the `defaultTemplate`
variable it already receives (`internal/case/rendernotepage/userspace.go:197-204`),
next to `Header` and `Footer`. The method returns the finished HTML, and an
empty string when no view is active, so a layout calls it without a condition.
The server does not splice anything into a layout's HTML. The federation guest
plan (PR #522) uses the same method for its guest bar.

An admin inside a layout that does not call the method has no reminder that
they are someone else. Nothing they do there can change anything, and the view
ends by itself after an hour.

The banner names the target by email, or by "user #id" when the user has none.
That costs one user lookup per request, only while a view is active.

### 4.7 Start, stop, audit

- **Start:** `admin.startViewAs(input: { userId })`, a use case under
  `internal/case/admin/startviewas/` shaped like `createhatlink`: admin gate,
  target checks, set the cookie, one audit line (`actorUserID`,
  `targetUserID`). No new table, so no migration.
- **Stop:** `POST /_system/view-as/stop`, a use case with an endpoint. It
  deletes the cookie, writes an audit line and redirects to the user's admin
  page in the admin. It is a plain HTTP endpoint so the banner works without
  JavaScript and in any layout.
- **Admin UI:** one more tool on the user page
  (`assets/ui/admin/user/show/show.view.tree:5-30`), wired like the
  "Sign-in link" button; on success the page navigates to `/`.

### 4.8 What the admin loses while viewing

While the override is present the session is the user's everywhere, not only
on reader pages:

- `/admin` and every `admin { ... }` GraphQL field refuse, as they do for that
  user.
- An admin session no longer stands in for an API key
  (`internal/case/checkapikey/resolve.go:40-47`), so a sync from the same
  browser fails instead of running as the admin.
- Fleet and codellm decide "is this an admin" by asking the monolith for
  `viewer { role }` with the browser's cookies
  (`internal/delegatedadmin/delegatedadmin.go:1-20`). They will see the user.

This is the simple rule, "you are X until you press Back to admin", and it is
the price of having one identity per request.

## 5. Files

| Area | Files |
|---|---|
| Token | `internal/usertoken/token.go` (override mint/parse/delete, `ViewAsBy`, `use` check) |
| Request | `internal/appreq/request.go` (resolver hook, `DenyViewAs`) |
| App | `cmd/server/auth.go` (resolver impl, cookie cleanup on sign-in and sign-out), `cmd/server/server.go` (non-GraphQL guard) |
| GraphQL | `internal/graph/schema.graphqls`, `internal/graph/handler.go`, generated files (`make gqlgen`) |
| Cases | `internal/case/admin/startviewas/`, `internal/case/stopviewas/` (+ `go generate ./internal/router/...`) |
| Reader page | `internal/case/rendernotepage/resolve.go` (skip view and move), banner |
| Template | `internal/defaulttemplate/views.html`, `langs/en.toml`, `langs/ru.toml`, `assets/defaulttemplate/src/index.scss` |
| Admin UI | `assets/ui/admin/user/show/`, a new `assets/ui/admin/user/viewas/` |
| Docs | `docs/en/user/` and `docs/ru/user/` page pair, a row in the docs map |

## 6. Options that lost

- **Open the existing sign-in link in a second browser profile.** Zero code.
  Loses on every point of section 2, and has no answer at all for a reader
  without an email.
- **Swap the admin's session for the user's and keep a "return ticket".** One
  identity cookie instead of two. Loses because the return ticket has to be
  able to restore an admin session, so it is an admin credential under another
  name, and coming back becomes a mint instead of a delete.
- **A `?view_as=<id>` parameter honoured for admins.** No cookie at all. Loses
  because links on the page and the `$mol` widgets' GraphQL calls do not carry
  the parameter: the page would be the user's and everything it loads would be
  the admin's.
- **An access report in the admin** ("this user can read these 214 notes, and
  here is why"). No impersonation, no session risk. It answers "what is open"
  better than clicking through pages does, and it loses only on "what does it
  look like". This plan's verdict is measured on the ground that favours it:
  the request was to look with the reader's eyes. For the padlock question
  alone the report may be the better tool, and it can be built later without
  touching this.
- **Full impersonation, writes included.** Lets an admin test a purchase or a
  form as the user. Refused by the owner for this PR: the admin's actions
  would be indistinguishable from the user's unless every write path learned
  to mark them.

- **Splicing the bar into every HTML page**, custom layouts included. One bar
  everywhere, and it was the plan author's recommendation, because the bar is
  the only way out a person can see. It lost on the owner's decision: the
  server would be editing HTML that a layout author wrote, and a layout with
  unusual markup could show it in an odd place. A method the layout calls
  leaves the author in charge of their page.

## 7. Decisions left to the owner

1. **Who may be a target.** The plan refuses admins (pointless, and it avoids
   an admin acting under another admin's name in the audit log) and banned
   users (a banned session answers 503 today, `internal/router/router.go:129-132`).
   Allowing a banned target would show exactly that broken page.
2. **Lifetime.** One hour, fixed. Shorter is safer against a forgotten tab;
   longer suits a long review.
3. **`signOut` while viewing.** The plan makes it end the view and keep the
   admin signed in. The alternative is to refuse it like every other mutation
   and leave the widget's button dead.
4. **Viewing as an anonymous visitor.** Not in this PR. It needs its own gate
   on the page cache, which treats every nil token as shareable
   (`pagecache.go:67`, `:126`).

## 8. Tests the build must bring

Written first, red before green.

- **In-process HTTP matrix** in `cmd/server`, on the helpers of
  `free_note_matrix_test.go:142-369`: an admin with an override gets the
  target's answer on page render, GraphQL `note`, site search, attachments,
  note queries and `viewer`; the same requests without the override answer as
  the admin.
- **Refusals:** an override alone (no session); an override with another
  admin's session; with a non-admin session; expired; with a target that is an
  admin, banned or missing; an override JWT in the session cookie and a session
  JWT in the override cookie. Each ends with the override cookie deleted.
- **Read-only:** every mutation refused, `signOut` ends the view, every
  non-GET endpoint in `internal/router/endpoints_gen.go` refused except stop.
- **No trace:** after a page render no row appears in `user_note_views` for
  the target and no reader move is published.
- **Losing the role:** remove the admin row mid-view; the next request is an
  ordinary user and the override is gone.
- **Use cases:** table tests with moq for `startviewas` and `stopviewas`.
- **Layouts:** a Jet layout that calls the method shows the banner during a
  view and nothing otherwise; one that does not call it shows none.
- **E2E:** one Playwright spec. The admin opens a user's page, presses the
  button, sees the banner, gets the closed page on a note the user cannot
  read, has a mutation refused, presses Back to admin and is an admin again.

Done is the scenario of section 3 walked by hand on a dev instance, with a
transcript and screenshots checked by an agent that has not seen the code.

## 9. What is not covered, and the experiment for each

- **The non-GraphQL guard may miss a write path.** The middleware list has
  paths with their own auth (git API, Telegram and payment webhooks, replica
  intake); they do not read the session, so they are not the reader's actions,
  but the list was read, not exercised. *Experiment:* a test that walks every
  registered endpoint and every middleware with a view-as request and a
  non-GET method, and fails when a new POST endpoint appears unclassified.
- **Order of the first `UserToken()` call.** The asset middleware resolves the
  identity before the router (`internal/case/serveasset/resolve.go:169`), and
  the result is cached per request. The override must already be applied by
  then. *Experiment:* an attachment of a note the target cannot read answers
  as it does for the target.
- **SSE subscriptions** copy the request with `Snapshot()`
  (`cmd/server/graphql.go:130`). *Experiment:* a note-changes subscription
  under view-as delivers the target's scope, not the admin's.
- **A sign-in in the middle of a view** (`?hat=`, purchase-token cookie,
  Telegram link) overwrites the session during the request
  (`cmd/server/server.go:90-108`, `routing.go:229-234`). The plan says the
  override is dropped on the next request; what the current request renders
  was not traced. *Experiment:* a test per path.
- **Fleet and codellm under a view.** Read from one file comment.
  *Experiment:* on a stand with fleet, open its UI during a view and note
  what it shows.
- **The MCP endpoint** builds an admin identity from an API key
  (`internal/case/mcp/endpoint.go:373-394`) and is POST, so the guard refuses
  it for a browser under view-as. Whether any browser-side feature relies on
  that POST was not checked. *Experiment:* grep the `$mol` sources for
  `/_system/mcp` before building.
- **A layout that does not call the method** leaves the admin with no visible
  way back. *Experiment:* during the hand-walked scenario, open a page with a
  custom layout under a view and note how a person gets out (the admin user
  page in another tab, or waiting an hour).
- **Expiry returns the admin's rights silently.** A tab left open for an hour
  shows a banner-less page on the next click with full admin rights. No fix is
  proposed; it is the direction of least surprise for the cookie, and the
  most surprising one for the person.
