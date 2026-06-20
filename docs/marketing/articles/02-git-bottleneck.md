# The Git bottleneck: why your Obsidian-Git setup is guaranteed to fail in the agentic era

*Audience: Obsidian power users, peer circles, anyone who's ever rage-quit a merge conflict. ~1,100 words. Voice: see [`../00_brief.md`](../00_brief.md).*

---

Open any Obsidian community forum and search for "Git sync conflict." You will not run out of results. The complaints are strikingly specific: `.obsidian/workspace.json` is mentioned by name, described as a file that records nothing more than which panel was open on screen. One user put it plainly: "This file alone guarantees a conflict 100% of the time between my laptop and my phone." Not sometimes. Guaranteed.

That's not bad luck. It's a structural problem. In the agentic era, where your notes are read and written by agents dozens of times a day rather than by you twice, it becomes fatal.

## What's actually inside that conflict

Git was built for source code, where two developers make deliberate changes to the same file and one of them has to decide whose logic wins. That model assumes:

1. Changes are intentional.
2. A human reviews the merge.
3. The files being merged are meaningful text.

Note-taking violates all three assumptions at once.

When you open Obsidian on your phone while it's also open on your laptop, both instances rewrite `workspace.json` with their own layout state. Neither change is "intentional" in any meaningful sense: they're side effects of the app running. No human should be reviewing a JSON diff that says `"activeLeaf": "leaf-4f2a"` vs `"activeLeaf": "leaf-7c8b"`. The file isn't content. It's ephemeral UI state that will be overwritten the next time you open the app anyway.

The same pattern plays out across `.obsidian/plugins/`, graph-view state, and any plugin that persists its own configuration. The Obsidian Git community has produced a long list of paths to add to `.gitignore` just to make the tool barely tolerable. It's a patch on top of a mismatch.

That's just the single-user case. Add a second person, and the conflict rate does not double. It compounds. Two people taking notes in parallel, across two machines each, through a Git remote: the merge queue never empties.

## The deliberate-commit model vs. the many-small-writes reality

The Ink & Switch researchers named this cleanly in their [Local-First Software](https://www.inkandswitch.com/local-first/) manifesto. The ideal they describe is one where collaboration and ownership don't require a central server or a merge-conflict tax. Git doesn't meet that ideal. It was designed for the deliberate-commit model: you finish a chunk of work, you write a message, you push. The note-taking reality is the opposite. Dozens of tiny writes per session, auto-saves every few seconds, plugin state updated constantly, never a good moment for a deliberate commit.

When you force that reality into Git's model, you pay in invisible labor. Not just the time to resolve a conflict, but the cognitive overhead of remembering to pull before you open the app, push before you close it, check the remote before a shared session. That tax compounds across every collaborator. A circle of four friends sharing an Obsidian vault over Git is, in practice, a part-time job nobody agreed to take.

## What the agentic era does to this problem

Until recently, the stakes were manageable. Human writers open Obsidian maybe five times a day. The conflict window is narrow.

Agents don't have that rhythm. A background agent monitoring your notes might read and annotate twenty files in an hour. An agent responding to a query might write a summary note, update a tag index, and add backlinks in thirty seconds. Several agents, across several users in a shared vault, do this concurrently.

Git was not built for this. The merge conflicts that were annoying in 2022 are load-bearing failures in 2025. A system that guarantees conflicts at human pace guarantees constant failures at agent pace.

There is also a subtler problem. When two agents write to the same vault through Git, their writes are buffered, batched, and delayed by the commit-push-pull cycle. An agent that writes a note at 2:00pm may not have that note visible to another agent (or to you) until someone remembers to push. For agents that depend on each other's outputs, or on fresh information, this isn't a minor annoyance. It breaks the whole point of having agents reason over shared knowledge.

## What "active middleware" means in practice

The alternative isn't a different sync tool. It's a different architecture.

Instead of a shadow copy of files moving through Git, the knowledge base becomes a live node. When a file changes, the change is immediately visible to anything querying that node: no commit, no push, no pull. The node exposes a search endpoint and an MCP interface that agents speak natively. Reads and writes happen against the live index, not a stale snapshot.

This is what trip2g does. It runs as a daemon alongside your Markdown files. The files stay on your disk. You own them; the server never holds a copy. But the node is always current: write a note, and the next agent query reflects it. No shadow copy. No merge queue.

For a peer circle, each person runs their own node. The nodes federate: one agent query can fan out across every node in the circle it's permitted to reach, and return one answer citing each source. The boundary between private and shared is defined per-folder, not per-repository. Because the node is live, an agent working with your notes and an agent working with a friend's notes can coordinate in real time, not after the next `git pull`.

## The honest caveat

This isn't zero complexity. Running a daemon means running a daemon: something that starts at boot, that you need to monitor, that can fail. If your collaboration model is genuinely simple (two people, one shared machine, a single Obsidian vault with no mobile access and no agents), a well-maintained `.gitignore` might be enough. The mesh earns its keep when you have separate owners, mobile devices, and agents in the picture. That description covers most real peer circles. It doesn't cover everyone.

## The point

Git's conflict guarantee isn't a bug you can configure away. It's the consequence of using a deliberate-commit tool for a continuous-write problem. That mismatch was a minor friction for solo note-takers in 2020. For a circle of four people with active agents reading and writing their shared knowledge base in 2025, it's structural breakage.

The fix isn't a better `.gitignore`. Remove Git from the sync layer entirely and replace it with a live node that answers queries in real time: no commits, no conflicts, no invisible labor.

---

*trip2g is open source and self-hosted. Source your own node (MIT, one binary) or read the [local-first research](https://www.inkandswitch.com/local-first/) that shaped the design. More in [the research notes](../../dev/market_research_notebooklm.md).*
