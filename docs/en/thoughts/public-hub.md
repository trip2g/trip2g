---
title: "One endpoint, many knowledge bases"
free: true
lang_redirect: "[[ru/thoughts/public-hub]]"
---

A bit over a month after the first federation commit, our public trip2g hub is
live: **one MCP endpoint that searches across several independent knowledge bases
at once.** The demo is the small part. The bet is the network — bases owned by
different people, reachable through a single address, queryable by any agent with
no central registry.

## The job we hired it for

Job-to-be-done: *when my agent needs to ground an answer, I want it to search every
base I trust — mine, my partners', a few curated public ones — through one
endpoint, and get the best evidence back, without me wiring and re-wiring
connections.*

Nobody wants "a federation feature." They want their agent to be smarter because it
can reach more good sources with zero plumbing.

## The problem federation solves

Everyone who runs trip2g has their own base, and an agent pointed at one of them is
blind to the rest. Wiring N endpoints into every agent's config is friction that
grows with the network. We wanted the opposite: add a base by writing one note, and
every agent that points at the hub can reach it.

That is what MCP federation does. A **KB-note** — a note with
`mcp_federation_kb_url` in its frontmatter — registers another MCP-compatible base.
`federated_search` fans out across all of them and merges the results. No central
registry: the notes in your vault *are* the topology.

## What's in the hub

- **Nick Senin Journal** — filtered cases from Code with Claude 2026, curated on
  Nikolai's own trip2g instance. Our proven first peer: an agent pulled those cases
  over MCP to harden a skill, without ever opening the talks.
- **Marcus Aurelius — Meditations** — our own base: all 12 books in Russian,
  English, and Greek, with commentary, thematic chains, and extracted principles.
- **Telegram channels** — a curated set of public channels behind a trip2g Telegram
  adapter.

Each entry is a bilingual pair listed in the [[en/hub/_index|hub index]]; adding the
next one is one note ([[en/hub/_create|how]]).

## Public discovery, gated content

Two layers, on purpose. **Discovery is public** — anyone can see a base exists and
target it by `kb_id`. **Content can be gated** — the Telegram adapter returns
*nothing* to an anonymous caller: no hits, no message bodies. Only a hub holding the
shared key gets results. So we can advertise a base in a public hub without leaking
what's inside it, while open reference bases need no key at all.

## What's next

Two moves, both bigger than the demo:

**Organize a team's work across this network.** Agents already publish what they're
doing; federation lets one agent see another's focus and goals through the same
endpoint — coordination without meetings, across organizations, no central server.

**Build a company on top of the network.** A hub of trusted bases, agents that
coordinate through it, content and access as the product — that is not a demo, it is
the shape of the business. The public hub is the first brick.

Month one: it works, and it's live. The interesting part starts now.

## Related

- [[en/harness|The Harness]] — how cross-base knowledge hardened a skill
- [[en/thoughts/agent-team-reporting|The agent reports to the team]]
- [[en/thoughts/knowledge-as-a-service|Knowledge-as-a-Service]]
