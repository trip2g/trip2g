I have the humanizer rules. Now I'll clean the doc, stripping the preamble and code fence, applying all hard constraints while preserving every fact, link, price, and roadmap caveat.

# Landing: Teams and B2B mesh

Audience: privacy-conscious orgs and B2B partners. Lead with sovereignty, federation, and agents. Honest about un-shipped enterprise features.
Voice/positioning: see [`../00_brief.md`](../00_brief.md).

---

## Hero

**Connect your team's knowledge and your partners' into one access-controlled mesh your agents can query. Nothing uploaded to a central server. Nothing you can't revoke.**

Your team's notes stay on your infrastructure. Your partner's data stays on theirs. trip2g connects the two into a federated mesh: one agent question reaches both sides, with answers cited back to their sources, and neither party has to merge their systems or hand files to a third party.

`[ Start free → ]`   `[ Talk to us ]`   `[ mcp:add https://trip2g.com/_system/mcp ]`

Open source. Self-hosted. You keep the keys.

---

## The problem (say it plainly)

Your team's best thinking is scattered, and sharing any of it with a partner is still a headache:

- Tribal knowledge lives in people's heads and local files. The engineer who onboarded last quarter knows where the decisions are. The one who left doesn't. New agents need that context, and it's not in your wiki.
- Corporate search tools require you to hand everything over. Glean and tools like it work by scraping your internal apps into one centralized index in a vendor's cloud. They're good at that. But the index is theirs, and if any of your data is sensitive, that's a problem you can't design around.
- Sharing with a partner is all-or-nothing. You can give a supplier a Notion page or a Drive folder, but you can't give their agent access to one specific folder of your research while keeping the rest of your vault sealed off. There's no "share a slice, revoke it later."
- Every silo is a blind spot for agents. Your agents work on what they can reach. If your R&D notes are in one system and your partner's specs are in another, your agent is working with half the picture, or none of it.

---

## How it works

1. Connect a knowledge base. Point trip2g at a Markdown/Obsidian folder on your infrastructure. One sync and it becomes a live knowledge-base node: search, an MCP endpoint, access rules.
2. Set the boundary. Mark what's internal, what's shared with a specific partner, what's public. Per-folder, per-node. Revocable without a support ticket.
3. Federate. Link your node to a partner's trip2g instance. Now your agent (or theirs) can run a federated query and get an answer assembled across both sides, citing both sources. Their data never leaves their server. Yours never leaves yours.

The mesh routes the query. It doesn't store a copy. Peering is direct, node to node, over HTTPS, authenticated per call with a signed token. No trip2g relay sits in the middle holding the data.

---

## What this actually feels like

> Your agent asks: *"What do we and Acme Corp have aligned on for the Q3 integration spec?"*
>
> Your agent queries your internal research notes on your server. At the same time, it peers with Acme's trip2g node and queries the folder they've shared with you. It returns a merged answer citing both sources: your internal docs and the section of theirs they've granted access to.
>
> Acme's full vault stays on their infrastructure. Your full vault stays on yours. The agent sees only what each party explicitly published.

That's the difference from a central corporate search tool: nothing was scraped, nothing lives in a vendor's index, and Acme can revoke the shared folder tomorrow without filing a request with you.

---

## Why this is different from Glean, Onyx, and tools like them

Glean and Onyx are single-org by design. They connect to one company's Slack, Drive, Confluence, and Jira, sync that company's access rules into one index, and let that company's team search across it. Onyx is the closest thing in the open-source world, and it does this well. But the architecture has one owner. There is no concept of peering Onyx with another company's Onyx, or Glean with another company's Glean, under a shared contract. They mirror one org's ACLs into one index. They can't reach across to a base your partner owns and runs.

trip2g is a different shape. There's no central index. Each node holds its own data. A query routes across nodes and assembles an answer on the fly. The server is active middleware, a semantic gateway, not a storage bucket. The one thing it does that the others structurally can't: peer two *separately-owned* bases under per-base access control, so a query can cross a company boundary without either side merging into the other.

We looked. No funded product does that today. The MCP gateways (Bifrost, TrueFoundry, AWS AgentCore) federate tools and agent registries, but inside one company's trust domain. The data-space consortia (Catena-X, IDS, Gaia-X) federate structured data under heavy standards committees, not human-authored markdown. Cross-company knowledge sharing under access control is, as far as we can find, an academic concept and not a shipping product. That's the gap trip2g is built for.

This matters when:
- You can't put certain data in a vendor's cloud, for regulatory, contractual, or plain preference reasons.
- You want to share one folder with a partner without merging your whole corpus into their system.
- You need to revoke access without depending on a vendor to honor the request.

What Glean does better today: 100+ native connectors for corporate SaaS (Slack, Salesforce, and the rest), a polished enterprise search UI, and a deployment model most IT teams already recognize. If your whole corpus lives in those tools and data residency isn't a constraint, Glean is more mature. We're not going to pretend otherwise.

---

## Own your data (for real)

