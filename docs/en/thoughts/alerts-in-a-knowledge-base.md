---
title: "Alerts belong in a knowledge base"
free: true
lang_redirect: "[[ru/thoughts/alerts-in-a-knowledge-base]]"
---

*What this is: we wrote [alert-sink](https://github.com/trip2g/alert-sink), a small open-source service that turns Alertmanager webhooks into incident notes in a trip2g instance. Every alert becomes a markdown note, the notes add up to a permanent searchable incident history, and the knowledge base can narrate incidents to a Telegram group on its own. Read it if your incident history currently lives in a chat scrollback.*

Alerts today are fire-and-forget. Prometheus notices a problem, Alertmanager pushes a message to Telegram or email, someone reacts or doesn't, and that's the end of the record. A month later you're asking "when did this node last flap?" and the honest answer is: scroll the chat and hope. Alertmanager itself only shows what is firing right now. History survives as a Prometheus time series, bounded by retention, or as a message that scrolled past.

That gap annoyed me for a while, because the fix is not a monitoring product. It's a place to write things down.

### One incident, one note

alert-sink is a single Go binary that sits behind Alertmanager as a webhook receiver. When an alert fires, the sink creates a markdown note in a trip2g knowledge base: `incidents/YYYY/MM/<timestamp>-<alertname>-<fingerprint>.md`, with the full label set and timing in frontmatter. When the alert resolves, the same note flips to resolved.

The path is a pure function of the alert identity. Re-deliveries and retries land on the same note, so an alert that fires, gets re-notified five times and resolves is still exactly one note. Not six messages in a chat.

What that buys you:

- **Permanent history.** Notes don't expire with Prometheus retention and don't scroll away.
- **Search.** Incident history is just notes in a knowledge base, so full-text search over it comes for free.
- **Postmortems that link back.** Every incident note carries a wikilink to a sibling postmortem note. A human or an agent writes the postmortem later, and the sink is careful never to touch that path.
- **Narration.** trip2g already knows how to publish notes to Telegram chats by tag. The sink stamps every incident note with the publish fields, so a wired-up knowledge base can post the incident to a group and edit the same message in place when it resolves. One incident, one message, flipping from firing to resolved on its own.

That last point is the part I find interesting. The knowledge base isn't just an archive that happens to hold incidents. It's the thing doing the talking. The sink writes a note; everything downstream, the magazine page, the search, the Telegram post, is the knowledge base behaving the way it behaves with any other note.

### Not an archive: live context for people and agents

The "search" bullet above is not just full-text. trip2g's [[en/user/search|search]] is hybrid: BM25 text search plus semantic vector search (OpenAI embeddings, RRF fusion). So incident history becomes queryable by meaning, not only by keywords. "Postgres connection pool saturated" surfaces notes about a `too many clients` incident even if those exact words aren't in the incident title.

The more interesting part is what this enables for agents. trip2g exposes an [[en/user/mcp|MCP server]]: connect it to any MCP-compatible client and the knowledge base becomes a tool your agents can call. For incident history that means an agent can ask `search("database connection spike")` and get the three most relevant past incidents back, ranked by semantic similarity. The workflow then is what you'd expect: an agent investigating a current alert can pull its own history before recommending anything.

A concrete example: a deploy agent sees a `CacheLatencyHigh` alert firing. It calls `search("cache latency high")` on the incident base, gets the postmortem from six months ago that traced the same symptom to a misconfigured eviction policy, and flags it in its report. That postmortem was written by a human into a note the sink would never touch. The knowledge base just held onto it.

Semantic search is opt-in. It's off by default; you turn it on by pointing trip2g at your own embedding server (any OpenAI-compatible endpoint, local or hosted) through the `vector_search` feature flag, see [[en/user/search|search]]. The demo ships with it off. Plain BM25 text search over incident history works with no setup at all; semantic ranking is the extra step. The MCP interface, on the other hand, is the same interface any trip2g instance exposes, so `search()` from an agent comes with the instance either way, it just uses whichever search you've enabled.

To make this concrete I ran it. I brought up the demo stack, fired three incidents through Alertmanager, pointed the instance at a local `bge-m3` embedding server, and searched. The queries below share no words with the incident notes, and two of them are in Russian against English notes. Real results from that run:

| Query (plain language) | Top hit (real incident note) | Why keyword alone misses it |
|---|---|---|
| `"database slow"` | `HighPostgresLatency` -- "p99 latency on postgres spiked to 4s" | note never says "database" or "slow" |
| `"база тормозит"` (RU) | `HighPostgresLatency` -- same note, English | cross-language, zero shared tokens |
| `"закончилось место"` (RU, "ran out of space") | `DiskFillingNode2` -- "root filesystem 96% on node-2" | note says "filesystem", not "место"/"space" |
| `"агент упал по памяти"` (RU, "agent died on memory") | `HermesOOM` -- "hermes-agent killed by OOM" | "OOM" vs "по памяти", RU vs EN |

Each of these returned the right incident as the top hit. The tell that it was the vector leg doing the work: before the embeddings finished loading, the same `"база тормозит"` query returned only the generic always-firing demo alert (BM25 noise); once the notes were embedded, `HighPostgresLatency` jumped to first. The actual matches depend on how your incidents are titled and labeled. The point is that a query phrased in plain language finds incidents named in monitoring-system language, which are rarely the same words, and rarely the same language.

```bash
git clone https://github.com/trip2g/alert-sink
cd alert-sink/demo
docker compose up --build
```

Wait a minute, then open http://localhost:8080/incidents. That's the incident magazine, a trip2g page listing every incident newest first. A demo alert is already there. To watch a full lifecycle:

```bash
docker compose stop demo-target    # DemoTargetDown fires after ~1 minute
docker compose start demo-target   # the same note flips to resolved
```

### A few engineering notes

Three decisions I'd defend.

**Two notes, clean separation.** The incident note is machine-owned: alert-sink writes and patches it entirely; no human edits it. The postmortem is a separate file the human (or an agent) creates later. The incident note carries a wikilink to the postmortem path; the unresolved link is the "create" affordance. The sink never writes or reads that path.

On resolve, the sink patches a single contiguous block at the bottom of the frontmatter and the visible status callout right after it. Frontmatter flips from `status: firing` to `status: resolved`; the callout flips from 🔴 FIRING to ✅ RESOLVED. Everything else in the note is untouched. This is also why one Telegram message suffices: trip2g edits the same post in place, so the channel member sees the flip without a new message.

**Self-mint auth.** trip2g has no long-lived scoped write tokens, so the sink holds the instance's JWT secret and signs itself a fresh 5-minute token for every write. No credential longer-lived than 5 minutes ever exists outside the secret. The honest caveat: that token is admin on its instance, trip2g has no path-level scoping. Least privilege comes from blast radius, so point the sink at a dedicated instance that holds only alert history.

**Alertmanager never waits.** The webhook handler validates, enqueues, and answers 200 in milliseconds. A worker goroutine does the actual writes with exponential backoff, so a trip2g outage doesn't back-pressure your alerting pipeline. Which matters, because the moments when trip2g might be down are exactly the moments alerts are firing.

The whole thing is Go stdlib plus one JWT library. No database, no state outside the knowledge base itself. MIT licensed: https://github.com/trip2g/alert-sink
