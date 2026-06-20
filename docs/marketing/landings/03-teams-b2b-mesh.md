# Landing: Teams & B2B mesh

Audience: privacy-conscious orgs and B2B partners. Lead with sovereignty + federation + agents. Honest about un-shipped enterprise features.
Voice/positioning: see [`../00_brief.md`](../00_brief.md).

---

## Hero

**Connect your team's knowledge and your partners' into one access-controlled mesh your agents can query. Nothing uploaded to a central server. Nothing you can't revoke.**

Your team's notes stay on your infrastructure. Your partner's data stays on theirs. trip2g connects the two into a federated mesh: one agent question reaches both sides, with answers cited back to their sources, without either party merging their systems or handing files to a third party.

`[ Start free → ]`   `[ Talk to us ]`   `[ mcp:add https://trip2g.com/_system/mcp ]`

Open source. Self-hosted. You keep the keys.

---

## The problem (say it plainly)

Your team's best thinking is scattered, and sharing any of it with a partner is still a headache:

- Tribal knowledge lives in people's heads and local files. The engineer who onboarded last quarter knows where the decisions are. The one who left doesn't. New agents need that context, and it's not in your wiki.
- Corporate search tools require you to hand everything over. Glean and tools like it are powerful, and they work by scraping your internal tools into a centralized index. That index sits in a vendor's cloud. If your data has any sensitivity, that's a problem you can't design around.
- Sharing with partners is all-or-nothing. You can give a supplier a Notion page or a Google Drive folder, but you can't give their agent access to one specific folder of your internal research while keeping the rest of your vault air-gapped. There's no "share a slice, revoke it later."
- Every silo is a blind spot for agents. Your agents work on what they can reach. If your R&D notes are in one system and your partner's specs are in another, your agent is working with half the picture, or none of it.

---

## How it works

1. Connect a knowledge base. Point trip2g at a Markdown/Obsidian folder on your infrastructure. One sync and it becomes a live knowledge-base node: search, an MCP endpoint, access rules.
2. Set the boundary. Mark what's internal, what's shared with a specific partner, what's public. Per-folder, per-node. Revocable without a support ticket.
3. Federate. Link your node to a partner's trip2g instance. Now your agent (or theirs) can run a federated query and get an answer assembled across both sides, citing both sources. Their data never leaves their server. Yours never leaves yours.

The mesh routes the query. It doesn't store a copy.

---

## What this actually feels like

> Your agent asks: *"What do we and Acme Corp have aligned on for the Q3 integration spec?"*
>
> Your agent queries your internal research notes on your server. At the same time, it peers with Acme's trip2g node and queries the folder they've shared with you. It returns a merged answer citing both sources: your internal docs and the section of theirs they've granted access to.
>
> Acme's full vault stays on their infrastructure. Your full vault stays on yours. The agent sees only what each party explicitly shared.

That's the difference from a central corporate search tool: nothing was scraped, nothing lives in a vendor's index, and Acme can revoke the shared folder tomorrow without filing a request with you.

---

## Why this is different from Glean, Onyx, and tools like them

Glean and Onyx are centralized scrapers. They connect to your Slack, Drive, Confluence, and Jira, pull everything into an enterprise index, and let your team search across it. They're good at that job. The tradeoff is that your data lives in their system, under their access model.

trip2g is a different architecture. There is no central index. Each node holds its own data. A query routes across nodes and assembles an answer on the fly. The server is active middleware, a semantic gateway, not a storage bucket.

This matters when:
- You can't put certain data in a vendor's cloud (regulatory, contractual, or just preference).
- You want to share one folder with a partner without merging your whole corpus into their system.
- You need to revoke access without depending on a vendor to honor the request.

What Glean does better today: 100+ native connectors for corporate SaaS tools (Slack, Salesforce, etc.), a polished enterprise search UI, and a deployment model that most IT teams already know. If your whole corpus is in those tools and data residency isn't a constraint, Glean is more mature.

---

## Own your data (for real)

