# Federation guest handoff: from the spike to a real version

**Status: pending approval. Nothing in this plan is built.** The spike it
grows from exists and runs, on a branch that is not for merge.

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

Turning it on shows one line under the checkbox: how many notes on this
instance have no subgraph, with a link to the "private by default" recipe.
The rule does not change; the owner learns its consequence before switching.
Section 7, decision 3.

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
Results that came through a second hop get no link in this version.

### 4.5 The neighbour's site address

The spike derives the neighbour's site from its MCP address by dropping
`/_system/mcp` (`MCPFederationSite` in `internal/model/mcp_federation_note.go`
*(spike)*). That is wrong for a neighbour reached by agents on an internal
address. A new optional frontmatter key, `mcp_federation_site_url`, overrides
it. Without the key the derivation stays.

### 4.6 A bar that says where the reader is

On every page of the default template a guest sees one line: "You are a guest
from sales.example.com · Back to sales.example.com · Leave". The hub's origin
is already signed into the handoff; the real version keeps it in the guest
session beside the edge's key id. On a second hop the bar names the base the
reader came from last, and "Back" goes one step, because that is all the
second neighbour knows.

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

## 5. Migrations

Two. Shown here for confirmation; neither file exists.

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

1. **The two migrations of section 5.** Confirm the SQL.
2. **The warning under the checkbox** (section 4.2): a count of notes with no
   subgraph, or nothing.
3. **Links for results from a second hop** (section 4.4): none in this
   version, or a design for a chained open link first.

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
- **Site override:** a KB-note with `mcp_federation_site_url` sends the
  browser there and still signs with the edge of its MCP address.
- **Bar:** names the hub, "Back" leads to the hub's origin, "Leave" ends the
  guest session; on a second hop it names the middle base. A Jet layout that
  calls the method shows the same bar for a guest and nothing for anyone else;
  a layout that does not call it shows none.
- **Dead ends:** an expired guest cookie gets the added line once; the refusal
  page links back only for a validly signed token.
- **The documented patch:** with it, a guest gets 403 on a note with no
  subgraph. This pins the recipe the docs point to.

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

1. Security review of the spike as it is; fix what it finds.
2. Rebase on `main`.
3. Migrations, guests table, per-edge setting, flag removed.
4. Twelve-hour guest session.
5. Landing page key, site override, open link, links in tool results.
6. Bar and dead ends.
7. Documentation.
8. The hand-walked scenario and its screenshots.
