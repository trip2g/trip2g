# Changelog

All notable releases of trip2g. Newest on top. Each release lists user-facing
changes with **What / Why / How to use** so an operator or admin can decide
whether to upgrade and how to start using a feature.

Older tags (`v0.2.0` and below) live in git history only.

---

## Unreleased

### Site search can leave out notes the reader cannot open

- **What.** A new `SEARCH_HIDE_UNREADABLE` setting (`--search-hide-unreadable`), off by default. When on, site search returns only notes the reader can open. When off, as before, every note the reader cannot open is listed after the results as a "Закрытый материал." placeholder with its real title and link.
- **Why.** The placeholders show that locked material exists, which suits a paid site. A site whose closed notes should stay unseen had no way to keep their titles and links out of search.
- **How to use.** Set `SEARCH_HIDE_UNREADABLE=true` and restart. Admins and readers with access are not affected, and neither is MCP search. See [[en/user/search#Notes a reader cannot open|How search works]].

### MCP `expand` declares `toc_path` as a list of strings again

- **What.** The `expand` tool's input schema says `toc_path` is an array of strings, and `last` is a plain number. Since v0.11.0 the schema carried `toc_path` as an array with no item type and put that item type on `last`.
- **Why.** An MCP client that checks tool schemas strictly could refuse `expand`, or the whole tool list, because of it.
- **How to use.** Nothing to do. A client that cached the tool list picks up the fixed schema on its next `tools/list`.

### A signed-in reader opens a `free: true` note

- **What.** A note with `free: true` opens for every reader: a guest, a signed-in user with no access to its subgraph, and an admin. Before, a signed-in user without a grant got a paywall on a free note that a guest could open, and a form on that note answered `form_not_found` to them. The page, the GraphQL `note` query, search, similar notes, MCP, assets, live note updates and form submission all read the same check, so all of them change together. Other notes of a paid or sign-in subgraph stay closed as before.
- **Why.** `free: true` publishes a note for everyone. Signing in must not close what a guest can read.
- **How to use.** Nothing to do.

### A guest opens a `free: true` note in a sign-in subgraph

- **What.** A `free: true` note in a subgraph marked **Require sign-in** opens for a guest too. Before, a guest got the sign-in wall on its page and in the GraphQL `note` query, its assets answered 401, it never reached the page cache, and RSS feeds and other `.Public()` lists left it out, while search, similar notes and MCP already showed it to that same guest. Now every path answers the same. A note in such a subgraph without `free: true` still shows a guest the sign-in wall. The sitemap lists such a note too, see below.
- **Why.** `free: true` publishes a note for everyone, whatever its subgraphs say ([[en/user/subgraphs]]). The sign-in wall contradicted the rest of the site.
- **How to use.** Nothing to do. Check your sign-in subgraphs for `free: true` notes you meant to keep for signed-in readers, and drop `free` from them. A free note shows the notes it embeds with `![[...]]` to whoever reads it, so a free note in a sign-in subgraph that embeds a closed note now shows that closed content to guests too.

### The sitemap lists a `free: true` note in a sign-in subgraph

- **What.** `sitemap.xml`, the main one and each custom domain's, lists a `free: true` note of a subgraph marked **Require sign-in**. Before, it left out every note of such a subgraph, free or not. The sitemap now asks the same question as the page, RSS feeds and the page cache, so it cannot disagree with them again. It still leaves out notes without `free: true`, `noindex: true` notes, hidden notes, `.html` files and anything under a `/_` path.
- **Why.** Such a note opens for a guest (see above), so a search engine should find it like any other free note.
- **How to use.** Nothing to do. To keep a free note out of search engines, add `noindex: true`.

## v0.11.1 (2026-10-07)

### Payment webhooks moved under `/_system/`

- **What.** NOWPayments notifications now arrive at `/_system/nowpayments/ipn`, Patreon webhooks at `/_system/patreon/webhook`. Nothing answers under `/api/` any more. A NOWPayments notification is refused when `--nowpayments-ipn-key` is not set.
- **Why.** Every system endpoint lives under `/_system/`; these two were the last ones outside it.
- **How to use.** NOWPayments: nothing to do, trip2g sends the address with every invoice. An invoice created before the upgrade and paid after it still notifies the old address, which no longer answers: after the upgrade, check pending purchases and grant access by hand to any that NOWPayments shows as paid. Patreon: nothing to do, trip2g registers its webhook itself. On a normal restart the old version removes the webhook at the old address when it stops, and the new version registers the new address when it starts. If the old version did not stop cleanly, the new one still registers the new address; the leftover webhook at the old address does nothing and can be deleted in the Patreon creator portal. Patreon members are also refreshed every hour either way.

### Revoke a subgraph access from the admin panel

- **What.** `/api/admin/revokeusersubgraphaccess` is removed. In its place, the admin mutation `revokeUserSubgraphAccess` and a **Revoke** button on an access page in the admin panel. A revoke needs a reason; the access page then shows when it was revoked, by whom and why, and the access list gains a "Revoked At" column.
- **Why.** The old endpoint did not check the admin role, and nothing in trip2g called it. A revoke now goes through the `admin` namespace, which does.
- **How to use.** Admin → Users → Subgraph Accesses → open the access → enter a reason → Revoke. Or call `revokeUserSubgraphAccess` with the access id and a reason ([[en/user/user_management]]).

### Docs: an app on top of trip2g

- **What.** [[en/user/spa|An app on top of trip2g]] now covers the whole path of a JavaScript app built on a custom layout: how the app is authorised by the visitor's session cookie, settings from the frontmatter, the site's standard header and footer around the app, saving with `expectedHash`, live updates over server-sent events, API calls and admin calls, MCP from the browser over the whole base, an API of your own that asks trip2g who may call it, forms, and a small shop as an example. An API of your own goes under `/_system/extra/`: trip2g serves nothing there, and a test in its router keeps it that way. [[en/user/templates#Standard header and footer in your own layout|Templates]] shows the standard header and footer in any custom layout.
- **Why.** The page described only the layout, the bundle and saving. Building a real app meant reading the source for auth, live updates and where to mount a backend.
- **How to use.** Nothing to do. Read [[en/user/spa|An app on top of trip2g]] before building an app.

### Docs: the default template draws no form

- **What.** [[en/user/forms|Forms]] now says that the default template only embeds a note's form as JSON: it draws no fields and no captcha. To show a form, use [form_template](https://github.com/trip2g/form_template), a ready-made layout that renders the form as a survey, or a layout of your own. The page also documents the `TurnstileRequiredPayload` result of `submitForm`.
- **Why.** The page said the default template rendered the form at the end of the page and handled Turnstile. It does neither, so a note with a form showed no form.
- **How to use.** Put `form.html` from form_template into `_layouts/` and set `layout: form` on the note. See [[en/user/forms|Forms]].

---

## v0.11.0 (2026-10-07)

### Previous/next links and breadcrumbs

- **What.** The default template shows breadcrumbs above the title and "Previous / Next" links at the end of a page. They come from the first sidebar note, left then right, that links to the page: the neighbouring links, and the heading above the link. Wikilinks and markdown links both count. Frontmatter `prev`, `next` and `breadcrumbs` override them; `false` turns each off.
- **Why.** Documentation read in order had no way to the next page except the sidebar, and no sign of which section a page belongs to.
- **How to use.** Nothing to do if your pages already have a sidebar note. To turn it off for a folder, set `prev: false`, `next: false`, `breadcrumbs: false` with a frontmatter patch. See [[en/user/default-template#Previous/next links and breadcrumbs|Default template]].

### Frontmatter links resolve like in Obsidian

- **What.** Wikilinks in `header`, `footer`, `left_sidebar` and `right_sidebar` resolve by the same rules as links in note text: by file name from the note that has the property, with `|alias` and `#heading` ignored.
- **Why.** These links were matched against the page URL only, so `[[Folder/_sidebar]]` with a non-Latin folder, `[[Site Footer]]` or `[[Note|alias]]` found nothing and the sidebar or header silently disappeared, even though Obsidian showed a valid link.
- **How to use.** Nothing to do. In the default global mode a link that worked keeps pointing at the same file; one that found nothing now works. In scoped mode a bare `[[Name]]` now picks the nearest file, as links in note text do. See [[en/user/wikilink-resolution#Links in frontmatter|Wikilink resolution]].

### Admins can hide notes

- **What.** `hideNotes` and the `hide` change in `updateNotes` work for a signed-in admin: a browser session, a personal token `t2g_…` in a header or in `?token=`, or a session token. The `hide` change also works with a webhook token, within its write patterns. The hide is recorded against the signed-in admin, the admin who created the API key, or the admin who created the webhook.
- **Why.** Both failed with an `errors` entry for everyone but an API key, and the `hide` change failed for a webhook token the same way: the hide had no admin to record.
- **How to use.** Nothing to do. A webhook token issued before the upgrade still gets `ErrorPayload` from the `hide` change until it expires. See [[en/user/update_notes#hide — remove a note from public view|updateNotes]].

### A base layout can mark a slot with a bare yield

- **What.** A base layout may mark a slot with `{{ yield main() }}` instead of `{{ block main() }}…{{ end }}`; the pages that extend it fill the slot with their own `{{ block main() }}`. Auto-import never pulls in a layout that starts with `{{ extends }}`: such a file is a page, not a component.
- **Why.** Auto-import found `main` in a page that extends the base and pasted that whole page into the base. The base failed to load, and every page extending it failed with `template /base could not be found`.
- **How to use.** Use a bare `yield` for a slot every page must fill, and `block` for a slot with a default. See [[en/user/templates#Layout inheritance: extends|Templates]].

### Wider reading column on large screens

- **What.** The default template's reading column grows on wide screens: 80 characters from 1440px. From 1536px the column is 56rem wide, the layout grows to 1680px with wider side columns, and the base font size stays at 18px instead of growing with the screen. Below 1440px nothing changes.
- **Why.** On a large monitor text was squeezed into a narrow strip and tables and code examples needed sideways scrolling. `wide: true` fixed that but hid both sidebars.
- **How.** Nothing to do. `wide: true` stays for pages that need the full width (boards, big diagrams).

### Wikilinks to a heading scroll to it

- **What.** `[[Note#Heading text]]` and `[[#Heading text]]` now link to the heading's id, the same one the page and its table of contents use. The heading text is matched exactly, then ignoring case and extra spaces; an id such as `faq-2` or one set with `{#id}` works too. A fragment that matches no heading now marks the link as broken, and `trip2g lint` reports it as `broken link: Note#Heading`.
- **Why.** The fragment went into the URL as typed (`#Heading%20text`) and matched no id, so the link opened the top of the page, and a link to a heading in the same note was marked as broken.
- **How to use.** Nothing to do: links written the Obsidian way now work. See [[en/user/markdown#heading-anchors|Markdown syntax]].

### "Links here" are sorted alphabetically

- **What.** The backlinks block at the end of a page and `nvs.BackLinks(note)` in templates list notes by title (case-insensitive), then by permalink.
- **Why.** The list was built from a map, so its order changed from one render to the next: a link was hard to find and the same page looked different each time.
- **How.** Nothing to do.

### Layouts escape output by default

- **What.** A custom layout now HTML-escapes what it prints. `{{ note.Title() }}`, a frontmatter value, `{{ note.ContentString() }}` and every other plain string reach the page as text: `<`, `>`, `&`, `"` and `'` become entities. HTML the server builds itself is still printed as is: `note.HTMLString()`, `FirstListHTML()`, a section's `TitleHTML` and `ContentHTML`, a code block's `HTML`, `FormSpecJSON()`, `SubgraphNamesJSON()`, `asset()` and `defaultTemplate.*`.
- **Why.** Layouts printed everything raw. A note whose markdown held `</textarea><script>` broke out of a `<textarea>{{ note.ContentString() }}</textarea>` and ran its script, and a title with `&` or `<` broke the markup around it. Each layout had to remember `| html` on every value, and one missed value was enough.
- **How to use.** Existing layouts keep working. `| unsafe` and `| raw` still print a string raw. `| html` and `html(x)` still escape, exactly once, so an RSS layout that writes `html(n.HTMLString())` gets the same output as before; the filter is no longer needed on text. `| json` keeps escaping `<`, `>` and `&` inside JSON, so `{{ x | json }}` stays safe in a `<script>`. `| url` escapes once.
- **Migration.** Check a layout that prints markup from a plain string: a block parameter such as `title="Line one<br>line two"`, or HTML glued with `+` or returned by `replace(...)`. That output now shows the tags as text; print it with `{{ title | unsafe }}`. Values from `htmlInjectionsHead` / `htmlInjectionsBodyEnd` stay plain strings, so keep `{{ injection.Content | unsafe }}`. `asset()` now percent-encodes `"`, `'`, `<`, `>`, a backslash, a backtick and whitespace in a URL; ordinary URLs, including signed storage URLs with `&`, are unchanged. See [[en/user/templates|Templates]].

### Your own heading anchors, and more of a note's content for templates

- **What.** A heading can set its own anchor: `## Pricing {#plans}` gets `id="plans"`; `{.class}` adds a CSS class. In templates, a section now has `ID` and `Level`, and `Section(...)` also finds a heading ignoring case and extra spaces, or by its anchor (`"plans"` or `"#plans"`). New: `Images()` (Markdown images and `![[...]]` embeds), `CodeBlocks(lang)`, `Task` and `TaskMark` on list items (Obsidian's custom statuses like `[/]` included), and the `parseJSON`, `parseYAML` and `parseCSV` functions.
- **Why.** A generated anchor changes whenever the heading text does, and breaks links to it. A template could split a note into sections but could not link to them, find a section by its anchor, or read a note's images, code blocks or task states.
- **How to use.** Add `{#id}` at the end of a heading, see [[en/user/markdown#heading-anchors|Markdown syntax]]. Template functions are in [[en/user/templates#own-toc|Templates]]. A heading that already ends with braces in this form, such as `## Setup {#install}`, loses them from its text and gets that id.

### An empty frontmatter field no longer breaks sync

- **What.** A note with `description:` or `redirect:` and no value (or `null`, `~`) now loads, with the field treated as not set.
- **Why.** Such a note used to fail the load, and the whole sync batch of a hundred notes failed with "invalid description type: <nil>". Several pages of Obsidian's own help docs are written this way.
- **How to use.** Nothing to do. If sync used to fail with this error, run it again.

### Search finds a note by its alias

- **What.** Site search and MCP search now read the `aliases` (and `alias`) frontmatter field. When the query as a whole equals a note's title or one of its aliases, that note gets a boost worth twice a text or vector match, which normally puts it first. Case, extra spaces and `ё`/`е` don't matter.
- **Why.** Aliases used to play no part in search: a note everyone calls "Starred" but titled "Bookmarks" could not be found by "Starred". On Obsidian's help docs, queries equal to an alias now find the right page almost every time (nDCG@5 0.81 → 0.98), and results for other queries did not change.
- **How to use.** List alternative names in the frontmatter, as in Obsidian: `aliases: [Starred, Favourites]`. Aliases do not create new URLs.

### MinIO image replaced with Silo

- **What.** The Compose files and the self-hosting guides now run `pgsty/silo` instead of `minio/minio` (`quay.io/minio/minio` in `docker-compose.yaml`).
- **Why.** MinIO ended its open-source distribution and deleted the `minio/minio` images from Docker Hub on 11 September 2026. `docker compose up` fails with `pull access denied for minio/minio`. Silo is the last open-source MinIO server, rebuilt and patched by the Pigsty project: same S3 API, same `MINIO_*` variables, same on-disk format.
- **How.** In your own `docker-compose.yml`, change `image: minio/minio:latest` to `image: pgsty/silo:latest` and run `docker compose up -d`. Keep the volume: Silo reads the existing data as is. See [[en/user/selfhosted|Self-hosting]].

### Sync no longer mistakes a note for a font or an image

- **What.** The server no longer rejects ordinary notes with an error like "Unsupported content type: application/vnd.ms-fontobject" (or `audio/mpeg`, `image/gif`, `text/xml`). It now checks one thing only: the content is UTF-8 text with no NUL bytes. A binary file posing as a note is still rejected, now with "File content must be UTF-8 text".
- **Why.** The file type was guessed from the first bytes, the way a browser does it. For text that check misfires: a note with the capital letters `LP` landing on bytes 35 and 36 (in a title such as "T HELPER" or "NLP", say) counted as an EOT font. A note starting with "ID3" counted as music, one starting with "GIF89a" as an image. Such a note never synced, and the error gave no clue what was wrong with it.
- **How to use.** Nothing to do. If a note used to fail to sync with this error, run the sync again.

### Embeds, preview blocks and links render reliably

- **What.** A note that embeds another note (`![[note]]`) no longer comes out blank when the embedded note has no body, when embeds form a loop, or when they are chained many levels deep. A loop now shows up as an "embed cycle" warning on the note instead. The free preview (`free_paragraphs`, `free_cut`) keeps tables and callouts, each counted as one block. Wikilinks to `javascript:`, `vbscript:`, `file:` and `data:` addresses render without the address, as ordinary Markdown links already did. An embedded tweet is now X's standard embed, built in the reader's browser: the server no longer fetches it. Quail widgets and ads (`quaily.com`, `quaily://`) and Dify chatbots (`udify.app`, `dify://`) are no longer embedded; such a URL in image syntax renders as a plain image.
- **Why.** Pages could lose all their content without any warning, a leading table vanished from the free preview, and a tweet embed made every sync wait on X. Pages rendered at the same moment for different visitors could also swap each other's image links in the footer and cards.
- **How.** Nothing to do. If a note shows the new "embed cycle" warning, remove one of the embeds that forms the loop.

### `noindex: true` keeps a page out of search engines and sitemap.xml

- **What.** A new `noindex` note property. A note with `noindex: true` is left out of `sitemap.xml` (the main one and every custom domain's), out of RSS feeds and any `.Public()` listing in a layout, and carries no JSON-LD. Its page gets `<meta name="robots" content="noindex">` in the default template and an `X-Robots-Tag: noindex` header on every response, custom layouts included.
- **Why.** A page meant for a direct link only (an offer, a draft, a demo) could print its own robots meta tag from a custom layout, but it still showed up in `sitemap.xml`, so its URL was public to anyone who read the sitemap.
- **How.** Add `noindex: true` to the note's frontmatter, or set it for a folder with a frontmatter patch. It is not access control: anyone with the link can still open the page. See [[en/user/seo|SEO]].

### Closed pages say they are closed, and offer sign-in instead of a wait list

- **What.** A page a visitor has no access to now reads "This page is closed", tells them to ask the owner of the knowledge base for access, and shows the sign-in form right there. The wait-list block (Telegram bot, e-mail field) is off by default and appears only if you turn on the new **show_waitlists** config value; it still shows only when nothing is on sale.
- **Why.** The wait-list e-mail field was the only input on the page, while the text above it said "please sign in", so visitors typed their sign-in address into a wait list and waited for a letter that was never coming. Most knowledge bases are closed to keep them private, not to sell access later.
- **How.** Nothing to do: closed pages get the new text and the sign-in form automatically. If you do collect contacts while offers are not ready, set **show_waitlists** to true in Admin → System → Config.

### Obsidian on the phone: open a note on the site, and close the warnings view (plugin v0.11.0)

- **What.** The note's `...` menu now has **Open on site** and **Copy site link** (only for notes that are actually published), plus a new **Open published URL in browser** command. The sync warnings view got a **Close** button, and its toolbar buttons are large enough to hit with a thumb.
- **Why.** Neither was reachable on mobile. The published-URL indicator lives in the status bar, which Obsidian mobile doesn't show, so the only route to a note's page was the "Copy published URL" command; and dismissing the warnings tab meant a detour through the tab switcher.
- **How.** Update the plugin to 0.11.0 or later, then tap `...` on a note → **Open on site**. For one-tap access, pin **Open published URL in browser** in Obsidian's Settings → Toolbar (mobile).

### Delivery chains: trace every webhook step across the agent graph

- **What.** Every webhook delivery now records which delivery caused it and which chain it belongs to. A new **System → Delivery Chains** screen lists the chains, and opening one walks its steps: what each step wrote, which write triggered it, its status, what it cost and how deep in the chain it sat. Chains span separate agent hosts (one host running the code role, another running the LLM roles) without either knowing the other exists.
- **Why.** When a change triggers a webhook that writes a note that triggers another webhook, the full sequence was previously invisible: you could see individual delivery records but not the chain that connected them. This makes the entire propagation path inspectable from a single admin screen.
- **How.** Open Admin → System → Delivery Chains and pick a chain to see its steps. Chains that wrote nothing are hidden: a schedule that finds no work still runs, and those runs would bury the rest. Turn on **Show empty** to see them. Nothing to configure: every delivery made after the upgrade is linked.

### Agents report what a run cost in their own units

- **What.** A webhook agent's response now carries a `costs` object instead of the fixed `tokens_used` and `steps` fields: `{"costs": {"tokens": 5186, "steps": 2}}`. The unit is the key, so an agent that bills money reports `{"usd": 0.004}` and one that counts anything else reports that. The admin shows whatever arrived, per step and summed per unit over the whole chain.
- **Why.** Tokens and steps are one executor's vocabulary. Anything else (a paid API, a credit balance, a queue quota) had nowhere to report itself, and trip2g carried a notion of "tokens" it has no business knowing about.
- **How.** Return `costs` as an object of numbers from your webhook agent. Every value must be a number: a string or an object under `costs` makes trip2g read the whole response as unparseable, so its `changes` are not applied either. The old `tokens_used` and `steps` fields are gone; an agent that still sends them simply reports no cost.

### Idempotent writes no longer fire change events

- **What.** An `updateNotes` call, a sync push or a change in a webhook agent's response whose content equals what is already stored creates no new version and raises no change event. No webhook delivery is queued, no SSE event is emitted, no re-embedding runs, and no Telegram publish-view refresh happens. Changing at least one byte still fires normally. Hiding a note still works and raises no event. Writing to a hidden path brings the note back (the only way to unhide); through `updateNotes` or a sync push that counts as a change, so the restored note fires the usual events.
- **Why.** An agent that re-applied identical content re-triggered the webhook that caused it on every cycle, at full LLM cost, bounded only by `max_depth`. The fix stops the loop at the write layer: if nothing changed, nothing downstream needs to know.
- **How.** Automatic. Agents that write back the full note content unchanged no longer cause runaway chains. The `max_depth` guard remains useful for agents that do produce genuine changes.

### Filter note paths by frontmatter

- **What.** `notePaths` accepts `filter: { frontmatter: [...] }`. Each predicate names a `key` and either `exists: true`, `exists: false` or `equals: "<string>"`. Every predicate has to match, and they also apply on top of `paths`, `like` and `search`. They check the frontmatter of each note's latest version after frontmatter patches. `equals` matches string values only: `equals: "5"` does not find `count: 5`, and `equals: "true"` does not find `draft: true`.
- **Why.** To find every note with a given property (for example, every role with `fleet_id: codellm`), a script had to load the notes one by one and parse their frontmatter.
- **How.** `notePaths(filter: { frontmatter: [{ key: "fleet_id", equals: "codellm" }] })`, with an API key in `X-Api-Key` as before. The server saves the frontmatter of each note version it loads. On startup it fills in the current latest and live versions. Older versions are not processed, so a key that exists only in old versions is not indexed.

### A note lookup that finds nothing is `nil` in a layout

- **What.** In a custom layout, `nvs.ByPath`, `nvs.ByPermalink`, `nvs.ByWikilink`, a query's `.First()` and `.Last()`, `note.LangAlternative(...)`, `PartialRenderer().Section(...)`, a section's `Section(...)` and `FirstList()` return `nil` when nothing matches. Both `{{ if x }}` and `{{ if x == nil }}` now test the result.
- **Why.** A miss returned an empty pointer of a concrete type. `{{ if x }}` treated it as empty, but `x == nil` was false, so a template that checked `x == nil` went on to use a note that did not exist.
- **How to use.** Nothing to do: templates written with `{{ if x }}` keep working. To look up and test in one tag, write `{{ if about := nvs.ByPermalink("/about"); about }}`. See [[en/user/templates#Assignment in if (Go-style)|Templates]].

### Layouts with try/catch, exec, `range _` or a hyphenated folder now load

- **What.** A layout that uses `{{ try }}…{{ catch err }}…{{ end }}`, `{{ return }}` in a file called with `exec("path", data)`, or `{{ range _, x := list }}` now loads and renders. A page that starts with `{{ extends "..." }}` gets its components auto-imported. `@lid` now turns every character a Jet block name cannot hold (`-`, `.`, a space) into `_`, and puts `_` in front of a leading digit: `my-theme/card.html` gives `my_theme_card`, `2col/card.html` gives `_2col_card`.
- **Why.** The loader's template walker stopped on these constructs, and the layout failed to load with a parse error. In a page with `extends`, auto-import put the components above the `extends` line, where Jet does not accept them. A component in a folder such as `my-theme/` got an `@lid` with a hyphen, which is not a valid block name.
- **How to use.** Nothing to do. `yield_blocks` prefixes follow the `@lid` form: `yield_blocks("_style_my_theme_")`. See [[en/user/templates#Data from another layout: exec and return|Templates]] and [[en/user/yield_blocks|yield_blocks]].

### Default template: code blocks, wide tables and the side menu

- **What.** A code block marked `jet` is highlighted as a Jet template. The copy button no longer scrolls away with a long line; its label follows the page `lang` (Russian for `ru`, English otherwise), and "Copied" is announced to screen readers. A block in a language the highlighter does not know is shown as plain text. A wide table scrolls inside its own box instead of widening the page. From 1024px the left menu scrolls on its own, opens scrolled to the current page, and marks the current page with a tinted background.
- **Why.** The copy button sat inside the scrolling code and moved with it. A wide table pushed the whole page sideways. In a long menu the current page could be out of view, and its mark was hard to see.
- **How.** Nothing to do.

### More of the interface in the reader's language

- **What.** The default template's 404 and 500 pages, the "Read in:" label of the language switcher, the menu buttons' labels, the logo's alt text and the "not supported yet" text for Canvas, Excalidraw and Bases files now follow the interface language. In Russian, the editor's "Edit page" button, file search, save list and version compare, the live-reload and follow-editor toggles and the theme switch are translated too.
- **Why.** These strings were hardcoded in English, or their Russian translations sat where the frontend build did not pick them up.
- **How.** Nothing to do. The interface language comes from the language cookie or the browser's `Accept-Language`.

### Readers no longer download the page editor

- **What.** The page editor is now a bundle of its own, `/assets/ui/editor/pane/-/web.js`. Only the editor frame loads it, when an admin opens the editor; the bundle every reader loads no longer contains it.
- **Why.** The editor was part of the public bundle, so every reader downloaded code that only an admin ever runs.
- **How.** Nothing to do with the release binaries or the Docker image. A custom layout that prints `{{ defaultTemplate.UserSpaceScripts() }}` passes the editor's URL along by itself. If you build the frontend yourself, also build `trip2g/editor/pane` (`npm start trip2g/editor/pane`), as the Dockerfile now does.

### Name the SSO button, and turn off sign-in by email

- **What.** The OIDC sign-in button can carry its own label instead of "Sign in with SSO": set `OIDC_DISPLAY_NAME` for the provider configured in the environment, or `displayName` in `createOIDCCredentials` for one created through the API. Sign-in by email code can be turned off: `DISABLE_EMAIL_SIGNIN=true` in the environment, or the `email_signin_enabled` switch in Admin → System → Config. When it is off, the sign-in screen hides the email field, its button and the captcha, and `requestEmailSignInCode` and `signInByEmail` answer with the `email_sign_in_disabled` error. New GraphQL: `Query.emailSignInEnabled`, `OAuthUrlPayload.label`, `AdminOIDCCredentials.displayName`.
- **Why.** "SSO" names a protocol, not the place the reader is about to go. On an instance where every account comes from an identity provider, the email form invited people to request a code for an account that does not exist.
- **How.** The environment variable wins: while `DISABLE_EMAIL_SIGNIN` is set, the admin switch cannot turn email sign-in back on. Make sure a provider works before you turn email off, or nobody can sign in, you included. An empty label keeps the default wording. See [[en/user/oidc#Rename the sign-in button|OIDC]] and [[en/user/oauth#Turning off sign-in by email|OAuth]].

### Sign-in returns to the page it started on, and redirects stay on this site

- **What.** After signing in with Google, GitHub or OIDC, the reader lands back on the page where they pressed the button, not on the home page. Sign-in redirects now send a relative `Location`, so behind a proxy that terminates TLS they no longer switch the browser from https to http. Every redirect target built from a request (the OAuth return address, and the clean URL after a `?hat=` sign-in link from a payment) is parsed and rebuilt as a path on this site; anything naming another host, a protocol-relative `//host`, `/\host` or a URL with control characters becomes `/`.
- **Why.** The button sent the full page URL and the server threw away every absolute URL, so each sign-in ended on `/`. Behind a TLS proxy the redirect pointed at `http://`, the browser did not send the `Secure` cookie it had just received, and a sign-in that worked looked like one that failed. The `?hat=` redirect copied the request line as is, so a request with an absolute-form URL could send the browser to another site.
- **How.** Nothing to do.

### Backlinks in GraphQL respect read access

- **What.** `NoteView.inLinks` lists only the linking notes the caller can read.
- **Why.** It returned every note that links to the current one, with its `content` and `html`, including notes in subgraphs the caller has no access to.
- **How.** Nothing to do.

### Vector search sees new notes without a restart

- **What.** Embeddings computed after a sync now reach vector search and similar notes while the server runs. The server re-reads the embeddings and chunks from the database after the embedding jobs finish, at most once per `EMBEDDING_RELOAD_INTERVAL` (default `5s`) while jobs keep finishing, and once more after the last one. No note is re-rendered.
- **Why.** Embeddings were loaded into memory with the notes, before the background jobs had written them, so a note synced after boot was invisible to vector search until the next restart.
- **How.** Nothing to do. During a long push search improves as jobs finish instead of after it ends.

### MCP: expand reads a leaf, lists only the ends of a long note, and federation errors name the bases

- **What.** `expand` on a section without subsections returns the section itself, the same text `note_html` gives for that `toc_path`, plus `section_html` in the structured payload. New `first` and `last` arguments on `expand` and `federated_expand` list only the oldest N and newest N subsections; the summary says the listing is partial ("newest 30 of 365 subsection(s)", "… 330 subsection(s) not listed …" between the two ends) and the payload carries `total_children` and `omitted`. A `federation_not_configured` answer now lists the bases the agent can address (`connected_kb_ids`), and when a peer reports the miss it also lists the bases connected directly to this hub. Search results and notes that point to another base show their `kb_id` in the text.
- **Why.** An agent reading a leaf had to call `note_html` with the same arguments. A note that gains a dated section a day returns a listing of tens of thousands of characters to someone who wants the latest entry. A not-configured error named no valid `kb_id`, so the agent could not correct itself.
- **How.** `{ "name": "expand", "arguments": { "path": "log.md", "first": 5, "last": 30 } }`. Ends that meet or overlap return the whole listing. See [[en/user/expand#A long listing: the ends only|expand]].

### A knowledge base in one container

- **What.** `Dockerfile.docs` builds an image with a vault baked in. `docker run` starts trip2g, pushes the vault into it once with the sync client from the instance's own onboarding archive, and keeps serving. No volume, no sign-in, no configuration: the database and the secrets are created inside the container at start. The internal listener is bound to `127.0.0.1`. By default the image holds this documentation.
- **Why.** Shipping a read-only knowledge base (for a team, or for an MCP client) took a running instance plus a separate sync step.
- **How.** `docker build -f Dockerfile.docs -t trip2g-docs .` then `docker run --rm -p 8080:8080 trip2g-docs`. Replace the `COPY` lines with your own folder, and pass `-e PUBLIC_URL=...` for the public address. See [[en/user/docs-image|A knowledge base in one container]].

### Fleet: pause a role, and safer code runs

- **What.**
  - A role note with `enabled: false` leaves the registry on the next poll, and its webhooks are removed, so a cron role stops firing; the note stays. `--dry-run` shows `STATUS: DISABLED (enabled: false)`. A value other than true/false (`yes`/`no`, `on`/`off`, `1`/`0` are accepted) is a parse error.
  - A retried call to codellm no longer runs the code twice: fleet sends one `Idempotency-Key` per call and repeats it on every retry, and codellm answers a repeat from its record for 10 minutes. Replays are counted in `codellm_exec_replays_total`.
  - Final stdout over the limit now fails the run with "stdout limit exceeded" (HTTP 422 from `/v1/chat/completions`) instead of passing on a cut-off prefix. The default limit is 10 MiB (was 1 MiB); set it with `CODELLM_MAX_STDOUT_BYTES` or `--max-stdout-bytes`.
  - The writes returned by the `exec` tool are checked as one batch before any is applied: an out-of-scope path is dropped and named in the tool result, and a patch whose `find` is missing or not unique rejects the whole batch.
  - codellm refuses to unseal a role whose `unseal_env_key` is covered by `CODELLM_EXPOSE_ENV_PREFIX`, as it already did for exact names in `CODELLM_EXPOSE_ENV`.
  - The agent system prompt puts the fixed text first and the per-delivery instruction last, so a provider's prompt cache can reuse it; `fleet_llm_tokens_total` gets `kind="cached"` (a part of `prompt`, not extra spend).
- **Why.** A cron expression that never comes due is not a reliable pause. A network retry could run code with side effects a second time. A truncated stdout looked like a result. A batch could end with half its writes applied and a success line in front of the model. A prefix allowlist could hand the seal key to the code it protects.
- **How.** Add `enabled: false` to a role's frontmatter to pause it, remove the line to resume. See [[en/user/fleet|Fleet]] and [[en/user/codellm-secrets|codellm secrets]].

### Admins can make a sign-in link for a user

- **What.** The user page in the admin has a **Sign-in link** button, and the admin GraphQL API has `createHatLink(input: { email, redirectUrl, expiresInMinutes })`. It returns a one-click `/_system/hat?token=...` URL that signs that person in. The link lives 5 minutes by default and at most 60, and `redirectUrl` must be a path on the same site. It signs in an existing user only: it never creates an account and never grants admin. A dead link now opens a short page that says what to do ("This link has expired", "This link doesn't work", "No account for this link") instead of an error.
- **Why.** Getting someone in without a mail round trip meant the server CLI and its signing secret. When a link failed, the visitor saw a raw error with nothing to act on.
- **How.** Open Admin → Users → the user → **Sign-in link**, copy the link and send it. The request is written to the audit log; the link itself is not.

### Personal access tokens for other users

- **What.** An admin can issue a personal token (`t2g_...`) for any user with `adminCreateUserToken(input: { userId, name, expiresInDays })` (expiry 1 to 365 days, optional) and revoke any token by id with `adminRevokeUserToken`. Admin → Users → **Personal Tokens** lists every token on the instance; the **Personal tokens** link on a user page opens it filtered to that user, with the new-token form prefilled. `CreateUserTokenPayload` has a new `instructions` field: a ready text with the MCP endpoint and the token, to hand to whoever will use it. In the account dialog, tokens got their own screens, and the account home shows the signed-in email and starred notes.
- **Why.** A token could only be created by the user for themselves, so giving an agent access meant signing in as that person first.
- **How.** Admin → Users → the user → **Personal tokens** → new token. Copy the instructions block once: it already carries the token. See [[en/user/mcp#Personal access tokens|MCP]].

### Closed pages answer 401 or 403 and leak no description

- **What.** A page shown behind the sign-in wall or the paywall now answers `401` to a visitor who is not signed in and `403` to one who is signed in but has no access, with `Cache-Control: no-store` on both. The page body is the same wall as before. A closed note also gets no automatic `<meta name="description">` from its first paragraph; only an explicit `description` in its frontmatter is used.
- **Why.** Walls were served with `200`, so crawlers, link previews and API clients took them for the real page. The description fallback was built from the note body and printed in the `<head>` of the wall itself, so the opening lines of a closed note were readable without access.
- **How.** Nothing to do. To give a closed page a description for search results, set `description` in its frontmatter.

### Site logo loads for anonymous visitors

- **What.** Images used in `_header.md` and `_footer.md` (in any folder) are served to visitors who are not signed in, whether or not those notes are `free`. Assets of other system notes follow the same rule as a normal note: anonymous when the note is readable without a session.
- **Why.** The site logo answered `401` to every anonymous visitor: assets were public only when an owning note could be listed publicly, and system notes such as `_header.md` never can. Admins are always signed in, so they did not see the problem.
- **How.** Nothing to do. Chrome attached under another name with the `header:` frontmatter field still needs `free: true` for its images to be public.

### Links keep the reader on the host they came to

- **What.** On a custom domain, the header, footer, sidebar, backlinks and template queries now render links the same way as the page body: a link to a note on the same host is relative, not an absolute URL. A note reached through its own `route` frontmatter is served at that path instead of being redirected to its permalink.
- **Why.** Shared chrome was rendered once for the main domain, so on a custom domain its links pointed back to the main domain or to an absolute URL that does not resolve on an internal or preview host. The route redirect depended on whether the file name had characters that get transliterated, and on a custom domain it led to a 404.
- **How.** Nothing to do. `rel=canonical`, `og:url` and JSON-LD stay absolute.

### MCP: standard transport, federated arguments and search fixes

- **What.** The MCP endpoint `/_system/mcp` now runs on the official Go MCP SDK (Streamable HTTP, stateless, JSON responses). Tool names, descriptions and schemas did not change. Along with it:
  - Federated calls pass every argument through to the peer. `federated_search` used to send only the query, so `limit`, `detail_limit` and similar were lost; `federated_note_html` now takes `toc_path`. A call to a single `kb_id` has the same timeout as a fan-out call.
  - An API key sees KB-notes (`mcp_federation_kb_url`) that are not `free`, as it sees every other note. Before, `federated_*` calls on such a KB answered "not configured".
  - Search chunks no longer start with a broken character in non-Latin text.
  - `expand` leaves out the title H1 and shows a short preview after headings that are too short to tell apart. `toc_path` from `search` is trimmed the same way.
  - Notes registered as tools (`mcp_method`) no longer show up in search results.
  - Each search match has `section_url`, a link to the matched heading on the site.
- **Why.** The hand-written protocol layer drifted from the spec, and federated agents got different results from the same call made locally.
- **Migration.** A request must send `Accept: application/json, text/event-stream`; without it the endpoint answers `400`. Real MCP clients send it. Update curl scripts and custom integrations (the stdio adapter in the docs is updated). An instance on an older version calling this one through federation does not send the header, so update both sides. Other visible changes: `notifications/initialized` answers `202` with an empty body, `tools/list` is sorted by name. See [[en/user/mcp|MCP]].

### Reranking is a per-search choice

- **What.** With a reranker configured, each search can ask for it: a `rerank` argument on the MCP `search` and `federated_search` tools, and `rerank` on the GraphQL `SearchInput`. A new `vector_search.reranker.default` in `FEATURES` decides what happens when a request says nothing; it is `false`. The MCP argument is shown only when a reranker is configured.
- **Why.** The cross-encoder costs about a second per candidate on CPU, so a search with `top_n` 20 took about 20 seconds. That is acceptable for an agent doing research, not for a person at a search box.
- **Migration.** `reranker.enabled: true` no longer reranks every search. To keep the old behaviour, add `"default": true` to the `reranker` object. This also applies to `memcli up --reranker`.

### Full-text search index can live on disk

- **What.** A new `SEARCH_INDEX_PATH` setting (`--search-index-path`) keeps the full-text index in that directory instead of in memory. A restart reopens the existing index instead of rebuilding it, and notes deleted while the server was down are removed from it on the next load. Empty keeps the in-memory index, as before.
- **Why.** The in-memory index costs about 35 times the size of the text. On a vault with 10 MB of markdown it held 350 MB of heap, and large vaults ran out of memory on restart. On disk the same vault takes 4 MB of heap and about 45 MB of disk, and a restart opens the index in milliseconds.
- **How.** Set `SEARCH_INDEX_PATH` to a directory on a persistent volume. Do not point two instances at the same directory.

### Federation: one handover key, rotated on install

- **What.** Adding an inbound secret now gives one handover key that already carries the address and key ID; the peer pastes it into Add Outbound. Before the row is stored, the receiving hub asks the issuing peer to adopt a fresh random key, so the value that went through a chat stops working. An outbound key's page has a **Rotate key** button (also `rotateFederationSecret`), shows when the key last rotated, and lists what the peer granted, read from the peer's new `/_system/mcp/federation` endpoint. Subgraphs have a one-line description that the peer sees next to the name. The admin list shows each key's direction.
- **Why.** A shared key handed over in a message stays in that message history, and in an agent's transcript when an agent relayed it. Operators also had no way to see what a pairing actually granted.
- **How.** Add Inbound → copy the handover key → send it → the other side pastes it into Add Outbound. The **Replace the key before storing it** switch is on by default; turn it off for a peer that cannot rotate (a public base, an adapter, an older instance), otherwise the peer refuses and nothing is stored. Rotation is refused against an `http://` peer unless private federation addresses are allowed. See [[en/user/federation#Key rotation|Federation]].

### Webhook agents: run logs, longer history, private addresses

- **What.** A webhook agent can return `logs`, a list of `{ts, level, msg, data}` entries, and the admin shows them on the chain step under **Log**. Up to 500 entries or 64 KB are kept per delivery; the rest is replaced by one entry saying how much was dropped. Delivery records and their request and response bodies are now kept 90 days, set with `--webhook-deliveries-retention` and `--webhook-delivery-logs-retention`. A new `--webhook-allow-private` (`WEBHOOK_ALLOW_PRIVATE`) lets webhooks reach private and internal addresses without turning on dev mode. Notes that an agent returns in its response body now trigger change webhooks, SSE and frontmatter indexing like any other write, and count one level deeper in the chain.
- **Why.** An operator could see that a run failed but not what the agent did. Records went away after 30 days and the bodies after one day, before anyone looked. Agents running next to trip2g on a private network were blocked by the SSRF guard. Notes written through the response body did not trigger the next role in a chain.
- **How.** Return `logs` next to `changes` and `costs`. Bodies are now stored 90 times longer, so plan disk space or set the retention lower.

### Fleet signs in with a personal token, and agents cannot write role notes

- **What.** Fleet authenticates with an admin's personal token (`--trip2g-admin-personal-token`, `TRIP2G_FLEET_TRIP2G_ADMIN_PERSONAL_TOKEN`) instead of the server's JWT secret. Set `OWNER_PERSONAL_TOKEN_VALUE` on the server to a `t2g_` value and it seeds that token for the owner (`OWNER_EMAIL`) at boot; `memcli up` generates one. Fleet refuses a write or patch from an agent that would create or edit a role note (any note with `fleet_id` in its frontmatter), and a patch applies only if the note is unchanged since that check. `--allow-role-authoring` turns the guard off. Fleet and codellm also serve Prometheus `/metrics`, pprof and health probes on a loopback listener: `--metrics-addr` (fleet `127.0.0.1:18090`, codellm `127.0.0.1:18087`, `CODELLM_METRICS_ADDR`); empty turns it off.
- **Why.** With the JWT secret, fleet could mint a session for anyone and could not be cut off without changing the secret. A role declares its own `write_patterns`, so an agent tricked by note content into writing a role note could give itself more access.
- **Migration.** `--jwt-secret`, `--admin-email` and `--admin-api-key` are gone from fleet. Set `OWNER_PERSONAL_TOKEN_VALUE` on the server and pass the same value to fleet. Revoking that token in the admin stops fleet within about half a minute. See [[en/user/fleet|Fleet]].

### Code roles: a toolbox, fleetkit and sealed secrets

- **What.** The codellm image is now Debian-based with Python 3, Node 24 and common tools (`jq`, `sqlite3`, `git`, `gh`, `ripgrep`, `curl` and others) plus libraries such as `requests`, `httpx`, `pydantic`, `jsonschema`, `axios`, `zod` and `ajv`. A `fleetkit` helper for Python and Node builds the output a code role prints (`note`, `write`, `patch`, `emit`) and reads its input (`bag`, `frontmatter`, `secrets`, `note_frontmatter`). A role note can carry encrypted values: list them in `unseal`, and codellm opens them with its `SEAL_KEY` (or the key named in `unseal_env_key`) for that run only. Values are sealed with `codellm seal` or the form at `/_system/codellm/seal` (`CODELLM_SEAL_PATH`). A role can narrow the env vars its code sees with `env_passthrough` and `env_prefix`, within the operator's allowlist.
- **Why.** Code roles had bare Python and no libraries, assembled notes and JSON by hand, and every credential meant editing the codellm deployment and restarting it.
- **How.** See [[en/user/fleet#Code roles|Fleet]] and [[en/user/codellm-secrets|Secrets for code roles]].

### Re-embed all notes from the admin, and copy a heading link

- **What.** The admin **Note Views** page has a **Re-embed all notes** button. It calls `regenerateNoteEmbeddings(input: { force: true })`, which queues every note even when its stored hash says it is up to date; chunks whose own hash did not change are not re-sent to the model. In the default template a `#` link appears next to a heading on hover.
- **Why.** After a change to chunking, every note still looked up to date, so the new chunks never reached the vector index. Linking to a section meant reading its id from the table of contents.
- **How.** Admin → Note Views → **Re-embed all notes**. Needs vector search to be on.

### trip2g-sync CLI: a graphql command, and a failed fetch stops the sync

- **What.** `trip2g-sync graphql '<query>' ['<variables>']` runs a GraphQL query against the instance the vault is set up for, using the key in `.obsidian/plugins/trip2g/data.json`; `graphql --introspect '<pattern>'` shows matching schema types. Queries go through MCP, where an API key carries its admin rights. If the CLI cannot fetch the server's note list, it now stops with an error and exit code 1. Bare asset links such as `![[logo.png]]` resolve to the file anywhere in the vault (the shallowest match wins), in the CLI and in browser sync. The downloaded onboarding vault now includes `AGENTS.md` and `CLAUDE.md` for AI agents working in it. The `trip2g-sync` CLI (plugin 0.12.0) prints what will happen to each non-empty line of the sync plan, for example that "Remote only" files are downloaded as new local files and "Local deleted" notes are hidden on the server. Its help now says that deleting a file locally hides the note on the server with no flag needed, and that `--two-way` downloads notes that exist only on the server.
- **Why.** A failed fetch was read as an empty server: the run reported every local note as deleted on the server and exited 0. An asset link without a folder found nothing unless the file sat next to the note.
- **How.** Run from the vault folder, for example `node .obsidian/plugins/trip2g/trip2g-sync.mjs graphql '{ viewer { id } }'`. These CLI changes ship with plugin and CLI 0.12.0.

### Files in notes are served by the site, with access checks

- **What.** Images and other files from notes and layouts are served at `/_system/assets/<sha256>/<file name>`. They are no longer presigned links to the storage server (`MINIO_PUBLIC_URL`) or, with local storage, `/_assets/...`. A file used by a layout or by a note anyone can read is public and cached for a year as immutable. Any other file needs a signed-in reader who can read at least one note that uses it, a valid API key in `X-API-Key`, or an admin. Such files are cached privately for 5 minutes, so the access check runs again after that. A file that no current note or layout uses is for admins only. Range requests and `ETag` work. URLs that leave the site (Telegram posts, `og:image`, JSON-LD, RSS, the asset URLs `pushNotes` returns) are made absolute with the site's public URL.
- **Why.** A presigned link expired after a few days and worked for anyone who had it, even for a file from a paid or closed note. It also needed the storage server to be reachable from readers' browsers.
- **How.** Nothing to do for content: pages use the new URLs as soon as the server runs the new version. `MINIO_PUBLIC_URL` and `MINIO_URL_EXPIRES_IN` are now ignored, and the storage server no longer needs a public address. Old links copied from pages before the upgrade stop working.

### Forms: submit checks note access, a misconfigured captcha blocks, `forms:` works

- **What.** `submitForm` first checks that the visitor can read the note the form belongs to. If not, it answers `form_not_found`, the same as for a missing form. Every text and email field is capped at 8192 characters, and `min_length` / `max_length` now count characters, not bytes. If `turnstile-site-key` is set but `turnstile-secret-key` is empty, the Turnstile check now fails instead of letting every request through; it also guards the email sign-in captcha. Several forms on one note through the `forms:` key now work.
- **Why.** The note version id in a submit is easy to guess, so a guest could post to a form on a paid, sign-in-only or subgraph note. A half-configured Turnstile looked like it was protecting forms but checked nothing. A field had no size limit other than the request body limit. `forms:` was never found in real frontmatter, because the YAML parser returns its keys in a type the code did not expect.
- **How.** Nothing to do. If you set only the Turnstile site key, add the secret key, or remove both to turn the captcha off. Non-Latin text that was close to its `max_length` now has more room, since the limit no longer counts bytes. See [[en/user/forms|Forms]].

### Without SMTP, sign-in codes go to the server log

- **What.** When `SMTP_HOST` is empty, requesting an email sign-in code works again. The code is written to the server log at warning level together with the email. `LOG_SIGN_IN_CODES=true` still logs codes when SMTP is configured.
- **Why.** The previous release made a code request without SMTP return an error unless `LOG_SIGN_IN_CODES` was set, which locked dev setups and fresh installs out of email sign-in.
- **How.** On a public server, configure SMTP: anyone who can read the server log can sign in as any user who asks for a code. See [[en/user/smtp|SMTP]].

### A custom layout that fails returns an error page

- **What.** A custom layout is rendered into a buffer first. If it fails, nothing from it reaches the reader. The response is status 500 with the default template's error page. An admin sees the layout name and the Jet error with its line on the same page. Everyone else sees a generic error.
- **Why.** Jet writes everything up to the failing expression before it stops. Visitors could get half a page, and admins got the raw error text glued to the end of that half page.
- **How.** Nothing to do. Open the page as admin to see what broke.

### Layout warnings reach the sync tool, which stops on critical ones

- **What.** `pushNotes` now returns the warnings of each layout file. Before, they were replaced with an empty list. The CLI sync tool (`trip2g-sync.mjs`) prints warnings grouped by file, with `CRITICAL` in red, and exits with code 1 if any warning is `CRITICAL` (for example, a layout that does not parse). A new `WARNING` reports a layout that writes its own expanded `@lid` name as plain text (`mesh-bar` in `mesh/bar.html`): a renamed file keeps the old name there, while the placeholder would follow the rename. Matches inside HTML comments and URLs are ignored.
- **Why.** A broken layout was pushed without a word, and the site found out at render time.
- **How.** Update the CLI and run it as before. In CI, a critical layout warning now fails the job. See [[en/user/cli|CLI sync tool]].

### `coalesce()` in layouts

- **What.** A new global function for Jet layouts: `coalesce(a, b, ...)` returns the first argument that is set and not empty, otherwise the last argument. A missing map key, `nil`, `""`, an empty list and an empty map count as empty; `0` and `false` count as values.
- **Why.** Jet's `||` returns a boolean, so `videos[lang] || videos["en"]` could not pick a fallback value.
- **How.** `{{ coalesce(videos[note.Lang()], videos["en"]) }}`. See [[en/user/jet-functions#Global functions|Jet functions]].

### Mermaid diagrams: a second renderer, zoom, fullscreen and export

- **What.** Flowchart, state, sequence, class, ER and xychart diagrams are drawn by beautiful-mermaid. Other types (gantt, pie, gitGraph, mindmap and so on), and any diagram it fails to parse, use mermaid.js as before. Each diagram gets buttons for zoom in, zoom out, reset, fullscreen and PNG / SVG export, and can be dragged. A diagram is at most 80% of the screen height and is scaled to fit its box.
- **Why.** Dragging a diagram never worked. Large diagrams shrank until they could not be read, and wide, short ones became a thin strip.
- **How.** Nothing to do. See [[en/user/mermaid|Mermaid]].

### Magazine: choose how many cards are large, medium and small

- **What.** Three frontmatter keys on the page that hosts the magazine: `magazine_featured` (large cards, default `1`), `magazine_grid` (medium cards after them, default `4`) and `magazine_grid_columns` (columns in the medium grid, default `2`). The rest of the notes go to the list. `0` skips a tier, and counts larger than the number of notes are capped. Card excerpts also skip tables now.
- **Why.** The layout was fixed at one large card, four medium ones and a list. A table at the start of a note broke its card.
- **How.** For example, `magazine_featured: 0` and `magazine_grid: 8` give no large card and eight cards in the grid. The defaults keep the old layout. See [[en/user/default-template#Magazine layout|Default template]].

### Widgets in the `content:` list

- **What.** The default template's `content:` list now accepts `toc`, `backlinks` (or `inlinks`), `outlinks` and `similar`, the same keywords as the sidebars. Each one renders its block in the main column.
- **Why.** The template could already draw these blocks in the main column, but `content:` treated the words as file names and showed nothing.
- **How.** For example, `content: [self, backlinks, similar]` puts backlinks and similar notes under the article.

### Callouts in Telegram posts

- **What.** Obsidian callouts (`> [!note] Title`) in a Telegram post become a quote with the title in bold on the first line. A callout without a title uses its type, capitalized. A collapsed callout (`[!type]-`) becomes an expandable quote that shows only the title until opened.
- **Why.** The Telegram converter did not know callouts: it dropped the whole block and logged `unexpected markdown node`.
- **How.** Nothing to do: write callouts as in Obsidian.

### Webhook tokens stay inside their write scope

- **What.** The short-lived token a webhook passes to an agent (**Pass API key**) is now held to its `write_patterns` everywhere it can write. `uploadNoteAsset` checks that the note belongs to a path the token may write. `pushNotes`, `hideNotes` and `commitNotes` refuse such a token with `ErrorPayload`, because they have no per-path scope. In `updateNotes`, a token with no `write_patterns` can no longer write anywhere.
- **Why.** These calls let a token write outside its `write_patterns`: attach files to any note, push or hide any note, or, in some cases, write anywhere through `updateNotes` when `write_patterns` was empty.
- **How.** Nothing to do for agents that use `updateNotes` within their scope. An agent that called `pushNotes`, `hideNotes` or `commitNotes` with its webhook token must switch to `updateNotes` (the `hide` change hides a note). See [[en/user/webhooks#API token for agents|Webhooks]].

### Note graph: groups, saved positions and live readers

- **What.** The admin note graph (Notes & Content → Note Graph) has **Auto layout**, **Save positions** and **Public outside** in its header, and a **Group by:** field. Notes with the same value of that frontmatter key (default `subgraph`) are pulled into one cluster. A dragged node keeps its position. While the graph is open, signed-in readers moving between notes show up as animated moves, also available as the admin-only GraphQL subscription `readerMoves` (admin session or instance API key). A reader appears under an anonymous key that changes every hour, never as a user id. The server does no extra work for this while no one is subscribed.
- **Why.** Notes were spread at random, the layout was lost on every reload, and there was no way to see how people move through the site.
- **How.** Open the graph, type a key into **Group by:** and press **Auto layout**. To group whole folders, add a frontmatter patch. See [[en/user/admin_graph|Note graph]].

### Fleet: roles pick a fleet by `fleet_id`, code runs in codellm

- **What.** A role chooses the fleet that runs it with the `fleet_id` frontmatter key. Fleet's `--fleet-id` (`TRIP2G_FLEET_FLEET_ID`) is now required and has no default (it used to be a marker that defaulted to `fleet1`). A fleet runs only roles with its own `fleet_id`. A role without `fleet_id` runs nowhere, and fleet reports it as an error during discovery. The `executor` key is gone. Code roles run in codellm, a separate service with an OpenAI-compatible API (`cmd/codellm`, image `Dockerfile.codellm`). A fleet whose `--llm-base-url` points at codellm runs role bodies as code. Fleet no longer runs code itself: its `--allowed-programs` flag is gone, and the interpreter list, sandbox and network access are codellm settings (`CODELLM_ALLOWED_PROGRAMS`, `CODELLM_SANDBOX`, `CODELLM_SANDBOX_NETWORK`). codellm can require a key (`CODELLM_API_KEY`), which fleet sends with `--llm-api-key` or `--exec-api-key`. Fleet also serves a read-only GraphQL API with the roles and how they trigger each other (`--graphql-addr`, default `127.0.0.1:9093`), open only to admins of the hub.
- **Why.** Code execution and its sandbox lived inside the agent process. Moving them to their own service lets a code role and an LLM role run on different hosts, and a role moves between them by changing one line.
- **How.** Before upgrading fleet: add `fleet_id` to every role, start one fleet per `fleet_id` with a matching `--fleet-id`, and for code roles run codellm and point that fleet's `--llm-base-url` at it. Move `--allowed-programs` to codellm's `CODELLM_ALLOWED_PROGRAMS`. See [[en/user/fleet#Code roles|Fleet]].

### Smaller fixes

- **A revoked personal token stops working at once.** `revokeUserToken` and `adminRevokeUserToken` now drop the token from the server's cache. Before, a revoked token kept working for up to 30 seconds.
- **A custom domain with nothing to index gets an empty sitemap.** Its `/sitemap.xml` is an empty `<urlset>`. Before, it served the main domain's sitemap, with the pages of another site. See [[en/user/multidomains#SEO on a custom domain|Multi-domains]].
- **Browsers pick up new built-in scripts and styles after an upgrade.** Files under `/assets/` now carry an `ETag` and a `Cache-Control` header: a URL with the file's current `?h=` hash is cached for a year as immutable, any other URL for an hour and then revalidated. Before, the server answered every `If-Modified-Since` with "304 Not Modified", because embedded files have no modification time, so a browser could keep an old script after an upgrade.
- **Faster asset sync.** `uploadNoteAsset` checks whether the file is already stored before it parses the note, and a layout version loads only the `_layouts/` files instead of every note. Re-syncing a vault full of images no longer parses each note once per image.
- **Commit in the startup log.** The server logs `trip2g starting` with the git `commit` it was built from.
- **Email check.** Creating or editing a user in the admin, and `createHatLink`, check only that an address is well formed. They no longer look up the domain's MX record, so a domain without mail no longer fails and the form no longer waits on DNS.
- **Long inline code.** In the default template a long URL or path in inline code wraps instead of making the page wider.
- **Admin dates.** The audit log and the wait-list screens show their dates correctly; they were formatted twice.
- **$mol in the header.** A $mol component in the header no longer takes the whole header row while its styles load.
- **Admin Sign out.** The **Sign out** button in the admin panel signs you out. Before, it did nothing.
- **Callout titles.** In the light theme, callout titles use the callout's own color and are readable. A callout type that starts with a non-Latin letter (`[!заметка]`) gets a correct capital letter in its default title, on the site and in Telegram.
- **Telegram rate limits.** Sending and editing Telegram posts (by bot and by account) makes at most three attempts when Telegram answers with a rate-limit error, and the wait between them stops when the server shuts down. Before, the retries had no limit.
- **Storage at startup.** The server waits up to 45 seconds for the S3 storage to come up instead of exiting on the first failed bucket check. Wrong credentials or a bad bucket name still fail at once. `MINIO_SECRET_ACCESS_KEY` is accepted as the secret key name.
- **Missing files in storage.** When a client uploads a file whose record already exists but whose data is gone from storage (for example, after the bucket was wiped), the server stores the file again. Before, it only linked the record and the file stayed missing.
- **Starter vault name.** `/_system/onboarding-vault?name=<name>` sets both the zip file name and the folder inside it. Letters, digits, dot, dash and underscore, up to 64 characters. An invalid name returns 400.

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
