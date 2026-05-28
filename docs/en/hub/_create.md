---
title: "Add a base to the hub"
free: true
lang: en
lang_redirect: "[[ru/hub/_create]]"
---

# Add a base to the hub

A hub entry is a **KB-note**: a note whose frontmatter points at another
MCP-compatible base (`mcp_federation_kb_url`). Once synced, your hub can reach that
base through `federated_search` and friends. See [[en/user/federation|MCP Federation]]
for how federation works end to end.

## 1. Create the bilingual pair

Each entry is two notes, one per language, cross-linked with `lang_redirect`
(which is **only** for language alternatives — not a multi-target dispatcher):

`docs/en/hub/<name>.md`

```yaml
---
title: "<Base name>"
free: true
lang: en
lang_redirect: "[[ru/hub/<name>]]"
mcp_federation_kb_url: https://<host>/_system/mcp
mcp_federation_kb_id: <kb_id>
---

What the base is and when an agent should use it.
```

`docs/ru/hub/<name>.md`

```yaml
---
title: "<Название базы>"
free: true
lang: ru
lang_redirect: "[[en/hub/<name>]]"
---

Что это за база и когда к ней обращаться.
```

**Put the federation fields on exactly one note** (here, the EN one). If both
language versions carried `mcp_federation_kb_url` / `mcp_federation_kb_id`, the hub
would register the same `kb_id` twice — duplicate fan-out and ambiguous targeting.
The other language version is just the alternative.

| Field | Required | Purpose |
|-------|----------|---------|
| `mcp_federation_kb_url` | yes | The remote MCP endpoint. Marks the note as a KB-note. |
| `mcp_federation_kb_id` | no | Slug used to target the base (`federated_search(kb_id=…)`). Defaults to the URL hostname. Keep it unique across the hub. |
| `free: true` | for public | Lets anonymous MCP callers route through this base. Omit to keep it subscriber/admin-only. |

> **Tip: sprinkle keywords into the note body.** A plain hub `search` matches note
> content, not the federated bases. List what the base covers in the body — topics,
> technologies, task types, signature terms. Then a plain `search` surfaces this
> entry, so an agent lands on the right base before even running a federated search
> (and from there, `federated_search(kb_id=…)`).

## 2. Public vs private base

- **Public base** (no auth) — nothing else to do. The hub proxies anonymously.
- **Private base / adapter** (needs a key) — add an **outbound federation secret**
  in Admin → Federation: paste the `kid` + secret the peer gave you, with the
  base's `kb_url`. The hub then signs outbound calls. Until the key is set,
  anonymous queries to that base return no results.

## 3. List it in the hub index

Add a line to both indexes:

- `docs/en/hub/_index.md`: `- [[en/hub/<name>|<Base name>]]`
- `docs/ru/hub/_index.md`: `- [[ru/hub/<name>|<Название базы>]]`

## 4. Sync

Sync your vault. Verify with a targeted call:
`federated_search(query: "…", kb_id: "<kb_id>")`, and confirm the entry shows up
in a fan-out `federated_search` (no `kb_id`).

## Related

- [[en/user/federation|MCP Federation]] — protocol, secrets, permissions
- [[en/hub/_index|Hub]] — the current list of bases
