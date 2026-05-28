---
title: "Foragent"
free: true
lang: en
lang_redirect: "[[ru/hub/foragent]]"
mcp_federation_kb_url: https://foragentadapter.2pub.me/_system/mcp
mcp_federation_kb_id: foragent
---

# Foragent

A knowledge base of **agent skills from open sources** ([foragent.ru](https://foragent.ru)),
federated into this hub. Skills gathered from across GitHub — collections like
**superpowers** and awesome-skills lists: code review, writing tests and TDD,
deployment and DevOps, refactoring, security, documentation. Each skill ships with
metadata, a GitHub link and tags.

Skills are provided **as is**; some are vetted by the foragent.ru team — a safety
scanner and policy gate set a verdict (`scanner_status`, `risk_score`) and a
`recommended` flag. The decision to use one is always yours: skills are **not
installed automatically**.

Like the Telegram channels, this is **not a trip2g instance** but a standalone node
implementing the MCP federation protocol. It has no public access of its own: the
only way in is through this hub, which holds its key (configured in the trip2g admin).

## Popular skills

The most-starred (by GitHub ⭐):

- **superpowers** (`obra/superpowers`) — brainstorming, writing and executing plans,
  test-driven development (TDD), systematic debugging, code review (requesting /
  receiving), subagent-driven development, git worktrees, verification before
  completion, dispatching parallel agents, using superpowers.
- **openclaw** (`openclaw/openclaw`) — coding agent skill, browser automation and
  scraping.

## Searching skills over MCP

Through the hub MCP endpoint (`trip2g.com/_system/mcp`):

1. **Search** — `federated_search(kb_id: "foragent", query: "…")`:
   - plain text: `{"kb_id":"foragent","query":"code review typescript"}`;
   - JSON DSL in the `query` field for filters:
     `{"op":"search","text":"deploy","technology_tags":["docker"],"sort":"stars"}`.
     Fields: `category`, `task_tags`, `technology_tags`, `capability_tags`,
     `workflow_stages`, `role_tags`, `licenses`, `sort` (`stars` | `updated` | `name`),
     `limit` (1–20).
2. **Skill details** — `federated_note_html(kb_id: "foragent", note_id: "<id from a result>")`:
   the full passport (author, category, tags, ⭐, license, scanner verdict,
   recommendation, SKILL.md link).
3. **Related** — `federated_similar(kb_id: "foragent", note_id: "<id>")`.

Each result carries `note_id`, `title`, `url` (GitHub), `category`, tags (`tasks`,
`capabilities`, `technologies`, `roles`), `stars`, scanner status and a `recommended`
flag. Skills are **not installed automatically** — this is a catalog for discovery
and review.

Source: [foragent.ru](https://foragent.ru)
