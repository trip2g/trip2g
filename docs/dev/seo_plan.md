# trip2g SEO Plan

**Scope:** trip2g.com — self-hosted markdown vault / MCP memory server for AI agents.  
**Audience:** Developers building AI agents, LLM-Wiki pattern followers, Obsidian power users.  
**Date:** June 2026.

---

## 0. Site Crawlability — Critical Finding

**The good news:** Pages appear to be **server-side rendered** — the fetched HTML includes actual
text content, not empty divs. Google and LLM crawlers can read the content.

**The bad news (observed on live site):**

| Problem | Evidence |
|---|---|
| No `<meta name="description">` on any page checked | Confirmed on `/en/user/agent_memory`, `/en/user/token_economy_bench`, `/en/user/use_cases`, `/en/thoughts` — none returned a meta description |
| No `<link rel="canonical">` | Confirmed in engine audit (`docs/dev/seo_audit.md:47`) — tag is entirely absent from all pages |
| No structured data / JSON-LD | Confirmed on all pages fetched — no SoftwareApplication, FAQ, HowTo, or Article schema |
| `/en/` homepage returns 404 | `https://trip2g.com/en/` is 404; the sitemap lists `/en/` paths but homepage lives at `/` |
| `robots.txt` returns only "open" | No crawl directives, no Sitemap: pointer — crawlers don't auto-discover the XML sitemap |
| Missing hreflang on bilingual pages | Both `/en/` and `/ru/` versions exist; no hreflang signals to avoid duplicate content penalty |
| No `og:description` or `twitter:description` | Falls back to nothing when `description` frontmatter is absent (confirmed in engine code) |

These are the highest-priority technical fixes — they undercut everything else.

---

## 1. Keyword Strategy

### The Audience's Search Behavior

Developers building AI agents search with high specificity. They are Googling, but increasingly
asking ChatGPT/Perplexity/Claude directly. Both channels must be served.

### Primary Clusters (High Intent, Reachable)

#### Cluster A — MCP Memory (HOTTEST, low-to-medium competition)
The MCP ecosystem is new. Competition is thin and the intent is perfect.

| Keyword | Intent | Competition | Priority |
|---|---|---|---|
| `MCP server memory` | Tool-shopping | Low | ★★★★★ |
| `MCP long-term memory` | Tool-shopping | Low | ★★★★★ |
| `Claude Code memory MCP` | Tool-shopping | Low | ★★★★★ |
| `persistent memory MCP server` | Tool-shopping | Low | ★★★★★ |
| `MCP knowledge base` | Tool-shopping | Low-Med | ★★★★ |
| `MCP agent context` | Informational | Low | ★★★★ |
| `claude desktop memory server` | Tool-shopping | Low | ★★★★ |

**Rationale:** MCP is ~18 months old. The directory of MCP servers is still sparse. A focused
landing page here can rank #1 within weeks, not months.

#### Cluster B — AI Agent Memory (Medium competition, growing fast)
| Keyword | Intent | Competition | Priority |
|---|---|---|---|
| `AI agent long-term memory` | Informational + shopping | Med | ★★★★ |
| `agent memory self-hosted` | Tool-shopping | Low | ★★★★★ |
| `LLM memory persistence` | Informational | Med | ★★★ |
| `agent knowledge base` | Tool-shopping | Med | ★★★★ |
| `self-hosted agent memory` | Tool-shopping | Low | ★★★★★ |
| `AI agent context window management` | Informational | Low-Med | ★★★ |

#### Cluster C — LLM Wiki / Karpathy Pattern (Niche, very low competition)
Karpathy's "LLM Wiki" concept has been circulating on X/HN. Direct targeting.

| Keyword | Intent | Competition | Priority |
|---|---|---|---|
| `LLM wiki` | Informational | Very Low | ★★★★★ |
| `Karpathy LLM wiki implementation` | Informational | Very Low | ★★★★★ |
| `personal knowledge base for AI` | Informational | Low | ★★★★ |
| `AI-queryable wiki self-hosted` | Tool-shopping | Very Low | ★★★★★ |