- Local-first. Your server is the primary copy. No vendor outages taking your knowledge offline.
- No shadow copy. trip2g publishes; it doesn't store. The server is a contentless middleware layer. The originals stay with you.
- Per-node access control. Grant access to a node, a folder, or a subgraph. Set it to expire. Revoke it on camera if you want.
- Markdown. If trip2g disappeared tomorrow, your files are still plain text on your disk.

---

## Who it's for

- Small technical teams who want their agents to know the internal context (decisions, architecture notes, runbooks) without pushing it all to a cloud vector store.
- Organizations with data residency constraints: regulated industries, legal, healthcare, defense-adjacent, where "no vendor ever touches this" is a hard requirement.
- Partner relationships where you need to share a specific research slice with a supplier, client, or joint-venture partner. Auditable, revocable, without merging IT systems.
- Teams already using Obsidian or Markdown who want to graduate from a shared folder to a queryable knowledge mesh their agents can actually use.

---

## What's shipped vs. what's on the roadmap

trip2g is a strong fit for technical teams willing to self-host. Some enterprise features aren't there yet.

Shipped:
- Per-base and per-folder access control, revocable
- Federated search across trusted nodes
- MCP endpoint for AI agents (Claude, Cursor, and anything that speaks MCP)
- Self-hosted, one Go binary + Docker
- Free sandbox to try without installation

On the roadmap (not shipped):
- SSO / SAML integration (required for most corporate IT deployments)
- SCIM provisioning
- Audit log export (CSV/SIEM integration)
- Enterprise SLA and support contracts

If SSO or SCIM is a hard requirement today, trip2g isn't ready for you yet. We'd rather say that than have you find out after setup.

---

## Pricing

- Self-host: free. Open source (MIT). Your infrastructure, your cost.
- Free sandbox: 100 MB, no installation, try it now.
- Managed hosting: when you'd rather not manage the server. No sandbox limit.

For context: Glean starts around $45-65/user/month with a ~$50k minimum contract. Onyx (Danswer) self-hosted is free; their cloud tier runs ~$20/user/month. trip2g's managed tier is a fraction of that. You're trading connector breadth and enterprise polish for data sovereignty and no scraping.

`[ Self-host → ]`   `[ Try the sandbox → ]`   `[ Talk to us ]`

---

## Honest FAQ

**You said no central server. Who runs the "federation" layer?**
Federation is direct, node-to-node. When your agent queries across your node and a partner's, it connects to their trip2g instance directly. There's no trip2g relay server in the middle holding the data. We provide the protocol and software; you run the infrastructure.

**What stops a partner from downloading everything we share with them?**
Access control limits what their agent (or their people) can query. But once a query returns text, that text is on their side. This is the same constraint as any API-based data sharing: you're sharing a queryable slice, not a file, but the answers cross the boundary. Set access at the folder level, log queries, and revoke when the relationship changes. We're working on query-level audit logs; they're not shipped yet.

**Our IT team needs SSO before they'll approve anything.**
Noted above: SSO isn't shipped yet. If that's a blocker, you'll need to wait or self-host behind your existing identity layer.

**We use Slack, Notion, Salesforce, not Markdown. Does this work?**
trip2g's native format is Markdown. For other sources, the current path is either exporting to Markdown or writing an adapter that bridges them. If your whole corpus lives in corporate SaaS and you have no Markdown files, trip2g isn't the right fit today. If you have a technical team and the willingness to maintain an adapter, it's doable.

**Is this actually different from running a private RAG pipeline?**
A private RAG pipeline uploads your documents to an embedding store, usually in a cloud vector database. trip2g doesn't store a copy. The server routes queries to your files and returns answers; the files stay where they are. It also federates across multiple nodes, which a single RAG pipeline can't do without merging everything into one index.

**Isn't a giant context window going to make this pointless?**
If frontier models ship 10M-token windows with no "lost in the middle" decay and sub-second latency, the retrieval argument weakens. The sovereignty and access-control arguments don't: you still don't want sensitive files in a vendor's cloud, and you still want to share a slice with a partner without merging your entire corpus into theirs. We say this plainly rather than pretend the question away.

---

`[ Start free → ]`   `[ Read the docs ]`   `[ Star on GitHub ]`
