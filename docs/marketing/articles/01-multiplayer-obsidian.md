# Multiplayer Obsidian: build a community brain without a SaaS middleman

*Audience: small trusted peer circles (the beachhead). ~1,100 words. Voice: see [`../00_brief.md`](../00_brief.md).*

---

Obsidian is the best single-player note app ever made. It is also, by design, single-player. The moment three friends try to share one brain, the tool fights back.

You know the workaround, because everyone reaches for it: put the vault in Git. And you know how that ends. The first time two people sync, Git tries to merge `.obsidian/workspace.json` (a file that records which pane you had open) and hands you a conflict. Not a content conflict. A *which-window-was-focused* conflict. Multiply that across plugins, graph state, and a dozen daily edits, and "shared vault over Git" becomes a part-time job resolving merges on files no human was supposed to touch.

This isn't a configuration mistake you can fix with a better `.gitignore`. Git was built to version source code with deliberate, reviewed commits. Real-time note-taking is the opposite: many small writes, constantly, by people who are not thinking about commits. The mismatch is structural. The local-first researchers at Ink & Switch made this case years ago. Collaboration and ownership shouldn't require a central server or a merge-conflict tax ([Local-First Software](https://www.inkandswitch.com/local-first/)).

So people give up on Git and move to the cloud. And the cloud has its own wall.

## The all-or-nothing wall

Notion, Drive, Dropbox. They all share the same primitive: a folder is either private or it's shared. There is no "let my study partner's AI read *this one* research folder, and nothing else." You either invite someone into the whole space, or you copy a subset out by hand and email it.

For a circle of three or four people who trust each other *partly* (share the research, keep the half-formed ideas private), that primitive is wrong. You don't want a shared workspace. You want **overlapping private ones**, with a controlled seam between them.

And now there's a third party at the table who makes the wall hurt more: your agent.

## Your agent only knows the room it's standing in

Point Claude or Cursor at your vault and it can answer questions about *your* vault. It cannot see your teammate's notes, the shared reading list, or the decision someone wrote down in a different app last Tuesday. So you do the thing we all do: you copy context into the chat by hand, every session, like a human RAG pipeline.

The irony is that bigger context windows don't fix this. They just let you paste *more* by hand. And the research says pasting more isn't even reliable: models degrade on information buried in the middle of long contexts ([Lost in the Middle](https://arxiv.org/abs/2307.03172)). The bottleneck was never the window size. It was that the knowledge is **fragmented and unreachable**, scattered across people and apps, with no endpoint an agent can actually query.

## What "multiplayer" should mean

Here's the model that works for a peer circle. Not a shared vault. A **mesh**.

- Each person keeps their own vault, on their own machine. They own the files. Full stop.
- Each vault becomes a *node*: it exposes a search endpoint and a clear boundary (this folder is mine, this one is shared with the circle, this one is public).
- Nodes link to the people you trust. One agent query can now fan out across every node it's allowed to see, and come back with a single answer that cites each source.

The decisive part is the boundary travels *with* the data. You're not uploading your notes to a collective silo and hoping the permissions hold. The notes stay home; only the *answers* cross the wire, and only for the bases each agent is permitted to read.

That's the difference between "share your files" and **route context**. Files are a liability to move. Context is the thing you actually want delivered.

## The 30-second version

> You ask your agent: *"How did our group decide to prioritize the Q3 reading?"*
> It pulls your prioritization note from your vault, peers with a friend's node, finds the thread where the call was made, and hands you one answer citing both. Neither of you exported a thing.

If you've ever rebuilt that answer by hand from three chat histories, you already feel the gap.

## How to actually do it (trip2g)

This is the job trip2g was built for. It's open source and self-hosted, so the "no SaaS middleman" part is literal. There's no company holding your notes.

1. **Each person connects their vault.** Point trip2g at a Markdown folder. One sync turns it into a node: search, an MCP endpoint agents speak natively, and access rules.
2. **Mark the seam.** Per-folder: personal, circle-shared, or public. Revocable. The server never keeps a private copy; it's a gate, not a vault.
3. **Federate the circle.** Link the nodes you trust. Now any member's agent can run one federated query across the shared surface, with citations, and zero manual copy-paste.

No central database. No "invite everyone to everything." No `workspace.json` conflicts at 11pm.

## The honest caveat

A mesh is more moving parts than a shared folder. If your circle is two people who already share one machine, you don't need this. Use a folder. The mesh earns its keep when you have *separate* owners who want to stay separate, trust each other *partly*, and want their agents to act across the seam. That's most real reading groups, research collabs, and small project crews. It's not everyone, and we'd rather you skip it than fight complexity you don't need.

## The point

The "second brain" was always going to go multiplayer. That was Tiago Forte's whole premise: knowledge compounds when it's organized for outcomes, not hoarded ([Building a Second Brain](https://www.buildingasecondbrain.com/)). What changed is *who* reads it. Your agent is now the heaviest user of your notes, and it's stuck in one room.

A community brain doesn't mean pooling your files into someone's cloud. It means keeping your vault yours, and letting trusted agents walk a graph that spans all of them.

That's trip2g: *trip to graph.*

---

*Try it: self-host (MIT, one binary), or `mcp:add https://trip2g.com/_system/mcp` and point an agent at it. Sources linked inline; more in [the research notes](../../dev/market_research_notebooklm.md).*