#### Cluster D — Obsidian Integration (Medium competition, large audience)
| Keyword | Intent | Competition | Priority |
|---|---|---|---|
| `Obsidian MCP server` | Tool-shopping | Low | ★★★★★ |
| `Obsidian AI agent integration` | Tool-shopping | Low-Med | ★★★★ |
| `Obsidian publish alternative` | Tool-shopping | Med | ★★★ |
| `Obsidian two-way sync self-hosted` | Tool-shopping | Low | ★★★★ |
| `Obsidian as knowledge base for AI` | Informational | Low | ★★★★ |

#### Cluster E — Head Terms (Aspirational, long-term)
These are competitive but worth targeting via content, not direct homepage optimization.

| Keyword | Intent | Competition | Priority |
|---|---|---|---|
| `AI agent memory` | Informational | High | ★★ (content-only) |
| `self-hosted RAG` | Tool-shopping | Med-High | ★★★ |
| `vector database alternative` | Tool-shopping | High | ★★ |
| `digital garden` | Informational | High | ★ (secondary) |

### Long-Tail Wins (Target immediately via content)
- "how to give Claude long-term memory"
- "MCP server for agent notes"
- "self-hosted memory for AI assistants without vector database"
- "Obsidian vault as MCP context"
- "token-efficient agent memory retrieval"
- "agent memory 15x cheaper than RAG"

---

## 2. Technical SEO

### Fix 1: Add `<link rel="canonical">` to Every Page (CRITICAL)

**Problem:** Absent entirely. The same content is reachable at multiple URLs (main domain,
custom domains, alternate-permalink redirects, `/index` aliases).

**Fix:** Emit canonical in `internal/defaulttemplate/views.html`. The logic already exists in
`ogURLForNote` (`endpoint.go:419-447`) — reuse it:

```html
<link rel="canonical" href="{% full absolute URL from ogURLForNote %}">
```

The `seo_audit.md` confirms this is the right reference point.

### Fix 2: Auto-generate Meta Descriptions (CRITICAL)

**Problem:** Descriptions only emit when `description:` frontmatter is set. Most pages have none.
Without this, Google generates its own snippet (usually bad) and LLM crawlers get no summary.

**Fix A (engine):** Add first-paragraph fallback in `resolve.go` — the `PartialRenderer().Introduce()`
method is already used for magazine cards (`views.html:362`). Truncate to 155 chars. Remove
the `// TODO` comment from `endpoint.go:401`.

**Fix B (content):** Add explicit `description:` frontmatter to the 10 most important pages
immediately (list in §3 below).

### Fix 3: Add hreflang for EN/RU pairs (HIGH)

The sitemap confirms full `/en/` and `/ru/` mirroring. Without hreflang, Google sees duplicate
content and may penalize or split ranking signals.

Add in `<head>` for every bilingual page:
```html
<link rel="alternate" hreflang="en" href="https://trip2g.com/en/user/agent_memory">
<link rel="alternate" hreflang="ru" href="https://trip2g.com/ru/user/agent_memory">
<link rel="alternate" hreflang="x-default" href="https://trip2g.com/en/user/agent_memory">
```

### Fix 4: robots.txt — Add Sitemap Pointer (HIGH)

Current robots.txt returns only "open" (likely a bare page, not a proper robots.txt).

**Fix:** Serve a proper text file at `/robots.txt`:
```
User-agent: *
Allow: /

Sitemap: https://trip2g.com/sitemap.xml
```

The sitemap at `/sitemap.xml` exists (680 URLs confirmed) but crawlers won't auto-discover it
without this pointer.

### Fix 5: Fix `/en/` 404 (HIGH)

`https://trip2g.com/en/` returns 404. The sitemap references `/en/` paths. This breaks
Googlebot's crawl of the English section's root. Either:
- Redirect `/en/` → `/en/user` (the docs index), or
- Create an English homepage at `/en/`

### Fix 6: Add Structured Data (MEDIUM)

No JSON-LD anywhere. Opportunities:

**SoftwareApplication** schema on the homepage:
```json
{
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  "name": "trip2g",
  "applicationCategory": "DeveloperApplication",
  "description": "Self-hosted MCP memory server and markdown vault for AI agents. 15× cheaper token retrieval than whole-note dumps.",
  "operatingSystem": "Linux, macOS, Docker",
  "offers": {"@type": "Offer", "price": "0", "priceCurrency": "USD"},
  "url": "https://trip2g.com",
  "license": "https://opensource.org/licenses/MIT"
}
```

**FAQPage** schema on the token economy benchmark page (the Q&A structure is already there).

