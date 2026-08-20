# Changelog

All notable releases of trip2g. Newest on top. Each release lists user-facing
changes with **What / Why / How to use** so an operator or admin can decide
whether to upgrade and how to start using a feature.

Older tags (`v0.2.0` and below) live in git history only.

---

## Unreleased

### Obsidian on the phone: open a note on the site, and close the warnings view (plugin v0.11.0)

- **What.** The note's `...` menu now has **Open on site** and **Copy site link** (only for notes that are actually published), plus a new **Open published URL in browser** command. The sync warnings view got a **Close** button, and its toolbar buttons are large enough to hit with a thumb.
- **Why.** Neither was reachable on mobile. The published-URL indicator lives in the status bar, which Obsidian mobile doesn't show, so the only route to a note's page was the "Copy published URL" command; and dismissing the warnings tab meant a detour through the tab switcher.
- **How.** Update the plugin to 0.11.0, then tap `...` on a note → **Open on site**. For one-tap access, pin **Open published URL in browser** in Obsidian's Settings → Toolbar (mobile).

### Delivery chains: trace every webhook step across the agent graph

- **What.** Every webhook delivery now records which delivery caused it (`parent_delivery_id`) and which end-to-end chain it belongs to (`chain_id`). A new **System → Delivery Chains** screen in the admin lists all chains, each with a timeline of its steps: what path each delivery wrote, what it triggered next, the depth at which it fired, and the wall-clock span. Chains are assembled across separate agent hosts — a fleet agent on one machine and a webhook consumer on another — without either process knowing about the other.
- **Why.** When a change triggers a webhook that writes a note that triggers another webhook, the full sequence was previously invisible: you could see individual delivery records but not the chain that connected them. This makes the entire propagation path inspectable from a single admin screen.
- **How.** Open Admin → System → Delivery Chains. Each row is a root delivery; expand it to walk the chain step by step. The screen refreshes as new chains arrive. No configuration is needed — all deliveries since the upgrade are automatically linked.

### Idempotent writes no longer fire change events

- **What.** A `updateNotes` call whose content equals what is already stored creates no new version and raises no change event. No webhook delivery is queued, no SSE event is emitted, no re-embedding runs, and no Telegram publish-view refresh happens. Changing at least one byte still fires normally. Hiding a note and unhiding it (writing to the hidden path is the only way to unhide) both still work; unhiding refreshes the served snapshot but deliberately raises no change event.
- **Why.** An agent that re-applied identical content re-triggered the webhook that caused it on every cycle, at full LLM cost, bounded only by `max_depth`. The fix stops the loop at the write layer: if nothing changed, nothing downstream needs to know.
- **How.** Automatic. Agents that write back the full note content unchanged no longer cause runaway chains. The `max_depth` guard remains useful for agents that do produce genuine changes.

### Frontmatter key index

- **What.** Note paths can be filtered by effective frontmatter keys and values through GraphQL. The server materializes frontmatter after patches and tracks whether each key is present in the latest or live note set.
- **Why.** Key-aware filtering can narrow the candidate notes before evaluating JSON values such as `fleet_id`, without exposing SQL through the API.
- **How.** Use `notePaths(filter: { frontmatter: [{ key: "fleet_id", equals: "codellm" }] })`. The index is built from latest/live snapshots; historical `note_versions` are not reparsed or backfilled, so metadata that existed only in an old version is not represented in the key index.

---

## v0.10.0 (2026-07-13)

### RSS is now a template, not a Go feature

