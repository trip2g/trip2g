# trip2g: marketing brief and content plan

Source of truth for positioning, voice, and the content we're producing here.
Grounding research (read it before writing): [`../dev/market_research_notebooklm.md`](../dev/market_research_notebooklm.md).

## What trip2g is (one line)
An open-source, self-hosted way to turn your Markdown/Obsidian notes into a **federated knowledge mesh** that AI agents can query across trust tiers (personal, friends, company, public) without uploading your files to anyone's cloud.

The name is *trip to graph*: every base is a graph, the bases connect into one larger graph, and agents walk it.

## Beachhead (who we sell to first)
**Small trusted peer circles**: 3-5 collaborators (co-readers, study partners, project builders) who already keep notes in Obsidian and are sick of the **Git-sync bottleneck**. They want a shared "community brain" their agents can search across, without any member giving up ownership of their files, and without a SaaS in the middle.

Why them first: they're the first group that actually *exercises* the two unique features, federation and per-base access control. (Solo users don't need federation; orgs are blocked by enterprise features we haven't shipped, like SSO/SCIM.)

Ranking from the research: 1) Peer circles · 2) Scattered researchers (easiest to reach, but frugal; they pick free Quartz) · 3) Privacy orgs (will pay $20-65/user, but need un-shipped enterprise features) · 4) B2B partners (high value, hardest to land).

## Headline to test first
> **Federate your local notes into a private memory mesh for your AI agents.**

## Voice (how we write, non-negotiable)
- Human, specific, concrete. Real examples, real numbers, real commands. If a sentence could appear on any SaaS site, cut it.
- Skeptical and honest. We name the trade-offs and the one real risk (below). No hype.
- Plain language. Short sentences. Say the thing.
- Ban-list (AI slop): "in today's fast-paced world", "seamless", "unleash", "elevate", "game-changer", "robust", "supercharge", "revolutionize", "harness the power", "dive in", "it's important to note", empty tricolons, em-dash pile-ups, and listicles that pad.
- Show, don't claim. Prefer a query trace / a diff / a screenshot over an adjective.
- Cite. Articles link to primary sources (see the essay library). Don't assert market claims without a link.

## Voice calibration (the founder's actual voice, match it)
Write like a builder's working log, not a brand page.
- First person, direct. "I did X. It broke at 3am. Here's the fix."
- Concrete over abstract: a real command, a file path, a number, a short code snippet, the actual URL. Show the thing, don't describe it.
- Honest and a little self-deprecating. Name what was hard, what failed, what came out cringe ("burned out for two years", "I did this badly the first time", "it looked cringe").
- Dry, deadpan humor. Understatement, not jokes.
- Short sentences. Plain words. Cut the throat-clearing.
- Ask the reader a real question now and then ("Do you automate this yet?", "What does the world look like in a year?").
- No hype, no corporate gloss. Assume a smart, technical reader who's tired of being sold to.

The energy: someone who has actually been awake debugging this, telling you what they found.

## Vocabulary that lands (use these; they're load-bearing)
- "Digital sovereignty": your knowledge should belong to you, not a platform.
- "Local-first": your device is the primary copy; the network is optional (Ink & Switch).
- "Contentless CMS / no shadow copy": the server never keeps a private copy of your notes.
- "Shared second brain / collective memory": the bridge from personal PKM to the mesh.
- "Active middleware": a real-time semantic API gateway for agents (vs a static site).
- "Federated query / query trace": one agent question, answered across trusted bases.

## Debates we take a side on (thought-leadership hooks)
- The Git-sync bottleneck: Git guarantees conflicts on files like `workspace.json`, which makes it broken for real-time note collaboration.
- RAG is a band-aid: passive retrieval is transitional; agents need structured, owned, federated context.
- The "original sin" of partitioning: folders + vector shards sever the connections an agent needs.
- Model layer vs context layer: the model is commoditizing; the context layer is the moat.
- All-or-nothing sharing: why must you choose between total privacy and total public exposure?