**HowTo** schema on the agent_memory quickstart page (numbered steps already present).

**Article** schema on all `/en/thoughts/` posts.

### Fix 7: Open Graph Tags Audit (MEDIUM)

Confirm all pages have `og:title`, `og:description`, `og:image`, `og:url`.
Currently `og:description` is absent when `description:` frontmatter is missing (same bug as §Fix 2).
The OG image appears to link Yandex Cloud signed URLs — ensure these are stable/permanent for
social sharing previews.

### Fix 8: Page Speed

No page speed data was observable. Given the server is Docker-based:
- Ensure static assets (CSS/JS) have long cache headers and are minified.
- The SSR approach is good for Core Web Vitals. Verify LCP (Largest Contentful Paint) image
  is preloaded via `<link rel="preload">`.
- Consider adding `<link rel="preconnect">` for Yandex Cloud CDN origin.

---

## 3. On-Page / Content

### Title & Meta Description Rewrites

**Priority pages:**

| Page | Current Title | Recommended Title (≤60 chars) | Recommended Meta Description (≤155 chars) |
|---|---|---|---|
| Homepage (`/`) | "trip2g" (implied) | `trip2g — Self-Hosted MCP Memory for AI Agents` | `Persistent long-term memory for AI agents. Self-hosted MCP server, Obsidian sync, and 15× cheaper token retrieval than whole-note RAG. Docker in 30 seconds.` |
| `/en/user/agent_memory` | `Long-term memory for AI agents with trip2g` | `AI Agent Long-Term Memory with MCP — trip2g` | `Give your AI agent persistent memory via MCP. search → expand → read_section costs 15× fewer tokens than full-note retrieval. Self-hosted, Docker, Obsidian sync.` |
| `/en/user/token_economy_bench` | `Token economy: measured on this site` | `MCP Memory Token Benchmark: 15–37× Savings vs RAG` | `Real benchmark: focused section retrieval costs 15× fewer tokens than whole-note dumps, 23× cheaper than grep. Open methodology, reproducible Python script.` |
| `/en/user/agent_status` | `Auto-broadcast your work status to the team` | `Team Status Broadcasting via Claude Code Hooks` | `Auto-sync your Claude Code session summaries to a shared trip2g vault. Session-end hook redacts secrets, LLM-summarizes, and broadcasts structured status to teammates.` |
| `/en/user/use_cases` | `Use cases` | `trip2g Use Cases: AI Agents, Digital Gardens & More` | `From AI consultant knowledge bots to Obsidian-powered digital gardens. See how developers use trip2g as self-hosted MCP memory and public knowledge base.` |
| `/en/user` (docs index) | `Documentation` | `trip2g Docs — MCP Memory Server & Obsidian Sync` | `Full documentation for trip2g: self-hosted MCP memory, Obsidian two-way sync, federation, token economy, and AI agent integration.` |

### Heading Structure

**Homepage H1:** "Your second brain was always meant to be shared." — This is poetic but SEO-blind.
Visitors arriving from "MCP memory server" search need to recognize the product instantly.

**Recommendation:** Add an invisible (sr-only) or secondary H1 alongside the tagline, OR change
the H1 to something like:
> "Self-Hosted MCP Memory for AI Agents"

And demote the tagline to a subheading or `<p class="hero-subtitle">`.

**agent_memory page H2s:** The structure is solid (numbered steps). Add `<h2>Why trip2g for AI
agent memory?</h2>` near the top to capture informational queries and give Google a clear topic.

### Internal Linking (Exploit the Wikilink Graph)

The `[[wikilink]]` graph is trip2g's biggest untapped internal-link asset. Every note already
links to related notes. SEO actions:

1. **Ensure wikilinks resolve to `<a href>` tags** in rendered HTML (confirm engine renders them,
   not just displays them as text). If they render, you already have a powerful internal link graph.

2. **Create hub pages** that aggregate clusters:
   - `/en/user/mcp-memory` — hub for all MCP agent memory content (links to agent_memory,
     token_economy_bench, agent_status, federation)
   - `/en/user/obsidian-mcp` — hub for Obsidian integration content

3. **Link from homepage** to the benchmark page and the agent_memory quickstart — currently the
   homepage is the only entry point and may not pass PageRank to these high-value pages.

4. **Cross-link the benchmark** from agent_memory quickstart: "See our token economy benchmark
   showing 15–37× savings vs whole-note retrieval."

