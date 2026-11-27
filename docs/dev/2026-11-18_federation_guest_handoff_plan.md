# Federation guest handoff: from the spike to a real version

**Status: approved by the owner on 2026-11-18, amended the same day after the
security review. Nothing in this plan is built.** The spike it grows from
exists and runs, on a branch that is not for merge.

- **Who builds:** a separate agent, after the owner has read and approved this
  plan. **Who reviews:** a third, with a security pass of its own.
- **Read from source:** the spike branch `spike/federation-guest-handoff` at
  `8149f0dd` (draft PR #518) and `main` at `e0e339c9`. A path marked *(spike)*
  exists only on the spike branch.
- **Measured live:** three local instances walked in Chromium, two hops,
  everything closed to anonymous readers; the Playwright spec
  `e2e/federation-guest.spec.js` *(spike)*, 7 tests, green inside the full
  parallel suite; the "private by default" frontmatter patch applied on one
  instance and checked as a guest.
- **Not measured:** listed in section 9, each with its experiment.
- **Amended after the security review** of the spike (a separate agent, the
  spike at `8149f0dd`, against the live stand): sections 4.5, 4.10 and 4.11
  and the rows marked *review* in section 8.1. The owner delegated two
  choices to this plan's author ("take the optimal options, I do not see all
  the consequences yet"); section 3 marks them, and both were made toward the
  smaller and reversible side.

## 1. The problem

A hub's agents can already search a neighbouring base over a federation edge.
A person cannot: the KB-note on the hub is a page with nothing to press, and
the neighbour does not know who they are. The spike closed that gap. A reader
signed in on the hub presses a button on the KB-note and is let into the
neighbour as a read-only guest, scoped to what the edge was granted.
`docs/dev/federation_guest_handoff.md` *(spike)* describes the mechanism in
full; this plan does not repeat it.

## 2. What the spike already settles

- **The handoff works in a browser:** `POST hub/handoff 200 → POST peer/guest
  303 → GET peer/guest 302`, and one hop further the same way.
- **A guest reads what the edge grants and is refused the rest**, on page
  render, GraphQL, search, attachments, note queries and similar notes
  (`TestFederationGuestAccessMatrix` *(spike)*).
- **A guest cannot write:** every mutation but `signOut` and every non-GET
  request is refused; MCP answers 403.
- **A form posted from another site is refused, and the same form from the
  hub's own site is accepted** (signed origin against the browser's `Origin`).
- **A ticket opens only in the browser that posted the form** (nonce cookie).
- **The reader's own session on the neighbour is left alone.**
- **Revoking the edge's key ends every guest session on the next request.**
- **A closed page tells a guest it is not shared over federation** and that
  access is requested separately; no sign-in or purchase widget.

## 3. Decisions already made by the owner

| Date | Decision |
|---|---|
| 2026-11-18 | The closed page for a guest carries text only, no request button. |
| 2026-11-18 | The KB-note button is an outline button in the site's colours. |
| 2026-11-18 | **A guest reads what an agent on the same edge reads.** Notes with no subgraph stay readable to any edge that grants at least one subgraph. An owner who wants otherwise applies the existing "private by default" patch. A stricter rule for guests would be a second access rule beside the first. |
| 2026-11-18 | **A guest session lives 12 hours**, not the 30 days of an ordinary session. |
| 2026-11-18 | **An agent's link to a neighbour's page goes through the reader's own hub**, which signs the handoff with the page as its target. |
| 2026-11-18 | **The two migrations of section 5 are confirmed** as written. |
| 2026-11-18 | **No count of untagged notes in the admin** when people are let in over an edge. The documentation carries that warning. |
| 2026-11-18 | **Results from a second hop get no open link in this version.** An agent that found a page two bases away reports it. With one hop only (below) the reader cannot follow it there in this version, unless their own base is paired with that one. |
| 2026-11-18 | **One hop only in this version** (section 4.10). *Chosen by the plan's author on the owner's delegation.* The other options were to sign the route into the token, or to let an owner open onward hops per edge; both keep chains and both need a design first. Reversible: a later plan opens chains. |
| 2026-11-18 | **No site address in the KB-note** (section 4.5). *Chosen by the plan's author on the owner's delegation.* The other option was to put the override on the key now, with a third migration. Reversible: a later change adds it there. |
| 2026-11-18 | **The guest bar is drawn by the default template only.** A custom layout shows it by calling a method. The server does not splice it into a layout's HTML. |

