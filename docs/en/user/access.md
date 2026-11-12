---
title: "Who can read a note"
description: "What each kind of reader sees for each frontmatter setting, and where a closed note still shows its title."
free: true
lang: en
lang_redirect: "[[ru/user/access]]"
---

This page says who can read a note and what everyone else sees in its place. After it you can look at a note's frontmatter and tell what a guest, a signed-in reader, a subscriber and an agent get: on the page, in search, in feeds and over MCP. Read it before you put paid or internal notes next to public ones. How to create subgraphs and grant access is on [[en/user/subgraphs|Subgraphs]]; this page is about the result.

## The rule

A note is closed unless something opens it. Two frontmatter keys decide:

```yaml
free: true       # everyone, guests included
subgraph: paid   # readers who hold access to "paid"
```

| Note | Guest | Signed in, no access | Access to another subgraph | Access to this subgraph | Admin |
|---|---|---|---|---|---|
| `free: true` | reads | reads | reads | reads | reads |
| no `free`, no `subgraph` | paywall | paywall | reads | n/a | reads |
| `subgraph: paid` | paywall | paywall | paywall | reads | reads |
| `subgraph: members`, and `members` has **Require sign-in** on | sign-in wall | reads | reads | reads | reads |

Four things follow from the table.

- **`free: true` wins over everything.** A free note in a paid subgraph is public. The subgraph still groups it; it no longer gates it.
- **A note with no subgraph is not private.** Any reader who holds access to at least one subgraph reads it. On a site with several audiences, tag every note. [[en/user/subgraphs#Make the whole vault private by default|Subgraphs]] has a patch that does it for the whole vault.
- **Several subgraphs mean "any of them".** `subgraphs: [beta, paid]` opens the note to a reader with access to either.
- **Access is checked on every request.** A grant that expires or is revoked stops working at the reader's next request.

"Access to a subgraph" is an active grant: bought through an offer, received through a Patreon or Boosty tier or a Telegram group, or given by hand in the admin. See [[en/user/monetization|Monetization]] and [[en/user/telegram-access|Telegram group access]].

Files ending in `.html` (layouts) are never served as a note to anyone but an admin.

## What a reader without access sees

The page opens, with a wall in place of the body.

| On the wall | Shown |
|---|---|
| Title | yes |
| `description` from frontmatter | yes |
| Sign-in form, or the offers that unlock the note | yes |
| Body, including the first paragraph | no |
| A description derived from the body | no |

The response status is `401` for a guest and `403` for a signed-in reader without access. Both walls carry `noindex` and are not cached.

The page header still names the note's preview image in `og:image`. The image file is checked like any other asset (below), so a link preview of a closed note shows the title and no picture unless the same image is also used in a free note.

**The title and the `description` of a closed note are public.** Do not put into them anything only subscribers should read.

## Where a closed note still appears

| Place | What a reader without access gets |
|---|---|
| The note's own page | the wall: title and `description` |
| Magazine cards | title, date and link; `description` if the note has one; no text from the body |
| Site search | title and link with a fixed placeholder instead of the snippet |
| A wikilink in a note they can read | the link, with the closed note's title |
| An embed in a note they can read | **the whole body** (see below) |
| Similar notes | nothing |
| RSS and the sitemap | nothing: both list free notes only |
| [[en/user/note-queries|Note queries]] | nothing: a query sees only notes its caller can read |
| MCP `search` | nothing |
| MCP `note_html`, `expand` | `Note not found`, the same answer as for a note that does not exist |

Start the server with `--search-hide-unreadable` to drop closed notes from site search instead of showing their titles.

## An embedded note is shown in full

`![[Other note]]` copies the other note's rendered body into the host note. The same goes for `content: [[Other note]]` in the default template. The host note's access decides who sees the result. The embedded note's own `free` and `subgraph` are not consulted.

```markdown
---
free: true
---

Intro everyone reads.

![[Paid lesson]]
```

Here every visitor reads the whole of `Paid lesson`, although its own page shows a paywall. The copy travels with the host note: into the page, the search snippet, RSS and MCP answers.

This is how the code behaves today. Embed a closed note only into a note with the same or narrower access. More on embeds: [[en/user/embedded-notes|Embedded notes]].

## Notes that start with `_`

A file or folder whose name starts with `_` is a system note. It is left out of magazine cards, site search, similar notes, RSS and MCP `search`, and its own URL answers `404`. It can still be embedded, and an agent can open it by path over MCP if the rule above lets it.

The underscore hides a note from listings. It is not an access setting: `free` and `subgraph` still decide who may read it.

## Hidden notes

`hide` in [[en/user/update_notes|updateNotes]] takes a note off the site without changing its content. A hidden note is not served to any reader: no page, no search hit, no feed entry.

## Assets

Images and attachments follow their notes.

- An asset used by at least one free note, by a layout, or by the site header and footer is served to everyone.
- Any other asset needs a signed-in reader who can read at least one note that uses it.
- A request with a valid `X-API-Key` header gets any asset.

## Drafts and releases

[[en/user/releases|Releases]] decide which version of a note readers get. The setting is **Show Draft Versions**, on by default.

| Show Draft Versions | Pages, site search, sitemap | MCP tools, note queries, similar notes |
|---|---|---|
| on | latest synced version | latest synced version |
| off | version in the live release | latest synced version |

With the setting off, the second column is a known gap: those readers do not wait for a release. A note you synced but have not released yet is already visible to them under the access in its newest frontmatter. If you add `free: true` to a draft, an anonymous agent can read that draft over MCP before the release goes live.

An admin always searches the latest versions and can switch a page between the two with `?version=latest` and `?version=live`.

## Keys and tokens

| Credential | Reads |
|---|---|
| No credential (anonymous MCP client) | free notes |
| Session cookie or a personal token (`t2g_…`) | what that user reads |
| API key in `X-API-Key` | every note and every asset |
| Scoped webhook token | notes matching the delivery's `read_patterns`, nothing else |

An API key is an owner's credential. Give a reader's agent a personal token instead; see [[en/user/graphql|GraphQL API]] for how to create one. Scoped tokens are described in [[en/user/webhooks|Webhooks]] and [[en/user/fleet|Fleet]].

## Custom layouts

A [[en/user/templates|layout]] you write can read any note in the vault and print any part of it. The access rule covers the note a visitor opens, not what the layout pulls in beside it. In a feed-like layout, add `.Public()` to the note list to keep only free notes:

```jet
{{ range i, n := nvs.ByGlob("blog/**").Public().All() }}
  <a href="{{ n.Permalink() }}">{{ n.Title() }}</a>
{{ end }}
```

`noteQuery(...)` in a layout is different: it runs with the visitor's access.

## Another base searching yours

A federated peer is one more kind of reader. Its key is scoped to a list of subgraphs, and the table above applies to it as to a reader who holds access to exactly those subgraphs. Who may search whom and what they see are set separately: [[en/user/context-separation|Context separation]] shows the pattern, [[en/user/federation|MCP Federation]] the setup.

## Related

- [[en/user/subgraphs|Subgraphs]]: tagging notes, granting and revoking access
- [[en/user/monetization|Monetization]]: offers, Patreon, Boosty, crypto
- [[en/user/embedded-notes|Embedded notes]]
- [[en/thoughts/three-zones|Three zones: private, public, paid]]
