---
title: Subgraphs
free: true
lang_redirect: "[[ru/user/subgraphs]]"
---

A subgraph is a named group of notes with its own access rule: add `subgraph: paid` to a note's frontmatter and only readers granted `paid` can open it. One vault, many audiences — public blog, paid course, internal wiki — without splitting into separate sites.

**Notes are private by default.** Only three kinds of reader see a subgraph-gated note: an admin, someone who bought access (a paid offer or subscription), or a user an admin granted access to by hand. Everyone else gets a paywall.

### What a subgraph is

A label in frontmatter. Any note can carry one or several:

```yaml
subgraph: paid
```

```yaml
subgraphs: [team, paid]
```

Both keys work (`subgraph` and `subgraphs`, string or list); a note with several labels belongs to all of them. The subgraph itself is created automatically the first time a note with its name syncs in — there is no separate "create subgraph" step.

Access checks run in this order:

1. **Admins** see everything.
2. **`free: true`** opens the note to everyone — anonymous visitors included — regardless of subgraphs. It's a separate axis: the subgraph groups the note, `free` publishes it.
3. A subgraph marked **Require sign-in** (in the admin) opens its notes to any signed-in user, no subscription needed.
4. Everyone else needs an **active access grant** to at least one of the note's subgraphs. No grant — the visitor sees a paywall or a sign-in wall.

So a note in a subgraph is closed by default. You open it per reader (a grant), per audience (an offer or a Telegram group), or entirely (`free: true`).

**One gotcha worth knowing.** A note with *no* subgraph is visible to every signed-in user who holds at least one grant. If you mix audiences, tag everything — or install a frontmatter patch that drops untagged notes into a subgraph nobody is granted. The patch recipe is in [[en/user/user_management]].

### Why use them

**A public blog plus a paid section.** Blog posts get `free: true`. Course chapters get `subgraph: paid`. Subscribers who bought the `paid` offer read the course; everyone else reads the blog and hits a paywall on the chapters.

**An internal area.** Put ops runbooks in `subgraph: internal` and grant it only to team members. The notes live in the same vault as your public docs but never leak to visitors or subscribers.

**Per-audience sections.** `subgraphs: [beta, paid]` — beta testers and paying users both read the note; each group holds a different grant.

**A scoped window for an agent or a peer.** Federation secrets are scoped to subgraphs: a partner's agent searching your base sees only the subgraphs you granted their key. This is the core of [[en/user/context-separation]]; the mechanics are in [[en/user/federation]].

### How to set it up

**1. Tag the notes.** Add `subgraph:` in Obsidian's properties panel or frontmatter, sync. You can also tag a whole folder at sync time: `--meta subgraph=team` (see [[en/user/advanced]]).

**2. Configure the subgraph.** Admin → Notes & Content → Subgraphs. Each subgraph has a color, a Hidden flag, and **Require sign-in** — check it to open the subgraph to any authenticated user without a subscription.

**3. Grant access.** Pick the route that fits:

| Route | Where |
|-------|-------|
| Sell it | Admin → Monetization → Offers: an offer lists the subgraphs it unlocks; a purchase grants them |
| Grant a person directly | Admin → Users, or the `createUserSubgraphAccess` mutation ([[en/user/user_management]]) |
| Telegram group members | Bind a group to a subgraph in the bot settings — members get access automatically |
| A federated peer | Scope their inbound secret to the subgraph ([[en/user/federation]]) |

### Tag a whole folder

Editing frontmatter note by note gets old once you have a `course/` folder with forty lessons. A frontmatter patch tags the whole folder in one shot. Create a note anywhere in the vault:

```markdown
---
type: frontmatter-patch
include: ["course/**"]
---

```jsonnet
{ subgraph: "course" }
```
```

Every note under `course/` gets `subgraph: course` at load time — you never touch the individual files, and new lessons dropped into the folder are tagged automatically. See [[en/user/frontmatter-patches|frontmatter patches]] for the full syntax (glob patterns, exclude, priority, multiple rules per file).

### Grant a user access by hand (no payment)

For a teammate, a client, or anyone you're not charging:

1. **Admin → Users → Create user.** Enter their email. No password to set — trip2g uses passwordless sign-in.
2. **They sign in.** The user goes to your site, enters their email, and gets a one-time sign-in code (or uses OAuth if you've configured Google/GitHub). No purchase, no offer involved.
3. **Admin → Users → pick the user → grant subgraph access.** Select the subgraph(s) and, optionally, an expiry date. This calls `createUserSubgraphAccess` under the hood — the same mutation an agent would call via the admin GraphQL API, documented step by step in [[en/user/user_management]].

The user can now read every note in that subgraph, nothing else.

### Per-subgraph sidebar and home page

A subgraph can look like its own site section:

- `_sidebar_<name>.md` — sidebar shown on every note of the subgraph (falls back to the global `_sidebar.md` if absent).
- A home note at `_index_<name>.md`, `_home_<name>.md`, or simply a note whose URL is `/<name>`.

So a `course` subgraph gets `_sidebar_course.md` with the lesson list and a landing note at `/course`.

### Worked example: a paid course with a free landing

```
course.md              ← landing: free: true
_sidebar_course.md     ← lesson navigation
lessons/lesson-1.md    ← subgraph: course
lessons/lesson-2.md    ← subgraph: course
```

`course.md`:

```yaml
---
free: true
---
```

`lessons/lesson-1.md`:

```yaml
---
subgraph: course
---
```

Then in the admin: create an offer that unlocks the `course` subgraph. Anyone can read the landing and see what they're buying; the lessons stay behind the paywall until purchase. Note the landing itself may also carry `subgraph: course` for the sidebar — `free: true` still wins and keeps it public.

**For agents:** an agent with admin GraphQL access can create the subgraph, grant the user, and skip the admin panel entirely — see [[en/user/subgraph_for_agents]].

### Related

- [[en/user/monetization]] — offers, Patreon, Boosty, crypto, Telegram-group access
- [[en/user/user_management]] — granting access by mutation, the untagged-notes warning
- [[en/user/context-separation]] — subgraphs as context boundaries between bases
- [[en/user/federation]] — scoping a peer's search to your subgraphs
- [[en/user/subgraph_for_agents]] — driving subgraph admin from an agent via GraphQL