### Content Gap: Comparison Pages (High-ROI)

These are the clearest opportunities to intercept high-intent queries:

**Article 1: "trip2g vs Vector Database (Pinecone / Weaviate / Qdrant) for Agent Memory"**
- Target: "self-hosted RAG alternative", "vector db for AI agents", "agent memory without Pinecone"
- Angle: trip2g uses full-text + vector hybrid; no embeddings infra required for most use cases;
  Markdown-native; 23× cheaper token retrieval than grep-over-files.
- Note: `docs/dev/why_not_qdrant.md` exists internally — this is the source material.

**Article 2: "Obsidian as AI Agent Memory: Complete MCP Setup Guide"**
- Target: "Obsidian MCP server", "use Obsidian with Claude", "Obsidian long-term memory AI"
- Angle: step-by-step with actual config, memcli one-liner, live demo GIF.

**Article 3: "How to Give Claude Code Persistent Memory (trip2g MCP Setup)"**
- Target: "Claude Code memory", "Claude Code long-term memory", "Claude Code MCP memory"
- Angle: Practical tutorial. This is a top-5 question in the Claude Code community right now.

**Article 4: "LLM Wiki Pattern: Karpathy's Idea, Production Implementation"**
- Target: "LLM wiki", "Karpathy personal wiki AI", "AI-queryable knowledge base"
- Angle: Acknowledge the origin, show trip2g as a concrete implementation with numbers.

**Article 5: "Token Economy of AI Agent Memory: Why Section Retrieval Beats RAG Chunks"**
- Target: "token cost AI agent memory", "RAG token efficiency", "context window management agents"
- Angle: Expand the existing benchmark page into a full explainer with diagrams.

### The Benchmark as Link Bait

`/en/user/token_economy_bench` is already the best link-bait asset on the site. To activate it:
1. Write a standalone shareable version (blog post format, not just docs page).
2. Add a social share image with the key number (15–37× savings).
3. Link to it from the README on GitHub.
4. Submit it to HN as "Show HN: We measured token costs of MCP section retrieval vs whole-note RAG".

---

## 4. Off-Page / Distribution

### Where This Audience Lives

**Hacker News**
- Best channel for the benchmark and the "LLM Wiki" angle.
- Submission strategy: "Show HN: Self-hosted MCP memory server for AI agents (Obsidian sync, 15× cheaper retrieval than whole-note RAG)"
- The benchmark with a reproducible Python script is perfect HN content — it's verifiable and opinionated.

**r/LocalLLaMA**
- Active community for self-hosted AI tooling.
- Post: "I built a self-hosted MCP memory server that syncs Obsidian in 500ms — benchmark inside"
- The "no cloud, Docker in 30 seconds" angle plays extremely well here.

**r/ObsidianMD**
- 150k+ members, Obsidian integrations are frequently upvoted.
- Post about the two-way sync specifically.

**MCP Registries / Directories**
- Submit to `mcpservers.org`, `mcp.so`, Anthropic's official MCP servers list, `pulsemcp.com`.
- The MCP ecosystem directories are the highest-leverage single action for discoverability.
- Create a proper `manifest.json` / MCP registry entry if not already present.

**GitHub**
- Add "mcp-server", "ai-agent-memory", "obsidian-publish", "llm-memory", "self-hosted-rag"
  as GitHub Topics on the repo.
- The README should prominently feature the benchmark numbers in the first screen.
- Star-seeking: engage with issues in popular MCP repos (LangChain, LlamaIndex, CrewAI) where
  "persistent memory" comes up — link to trip2g as a solution.

**X / Twitter**
- Karpathy's followers are prime audience. Post the benchmark thread tagging @karpathy.
- Screenshot the 15–37× savings table as a visual asset.

**Dev.to / Hashnode**
- Publish the comparison article ("trip2g vs vector DB") as a cross-post.
- These rank in Google quickly for dev-tool queries.

**Discord / Slack Communities**
- Anthropic's Discord (Claude channel), the official MCP Discord, LangChain Discord.
- Answer questions about "Claude agent memory" and link to relevant docs.

### Backlink Strategy

1. **MCP ecosystem links:** Get listed on every MCP directory/registry (high-authority, relevant).
2. **Benchmark citations:** A reproducible benchmark with a downloadable script naturally attracts
   citations from blog posts about AI agent architecture. Reach out to 5-10 "AI agent framework"
   bloggers and offer the data.