## 4. What the real version adds

### 4.1 A table for guests

The spike keeps the pair "edge, reader" as a string in `users.created_via`,
finds it by scanning, and two first handoffs at once can create two rows. The
real version has a table with a unique key (section 5). A guest is then "a
user with a row in `federation_guests`", and the admin user list, counts and
exports leave those users out by that join. No marker column on `users`.

### 4.2 Accepting people is a setting of the edge

The spike has one flag for the whole instance, `--federation-accept-guests`
*(spike, `internal/appconfig/config.go`)*, and turning it needs a restart. The
real version has a checkbox on the inbound key in the admin, off by default,
stored on the key's row (section 5). The flag is removed. One neighbour can
then accept people from Sales and not from Support.

### 4.3 Twelve hours

A guest session gets its own lifetime, a constant, not a flag. The session
cookie and the JWT both expire 12 hours after the handoff and are not renewed
by activity. A reader removed from the hub therefore stops being a guest
within 12 hours instead of 30 days. Revoking the edge still ends sessions at
once.

### 4.4 A page as the target, and a link an agent can give

The handoff token already carries the path the neighbour opens; the hub signs
`/` every time (`targetPath` in `internal/case/federationhandoff/resolve.go`
*(spike)*). Two additions use that one field:

- **A landing page in the KB-note.** A new optional frontmatter key,
  `mcp_federation_kb_path`, names the page a guest lands on. Without it the
  target stays `/`. This closes the worst hole of the spike: a guest whose
  first page is the neighbour's front page, closed to their edge.
- **An "open" link on the hub.**
  `GET /_system/federation/open?note=<KB-note path>&path=<page on the neighbour>`
  answers with a small page: the neighbour's name and a button. The button
  posts to the existing handoff endpoint with that path. A reader who is not
  signed in gets the hub's sign-in first.

The path is checked like the redirect of a sign-in link is checked today
(`internal/case/admin/createhatlink/resolve.go:55-60`): a path on the site,
one leading slash, never `//host`.

The federated search and note tools add such a link to every result that came
from a direct neighbour, so an agent passes it on without building anything.
Results that came through a second hop get no link in this version, and a
reader cannot follow them there (section 4.10).

### 4.5 The neighbour's site address comes from the key, never from the note

The hub sends a reader, and the signed token with them, to one address only:
the site of the MCP address its outbound key is bound to. The spike builds
that address from the KB-note being opened and picks the key separately, by
the note's `kb_id`; the two are never compared. The real version refuses to
sign for a KB-note whose `mcp_federation_kb_url` is not the key's own address.
A note is something any writer on the hub can push; a key is something only
an admin creates.

The first draft of this plan added a frontmatter key,
`mcp_federation_site_url`, to override the derived address. It is withdrawn:
it would let a note name the destination again. A neighbour whose MCP address
is not the address browsers use (an internal address, a separate site domain)
cannot accept people in this version. Agents reach it as before. The override
belongs on the key, in the admin, and is a later change with a migration of
its own.

### 4.6 A bar that says where the reader is

On every page of the default template a guest sees one line: "You are a guest
from sales.example.com · Back to sales.example.com · Leave". The hub's origin
is already signed into the handoff; the real version keeps it in the guest
session beside the edge's key id.

