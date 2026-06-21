# Landing: Obsidian Sync / Publish alternative

Audience: Obsidian users searching for alternatives to Sync and Publish (solo or small circle). Lead with what's different, not what overlaps.
Voice/positioning: see [`../00_brief.md`](../00_brief.md).

---

## Hero

**Publish your vault, make it AI-queryable, and share a slice with your circle. Without uploading your files to anyone.**

Obsidian Sync keeps your notes on your devices. Obsidian Publish puts them on a website for humans to read. trip2g does something else: it turns your vault into an endpoint your AI agent can query, federates it with the people you trust, and gives you per-folder access control. Same Markdown files. No cloud copy.

`[ Start free → ]`   `[ mcp:add https://trip2g.com/_system/mcp ]`

Open source. Self-hosted. Your notes never leave your disk.

---

## What Obsidian Sync and Publish actually do (and don't do)

These tools do their jobs well. Worth being specific about what those jobs are.

Obsidian Sync (~$4-10/mo) is multi-device sync for one person. End-to-end encrypted, works on mobile, handles conflicts on most files reasonably well. If "I want my notes on my phone and laptop" is the whole job, Sync is fine and you probably don't need trip2g.

Obsidian Publish (~$8-16/mo per site) is a static website for humans. You pick which notes go public, Obsidian hosts them, people can read them in a browser. It looks good. If "I want a public-facing digital garden humans can browse" is the whole job, Publish is fine.

Where they stop: neither one makes your vault queryable by an AI agent. Neither one lets you share one folder with one person's agent while keeping the rest private. Neither one federates. Your Sync vault and your collaborator's Sync vault are two separate islands.

---

## The problem (say it plainly)

You've got notes worth sharing, but sharing them is still a mess:

- Git breaks on real vaults. The moment two people sync an Obsidian vault through Git, `workspace.json` conflicts on every pull. Guaranteed. You spend time resolving merge noise instead of writing.
- Publish is for humans, not agents. A static site can't answer a question. Your AI can't query it, filter by date, or pull a specific section. It's a brochure, not an endpoint.
- Sync is personal. It's one vault, one account. There's no "share this folder with my study partner's Claude without giving them everything."
- Your agent is still blind. Even if your notes are beautifully organized, your AI is working from a copy you pasted into the chat, or from a local file it happened to be pointed at. Not from a live, queryable endpoint it can reach at any time.

---

## How trip2g is different

1. Connect a vault. Point trip2g at your Obsidian/Markdown folder. One sync and it becomes a live knowledge-base node: an MCP endpoint your agent can call, with search and access control built in.
2. Set the boundary. Mark what's personal, what's shared with your circle, what's public. Per-base, per-folder. Revocable anytime.
3. Federate. Link your node to a collaborator's. Now your agent (or theirs) can run one query and get an answer assembled across both vaults, with citations, without anyone exporting a file.

No shadow copy. The server publishes; it doesn't store your notes.

---

## What this actually feels like

> You're working with a reading partner. You ask your agent: *"What did we each pull from the Kahneman chapters last month?"*
>
> Your agent queries your vault, peers with your partner's node, and returns a merged answer with citations pointing back to the specific notes in each vault. Your partner's notes stayed on their machine. Yours stayed on yours.

That's the difference from Publish: a human can read a published note. An agent can query a trip2g node, filter it, combine it with another base, and give you a synthesized answer.

That's the difference from Sync: Sync keeps one person's files in sync. trip2g connects two people's knowledge without merging their vaults.

---

## Own your data (for real)

- Local-first. Your device is the primary copy. The network is optional. trip2g is the gate, not the vault.
- No shadow copy. The server is contentless middleware. It never keeps a private mirror of your notes. If trip2g shut down tomorrow, your Markdown files are still on your disk, unchanged.
- Markdown, forever. No proprietary format, no lock-in. Plain files.

---

## Who should stay on Obsidian Sync and Publish

Honest answer: most people. If you want your notes on multiple personal devices, Sync does that job cleanly. If you want a public website for human readers, Publish is polished and works well. Both are reasonable for what they are.

trip2g makes sense when you want at least one of these:

- Your agent can query your vault live (not just a file it was pointed at once).
- You want to share a specific folder with a collaborator's agent, not your whole vault and not a public website.
- You're working in a small circle and want federated search across everyone's notes without a central server holding copies.
- You want to publish notes for human readers and have an AI-queryable endpoint at the same time.

If none of those apply, you probably don't need trip2g yet.

---

## Pricing

- Self-host: free. Open source (MIT). One Go binary + Docker.
- Free sandbox: 100 MB, no install, to try it now.
- Managed hosting: when you'd rather not run Docker. No sandbox limit.

For comparison: Obsidian Sync runs $4-10/mo for personal multi-device sync. Publish runs $8-16/mo per site for a static human-readable website. trip2g's managed tier covers a different job: an agent-queryable, federated, access-controlled endpoint.

`[ Self-host → ]`   `[ Try the sandbox → ]`

---

## Honest FAQ

**Is this just Obsidian Publish with AI features bolted on?**
No. Publish compiles your notes into a static site. There's no runtime, no query endpoint, no access control per collaborator. trip2g is active middleware: your agent can call it, ask a question, and get a structured answer from a live knowledge base. Different architecture, different job.

**Can I use trip2g alongside Obsidian Sync?**
Yes. trip2g sits behind your vault folder, whatever is keeping that folder in sync. Sync handles your multi-device personal copy; trip2g adds the agent endpoint and federation on top.

**Does trip2g replace Obsidian Sync for multi-device personal sync?**
Not today. trip2g's focus is agent queryability and federation, not device sync. Use Sync (or iCloud, or Syncthing) for your personal multi-device setup. trip2g handles the "my agent and my collaborators' agents" layer.

**What happens if the trip2g server goes down?**
Your notes are still on your disk, untouched. The agent endpoint goes offline, but nothing is lost. Local-first means the network is optional.

**Isn't a giant context window going to make this pointless?**
The retrieval argument weakens if models get 10M-token windows with no performance decay. The ownership and access-control arguments don't: you still don't want your private notes living in someone else's cloud, and you still want to share one folder with one person's agent without sharing everything. We'd rather say that plainly than pretend the question away.

**Do I need to know Docker?**
To self-host, yes. It's one binary and a Compose file, but it's still a terminal. The free sandbox skips that entirely; try it there first.

---

`[ Start free → ]`   `[ Read the docs ]`   `[ Star on GitHub ]`