## The one risk we don't hide
If frontier models ship 10M+ token windows with sub-second latency and no "lost-in-the-middle" decay, a federated mesh becomes overhead for everyone except air-gapped/extreme cases. Our honest answer: ownership + access control + cost don't go away even if context windows grow. We say this plainly. We don't pretend the risk isn't real.

---

## Content plan (what lives in this folder)

### Landings (`landings/`)
1. `01-main.md`: primary trip2g.com hero. Beachhead = peer circles; lead with the mesh/graph + agents. (drafted)
2. `02-obsidian-sync-alternative.md`: capture the "Obsidian Publish/Sync alternative" search. Position trip2g as publish + AI-queryable + federated, honest about overlap and who NOT to target.
3. `03-teams-b2b-mesh.md`: privacy orgs + B2B partners. "Connect your team's and partners' knowledge into one access-controlled mesh your agents can query, no central silo." (the willing-to-pay bet.)

### Articles (`articles/`), 5 total, each links to primary sources
1. `01-multiplayer-obsidian.md`: *Multiplayer Obsidian: build a community brain without a SaaS middleman.* (Peer circles / beachhead.) (drafted)
2. `02-git-bottleneck.md`: *The Git bottleneck: why your Obsidian-Git setup is guaranteed to fail in the agentic era.* (Top pain; cite the conflict mechanics.)
3. `03-rag-is-a-bandaid.md`: *RAG is a band-aid: from passive retrieval to a federated context mesh.* (Cite Lost-in-the-Middle, MemGPT, GraphRAG.)
4. `04-share-a-slice.md`: *Digital sovereignty by design: share a slice of your research without merging your whole life.* (Privacy/B2B; cite Local-First, Kleppmann.)
5. `05-fragmentation-tax.md`: *The fragmentation tax: why your AI is blind to your 12 digital silos.* (Top-of-funnel; cite cross-app context discourse.)

### Video scripts (`video-scripts/`), 5 short scripts (60-90s, hook + demo + CTA)
1. The 30-second federated query trace (agent pulls from your vault + a friend's node, cites both, no exports).
2. Git-conflict horror to trip2g fix (show a `workspace.json` merge conflict, then the same edit syncing clean).
3. "Watch your agents work": open several bases, see agents reading/editing live (the wow demo; frame as transparency/control, not gimmick).
4. Share a slice: give a partner's agent one folder, revoke it on camera; the rest stays air-gapped.
5. From notes to API in 60s: `mcp:add` a vault, then an agent answers a question grounded on it.

## Essay library: mirror their tone/structure and cite them
- Local-First Software: You Own Your Data, in spite of the Cloud (Ink & Switch, https://www.inkandswitch.com/local-first/): coined a category + a 7-ideal checklist.
- Building a Second Brain (Tiago Forte, https://www.buildingasecondbrain.com/): shifted "collect" to "organize for outcomes" (PARA).
- MemGPT: Towards LLMs as Operating Systems (Packer et al., https://arxiv.org/abs/2310.08560): the "LLM OS" RAM-vs-disk metaphor.
- Lost in the Middle: How Language Models Use Long Contexts (Liu et al., https://arxiv.org/abs/2307.03172): hard evidence against the "infinite context" hype.
- Evergreen Notes (Andy Matuschak, https://notes.andymatuschak.org/Evergreen_notes): knowledge as compound interest.
- A Brief History & Ethos of the Digital Garden (Maggie Appleton, https://maggieappleton.com/garden-history): library to garden metaphor.
- Data agency / AT Protocol (Martin Kleppmann, https://martin.kleppmann.com/): protocol shifts framed as data-agency rights.
- From Local to Global: A GraphRAG Approach (Edge et al., Microsoft, https://arxiv.org/abs/2404.16130): graph structure fixes a core RAG gap.

> Structure to steal from the best of these: **name a new category or metaphor → one sharp, falsifiable claim → evidence → a checklist or framework the reader can act on.** That's why they spread.
