# Changelog

All notable releases of trip2g. Newest on top. Each release lists user-facing
changes with **What / Why / How to use** so an operator or admin can decide
whether to upgrade and how to start using a feature.

Older tags (`v0.2.0` and below) live in git history only.

---

## v0.5.1 — 2026-05-27

### Per-webhook secrets injected into delivery payload (`0b72acf2`)

- **What.** Each change webhook and cron webhook now has a **Secrets** panel in the admin. Add named key-value pairs (e.g. `auth_token`, `api_key`) — they are stored encrypted and sent in every delivery payload under `payload.secrets`. The webhook consumer can read them without any extra API calls.
- **Why.** Cron webhooks are the foundation of a plugin system. A plugin is a web server (or serverless function) that receives a payload with a short-lived API token and processes it — often in the background — then patches the vault when ready. Because plugins are stateless, they have no safe place to store their own credentials. Secrets solve this: trip2g holds them encrypted and delivers them on every call, so the plugin stays credential-free on its end. A plugin that needs more time can use the API token to push updates back asynchronously; secrets give it everything else it needs to talk to external services.
- **How.** Open Admin → Change Webhooks (or Cron Webhooks) → select a webhook → scroll to the **Secrets** section. Enter a name and value, click **Add Secret**. To update a value, type in the row's value field and click **Save**. To remove, click the trash icon (confirm on second click). Secrets appear in the delivery payload as `{ "secrets": { "auth_token": "...", "api_key": "..." } }`.

## v0.5.0 — 2026-05-26

### In-browser file editor (admin)

- **What.** Admins can now edit any page right on the site. An editor icon appears in the top-right of the admin panel and next to the search on every page. Open it to browse every uploaded file as a folder tree, view and edit any one, and save. You can also roll a file back to an earlier version.
- **Why.** Fix a typo or update a page in seconds — no Obsidian, no re-sync.
- **How.** Click the editor icon (admins only). The current page's note opens by default; pick any other file from the tree on the left. Edits stay in your browser until you press **Save**, and the versions panel lets you load and restore an older version.

### Public hub of curated bases

- **What.** A `hub/` section with a bilingual index of the knowledge bases reachable through the hub (first entry: the Nick Senin Journal — filtered Code with Claude 2026 cases).
- **Why.** A browsable, public entry point to federated bases.
- **How.** See [`docs/en/hub/_index.md`](./en/hub/_index.md); add your own with [`docs/en/hub/_create.md`](./en/hub/_create.md).

## v0.4.1 — 2026-05-25

### MCP Federation — one hub across many knowledge bases

- **What.** Your instance can act as a federation hub. A **KB-note** (a note with `mcp_federation_kb_url` in frontmatter) registers another MCP-compatible base, and `federated_search` / `federated_similar` / `federated_note_html` reach across all of them through your single MCP endpoint. Public bases need no auth; private peers use a shared HMAC secret.
- **Why.** One endpoint, one auth surface — your agent searches your own notes, partner instances, and external adapters (GitHub, Telegram) together, without rewiring `.mcp.json`.
- **How.**
  - User docs: [`docs/en/user/federation.md`](./en/user/federation.md), [`docs/ru/user/federation.md`](./ru/user/federation.md)
  - Public base: create a note with `mcp_federation_kb_url` (+ optional `mcp_federation_kb_id`) and `free: true`.
  - Private peer: exchange a federation secret in Admin → Federation, then add the KB-note.

### Canvas files (Base & Excalidraw coming later)

- **What.** `.canvas` files sync and render. `.base` and `.excalidraw` files are accepted by sync too, but rendering them is planned for later — for now they show a clear placeholder instead of breaking the page.
- **Why.** Canvas vaults work today; Base and Excalidraw vaults sync without errors while full support is on the way.
- **How.** Just sync — the plugin and CLI accept all three extensions; Canvas renders now.

### Telegram navigation & canvas bots

- **What.** A wikilink-browser bot and canvas-driven navigation over a Telegram business connection.
- **Why.** Readers can browse your vault graph from inside Telegram.
- **How.** See the Telegram docs; enable on a business connection.

### Admin & config

- **What.** GraphQL API for note **version history**; admin **filter for form submits** (status / date / processed); environment variables accept a `TRIP2G_` prefix (unknown vars warn).
- **Why.** Inspect and roll context, triage submissions, and configure self-hosted instances more safely.
- **How.** Admin panels; prefix any env var with `TRIP2G_`.

### Obsidian sync plugin + CLI

- **What.**
  - Accepts `.canvas`, `.base`, and `.excalidraw` files.
  - Surfaces GraphQL error details on a failed push (no more silent failures).
  - **New `--exclude <glob>` flag** (repeatable). Excluded paths are never pushed; if they already exist on the server they are **hidden**. A bare name like `dev` matches that directory and everything under it. Default: nothing is excluded — everything uploads.