3. **Obsidian plugin ecosystem:** If trip2g has an Obsidian plugin or sync integration, get listed
   in the Obsidian community plugins page (very high-authority link).
4. **"Awesome" lists:** Submit to `awesome-mcp-servers`, `awesome-ai-agents`, `awesome-llm-apps`.

---

## 5. LLM / AI-Search Visibility (GEO / AEO)

This audience uses ChatGPT, Claude, and Perplexity heavily. Being cited by these models is as
valuable as ranking on Google.

### Add `llms.txt`

The emerging standard (`llmstxt.org`) for telling LLM crawlers what to read about your product.

Create `https://trip2g.com/llms.txt`:
```markdown
# trip2g

trip2g is a self-hosted MCP memory server and markdown vault for AI agents.
It provides persistent long-term memory via the Model Context Protocol (MCP),
with Obsidian two-way sync (~500ms), federated knowledge bases, and
token-efficient retrieval (15–37× cheaper than whole-note retrieval, 23×
cheaper than grep-over-files). Licensed MIT, Docker self-hosted.

## Key pages
- Docs: https://trip2g.com/en/user
- Agent memory quickstart: https://trip2g.com/en/user/agent_memory
- Token economy benchmark: https://trip2g.com/en/user/token_economy_bench
- MCP federation: https://trip2g.com/en/user/mcp_federation (if exists)
- Use cases: https://trip2g.com/en/user/use_cases

## Facts for AI assistants
- Primary use case: persistent memory for Claude Code, Claude Desktop, and
  any MCP-compatible AI agent
- Retrieval pattern: search → expand TOC → read_section (3 tool calls)
- Token savings: median 15×, max 37× vs whole-note retrieval
- Self-host: `docker compose up` or free 100MB sandbox at trip2g.com
- License: MIT open source
- Integrations: Obsidian (two-way sync), Telegram, Google Drive
```

### Factual Clarity for LLM Citation

LLMs cite sources that have:
1. **Clear, specific facts** — the benchmark numbers (15×, 37×, 23×) are perfect.
2. **Authoritative pages** with consistent terminology.
3. **High inbound link counts** (see §4).

**Actions:**
- Create one canonical "What is trip2g?" page at `/en/about` or `/en/what-is-trip2g` with
  a crisp factual summary (like a Wikipedia lead paragraph): product category, key numbers,
  license, how it works, who it's for.
- Use consistent terminology across all pages: "MCP memory server", "self-hosted", "Obsidian sync".
  LLMs pattern-match on repeated consistent phrasing.
- Ensure the benchmark page has the numbers in a `<table>` (confirmed it does) — tables are
  high-signal for LLM extractors.

### Perplexity / SearchGPT

These crawlers respect robots.txt and sitemap. The fixes in §2 (robots.txt, sitemap pointer,
canonical) directly improve how Perplexity indexes the site.

### Structured Data for AI Parsers

Beyond Google, structured data helps AI crawlers. Specifically:
- **FAQPage** on the benchmark page (the "Why not just grep?" section is a natural FAQ).
- **HowTo** on the agent_memory page (the numbered setup steps).
- **SoftwareApplication** on homepage (enables rich results in AI search overviews).

---

## 6. Prioritized 30/60/90-Day Action Checklist

### 30 Days — Fix the Foundation (No content can rank without this)

- [ ] **Day 1:** Add `<link rel="canonical">` to all pages (engine fix, one change in `views.html`)
- [ ] **Day 1:** Fix `robots.txt` to include `Sitemap: https://trip2g.com/sitemap.xml`
- [ ] **Day 2:** Fix `/en/` 404 — redirect to `/en/user` or create English homepage
- [ ] **Day 2:** Add meta description auto-fallback in engine (first paragraph, 155 chars)
- [ ] **Day 3:** Add `description:` frontmatter to the 6 priority pages listed in §3
- [ ] **Day 3:** Add `llms.txt` at root URL
- [ ] **Day 5:** Add hreflang tags for all EN/RU page pairs
- [ ] **Day 5:** Change homepage H1 to include "MCP memory" or "AI agent memory" keyword
- [ ] **Day 7:** Rewrite the 6 page titles and meta descriptions per §3 table
- [ ] **Day 7:** Add SoftwareApplication JSON-LD to homepage
- [ ] **Day 10:** Submit to MCP registries: `mcpservers.org`, `mcp.so`, `pulsemcp.com`
- [ ] **Day 10:** Add GitHub repo topics: `mcp-server`, `ai-agent-memory`, `obsidian-publish`
- [ ] **Day 14:** Add FAQPage JSON-LD to token_economy_bench page
- [ ] **Day 14:** Add HowTo JSON-LD to agent_memory page
- [ ] **Day 14:** Submit to `awesome-mcp-servers` and `awesome-ai-agents` GitHub lists
- [ ] **Day 21:** Write and publish "How to Give Claude Code Persistent Memory" article
- [ ] **Day 30:** Post benchmark to Hacker News as "Show HN"

