---
title: "A network of bases instead of one shared workspace"
description: "How trip2g lays out the knowledge of several owners: every owner keeps a base of their own, and a base shows a neighbour only what its owner granted. What that buys, what it costs, and the one access rule behind it."
free: true
lang_redirect: "[[ru/user/federation-model]]"
---

This page explains the idea behind federation in trip2g. Most team tools put everyone's notes into one shared workspace and then cut it up with permissions. trip2g does the opposite: every owner keeps a base of their own, and bases are joined by edges that carry only what an owner granted. After reading it you can decide whether your knowledge belongs in one base or in several, and you will know exactly what a neighbour reads once you pair with it. The mechanics of pairing are in [[en/user/federation|MCP Federation]].

In this article:

- [Two ways to hold shared knowledge](#two-ways-to-hold-shared-knowledge)
- [What an edge is](#what-an-edge-is)
- [One access rule for every reader](#one-access-rule-for-every-reader)
- [A worked example](#a-worked-example)
- [What crosses an edge today](#what-crosses-an-edge-today)
- [A mess stays where it was made](#a-mess-stays-where-it-was-made)
- [What this costs](#what-this-costs)
- [Where to go next](#where-to-go-next)

## Two ways to hold shared knowledge

| | One shared workspace | A network of bases |
|---|---|---|
| Where notes live | In one place, kept by one vendor or one admin | In each owner's own base: markdown files and one trip2g instance |
| Who decides what a colleague reads | The workspace admin, page by page | The owner of the base that is being read |
| What a newcomer sees by default | Whatever the workspace shows to members | Nothing of a neighbour's, until that neighbour grants it |
| What an untidy corner costs | Everyone's search and navigation | Its own owner |
| How you find something | One search over everything you may see | A search that asks every base you are paired with |
| What you administer | Members and page permissions | A pair of keys for every two bases that talk |

Neither column is better in general. One workspace is simpler when one group writes together and agrees on how things are filed. A network fits when the writers have different owners: departments that answer for their own knowledge, a company and its contractor, a person's own base next to their team's.

In trip2g a base is one instance with its own notes, its own readers and its own admin. Nothing is shared between two instances until both owners set it up.

## What an edge is

An edge joins two bases in one direction. It is a pair of keys with one key ID: an **outbound** key on the base that asks, and an **inbound** key on the base that answers.

- **The answering side decides what the edge carries.** Its owner opens the inbound key in the admin and ticks subgraphs. A new key is granted nothing.
- **Each direction is its own edge.** Sales reading Engineering and Engineering reading Sales are two pairs of keys with two separate grants.
- **A neighbour learns about its own edge only.** It can ask which subgraphs it was granted and read their descriptions. It is told nothing about your other neighbours.
- **A grant is read on every request.** Remove a subgraph from the key, or revoke the key, and the next request from that neighbour already gets less.

Because an edge is granted by subgraph, a base that wants to share a part has to say which notes are in that part. That is the work a network asks of each owner, and it is the only place where order is required.

## One access rule for every reader

There is no separate rule for neighbours. A key is treated like a reader who holds access to the subgraphs the key was granted.

| Note | Guest | Signed in, no access | Reader with access to another subgraph | Key granted another subgraph | Reader or key with this subgraph | Admin |
|---|---|---|---|---|---|---|
| `free: true` | reads | reads | reads | reads | reads | reads |
| no `free`, no `subgraph` | closed | closed | reads | reads | n/a | reads |
| `subgraph: specs` | closed | closed | closed | closed | reads | reads |
| in a subgraph with **Require sign-in** | closed | reads | reads | reads | reads | reads |

The reader columns are explained in [[en/user/access|Who can read a note]]. Two rows matter more once you have neighbours.

- **A note with no subgraph is not private.** The first subgraph you grant to a key also opens every note that has no subgraph. If your base holds internal notes that were never tagged, a neighbour granted `public-specs` reads all of them.
- **A subgraph with Require sign-in is open to every key that was granted anything.** It was made for "any signed-in reader", and a key with a grant counts as one.

To close the first row, put every untagged note into a subgraph you grant to nobody. One patch does it for the whole base: [[en/user/subgraphs#Make the whole vault private by default|Make the whole vault private by default]]. The patch also changes what your own readers see: a subscriber who read untagged notes before now needs that subgraph.

Nothing closes the second row for a key. Keep in a **Require sign-in** subgraph only what any paired base may read.

## A worked example

Three departments, three bases. Engineering pairs with Sales and grants the Sales key one subgraph, `shared`.

Engineering's notes:

```yaml
# specs.md
subgraph: shared
```

```yaml
# salaries.md
subgraph: internal
```

```yaml
# roadmap-draft.md   (no frontmatter)
```

What a Sales agent gets from Engineering:

| Note | Before the patch | After "private by default" |
|---|---|---|
| `specs.md` | reads | reads |
| `salaries.md` | closed | closed |
| `roadmap-draft.md` | **reads** | closed |

Engineering did not grant the draft. It became readable because it carries no subgraph. This is the one surprise of the model, and it is why the patch is worth applying before the first grant, not after.

Design pairs with Engineering the same way and grants `partners`. Sales has no edge to Design, so a Sales agent reaches Design only through Engineering. Two things limit that: the Sales key must be able to read Engineering's KB-note for Design, and what comes back is what Design granted to Engineering's key.

## What crosses an edge today

**Agents cross.** An agent connected to its own base searches paired bases, reads their notes, walks their tables of contents and fetches their attachments through the federated tools. Every answer is limited by the grant of the edge it travelled. A question fans out to neighbours of neighbours, to a depth of three by default.

**People do not.** A reader signed in on Sales has no session on Engineering. A link in an agent's answer leads to Engineering's site, and that site treats the reader like any other visitor: a free note opens, anything else asks them to sign in there. Giving a person the same passage an agent has is not built.

## A mess stays where it was made

In a shared workspace, order is everyone's job, and so it is often nobody's: one team's abandoned pages show up in another team's search. A network does not make anyone tidy. It changes who pays for the mess.

- A base that nobody tends is still one base. Its neighbours see the subgraphs it granted and nothing else.
- An owner who cares keeps their base in order without waiting for the others.
- An owner who does not care has nothing to grant that is worth reading, and that shows on the first search.

The limit of this is the row from the access table above: an untidy base is exactly the one with many untagged notes, and untagged notes are what an edge carries beyond its grant. A careless owner who grants one subgraph shares more than they meant to. The patch closes that.

## What this costs

- **Access is granted to a base, not to a person.** Everyone on the asking side who can use the KB-note for that neighbour gets the whole grant of the edge. The KB-note is an ordinary note, so the asking base decides who that is with its own subgraphs. To give two audiences different parts, hold two keys with two grants and one KB-note for each.
- **There is no single index.** A search asks each neighbour and merges the answers. A neighbour that does not answer in time is skipped, so a result can be missing without an error. By default one request to a neighbour is cut off after two seconds.
- **Keys to keep.** Every pair of bases that talk needs a key in each direction, and each key has its own grant to review.
- **Names are local.** A subgraph called `internal` on one base has nothing to do with `internal` on another. A neighbour sees a subgraph's name and the description its owner wrote, so write the description.
- **People stop at the edge.** See the section above.

## Where to go next

- [[en/user/federation|MCP Federation]]: pair two bases, rotate and revoke keys, the federated tools.
- [[en/user/subgraphs|Subgraphs]]: tag notes, and the patch that closes untagged ones.
- [[en/user/access|Who can read a note]]: the access table in full, and what a closed note still shows.
- [[en/user/org-knowledge-base|An organisation's knowledge base]]: how many bases a company needs and who writes to each.