- **Why.** Keep test/demo or internal folders (e.g. `dev/`, `demo/`) in your repo but out of the published site — and reversibly hide them on the server.
- **How.** `trip2g-sync ./docs --exclude dev --exclude demo`. Re-including a path re-publishes and automatically unhides it.

---

## v0.4.0 — 2026-05-21

### updateNotes mutation — atomic find/replace across notes

- **What.** New GraphQL mutation that patches multiple notes in one transaction via a `PathMap` of `{path → [{find, replace}]}`.
- **Why.** Lets external tools, agents, and scripts apply consistent edits across a vault without orchestrating per-note round-trips. Avoids partial states when one of the replacements fails.
- **How.**
  - User docs: [`docs/en/user/update_notes.md`](./en/user/update_notes.md), [`docs/ru/user/update_notes.md`](./ru/user/update_notes.md)
  - Example: send `updateNotes(input: { pathMap: { "notes/post.md": [{ find: "old", replace: "new" }] } })`.
  - End-to-end spec: `e2e/updatenotes/*` (see `test(e2e): add updateNotes e2e spec and demo fixture`).

### Forms admin — submit processing

- **What.** New `markFormSubmitProcessed` mutation and `processed` fields on form submits; admin can mark a submit as handled, the UI hides processed entries by default.
- **Why.** Closes the form-handling loop inside the admin instead of forcing external triggers.
- **How.**
  - User docs: [`docs/en/user/forms.md`](./en/user/forms.md), [`docs/ru/user/forms.md`](./ru/user/forms.md)
  - Dev reference: [`docs/dev/forms.md`](./dev/forms.md)
  - Use `can_submit` / `success_url` frontmatter on form notes to gate submissions and customize the thank-you page.

### Layout smoke-render — surface Jet runtime errors at load

- **What.** When notes are loaded, each parsed Jet layout is executed against up to **10 first notes** that select it via frontmatter `layout:`. Runtime errors and panics become `NoteWarning` entries on the layout.
- **Why.** Previously, a broken layout that *parsed* (e.g. `{{ note.NoSuchField }}`) only failed when a user opened the page in the browser. Agents pushing notes had no signal. Smoke-render moves the failure to load time so warnings show up in the same channel as parse errors — visible via `pushNotes` / admin layout listings without a browser request.
- **How.** Automatic; no flags. Watch for `smoke render error` / `smoke render panic` in layout warnings after a sync. Layouts without parsed templates and layouts no note uses are skipped.

### Template debugging — `Meta.Debug()` and global `debug()`

- **What.** Inside Jet templates: `{{ Meta.Debug() }}` dumps note metadata; global `{{ debug(<any>) }}` prints type, value, and the method set of any expression via reflection.
- **Why.** Removes the "guess what the template sees" loop when authoring layouts and components.
- **How.**
  - User docs: [`docs/en/user/jet.md`](./en/user/jet.md), [`docs/ru/user/jet.md`](./ru/user/jet.md) (debugging section).
  - Example: `{{ debug(note.M()) }}` or `{{ debug(note.Title) }}`.

### `renderlayout.py` CLI — render a layout against a note

- **What.** Standalone script in `scripts/renderlayout.py` plus a `check_templates` agent skill.
- **Why.** Lets you preview layouts and reproduce template issues from the terminal — useful in CI and when iterating on `_layouts/`.
- **How.**
  - User docs: [`docs/en/user/renderlayout.md`](./en/user/renderlayout.md), [`docs/ru/user/renderlayout.md`](./ru/user/renderlayout.md)
  - Skill: [`docs/skills/check_templates.md`](./skills/check_templates.md)

### Fixes
- `layoutloader`: nil-guard for `YieldNode.Parameters`; layout ID normalized in preview to match production.
- `renderlayout` preview: autoimport, `yield_blocks` wiring, and `htmlInjections` now match production behavior.
- `renderpreview`: parses YAML frontmatter from `note.src` like the real loader.
- `templateviews.GetStrings`: returns empty slice instead of `nil` (no more nil-iteration surprises in templates).

### Docs & chore
- Forms dev reference + roadmap, BEM rendering skill, Jet `debug()` section.
- Lint passes across `updatenotes`, `layoutloader`, `noteloader`.

---

## v0.3.1 — historical (backfill)

### Forms in notes (initial)

- **What.** Forms can be embedded in vault notes via frontmatter and rendered by the default template. Multiple forms per note supported, `note_version_id` and `form_id` tracked per submit.
- **Why.** Lets a published site collect input (signups, contact, polls) without an external service.
- **How.** See `docs/dev/forms.md` and `docs/{en,ru}/user/forms.md`.