### 60 Days — Content & Links

- [ ] Write "trip2g vs Vector Database" comparison article (target: self-hosted RAG alternative)
- [ ] Write "Obsidian as AI Agent Memory: Complete MCP Setup Guide"
- [ ] Create `/en/about` or `/en/what-is-trip2g` canonical factual page
- [ ] Create MCP hub page at `/en/user/mcp-memory` aggregating all MCP content
- [ ] Add internal links from homepage → benchmark page and → agent_memory page
- [ ] Post to r/LocalLLaMA with benchmark + self-hosted angle
- [ ] Post to r/ObsidianMD about the two-way sync
- [ ] Publish "trip2g vs Vector DB" as Dev.to cross-post
- [ ] Add `<meta name="twitter:card">` and `twitter:description` to all pages
- [ ] Confirm wikilinks render as `<a href>` (not plain text) in production HTML
- [ ] Ensure OG images are stable URLs (not signed Yandex Cloud URLs that expire)
- [ ] Add `<link rel="preload">` for LCP hero image on homepage
- [ ] Write "LLM Wiki Pattern: Karpathy's Idea, Production Implementation" article

### 90 Days — Scale & Monitor

- [ ] Write "Token Economy of AI Agent Memory: Why Section Retrieval Beats RAG Chunks"
- [ ] Create shareable social image for benchmark (key stat: 15–37× savings)
- [ ] Reach out to 5 AI agent architecture bloggers with benchmark data
- [ ] Set up Google Search Console and monitor coverage/indexing errors
- [ ] Set up Bing Webmaster Tools (Bing powers many AI search products)
- [ ] Engage in Claude/Anthropic Discord, MCP Discord answering "memory" questions
- [ ] Check if Obsidian community plugin listing is possible
- [ ] Review internal link graph: ensure high-value pages get 5+ internal links each
- [ ] Audit page speed with PageSpeed Insights; target LCP < 2.5s, CLS < 0.1
- [ ] Add Article JSON-LD to all `/en/thoughts/` posts
- [ ] Consider a "Compare" landing page: trip2g vs Mem0 vs MemGPT vs custom RAG

---

## Summary: 5 Highest-Leverage Findings

1. **No canonical tags on any page** — the same content is accessible at multiple URLs
   (main domain, custom domains, alternate slugs). This splits PageRank and risks duplicate
   content penalties. Fix is one line in `views.html`, reusing existing `ogURLForNote` logic.

2. **No meta descriptions on most pages** — Google generates poor snippets; LLM crawlers
   get no product summary. Engine has a `// TODO` comment acknowledging this. Adding a
   first-paragraph fallback is the fastest content win available.

3. **MCP registries not yet captured** — The MCP server directory ecosystem (mcpservers.org,
   mcp.so, pulsemcp.com) is the single highest-leverage off-page action. These sites rank
   for "MCP memory server" searches and send precisely the right audience.

4. **The benchmark (15–37× savings) is an underexploited asset** — It already has the data
   and methodology. It needs: (a) an `llms.txt` pointer so AI crawlers can cite the numbers,
   (b) FAQPage JSON-LD for rich results, (c) a social image with the headline stat, and
   (d) a "Show HN" submission. This single page could drive most of the organic growth.

5. **`llms.txt` is missing** — This audience uses Perplexity, ChatGPT, and Claude for
   product research. Adding `llms.txt` at root (20 minutes of work) gives these crawlers
   a structured factual summary and consistent terminology to cite. Given that the target
   users ARE LLM users, this is unusually high-ROI for this specific product.