The default template renders the bar itself. A custom Jet layout gets it by
calling one method on the `defaultTemplate` variable it already receives
(`internal/case/rendernotepage/userspace.go:197-204`), next to `Header` and
`Footer`. The method returns the finished HTML, and an empty string for a
reader who is not a guest, so a layout calls it without a condition. The
"view as a user" plan (PR #521) uses the same method for its banner: one call
in a layout shows whichever applies.

A guest's page is never stored in the page cache, because the cache holds
pages for a nil session only
(`internal/case/rendernotepage/pagecache.go:67-69`), so a layout that calls
the method needs no new cache rule.

### 4.7 Dead ends

- **An expired or revoked guest session** today shows the ordinary sign-in
  wall to a person who has no account here. A guest cookie that fails its
  check leaves a one-request marker, and the wall adds a line: "You were here
  as a guest. Open this base again from your own knowledge base."
- **The refusal page** ("This knowledge base didn't let you in") gets a link
  back to the hub's origin when the token's signature was valid, and no link
  otherwise.

### 4.8 Documentation

No new page for the access nuance: it exists.

- `docs/{en,ru}/user/subgraphs.md` already has "Make the whole vault private
  by default" with the patch. Confirmed on a stand on 2026-11-18: after the
  patch a guest gets 403 on a note with no subgraph and still reads the
  granted ones.
- `docs/{en,ru}/user/federation.md` does not mention it where the owner grants
  subgraphs to a key. One sentence and a link go there.
- **Subgraphs that only ask visitors to sign in** stay readable to every edge
  that grants anything, and no patch closes that. None of `subgraphs.md`,
  `access.md`, `org-knowledge-base.md` or `federation.md` says so. It goes
  into `federation.md` beside the sentence above.
- A new section in `federation.md` for people: what each owner switches on,
  what a guest sees, the 12 hours, the landing page key.

### 4.9 Review and rebase

The spike's code was written by one agent and reviewed by none. Before any of
the above, a security review of the handoff, the guest session and the two
guards reads the spike as it is. The branch is rebased on `main`; one of the
spike's own findings, the response handler that never ends on the SSE
transport, is already fixed on `main` (#519) and will conflict in
`internal/graph/handler.go`.

### 4.10 One hop

A guest cannot go further. A base refuses to sign a handoff for a reader who
is a guest here, and refuses to accept a token whose hop count is above 1.
The hop count stays in the token, so that a later version can open chains
without changing its format.

The spike allows chains up to the federation depth limit, and the review
showed what that gives a reader: the union of every grant reachable within
the limit. A chain can come back to a base it passed through, over a
different edge with a wider grant. Opening chains safely needs either the
route signed into the token or an owner's consent per edge, and each is a
design of its own.

### 4.11 What the review asks of the session, the nonce and the ticket

- **A guest session belongs to one key, not to a name.** It carries the id of
  the inbound key it was opened under, and every request compares it. A key
  created later under the same `kid` does not revive a session opened under
  the revoked one. Rotation keeps the row and so keeps the session. Creating
  an inbound key under a `kid` that has existed is refused.
- **The nonce cookie is host-only.** Its name takes the `__Host-` prefix
  whenever the cookie is `Secure`, so a page on a sibling subdomain cannot set
  it. Note pages send `frame-ancestors 'self'`.
- **The ticket is opaque.** A random id in the redirect, with what it stands
  for kept on the server beside the spent ids. Today the ticket is a readable
  token that carries the reader's hash through access logs and browser
  history.

## 5. Migrations

Two. The owner confirmed the SQL on 2026-11-18; neither file exists yet.

```sql
-- migrate:up
create table federation_guests (
  id integer primary key autoincrement,
  kid text not null,
  reader_hash text not null,
  user_id integer not null references users(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  unique (kid, reader_hash)
);

-- migrate:down
drop table federation_guests;
```

```sql
-- migrate:up
alter table federation_secrets add column accept_guests boolean not null default false;

-- migrate:down
alter table federation_secrets drop column accept_guests;
```

No backfill: the spike was never deployed, so no instance has guest rows in
`users.created_via`.

## 6. Options that lost

- **A count of notes with no subgraph under the "accept people" checkbox.** It
  would tell an owner what the first grant brings with it at the moment they
  switch people on. The owner called it unnecessary: the rule is not new, the
  subgraphs page says it, and PR #524 adds it to the federation page.
- **A chained open link**, one link that walks a reader through two handoffs
  to a page two bases away. It would let a person reach everything an agent
  found. It lost for this version: the whole route has to be signed, every
  base in the middle has to agree to pass the reader on, and that is new
  ground for a mistake in security. Results from a second hop are the rarer
  case.
- **Chains in this version**, guarded by a signed route or by a per-edge
  "pass guests on" setting. Either would keep the second hop the spike
  demonstrated. They lost to "one hop" on the owner's delegation: each is a
  new mechanism whose consequences nobody has walked through yet, and the
  first version is safer without it.
- **A site address in the KB-note** (`mcp_federation_site_url`), this plan's
  own first draft. It lost to the review: the destination of a signed token
  must not be something a note author chooses.
- **A stricter access rule for guests** (free notes and granted subgraphs
  only). It was this plan's author's recommendation: a person follows links
  further than an agent searches, and an owner expects to have shared only
  what they granted. The owner refused it: one rule for a subscriber, an agent
  and a guest is worth more than a tighter default, and the patch already
  gives the tighter behaviour to those who want it. The cost that remains is
  section 4.8's third point, which the patch does not cover.
- **Replay state in the database.** Spent token and ticket ids live in memory
  (`federatedassetlink.Spent`, the same store `main` uses for federated asset
  links). A table would survive a restart. It lost because the two things it
  protects are already bound elsewhere: a ticket opens only with the nonce
  cookie of the browser that posted the form, and a form is accepted only from
  the hub's own origin. Replay inside 30 seconds after a restart would let a
  reader repeat their own handoff. Announced to the owner as a third migration
  on 2026-11-18 and withdrawn here.
- **A marker column on `users` for synthetic users.** The join with
  `federation_guests` says the same thing without a second source of truth.
- **A capability field** in the neighbour's description, so the hub shows the
  button only when the neighbour accepts people. It needs the hub to store
  what it learned per outbound key, a third migration, and a refresh. The
  refusal page with a way back (section 4.7) covers the case. Later, if
  refusals turn out to be common.
- **A direct link to the neighbour that sends an anonymous visitor to their
  hub and back.** It would make bookmarks and forwarded links work. It lost
  for now: the neighbour would have to show every visitor the list of bases it
  is paired with, and learn each hub's site address.
- **Access per reader instead of per edge.** The hub would send the reader's
  own subgraphs and the neighbour would intersect them with the grant. Two
  instances would have to agree on subgraph names. The gate that exists is
  enough for a first version: who can read the KB-note on the hub.
- **An "open" link that hands off at once**, with no button. One click fewer.
  It lost because the spike made the handoff a form post on purpose: a link on
  any other site could otherwise walk a reader's browser into a guest session.

- **Splicing the bar into every HTML page**, custom layouts included. One bar
  everywhere, and it was the plan author's recommendation, because the bar is
  the only way out a person can see. It lost on the owner's decision: the
  server would be editing HTML that a layout author wrote, and a layout with
  unusual markup could show it in an odd place. A method the layout calls
  leaves the author in charge of their page.

## 7. Decisions left to the owner

None. Every question this plan raised was answered on 2026-11-18; section 3
has the answers. What remains is the approval of the plan as a whole.

## 8. Tests the build must bring

The spike's tests move with the code: the in-process tests in `cmd/server`
*(spike)*, the access matrix and the Playwright spec. Written first for what
is new:

- **Guests table:** two first handoffs for one reader at once leave one row;
  a guest does not appear in the admin user list or its count.
- **Per-edge setting:** with it off for one edge and on for another, the first
  is refused and the second admitted; switching it off ends that edge's guest
  sessions on the next request and leaves the other edge's alone.
- **Twelve hours:** a guest session minted 12 hours and one second ago is
  anonymous; an ordinary session of the same age is not.
- **Landing page:** a KB-note with `mcp_federation_kb_path` lands on that
  page; without it, on `/`.
- **Open link:** lands on the page after the button; refuses `//host`, an
  absolute URL and a path with no leading slash; asks an anonymous reader to
  sign in and then continues; a reader who cannot read the KB-note is refused.
- **Tool results:** a federated search result from a direct neighbour carries
  an open link that the test then follows in a browser.
- **Bar:** names the hub, "Back" leads to the hub's origin, "Leave" ends the
  guest session. A Jet layout that
  calls the method shows the same bar for a guest and nothing for anyone else;
  a layout that does not call it shows none.
- **Dead ends:** an expired guest cookie gets the added line once; the refusal
  page links back only for a validly signed token.
- **The documented patch:** with it, a guest gets 403 on a note with no
  subgraph. This pins the recipe the docs point to.


### 8.1 Security tests

The owner asked for these by name on 2026-11-18: end-to-end tests that try to
break in. Each row is an attack with the answer it must get. "Browser" means
the Playwright spec against three real instances; "in-process" means a Go test
in `cmd/server` that sends real HTTP requests to an app built in the test.
A browser test is used where the browser itself is part of the wall (cookies,
`Origin`, frames); everything else is cheaper and stricter in-process.

**Getting in without the right**

| Attack | Must answer | Level | In the spike |
|---|---|---|---|
| *review:* open a KB-note that carries a real edge's `kb_id` and another address | no token is signed; the page says the note does not match its key | in-process and browser | no |
| *review:* present a guest cookie opened under a key that was revoked and re-created with the same `kid` | anonymous, cookie cleared | in-process | no |
| *review:* create an inbound key under a `kid` that has existed | refused | in-process | no |
| *review:* a guest presses a KB-note button | refused by the base they are a guest on | browser | the spike allows it |
| *review:* a token with hop 2 | 403 | in-process | the spike allows it |
| *review:* set the nonce cookie for the parent domain from a sibling subdomain, then open a ticket | the host-only cookie is not overwritten; 403 | browser, on the deployed pair of section 9 | no |
| Post a reader's own handoff form from another site | 403, no cookie; the same form from the hub's site is accepted | browser | yes |
| Post the same form twice | second post 403 | browser | in-process only |
| Post a form older than 30 seconds | 403 | in-process, with the clock moved | yes |
| Change the path, the origin, the hop or the reader in a signed token | 403 | in-process | a wrong signature, a wrong origin and a tampered ticket; not each field in turn |
| Open a copied ticket link in another browser profile | 403, and the first browser's ticket still works once | browser | by hand and in-process |
| Open a ticket twice | second 403 | browser | by hand and in-process |
| Use a handoff token as an MCP bearer token, and an MCP token as a handoff | refused both ways | in-process | yes |
| Present a guest cookie whose edge does not exist here (two instances sharing a JWT secret by mistake) | anonymous, cookie cleared | in-process | no |
| Hand off over an edge whose "accept people" is off | 403 | browser | flag only |
| Frame the hub's KB-note in a page on another site, or on a sibling subdomain, and get the click | the page refuses to be framed (`frame-ancestors 'self'`). The first draft relied on `SameSite=Lax` alone, which does not hold between subdomains of one site | browser | no |
| Follow an "open" link from another site without pressing anything | no guest session is opened; the page waits for the button | browser | not built |

**Going where the link should not lead**

| Attack | Must answer | Level | In the spike |
|---|---|---|---|
| `path=//evil.example`, `path=https://evil.example`, `path=/\evil.example`, a path with a line break | refused before signing, on the hub; refused again on the neighbour | in-process | the neighbour refuses an absolute and a `//` target; the hub side is not built |
| A neighbour's name or origin with markup in it, shown in the bar | escaped | in-process | not built |

**Doing more than reading**

| Attack | Must answer | Level | In the spike |
|---|---|---|---|
| Every mutation in the schema, sent by a guest | refused with the guest error, before the mutation's own validation | in-process, **the list is read from the schema**, so a mutation added later is covered without anyone remembering | a sample only |
| Every registered non-GET endpoint, sent by a guest | 403, except the handoff and guest endpoints | in-process, the list is read from the router | a sample only |
| Any `admin { ... }` query or mutation | refused | in-process | one admin mutation |
| Mint a personal token, an API key or a sign-in link | refused | in-process | not checked one by one |
| Reach MCP with the guest's browser session | 403 | browser | yes |
| Sign in to the guest's synthetic user by email code, HAT link, OAuth or Telegram | no path reaches a user with no email and no Telegram id | in-process | argued in the spike's document; one test shows a reader id equal to a local email gets a row of its own |

**Reading more than was granted**

One test seeds a note in a subgraph the edge was not granted, with a marker
string in its body, its attachment and its frontmatter, and then asks for it
every way a reader can. The marker must appear in none of the answers.

| Read path | Level | In the spike |
|---|---|---|
| The page, and the same page with `?version=latest` | in-process | page only |
| GraphQL `note`, `noteQuery`, similar notes, backlinks | in-process | all but backlinks |
| Site search, including the snippet text | in-process and browser | yes |
| The attachment by its direct address | in-process | yes |
| RSS, sitemap, the note-changes subscription | in-process | RSS |
| A free note that embeds the closed one | in-process | yes: the embed is a link |

Two leaks are known and stay unless the owner decides otherwise; the tests pin
them so that they do not grow:

- **A closed note's title is shown** on its closed page and in search results.
- **Subgraph names are in the HTML of every link** (`class="paywall paywall-internal"`),
  for every visitor, guests and anonymous readers included.

**Staying in after the right is gone**

| Event | Must happen on the next request | Level | In the spike |
|---|---|---|---|
| The inbound key is revoked | anonymous, cookie cleared, the button leads to the refusal page | browser | yes |
| "Accept people" is switched off for the edge | anonymous for that edge's guests only | browser | flag only |
| A subgraph is removed from the key | its notes are closed | in-process | no |
| Twelve hours pass | anonymous | in-process, with the clock moved | not built |
| The guest user is banned on the neighbour | refused; what the page shows was never looked at | browser | no |

**What the browser carries**

| Check | Level | In the spike |
|---|---|---|
| Session and nonce cookies are `HttpOnly`, `SameSite=Lax`, and `Secure` on an HTTPS instance | browser, on the deployed pair of section 9 | attributes read in code only |
| The nonce cookie is limited to the guest endpoint's path and is gone after the ticket leg, on success and on refusal | browser | on success |
| The handoff page is `no-store` and sends `Referrer-Policy: strict-origin` | in-process | not confirmed for this plan |
| The reader's own session on the neighbour survives a handoff | browser | yes |

**What these tests do not show.** They pin the attacks listed here. They do
not show that no other attack exists. Three things cover what a list cannot:
the two tests that read their list from the code (mutations, endpoints), the
security review that opens the work (section 4.9), and section 9, which names
what nobody has tried. A wall counts as standing only while a test in this
section still fails when the wall is taken out; the build shows that once for
each row by breaking the check and watching the test go red.

Done is the scenario walked by hand on three instances, with a transcript and
screenshots checked by an agent that has not seen the code.

## 9. What is not covered, and the experiment for each

- **Only Chromium was tried.** The origin check and the nonce cookie depend on
  browser behaviour. *Experiment:* run the spec in Firefox and WebKit
  projects.
- **No deployed pair.** Everything ran on localhost names, plain HTTP, with
  `--user-token-insecure`. *Experiment:* two instances behind TLS proxies on
  real names; walk the handoff and the open link.
- **Read replicas.** A replica forwards write methods to the primary
  (`cmd/server/routing.go:198-207`); the ticket leg is a GET. Whether a ticket
  minted on the primary opens on a replica, with spent ids held in memory per
  process, was not tried. *Experiment:* the spec against the replica ports of
  the test stack.
- **Where tool results get their links.** Section 4.4 says the federated tools
  add an open link per result. The place that assembles result URLs was not
  read for this plan (`prefixKBID` in
  `internal/case/mcp/federation_helpers.go:294` was seen, not traced).
  *Experiment:* trace one `federated_search` answer from the peer's payload to
  the tool's text before building.
- **A proxy that forces `Referrer-Policy: no-referrer`**, and an HTTPS hub
  with an HTTP neighbour. Both are expected to send `Origin: null` and be
  refused. Inferred from the Fetch standard. *Experiment:* a proxy header in
  the deployed pair above.
- **A refused page still counts as a view** for every signed-in reader,
  guests included (`internal/case/rendernotepage/resolve.go:286-301` runs
  before the access check). Not changed here. *Experiment:* none needed to
  confirm; it needs a decision of its own.
- **The reader id is an unsalted hash of the email.** The neighbour's admin
  can test whether an address visited. Left as is. *Experiment:* none; the
  remedy is a keyed hash, at the price of not recognising the same person
  across instances.
- **A hub can mint guests without limit.** Each new reader hash is a row.
  *Experiment:* a loop of handoffs with random readers and a look at the table
  size; then a per-edge cap if the owner wants one.
- **Peers that serve several domains.** The session cookie is set on one host.
  *Experiment:* a multi-domain test stack; not planned for this version.

## 10. Order of work

1. Security review of the spike as it is: done on 2026-11-18. Fix what it
   found first (sections 4.5, 4.10, 4.11), each with its test from 8.1.
2. Rebase on `main`.
3. Migrations, guests table, per-edge setting, flag removed.
4. Twelve-hour guest session.
5. Landing page key, open link, links in tool results.
6. Bar and dead ends.
7. Documentation.
8. The hand-walked scenario and its screenshots.