- **What.** The old automatic `<permalink>.rss.xml` feed (one per note, items = the note's links) is gone, along with its `enable_rss` site setting. RSS is now a normal Jet layout: a note with `layout: rss` + `content_type: application/rss+xml; charset=utf-8` frontmatter renders an RSS 2.0 feed, served wherever you route it (default: `/feed.xml`). Which notes appear is controlled by `rss_glob`/`rss_limit` frontmatter, and only publicly readable notes (free, not sign-in-gated, not system) are ever included.
- **Why.** The per-note curated-links feed was rigid and undiscoverable. A template-based feed is fully customizable — edit `_layouts/rss.html` to change item shape, or build several feeds with different scopes — using the same layout system as everything else on the site.
- **How.** See [[en/user/rss]]. Existing `.rss.xml` subscriber URLs 404 now; there's no redirect — set up a `/feed.xml` note to replace them.

### Theming the default template + a live theme editor

- **What.** Two new pages document how to re-skin the default template. [[en/user/themes]] explains that the whole look comes from Pico CSS `--pico-*` variables — override them in a `<style>` block pasted into admin **HTML Injection** and the site re-themes at once, no rebuild or fork. [[en/user/theme-editor]] renders a live editor right on the site: drag the variables, watch the preview, and (signed in as admin) click **Save** to write the theme into your site's HTML injection. The editor is a custom Jet layout (`layout: theme_editor`), installable on any site from [trip2g/theme_editor_template](https://github.com/trip2g/theme_editor_template).
- **Why.** Changing the look used to mean reading the stylesheet and guessing which variable to touch. The editor makes theming a drag-and-save loop, and doubles as a worked example of a custom template that talks to the admin API (same pattern as the kanban board).
- **How.** Read [[en/user/themes]] for the variable list and where to paste. To try the editor, open [[en/user/theme-editor]]; to install it on your own site, `curl` the layout from the template repo and add a note with `layout: theme_editor`.

### Embedded notes with a custom CSS class

- **What.** New page [[en/user/embedded-notes]] documents `![[note-name]]` — the target note's content renders inline, wrapped in `<div class="embedded-note">`. Add `embed_class: my-class` to the embedded note's frontmatter and the wrapper also gets `embedded-note__my-class`, a hook you can style from the admin. Any note becomes a reusable, styled block: a callout, a promo card, a shared footer.
- **Why.** Reusable partials (a signup banner, a shared header) had no documented pattern. Combined with `_`-hidden notes, one edit updates the block everywhere it's embedded.
- **How.** See [[en/user/embedded-notes]]. Keep the block in a `_hidden` note, add `embed_class:` for styling, and embed it with `![[_block]]` wherever you need it.

### Telegram group ↔ subgraph access (two-way)

- **What.** New page [[en/user/telegram-access]] documents linking a Telegram group to a [[en/user/subgraphs|subgraph]] so access flows both ways: group members can read the subgraph's notes on your site, and readers with subgraph access can be invited into the group. Each direction is a separate switch, and both are live — leave the group and the notes close; let access lapse and a background job removes you from the group. Backing this, membership now tracks Telegram `chat_member` updates so joins and leaves register immediately, and the bot's `/content` menu points at the right subgraphs.
- **Why.** Gating content by paid-group membership was possible but underdocumented, and membership changes weren't always reflected. This makes a Telegram group a first-class access source, on equal footing with a paid offer or a hand grant.
- **How.** Add your bot as a group admin, link the group to a subgraph in Admin → **TG bots** (one section per direction), tag the notes with `subgraph:`, and members use `/content` to open the gated notes. Full walkthrough in [[en/user/telegram-access]].

### Git access to your site

- **What.** New page [[en/user/git]] documents that every trip2g site is a git repository: `git clone https://your-site.com/_system/git`, edit markdown with any tool or agent, commit, and push — the server applies your commits to the live site. Auth is HTTP Basic (`user` + a git token created in Admin → **Integrations → Git tokens**), and token scopes now enforce pull vs. push separately, so a read-only token can clone but not write.
- **Why.** Batch edits, CI jobs, and coding agents need standard git tooling, not a bespoke plugin. Scoped tokens make it safe to hand a read-only clone URL to an automation.
- **How.** Create a git token in the admin, `git clone` the `_system/git` URL, and push to the `master` branch (the only branch the server accepts). For day-to-day writing in Obsidian, the sync plugin is still the better fit. See [[en/user/git]].

### Subgraph recipes: private-by-default, all-free, and `_`-hidden notes

- **What.** [[en/user/subgraphs]] gains three practical recipes. A **private-by-default** frontmatter patch assigns every untagged note to a reserved subgraph nobody is granted, so nothing leaks unless you publish it explicitly. A symmetric **all-free** patch sets `free: true` vault-wide for a fully public site. And a clarified section documents that any file or folder whose name starts with `_` is hidden from listings, search, and RSS (but still reachable and embeddable).
- **Why.** "Which notes are visible to whom" is the most common subgraph question. These give copy-paste starting points for the two ends of the spectrum — private-first and public-first.
- **How.** In Admin → **Notes & Content → Frontmatter Patches**, create a rule with the jsonnet shown in [[en/user/subgraphs]] (private-by-default or all-free), and create the reserved `private` subgraph with no grants.

## v0.9.0 (2026-07-12)

### First login on a fresh self-host box

- **What.** Bringing up the first admin login on a brand-new box no longer needs an email server. The recommended path: `trip2g-server login-link` prints a one-time, 5-minute sign-in link you open straight from the terminal — `/_system/hat` now accepts the token over GET, so the link is clickable into a browser. Prefer the email-code flow without a mail server? Set `LOG_SIGN_IN_CODES=true` and the server writes sign-in codes to its log (read them from `journalctl` or `docker logs`). Two related fixes back this up: requesting a code with no email transport now returns a clear error instead of silently pretending to send one, and `SMTP_STARTTLS=false` is honored so a local plaintext relay (Postfix, MailHog) works.
- **Why.** First login on a fresh box — no domain, no email service — used to be a dead end: the server accepted the request but the code never appeared anywhere, and even a local relay failed on an unwanted STARTTLS upgrade. Now one command gives you a working login link, with a log-based fallback and honest errors.
- **How.** Run `trip2g-server login-link` on the box (or `docker exec` into the container) and open the printed URL within 5 minutes; re-run it anytime for a fresh link. Fallback: set `LOG_SIGN_IN_CODES=true`, request a code from the login page, and copy it from the log. Remove these bootstrap options once you set up OAuth or a real email transport — see [[en/user/smtp]].

### Search: one retrieval engine for site and MCP

- **What.** The site (GraphQL) search and the MCP `search` tool now share one retrieval engine (text + vector + rank fusion) instead of two divergent copies. Two behavior changes for MCP clients: anonymous (and federation) clients now search **published (live)** notes, like anonymous site visitors — previously they searched the latest versions, including drafts of public notes; and vector search now skips stale embeddings after an embedding-model switch instead of ranking them arbitrarily. The unused server-rendered `/search` page is removed (the site search widget uses GraphQL and is unaffected).
- **Why.** Retrieval fixes and tuning landed on one surface and silently missed the other; agents and visitors could see different rankings for the same query. On sites with draft previews enabled, anonymous MCP clients could read draft content of public notes.
- **How.** Automatic. If an agent integration relied on anonymous MCP search seeing drafts, authenticate it with an API key — API-key clients still search the latest corpus.

### SMTP provider guide

- **What.** A new bilingual documentation page, [[en/user/smtp]], covers how to pick a transactional-email provider for a self-hosted trip2g instance. It lists the envelope configuration (`SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, etc.), rates the main provider categories (dedicated transactional services, domain registrar mail, free tiers), and gives the verdict: don't run your own outbound MTA.
- **Why.** Email deliverability for self-hosted instances is a frequent setup question. The available config keys were documented piecemeal; there was no single place explaining the tradeoffs.
- **How.** Read [[en/user/smtp]] before configuring email on a new instance.

### Instagram frames (instaframes)

- **What.** The instaframes feature now has a bilingual user guide and a gallery landing page. Carousel pages also gain an **in-page edit widget**: a button that opens the slide deck for editing without leaving the published page. The deprecated reel/teleprompter layout is removed — the carousel layout is the only instaframes layout.
- **Why.** The guide and gallery make the feature discoverable. The edit widget closes the feedback loop between publishing and editing: you see a slide, notice a typo, and fix it on the spot. The reel layout had no active users and duplicated the carousel's code path.
- **How.** See [[en/user/instaframes]] for the full guide and gallery. The in-page edit widget appears automatically on carousel pages when you are logged in as the site owner.

### Admin dashboard fixes

- **What.** Two small corrections to the admin dashboard: the onboarding documentation link now points to the correct page (the previous link was a typo that 404'd), and the storage-usage figure is now labeled **MiB** to match the actual byte count displayed.
- **Why.** The onboarding link was the first thing a new operator clicked; landing on a 404 was a poor first impression. The old "MB" label implied decimal megabytes but the code reported binary mebibytes.
- **How.** Automatic on upgrade.

### MCP / federation improvements

- **What.** Four targeted improvements to the MCP endpoint and federation layer. (1) A new `federated_instructions` tool lets a connected client fetch the instruction text for any route, with a per-route cache to avoid redundant fetches. (2) Federation hop depth now uses **inclusive semantics**: a request with `max_depth=2` may traverse exactly 2 hops; previously it stopped one hop short. Requests that exceed the limit now return a clean rejection instead of an opaque error. (3) Hop-rewrite propagates the caller's `kb_id` frame into federated results and errors, so the caller can always tell which peer an answer came from. (4) Note-read tools (`note_html`, `note_text`) now steer by `path` or `match_id` and document that `note_id` is an internal integer not suitable for long-term references.
- **Why.** The depth off-by-one meant that `max_depth=1` allowed zero federation hops — a configuration that appeared to work but never reached any peers. The hop-rewrite makes it possible to build attribution chains across federated vaults. Clear note-ID guidance prevents agents from caching internal IDs across syncs.
- **How.** No configuration change needed. If you relied on the old depth semantics, decrement your `max_depth` value by one to preserve the previous behavior.

### MCP graph-walk visualizer

- **What.** A new interactive documentation page, [[en/search_visualizer]], lets you watch an AI model walk the federation graph in real time. It shows each tool call as a node, a mini-map of the federation topology, model chips to switch between providers mid-session, a step-spine axis timeline, and a trace-import panel to replay downloaded walk JSONs. The step budget is 50, enough for deep multi-hop descents.
- **Why.** Federation graph traversal is opaque: it is hard to reason about why an agent took a particular path or where latency came from. The visualizer makes the walk inspectable.
- **How.** Open [[en/search_visualizer]], pick a model and a starting knowledge base, and run a query. The walk unfolds node by node. Download the trace JSON to replay or share it.

### memcli: local embedded vector search

- **What.** `memcli` now bundles an arm64-native retriever sidecar that runs vector search locally, without sending text to an external service. The sidecar handles both embedding and (optionally) reranking.
- **Why.** Vector search on a local memcli instance previously required a network call to an embeddings server. The bundled retriever makes local semantic search self-contained on Apple Silicon.
- **How.** The sidecar starts automatically with `memcli up` on arm64 macOS. No extra configuration is required. Guide: [[en/user/memcli]].

### Retriever: Metal/MPS support and UTF-8 snippet safety

- **What.** The retriever sidecar now runs inference on Metal (Apple GPU) and MPS backends with fp16 precision and micro-batch stability fixes. Search result snippets are cut on rune boundaries so non-ASCII text (Cyrillic, CJK, emoji) is never truncated mid-codepoint.
- **Why.** Inference on CPU was 3–10× slower than on the GPU. Truncated UTF-8 caused garbled snippet text in multilingual vaults.
- **How.** Automatic. The retriever selects the best available device at startup. No configuration change is needed.

### Ops metrics: embedding pipeline and job queue

- **What.** Two new Prometheus metric groups are now exposed on the internal metrics endpoint. (1) Embedding-pipeline metrics track chunk count, latency, and error rate for each embedding pass. (2) Job-queue depth metrics report how many jobs are waiting in each named queue. Additionally, chunk embedding requests are now sub-batched to fit the retriever server's declared maximum batch size, preventing a re-embed storm when the batch limit is lower than the chunk count.
- **Why.** Without queue-depth metrics, operators had no early warning of a backing-up embedding pipeline. The sub-batching fix prevented a crash loop where an oversized batch caused the embedder to reject the request, which triggered a full re-embed, which produced another oversized batch.
- **How.** New metrics appear on the existing internal metrics endpoint with no configuration change. Update your dashboards to add queue-depth and embedding-latency panels.

### Telegram: navigation browser limited to public live notes

- **What.** The in-Telegram navigation browser now shows only **public, live** notes. Previously it could surface notes that were private or in draft state.
- **Why.** A Telegram user clicking a navigation link should land on content that is actually accessible to them. Draft and private notes produced broken or access-denied pages.
- **How.** Automatic. Draft and private notes no longer appear in the Telegram navigation browser.

---

## v0.8.0 (2026-07-01)

### Fleet agent runtime

- **What.** Trip2g notes can now trigger LLM agents. You write "role notes" in a folder (default `roles/`): frontmatter is the config (model, tools, read/write path patterns, trigger, budget limits, concurrency policy), body is the instruction rendered as a Jet template against the trigger context. The `fleet` daemon discovers role notes, reconciles change/cron webhooks to itself, and when a trigger fires runs a scoped agent loop with tools: search, read_note, patch_note, write_note. An optional `executor: code` tool (e.g., Python) is off by default and gated by `--allowed-programs` / `TRIP2G_FLEET_ALLOWED_PROGRAMS`.
- **Why.** A vault becomes an automation surface (transcript ingestion, knowledge-base construction, triage, summaries) without external orchestration. Trip2g stays a plain event source; the fleet is the only moving part.
- **How.** The `fleet` binary ships inside the trip2g image at `/fleet` and as a standalone downloadable. Configure with `TRIP2G_FLEET_*` env vars (or `cmd/fleet` flags); auth uses an admin HAT minted from the app's JWT secret. Guide: [[en/agents_how_it_works]].

### Downloadable binaries with checksums

- **What.** Prebuilt archives are now attached to every GitHub Release: linux amd64/arm64, macOS arm64/amd64, Windows amd64. Each archive contains `trip2g-server` and `fleet`, plus a `.sha256` checksum file.
- **Why.** Run trip2g without Docker: a single binary on any of the five supported platforms.
- **How.** Download the archive for your OS from the Release page, verify with `sha256sum -c *.sha256`, extract, and run. macOS binaries are unsigned; Gatekeeper warns on first launch (right-click the binary and choose Open to allow it).

### `trip2g lint docs`

- **What.** A new `lint docs` subcommand runs the real note-loader over a docs tree and reports issues: cross-language wikilink leaks (a `ru/` note linking to an `en/` page), layout render errors, and broken links (advisory). It replaces the old `check-doc-lang-links.sh` bash script and is now wired into CI.
- **Why.** Catches publish-time problems before they go live: a Russian note accidentally linking to an English page, a layout that fails to render.
- **How.** `trip2g lint docs` (or `go run -tags dev ./cmd/server lint docs` from source). Exit code is non-zero on warnings; broken links to not-yet-created notes are advisory and do not block the check.

### Database metrics and stricter SQL

- **What.** Three new Prometheus metrics are now exposed on the existing internal metrics endpoint: DB connection pool stats, database file size, and WAL file size. Double-quoted string literals in SQL are rejected at startup via the `_dqs` pragma to surface typos early. The last direct `mattn/go-sqlite3` import was dropped from test tooling; the app has run on pure-Go modernc since well before this release.
- **Why.** The metrics give operators visibility into DB pressure before it becomes an outage. The `_dqs` pragma turns a class of silent SQL bugs into a startup error.
- **How.** Metrics appear on the existing internal metrics endpoint with no config change. The `_dqs` rejection may surface a pre-existing SQL typo on upgrade; check startup logs if the server refuses to start.

### Search: cross-encoder reranker removed

- **What.** The optional second-stage cross-encoder reranker (shipped off-by-default in a prior release) is removed, along with the Python sidecar (`reranker-server/`).
- **Why.** Two rounds of benchmarking showed it hurt search quality: nDCG dropped from 0.9221 to 0.8881 with 512-char passages and to ~0.39 with full-note passages. The cross-encoder over-weighted surface term overlap and promoted near-neighbour distractors that the existing bi-encoder + BM25 + RRF first stage had correctly ranked lower. Rationale is in `docs/dev/reranker.md`.
- **How.** Nothing to do. The feature was off by default. Any existing `vector_search.reranker.*` config keys are now ignored.

---

## v0.7.1 (2026-06-26)

### Wide pages (`wide: true`)

- **What.** Any note can now render full-width by adding `wide: true` to its frontmatter. The page drops both sidebars and the narrow reading column so the content fills the whole main area.
- **Why.** Wide content (kanban boards, big Mermaid diagrams, large tables) was cramped in the default 65ch reading column.
- **How.** Add `wide: true` to the note's frontmatter. Guide: [[en/user/default-template]].

### Live layout preview tool: `trip2g-preview`, replaces `renderlayout.py` (`19483421`, `d0fe4179`)

- **What.** A new standalone node CLI, `scripts/trip2g-preview.mjs`, renders a Jet layout against a note via `/_system/renderlayout` and adds a `--watch` mode: it re-POSTs on every save while a browser parked on the `?live` URL reloads itself. It replaces the old `scripts/renderlayout.py` (removed). No Python dependency.
- **Why.** Tightens the layout-developer loop: edit a `_layouts/*.html`, see the result live, catch Jet errors in the terminal. It targets any server: a local memcli (auto-reads `data.json`) or a remote/staging instance via `--api-url`/`--api-key`.
- **How.** `node scripts/trip2g-preview.mjs --watch --layout-file _layouts/article.html --note-path /about`, then open the printed `?live` URL (with memcli, run `memcli open` first so the browser is signed in). Guide: [[en/user/renderlayout]]; the two local design loops are documented in `docs/dev/local_design_iteration.md`.

### memcli: isolated instances with `--name` (`5f0542f8`, `353841e0`)

- **What.** `memcli up --name <id>` boots a second instance in its own container `trip2g-memory-<id>`. Pass the same `--name` to `down`/`status`/`logs`, and give it a distinct `--port`. With no `--name` the default (`trip2g-memory`) is unchanged.
- **Why.** Lets several local memcli instances run side by side (e.g. one per vault) instead of fighting over the single hardcoded container name.
- **How.** `node cli/memcli/dist/memcli.js up --folder ./v2 --name v2 --port 24381`. State stays per `--folder`. Guide: [[en/user/memcli]].

---

## v0.7.0 (2026-06-23)

### Read replica mode (`a4423895`)

- **What.** A second trip2g instance can now run as a read-only replica. Set `TRIP2G_LEADER_ADDR` on it and it will serve all GET requests from a LiteFS-replicated local SQLite copy and forward every write to the primary.
- **Why.** This halves read latency when the primary is under load, lets you restart the primary without dropping a single reader request, and opens the door to horizontal read scaling on separate machines. During a leader restart in tests, zero read failures were observed on the replica.
- **How.** Run LiteFS on both machines so the replica gets a streaming copy of the SQLite WAL. Start the replica with `--leader-addr=http://10.x.x.x:8082 --leader-shared-secret=...`. The `--leader-shared-secret` must match `TRIP2G_LEADER_REPLICA_SECRET` on the primary. Full wiring guide: [[en/user/read-replica]].

### OIDC / Corporate SSO login (`1d2b12b5`, `b6446f76`, `e6d7afe9`)

- **What.** Users can now sign in with a corporate identity (Authentik, Keycloak, any standards-compliant OpenID Connect provider). A "Sign in with SSO" button appears on the login page when an OIDC provider is configured. Auto-provisioning is optional: turn it on and a valid IdP identity creates the trip2g user automatically; leave it off and only pre-existing accounts are admitted.
- **Why.** Teams that already have a company IdP no longer need to manage separate trip2g passwords. One click, IdP MFA included.
- **How.** Set `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, and `OIDC_DISCOVERY_URL` on the server. No database row is needed; the provider is read from env at request time. Admin credential CRUD is available via GraphQL mutations for setups that need it. Full guide: [[en/user/oidc]].

### Local filesystem storage backend (`0f5b84e2`)

- **What.** Trip2g can now store uploaded assets (images, attachments) on the local filesystem instead of S3 or MinIO. Set `STORAGE_BACKEND=local` (or `--storage-backend=local`) and optionally `STORAGE_LOCAL_PATH` to pick the directory.
- **Why.** Running without an object storage service removes the main external dependency for single-server self-hosting. A $4 VPS with a volume attached is now a complete, standalone deployment.
- **How.** Add `STORAGE_BACKEND=local` to your env file. The default path is `./data/storage`; override with `STORAGE_LOCAL_PATH`. Switching from an existing S3 setup requires copying the bucket contents to the local path first. See [[en/user/local-quickstart]] for the full single-server setup.

### Zero-downtime deploys (`08a6cfde`, `22dd8cf3`)

- **What.** Trip2g now supports systemd socket activation (`LISTEN_FDS`) and exposes two health probes on a separate internal port: `/livez` (always 200, even during warmup) and `/readyz` (503 until the instance is warmed up and again during drain). Together they give a load balancer or orchestrator the signals it needs to cut traffic from old to new without dropping requests.
- **Why.** Before this, a rolling restart caused a gap: the old process was gone before the new one finished warming the note cache. Now the new instance warms up in parallel and traffic only moves once `/readyz` returns 200.
- **How.** Add the socket unit to systemd so the OS holds the port during the restart gap. Set `--internal-listen-addr=:8081` and wire your load balancer to wait for `/readyz` before routing. `--shutdown-grace-period` controls how long the old instance keeps serving after `/readyz` flips to 503. Use `--simple-backup-on-shutdown=false` to skip the shutdown backup during rolling restarts. Full recipes (Nomad/Traefik, Caddy, k3s): [[en/user/zerodowntime]].

### memcli: agent-memory bootstrap CLI (`a4e6fd1e`, `951d52ed`)

- **What.** `memcli` is a single-command tool that boots a trip2g instance as persistent long-term memory for AI agents. One command (`node memcli.js up --folder ./memory`) starts the Docker container with local storage, mints an admin key via HAT, starts the `trip2g-sync --watch` sidecar, and writes a `hub.md` federation note into the memory vault. The new `hub` subcommand and `memory_bind_hub` MCP tool bind the memory instance to a remote federation hub, so agent searches reach federated knowledge bases through a single MCP endpoint.
- **Why.** Setting up a standalone trip2g for an agent previously required four separate manual steps (server, key, sync, MCP config). memcli collapses them to one command. The compiled `dist/memcli.js` ships in the repository; no build step is required.
- **How.** `node cli/memcli/dist/memcli.js up --folder ./memory-vault`. When it finishes, the MCP endpoint is at `http://localhost:24081/_system/mcp`. Add it to your agent's `.mcp.json`. Full guide: [[en/user/agent-memory]].

### gitapi mirror stability (`2845bb82`, `8082c24d`)

- **What.** Two fixes to the DB→git mirror (gitapi). When a `materialize` call fails mid-way, orphaned loose objects are now cleaned up before returning the error. After a successful materialize, gc runs immediately to pack loose objects. Previously, failed runs could leave orphaned files that accumulated across nightly rebuilds and eventually filled the disk; objects were packed only on the next gc cycle.
- **Why.** On instances with many notes, repeated materialize failures left the git data directory growing unbounded. Operators with a small VPS saw disk pressure that required manual cleanup.
- **How.** No action needed. Both behaviors are automatic after upgrading. If you saw disk growth from orphaned objects, a `git gc --prune=now` in the repo directory (or a server restart that triggers the next materialize) will clear the backlog.

---

## v0.6.1 (2026-06-22)

### Live updates in the Obsidian plugin (plugin v0.5.0)

- **What.** The trip2g sync plugin now consumes the `noteChanges` subscription (shipped in v0.6.0) and pulls server-side changes into your vault in real time. No sync click, no waiting for the periodic check.
- **Why.** For multi-device and agent-driven editing, a change made on the server (or another device) shows up in Obsidian within a second instead of after the next poll. The benefit is instant freshness and less idle traffic. This is a UX change, not a backend speed gain: the underlying note-list query was already a ~10 ms indexed read.
- **How.** Update the plugin to 0.5.0 (via BRAT), turn on **Two-way sync**, then set **Live pull patterns** in the plugin settings (include/exclude globs, e.g. `**`, or `blog/**` excluding `drafts/**`). Safety is preserved: local edits are never overwritten (you get a conflict prompt), and server-side deletions ask before removing locally. The 60-second background poll becomes a lighter 5-minute reconciliation backstop. User docs: [`docs/en/user/two-way-sync.md`](./en/user/two-way-sync.md). The story: [`docs/en/thoughts/sync-benchmark.md`](./en/thoughts/sync-benchmark.md).

### Much faster bulk and CLI sync of notes with assets

- **What.** Syncing many notes that embed images is dramatically faster from the CLI and browser-sync. A cold push of 2000 notes with 2000 images dropped from **231.8 s to 8.8 s (~26×)**.
- **Why.** Each asset upload was triggering a full server-side note reload, because the CLI and browser-sync didn't batch uploads the way the Obsidian plugin already did. One missing flag (`skipCommit`) turned 2000 uploads into 2000 full reloads.
- **How.** No action needed beyond updating to plugin/CLI 0.5.0. The interactive Obsidian plugin was already unaffected.

### Stability: the real-time subscription could crash the server (`c283963b`)

- **What.** A race in the in-process event bus behind the `noteChanges` subscription could panic the whole server when a subscriber disconnected during a save (`send on closed channel`).
- **Why.** It never fired while nothing subscribed, but the new live-pull plugin connects and disconnects routinely, making it a real risk for anyone running the subscription.
- **How.** No action needed; the bus now uses a per-subscriber done channel and is race-tested under stress. A **Sync now** command was also added to the plugin (command palette).

---

## v0.6.0 (2026-06-17)

### Mermaid diagrams

- **What.** A ` ```mermaid ` fenced code block in any note renders as a diagram on the published page: flowcharts, sequence diagrams, pie charts, Gantt charts, class diagrams, state diagrams, ER diagrams, and every other type Mermaid supports.
- **Why.** Write diagrams the same way Obsidian renders them. The block just works.
- **How.** Add a code block with the `mermaid` language tag and paste your diagram syntax. The Mermaid library loads lazily and only on pages that contain a `mermaid` block; pages without diagrams pay no loading cost. User docs: [`docs/en/user/mermaid.md`](./en/user/mermaid.md), [`docs/ru/user/mermaid.md`](./ru/user/mermaid.md).

### Charts from `datachart` blocks (`7cadad3e`, `cb1403f5`)

- **What.** A ` ```datachart ` fenced code block becomes an interactive chart on the published page, powered by Apache ECharts. Data can come from four sources: `inline` rows bundled in the block, a vault file referenced via a frontmatter `[[wikilink]]` (`frontmatter`), an external HTTP-JSON endpoint fetched on the server and cached (`url`), or your site's own content via SQL (`internal`, coming soon). URL fetch errors are recorded and surfaced to authors so a broken endpoint is visible at sync time, not only in the browser (`97997bb4`).
- **Why.** Publish live charts as naturally as you write any other Obsidian note. The ECharts widget loads lazily, only on pages that contain a `datachart` block.
- **How.** Add a fenced block with the language tag `datachart` containing a JSON object with `data` and `config` keys. The `config` object is passed directly to ECharts, so any chart type and option it supports works here. User docs: [`docs/en/user/chartdata.md`](./en/user/chartdata.md), [`docs/ru/user/chartdata.md`](./ru/user/chartdata.md).

### MCP Federation: admin topology endpoint and configurable `kb_id` (`4485b6cd`, `5c7492ae`)

- **What.** Two additions to the federation layer introduced in v0.4.1. A new admin-gated `GET /_system/federation/admin` endpoint returns the full federation topology: all registered KB peers, their scopes, and their reachability status. The `kb_id` for your instance is now configurable; if not set explicitly it falls back to the public URL host.
- **Why.** Operators running multiple federated instances can inspect the topology without digging through the database. A configurable `kb_id` lets you assign a stable, human-readable identifier that stays correct regardless of which domain the instance answers on.
- **How.** The topology endpoint is admin-only (requires an admin API key or session). Set `kb_id` in your instance config to override the default host-derived value. No changes needed to existing federation setups.

### Live note-change SSE subscriptions (`854c56b0`)

- **What.** A new `noteChanges` GraphQL subscription streams note upsert and removal events to connected clients over SSE, with optional glob filtering. Each event carries the changed HTML selectors diff so clients can patch the DOM without a full page reload.
- **Why.** Enables live-updating UIs (an admin editor, a preview pane, or a custom dashboard) that reflect vault changes the moment they land on the server.
- **How.** Subscribe to `noteChanges(glob: "posts/**")` via the GraphQL SSE endpoint. The `changedHtmlSelectors` field on each event lists the CSS selectors whose rendered HTML changed, making surgical DOM updates possible.

### In-browser editor: Ctrl+Click wikilink navigation (`a065fea2`)

- **What.** In the in-browser file editor (introduced in v0.5.0), Ctrl+Click (or Cmd+Click on macOS) on a wikilink opens the linked note directly.
- **Why.** Matches the Obsidian editing experience and removes the need to search for a linked file manually.
- **How.** Open the editor, hover over any `[[wikilink]]`; the cursor changes to a pointer. Ctrl+Click to navigate.

---

## v0.5.1 (2026-05-27)

### Per-webhook secrets injected into delivery payload (`0b72acf2`)

- **What.** Each change webhook and cron webhook now has a **Secrets** panel in the admin. Add named key-value pairs (e.g. `auth_token`, `api_key`); they are stored encrypted and sent in every delivery payload under `payload.secrets`. The webhook consumer can read them without any extra API calls.
- **Why.** Cron webhooks are the foundation of a plugin system. A plugin is a web server (or serverless function) that receives a payload with a short-lived API token and processes it (often in the background), then patches the vault when ready. Because plugins are stateless, they have no safe place to store their own credentials. Secrets solve this: trip2g holds them encrypted and delivers them on every call, so the plugin stays credential-free on its end. A plugin that needs more time can use the API token to push updates back asynchronously; secrets give it everything else it needs to talk to external services.
- **How.** Open Admin → Change Webhooks (or Cron Webhooks) → select a webhook → scroll to the **Secrets** section. Enter a name and value, click **Add Secret**. To update a value, type in the row's value field and click **Save**. To remove, click the trash icon (confirm on second click). Secrets appear in the delivery payload as `{ "secrets": { "auth_token": "...", "api_key": "..." } }`.

## v0.5.0 (2026-05-26)

### In-browser file editor (admin)

- **What.** Admins can now edit any page right on the site. An editor icon appears in the top-right of the admin panel and next to the search on every page. Open it to browse every uploaded file as a folder tree, view and edit any one, and save. You can also roll a file back to an earlier version.
- **Why.** Fix a typo or update a page in seconds. No Obsidian, no re-sync.
- **How.** Click the editor icon (admins only). The current page's note opens by default; pick any other file from the tree on the left. Edits stay in your browser until you press **Save**, and the versions panel lets you load and restore an older version.

### Public hub of curated bases

- **What.** A `hub/` section with a bilingual index of the knowledge bases reachable through the hub (first entry: the Nick Senin Journal (filtered Code with Claude 2026 cases)).
- **Why.** A browsable, public entry point to federated bases.
- **How.** See [`docs/en/hub/_index.md`](./en/hub/_index.md); add your own with [`docs/en/hub/_create.md`](./en/hub/_create.md).

## v0.4.1 (2026-05-25)

### MCP Federation: one hub across many knowledge bases

- **What.** Your instance can act as a federation hub. A **KB-note** (a note with `mcp_federation_kb_url` in frontmatter) registers another MCP-compatible base, and `federated_search` / `federated_similar` / `federated_note_html` reach across all of them through your single MCP endpoint. Public bases need no auth; private peers use a shared HMAC secret.
- **Why.** One endpoint, one auth surface: your agent searches your own notes, partner instances, and external adapters (GitHub, Telegram) together, without rewiring `.mcp.json`.
- **How.**
  - User docs: [`docs/en/user/federation.md`](./en/user/federation.md), [`docs/ru/user/federation.md`](./ru/user/federation.md)
  - Public base: create a note with `mcp_federation_kb_url` (+ optional `mcp_federation_kb_id`) and `free: true`.
  - Private peer: exchange a federation secret in Admin → Federation, then add the KB-note.

### Canvas files (Base & Excalidraw coming later)

- **What.** `.canvas` files sync and render. `.base` and `.excalidraw` files are accepted by sync too, but rendering them is planned for later; for now they show a clear placeholder instead of breaking the page.
- **Why.** Canvas vaults work today; Base and Excalidraw vaults sync without errors while full support is on the way.
- **How.** Just sync. The plugin and CLI accept all three extensions; Canvas renders now.

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
  - **New `--exclude <glob>` flag** (repeatable). Excluded paths are never pushed; if they already exist on the server they are **hidden**. A bare name like `dev` matches that directory and everything under it. Default: nothing is excluded. Everything uploads.
- **Why.** Keep test/demo or internal folders (e.g. `dev/`, `demo/`) in your repo but out of the published site, and reversibly hide them on the server.
- **How.** `trip2g-sync ./docs --exclude dev --exclude demo`. Re-including a path re-publishes and automatically unhides it.

---

## v0.4.0 (2026-05-21)

### updateNotes mutation: atomic find/replace across notes

- **What.** New GraphQL mutation that patches multiple notes in one transaction via a `PathMap` of `{path → [{find, replace}]}`.
- **Why.** Lets external tools, agents, and scripts apply consistent edits across a vault without orchestrating per-note round-trips. Avoids partial states when one of the replacements fails.
- **How.**
  - User docs: [`docs/en/user/update_notes.md`](./en/user/update_notes.md), [`docs/ru/user/update_notes.md`](./ru/user/update_notes.md)
  - Example: send `updateNotes(input: { pathMap: { "notes/post.md": [{ find: "old", replace: "new" }] } })`.
  - End-to-end spec: `e2e/updatenotes/*` (see `test(e2e): add updateNotes e2e spec and demo fixture`).

### Forms admin: submit processing

- **What.** New `markFormSubmitProcessed` mutation and `processed` fields on form submits; admin can mark a submit as handled, the UI hides processed entries by default.
- **Why.** Closes the form-handling loop inside the admin instead of forcing external triggers.
- **How.**
  - User docs: [`docs/en/user/forms.md`](./en/user/forms.md), [`docs/ru/user/forms.md`](./ru/user/forms.md)
  - Dev reference: [`docs/dev/forms.md`](./dev/forms.md)
  - Use `can_submit` / `success_url` frontmatter on form notes to gate submissions and customize the thank-you page.

### Layout smoke-render: surface Jet runtime errors at load

- **What.** When notes are loaded, each parsed Jet layout is executed against up to **10 first notes** that select it via frontmatter `layout:`. Runtime errors and panics become `NoteWarning` entries on the layout.
- **Why.** Previously, a broken layout that *parsed* (e.g. `{{ note.NoSuchField }}`) only failed when a user opened the page in the browser. Agents pushing notes had no signal. Smoke-render moves the failure to load time so warnings show up in the same channel as parse errors, visible via `pushNotes` / admin layout listings without a browser request.
- **How.** Automatic; no flags. Watch for `smoke render error` / `smoke render panic` in layout warnings after a sync. Layouts without parsed templates and layouts no note uses are skipped.

### Template debugging: `Meta.Debug()` and global `debug()`

- **What.** Inside Jet templates: `{{ Meta.Debug() }}` dumps note metadata; global `{{ debug(<any>) }}` prints type, value, and the method set of any expression via reflection.
- **Why.** Removes the "guess what the template sees" loop when authoring layouts and components.
- **How.**
  - User docs: [`docs/en/user/jet.md`](./en/user/jet.md), [`docs/ru/user/jet.md`](./ru/user/jet.md) (debugging section).
  - Example: `{{ debug(note.M()) }}` or `{{ debug(note.Title) }}`.

### `renderlayout.py` CLI: render a layout against a note

- **What.** Standalone script in `scripts/renderlayout.py` plus a `check_templates` agent skill.
- **Why.** Lets you preview layouts and reproduce template issues from the terminal, useful in CI and when iterating on `_layouts/`.
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

## v0.3.1 (historical backfill)

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
- **Why.** Drop-in setup for AI agents over an Obsidian vault. No manual wiring.
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

## v0.3.0 (historical backfill)

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
- **Why.** Authors edit chrome the same way they edit content. No template hacks.
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
