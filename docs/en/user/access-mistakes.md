---
title: "Access mistakes: you think it is closed, and it is open"
description: "Seven ways an owner opens more than they meant to, why the owner is the last to notice, and how to check what a reader or a paired base really gets."
free: true
lang_redirect: "[[ru/user/access-mistakes]]"
---

This page lists the mistakes that open a note to more readers than its owner meant. None of them is a bug: each is a rule working as written, met by an owner who expected something else. After reading it you can look at your own base and find these cases before a reader does. The rules themselves are in [[en/user/access|Who can read a note]]; this page is about where people trip over them.

In this article:

- [Why the owner is the last to notice](#why-the-owner-is-the-last-to-notice)
- [The seven mistakes](#the-seven-mistakes)
- [How to see what is open](#how-to-see-what-is-open)
- [Before you grant the first subgraph to a neighbour](#before-you-grant-the-first-subgraph-to-a-neighbour)

## Why the owner is the last to notice

- **An admin reads everything.** Every note opens for you, so nothing on your screen changes when a note opens for someone else.
- **The padlock means "not free", not "closed to everyone".** A link to a note with no subgraph carries the same padlock as a link to a note in a private subgraph. The first is readable by every reader with any access; the second is not.
- **The mistake shows up on someone else's screen.** A reader who sees more than they should has no reason to tell you.

So a base can look locked from the inside and be open from the outside. The only reliable check is to look as someone else: [How to see what is open](#how-to-see-what-is-open).

## The seven mistakes

### 1. `free: true` on a note in a closed subgraph

```yaml
---
subgraph: internal
free: true
---
```

**What you expect:** the note stays inside `internal`, and `free` means something small, like "no payment".

**What happens:** every visitor reads it, signed in or not. `free: true` wins over any subgraph. The subgraph still groups the note; it no longer closes it.

**How to fix:** remove `free`. If a patch sets `free` for a whole folder, check which folders it covers.

### 2. One subgraph granted, and every untagged note goes with it

**What you expect:** a reader or a paired base that was given `public-specs` reads `public-specs`.

**What happens:** they also read every note that has no subgraph. A note with no `subgraph` is not private: any reader who holds access to at least one subgraph reads it, and a federation key with at least one granted subgraph is such a reader. A signed-in reader with no grant at all does not read these notes.

**How to fix:** put untagged notes into a subgraph you grant to nobody, with one patch for the whole base: [[en/user/subgraphs#Make the whole vault private by default|Make the whole vault private by default]]. Your own subscribers who read untagged notes before will then need that subgraph.

### 3. "Require sign-in" taken for "our people only"

**What you expect:** a subgraph marked **Require sign-in** is for staff.

**What happens:** it is for anyone who can sign in. Any signed-in reader reads its notes with no grant at all, and so does every paired base whose key was granted at least one subgraph.

**How to fix:** keep in such a subgraph only what any signed-in visitor and any paired base may read. For staff, use an ordinary subgraph and grant it to them. The "private by default" patch does not help here: it only touches notes with no subgraph, and these notes have one.

### 4. A closed note embedded in an open one

```markdown
---
free: true
---

![[Paid lesson]]
```

**What you expect:** a reader without access sees a wall where the embed is.

**What happens:** they see a link to `Paid lesson`, not its text. An embed is copied into the host only when everyone who reads the host can read the embedded note too. Otherwise a link takes its place.

**What to check:** if you meant the text to show, the host must have the same or narrower access than the embedded note. Details: [[en/user/access#An embedded note is shown only to those who can read it|An embedded note is shown only to those who can read it]].

### 5. An API key handed to an agent that should see a part

**What you expect:** the agent reads the folders you had in mind.

**What happens:** over MCP an API key reads every note in the base, the same as a site admin. An API key is the owner's access; it cannot be narrowed to a part.

**How to fix:** give a reader-agent a federation key with the subgraphs it needs, or a personal token of a user who holds the grant. See [[en/user/access#Keys and tokens|Keys and tokens]] and [[en/user/org-knowledge-base|An organisation's knowledge base]].

### 6. A KB-note for a neighbour left untagged

**What you expect:** the note that points at a paired base is for you and your agents.

**What happens:** a KB-note is an ordinary note. With no `free` and no `subgraph`, every reader of yours with any access can route their agent through it and gets everything the neighbour granted to your key. With `free: true`, an anonymous caller can.

**How to fix:** put the KB-note into a subgraph and grant that subgraph to the people who should reach the neighbour. See the visibility table in [[en/user/federation#Permissions|MCP Federation]].

### 7. A title or a subgraph name that says too much

**What you expect:** a closed note is invisible.

**What happens:** two things about it are public.

- **Its title** is shown on the wall that replaces the page, on magazine cards, in wikilinks from notes a reader can open, and in site search. **Its `description`** is shown on the wall and on magazine cards.
- **The names of its subgraphs** are in the HTML of every link to it: a link to a note in the subgraph `layoffs` carries the class `paywall-layoffs`, for every visitor.

**How to fix:** write titles and descriptions of closed notes as if strangers read them, and name subgraphs by audience, not by content. To keep closed notes out of site search, start the server with `--search-hide-unreadable`.

## How to see what is open

None of these replaces the others; each shows a different slice.

| Check | Shows | Does not show |
|---|---|---|
| A private browser window | What a visitor with no account gets: free notes, walls with titles | What a subscriber or a paired base gets |
| Admin → Notes & Content → Note Graph | Every note at once: a green square is `free`, a coloured circle is in a subgraph of that colour, a grey circle has no subgraph or is in a subgraph with no colour set. The home page is a diamond. After **Public outside**, the next **Auto layout** moves the free notes to an outer ring | Who holds access to which subgraph. A note in several subgraphs shows the colour of the first only |
| Admin → Notes & Content → Note Views | A list of notes; each row shows whether the note is **Free** | Subgraphs, and who reads them |
| A test user with the same grant as a real reader | Exactly what that reader gets. Create the user, grant the subgraphs, then use **Sign-in link** on the user's page and open the link in another browser profile. The link lives five minutes by default | What a paired base gets |
| Admin → Integrations → Federation Secrets, the key's page | The subgraphs granted to a paired base, under **Subgraph Access**; the list itself shows only how many | The untagged and **Require sign-in** notes that come with the first grant |

Give every subgraph a colour first (Admin → Notes & Content → Subgraphs). Then the grey circles on the graph are the notes of mistake 2, and the green squares are where to look for mistake 1.

Signing in through the link is a full session as that user: pages you open are recorded as their views, and it replaces your own session in that browser profile. Use a test user, not a real reader's account.

## Before you grant the first subgraph to a neighbour

1. Give every subgraph a colour, open the Note Graph and look through the grey circles. If any of them should not leave the base, apply the "private by default" patch first.
2. List your **Require sign-in** subgraphs. Everything in them goes to the neighbour with the first grant.
3. Go down the Note Views list and check every note marked **Free** that sits in a folder you think of as internal.
4. Check that the notes you are about to share do not embed anything closed.
5. Read the titles and descriptions of the closed notes that the shared notes link to.
6. Grant the subgraph, then look at the grant from the other side: ask the neighbour's owner what their agent finds for a query that should return nothing.
