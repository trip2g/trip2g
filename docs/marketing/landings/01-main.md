# Landing (trip2g.com)

Audience: small trusted peer circles (Obsidian power users, 3-5 collaborators). Lead with the mesh + agents.
Voice/positioning: see [`../00_brief.md`](../00_brief.md).

---

## Hero

**Federate your local notes into a private memory mesh for your AI agents.**

Your notes stay in your vault, on your machine. trip2g turns them into an endpoint your agent can query, and connects to your friends' bases the same way. One question reaches everyone's knowledge without anyone uploading a file.

`[ Start free → ]`   `[ mcp:add https://trip2g.com/_system/mcp ]`

Open source. Self-hosted. Your data never leaves your control.

---

## The problem (say it plainly)

You and three friends keep your best thinking in Obsidian. Sharing it is still miserable:

- **Git breaks.** Sync a real vault across people and you get conflicts on files like `workspace.json`. Guaranteed, every time. You spend your evening resolving merge conflicts instead of thinking.
- **Cloud is all-or-nothing.** Notion and Drive make you choose: fully private, or fully shared. There's no "share this one folder with this one person's agent."
- **Your AI is blind.** Your agent can read the vault it's pointed at. Not your teammate's, not the shared research, not the decision someone made in a different app last week.

So context lives in twelve places, and you keep copy-pasting it into the chat box by hand.

## How it works

1. **Connect a vault.** Point trip2g at an Obsidian/Markdown folder. One sync turns it into a live node: search, an MCP endpoint, access rules.
2. **Set the boundary.** Mark what's personal, what's shared with your circle, what's public. Per-base, per-folder. Revocable.
3. **Federate.** Link your node to the people you trust. Now any agent (yours or theirs) can run one `federated_search` and get an answer assembled across every base it's allowed to see, with citations.

No central server holds your notes. trip2g is the gate, not the vault.

## What this actually feels like

> You ask Claude: *"How did our group decide the Q3 priorities?"*
> Your agent pulls the prioritization framework from your vault, then peers with a teammate's node and finds the thread where the call was made. It hands you one answer, citing both. Neither of you exported or uploaded anything.

That's the whole product in one query.

## Own your data, for real

- **Local-first.** Your device is the primary copy. The network is optional. No spinners, no lock-in.
- **No shadow copy.** The server publishes; it doesn't store a private mirror of your notes.
- **Markdown, forever.** If trip2g disappears tomorrow, your notes are still plain files on your disk.

## Who it's for

- **Reading and research circles** who want a shared brain their agents can search.
- **Small project teams** who need "this folder, these people" sharing (not a wiki migration).
- **Anyone** who wants their AI to know what they know, without renting that knowledge back from a platform.

## Pricing

- **Self-host:** free. Open source (MIT). One Go binary + Docker.
- **Free sandbox:** 100 MB, no install, to try it.
- **Managed hosting:** when you'd rather not run Docker. (Removes the sandbox limit.)

`[ Self-host → ]`   `[ Try the sandbox → ]`

## Honest FAQ

**Isn't a giant context window going to make this pointless?**
If models ship 10M-token windows that never lose the middle and cost nothing, the retrieval argument weakens. The ownership and access-control arguments don't: you still don't want your private notes living in someone's cloud, and you still want to share a slice without sharing everything. We'd rather tell you that than pretend the question doesn't exist.

**Is this just Obsidian Publish?**
Publish makes a website for humans. trip2g makes an endpoint for agents: queryable, access-controlled, and federated across bases. Different job.

**Do I have to give up Obsidian?**
No. trip2g sits behind your existing vault.

---

`[ Start free → ]`   `[ Read the docs ]`   `[ Star on GitHub ]`