- Local-first. Your server is the primary copy. No vendor outage takes your knowledge offline.
- No shadow copy. trip2g publishes; it doesn't store. The server is a contentless middleware layer. The originals stay with you.
- Per-node access control. Grant access to a node, a folder, or a subgraph. Set it to expire. Revoke it on camera if you want.
- Markdown. If trip2g disappeared tomorrow, your files are still plain text on your disk.

---

## Who it's for

- Small technical teams who want their agents to know the internal context (decisions, architecture notes, runbooks) without pushing it all to a cloud vector store.
- Organizations with data residency constraints: regulated industries, legal, healthcare, defense-adjacent, where "no vendor ever touches this" is a hard line.
- Partner relationships where you need to share a specific research slice with a supplier, client, or joint-venture partner. Revocable, without merging IT systems. This is the case nobody else serves, and it's the one we'd most like to hear about.
- Teams already on Obsidian or Markdown who want to graduate from a shared folder to a queryable knowledge mesh their agents can actually use.

---

## What's shipped vs. what's on the roadmap

trip2g is a strong fit for technical teams willing to self-host. Several enterprise features aren't there yet, and the federation-first path exists partly because the enterprise-trust path isn't finished. We'd rather say that here than have you discover it in a security review.

Shipped:
- Per-base and per-folder access control, revocable
- Federated search across trusted nodes, including across separate owners
- MCP endpoint for AI agents (Claude, Cursor, anything that speaks MCP)
- Self-hosted, one Go binary plus Docker, embedded SQLite
- Free sandbox to try without installation

On the roadmap (not shipped):
- SSO / SAML / OIDC-as-IdP and SCIM provisioning. Today the OAuth login is GitHub/Google for the panel only. No MFA.
- Comprehensive audit trail. A few events are logged; query-level audit is not.
- Enforced network isolation. Agents currently run on host-network and can reach infrastructure they shouldn't. The isolation plan is designed, not deployed.
- HA control-plane. Today it's a single binary and a single SQLite file. Failover is manual.
- Enterprise SLA and support contracts.

If SSO, SCIM, a real audit trail, or enforced isolation is a hard requirement today, trip2g isn't ready for your security team yet. We're building toward it. We're saying so upfront so you can decide now rather than after setup.

---

## Pricing

We price the box and the federation, never per seat.

- Community: free, self-host. Open source. Full single-instance node, MCP-native, per-folder access control. Your infrastructure, your cost.
- Free sandbox: 100 MB, no installation, try it now.
- Federation / Team: a flat monthly fee per operating organization for instance-to-instance federation at scale. Flat per org, not per user, because the value is "I run a node and peer with partners," not "I added a seat."
- Sovereign / Enterprise: custom, and it doesn't exist until the enterprise-trust controls above ship. We won't sell you a tier whose security features aren't built.

For context: Glean runs around $45-50/user/month with six-figure minimums and a loaded TCO north of $300k/year. Onyx self-hosted is free; their cloud tiers run roughly $20-200/month. trip2g's per-org federation tier is anchored well below Glean's per-seat math. You're trading connector breadth and enterprise polish for data sovereignty, cross-company federation, and no scraping.

`[ Self-host → ]`   `[ Try the sandbox → ]`   `[ Talk to us ]`

---

## Honest FAQ

**You said no central server. Who runs the "federation" layer?**
Federation is direct, node to node. When your agent queries across your node and a partner's, it connects to their trip2g instance directly, authenticated per call with a signed token. No trip2g relay sits in the middle holding the data. We provide the protocol and software; you run the infrastructure.

**What stops a partner from downloading everything we share with them?**
Access control limits what their agent or their people can query. But once a query returns text, that text is on their side. This is the same constraint as any API-based sharing: you're sharing a queryable slice, not a file, but the answers cross the boundary. Set access at the folder level and revoke when the relationship changes. Query-level audit logs are on the roadmap, not shipped yet, so don't count on them today.

**Our IT team needs SSO before they'll approve anything.**
Then trip2g isn't ready for you yet. SSO/SAML/SCIM aren't shipped. You'd either wait for them or self-host behind your existing identity layer at the network edge. We list this as a roadmap item, not a feature.

**We use Slack, Notion, Salesforce, not Markdown. Does this work?**
trip2g's native format is Markdown. For other sources the current path is exporting to Markdown or writing an adapter that bridges them. There's no native connector catalog yet. If your whole corpus lives in corporate SaaS and you have no Markdown, trip2g isn't the right fit today. If you have a technical team and the willingness to maintain an adapter, it's doable.

**Is this actually different from a private RAG pipeline?**
A private RAG pipeline uploads your documents into an embedding store, usually a cloud vector database. trip2g doesn't store a copy. The server routes queries to your files and returns answers; the files stay where they are. It also peers across multiple nodes, including ones another company owns, which a single RAG pipeline can't do without merging everything into one index under one owner.

**Isn't a giant context window going to make this pointless?**
If frontier models ship 10M-token windows with no "lost in the middle" decay and sub-second latency, the retrieval argument weakens. The sovereignty and access-control arguments don't: you still don't want sensitive files in a vendor's cloud, and you still want to share a slice with a partner without merging your entire corpus into theirs. We say this plainly rather than pretend the question away.

---

`[ Start free → ]`   `[ Read the docs ]`   `[ Star on GitHub ]`
