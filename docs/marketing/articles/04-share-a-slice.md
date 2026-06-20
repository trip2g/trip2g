# Digital sovereignty by design: share a slice of your research without merging your whole life

*Audience: privacy-conscious users, B2B partners. ~1,100 words. Voice: see [`../00_brief.md`](../00_brief.md).*

---

Every cloud collaboration tool offers the same deal: give us everything, and we'll let you share it.

Dropbox, Google Drive, Notion, the primitive is identical. A folder is private or it's shared. You pick one. There is no "let my partner's agent read this one research directory, and nothing else." You either open the vault or you don't.

For a decade that was a livable compromise. Humans are slow readers. The blast radius of sharing too much was bounded by someone's attention span.

Your agent has no attention span. It reads everything it can reach, all at once, on every query. So "share everything or share nothing" is no longer a mild annoyance. It is a structural security hole.

## The joint-R&D scenario

Two biotech companies are running a shared drug discovery project. Call them Alpha and Beta. They have a joint research folder: study notes, assay results, interim hypotheses. They also each have internal notes that belong only to them. Patent strategy, supplier contracts, things that should never cross the fence.

Under today's tools they have three choices. One: set up a shared Notion or Drive workspace and pour the joint material in, which means entrusting both companies' agents to a vendor's cloud, no air-gap, no guarantee the data won't end up in a training pipeline. Two: email files back and forth and let someone on each side maintain the "official copy," a coordination tax that scales badly when agents start reading at 50,000 tokens per second. Three: give each other full access to internal systems, which no legal team will sign.

None of these are good. What they actually need is simpler: share one folder, revocably, with clear borders that travel through the query itself, while everything outside that folder stays air-gapped on each company's own machine.

That is a different primitive. The question is whether any software actually builds it.

## Why the cloud can't give you this

Cloud platforms centralize first and slice second. The data moves to a server; the server then decides what each user sees. That model requires trusting the intermediary with your files, with your access rules, and with whatever the platform decides to do with both after you've uploaded them.

Martin Kleppmann has spent years arguing that this ordering is backwards ([martin.kleppmann.com](https://martin.kleppmann.com/)). If the data lives on a central server, you are renting access to your own knowledge. The platform holds the master copy. Your "ownership" is a permission the vendor can revoke, change, or ignore in a terms-of-service update.

The local-first researchers at Ink & Switch made the same case from a different angle. Local-first software, in their framing, keeps your device as the primary copy. The network is optional. Collaboration happens at the edges, not by funneling everything through a middle ([Local-First Software](https://www.inkandswitch.com/local-first/)). Sharing should mean routing the *query* through the parts you choose, while the originals stay home. That is not the same as uploading files to a third party and clicking a permissions checkbox.

The distinction matters more now than it did in 2019 when they wrote that. In 2019 the question was "who reads your notes?" Now the question is "which agents can query which parts of your knowledge graph, and who can prove it?"

## How per-subgraph access control actually works

A federated mesh inverts the cloud model. The data never leaves your machine. What leaves is a semantic answer to a specific question, scoped by the access rules you set.

In trip2g terms, your knowledge base is divided into zones. Personal: only your agents see it. Shared: specific trusted nodes can query it. Public: open. Each zone is a subgraph with explicit rules attached to it. Those rules travel with the query through the federation mesh. There is no shadow copy on a central server. The server is a gate, not a vault.

The joint-R&D slice for Alpha and Beta looks like this:

- Alpha runs trip2g on their own infrastructure. Beta runs it on theirs.
- Alpha marks the `joint-research/` folder as a shared subgraph and grants Beta's node read access.
- Beta's agent can now query `joint-research/` and get answers grounded in Alpha's actual files.
- Alpha's `patent-strategy/` and `supplier-contracts/` folders are in a personal zone. Beta's node never receives a query result from those paths, because no query is ever routed there.
- Alpha revokes access by changing the zone rule. No files to chase down, no accounts to suspend, no vendor to call.

A human collaborator could set this up in a single configuration file. An agent on either side would never know the other zones existed. They are not visible, not reachable, not hinted at in any response.

## The researcher-and-collaborator version

You don't need two corporate IT departments for this to matter. Consider a PhD student and their supervisor. The student has three years of interview notes, half-formed hypotheses, dead ends they'd be embarrassed to share. The supervisor has their own reading notes and a folder of papers they've annotated. They share one folder: the dissertation outline and the citation library.

Today, the student exports a subset to a shared Notion page and keeps it manually in sync. The supervisor's agent queries Notion. The student's agent queries their local vault. The two agents never talk. Every synthesis happens by hand.

With per-subgraph access control: the student marks `dissertation/` as shared, grants the supervisor's node access, and the supervisor's agent can run a query that pulls from both their annotated papers and the student's outline simultaneously, without the student ever uploading their interview notes anywhere.

The boundary is clear. It is enforced at the protocol level, not by trusting a cloud vendor to honor a permissions checkbox.

## The honest caveat

A federated mesh adds moving parts. You are running infrastructure: two trip2g nodes, each on its own machine, each needing to be online when a cross-node query fires. For two people who just want to share a Google Doc, this is the wrong tool. The complexity earns its keep when the access boundary matters: when one side of the collaboration is legally sensitive, when you need a revocation path that doesn't depend on a platform's API, when the agents reading your notes are aggressive enough that "share everything" is genuinely dangerous.

If you're at the Google Doc stage of collaboration, stay there. If you've hit the wall where all-or-nothing sharing is blocking you, this is what the alternative looks like.

## What this actually is

Cloud sharing is a product decision dressed up as infrastructure. Someone decided it was simpler to centralize everything and manage access from the top. For casual collaboration, they were right. For the world where agents are querying your knowledge graph a hundred times a day, they were wrong.

When you share a slice in a federated mesh, the original stays on your machine. The query crosses the wire. The answer comes back. Nothing else moves, and there is no platform in the middle holding your data as leverage.

Is your current setup giving you that kind of control?

---

*trip2g is open source and self-hosted. To set up per-subgraph access control, start with `mcp:add <your-node-url>` on each side and configure the shared zone in the base settings. Sources linked inline; more in [the research notes](../../dev/market_research_notebooklm.md).*