### Layouts get HTML injections

- **What.** Custom HTML injections (head / body_end placements) now apply inside Jet layouts, not just the default template.
- **Why.** Analytics, custom scripts, and SEO tags work for custom-layout pages.
- **How.** Admin → HTML injections; pick placement `head` or `body_end`.

### Language switcher rework

- **What.** Dropdown showing full native language names with normalization; US flag for English; multilingual docs.
- **Why.** Multilingual sites look correct and pick the right alternate per language.
- **How.** Use `lang:` / `lang_redirect:` frontmatter; switcher renders automatically in default template.

### renderlayout preview endpoint (`/_system/renderlayout`)

- **What.** Admin endpoint to render an arbitrary layout against a note for preview.
- **Why.** Editor / IDE integrations can show a live preview of `_layouts/*` changes.
- **How.** Spec: `docs/superpowers/specs/2026-05-10-renderlayout-endpoint-design.md`.

### Admin API keys: enable/disable

- **What.** GraphQL mutation + admin button to toggle API key state; auto-cleanup of API key logs after 90 days.
- **Why.** Pause a key without rotating it; keep logs bounded.
- **How.** Admin → API keys panel.

### Onboarding vault: agent config files

- **What.** Vault download now ships with `.mcp.json`, `codex.json`, antigravity config and `AGENTS.md`.
- **Why.** Drop-in setup for AI agents over an Obsidian vault — no manual wiring.
- **How.** Download the onboarding vault; configs are already inside.

### Cronjobs lock by default

- **What.** Cronjob editing is disabled unless `--cronjobs-allow-edit` is passed.
- **Why.** Reduces footgun in shared / production deployments.
- **How.** Pass the flag in your start script if you do want to edit cron entries through the admin.

### Notable fixes
- Backlinks / similar notes exclude system notes (paths starting with `_`).
- TOC anchors work again (heading `id` emitted in HTML).
- Mesh template: `yield_blocks` moved to `<head>` with documented limitations.

---

## v0.3.0 — historical (backfill)

### Self-hosted deployment path documented

- **What.** End-to-end deployment guide for running trip2g on your own infrastructure.
- **Why.** Reproducible, hands-off setup outside the hosted offering.
- **How.** `docs/{en,ru}/user/hosting.md` and related guides.

### Sign-in wall + captcha (Auth phase 1A)

- **What.** Per-note sign-in requirement, captcha on auth flows, hardened sessions.
- **Why.** Gate private content; resist abuse on public auth endpoints.
- **How.** Frontmatter / subgraph `require_signin`; default template renders the wall.

### Vault-based layout sections

- **What.** Header / footer / sidebar can be sourced from vault notes.
- **Why.** Authors edit chrome the same way they edit content — no template hacks.
- **How.** Place notes in the conventional paths (see `docs/dev/default_template.md`).

### Vault-based frontmatter patches

- **What.** Markdown files in the vault can declare patches that apply to other notes' frontmatter at load time.
- **Why.** Bulk-tag, set layouts, or normalize metadata without editing every note.
- **How.** `docs/dev/frontmatter_patches.md`.

### Search: bge-m3 embeddings + embedding microservice

- **What.** Switched to `bge-m3` embedding model; introduced an embedding-server microservice; vector search top-K results with `matchOrigin`.
- **Why.** Better semantic retrieval and decoupled embedding workload.
- **How.** Configure embedding endpoint; vector search exposed via existing search APIs.

### MCP server upgrades

- **What.** MCP search results are addressable, self-describing, and openable as focused chunk reads; text-search hits map to nearest chunks.
- **Why.** RAG clients get richer, navigable results.
- **How.** Connect via MCP; consult `docs/user/ai-agent-docs-setup.md` (if present in your vault) for client wiring.

### "Read in Telegram" + Telegram UX

- **What.** Button on note pages, public TG links preferred for published posts, UTM tags on Telegram-originated traffic.
- **Why.** Closes the loop between site posts and Telegram audience attribution.
- **How.** Automatic for posts published through TG; UTM scheme documented in `docs/superpowers/specs/`.

### exporttgchannel CLI

- **What.** New CLI command to export a Telegram channel to Obsidian-flavored markdown.
- **Why.** One-step import of an existing channel into a vault.
- **How.** `go run ./cmd/exporttgchannel --help`.

### Notable fixes
- Configurable URL normalization with 301 redirects for alternate variants.
- Audio rendered as `<audio>`; documents rendered as links.
- Queue: prevent goroutine leak on double start and deadlock on stop.
- Wikilink `[[slug#anchor]]`: strip anchor before note lookup.
- All `golangci-lint` warnings resolved, pre-push hook added.
