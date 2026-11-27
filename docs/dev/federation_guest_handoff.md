# Federation guest handoff: a person opens a neighbouring base

A reader signed in on a hub presses a button on a KB-note and arrives in the neighbour's base in their browser as a read-only guest, scoped to what the federation key was granted. This page describes what is built. The plan it was built from is `docs/dev/2026-11-18_federation_guest_handoff_plan.md`, with the decisions and the options that lost.

## The flow

1. **Hub.** `POST /_system/federation/handoff` (`internal/case/federationhandoff`) takes `note` (the KB-note path) and an optional `path`. It refuses a post whose Origin is not the hub's own (a sibling site can send the session cookie), an anonymous reader, a guest, a reader who cannot read the KB-note, and a KB-note whose address has no outbound key. The site address comes from the key, never from the note: the key is selected by the note's own address, and a note naming another address gets no token. The hub signs, with the key's secret, a single-purpose JWT (`use: guest`) carrying the target path, hop `1`, the reader id (a hash of the reader's email, or of the hub and the user id) and the hub's origin. It answers with a page that posts the form to the neighbour.
2. **Neighbour, POST.** `POST /_system/federation/guest` (`internal/case/federationguest`) verifies the signature like a tool call's bearer, then insists on the handoff use, a lifetime of one handoff, an origin that equals the browser's `Origin`, the signed path, hop `1`, a reader id that is a SHA-256 hex, a key that accepts people, and an id not spent before. It stores what the ticket stands for on the server and answers `303` to the GET leg with an opaque ticket id; the nonce that binds the ticket to this browser goes in a cookie.
3. **Neighbour, GET.** `GET /_system/federation/guest?ticket=...` opens the ticket only with that nonce, once. If the browser already holds a session of its own, it is left alone. Otherwise a guest session is stored and the browser is sent to the target path.

## What a guest is

- A user row with no email and no Telegram id, and a row in `federation_guests` (`kid`, `reader_hash`, unique together). The admin user list leaves those users out.
- A session with role `federation_guest`, carrying the key's row id (`ki`), the key's name (`k`), the hop, the reader id and the hub's origin. It lives 12 hours from the handoff (`usertoken.GuestSessionLifetime`), measured from the issue time, and is not renewed.
- Read access is the key's live subgraph grant plus what the same key reads over MCP: free notes, notes with no subgraph, and notes in sign-in-only subgraphs (`CurrentFederatedScope`).
- Every request checks the session against the key: a key that is revoked, replaced by another under the same name, or switched off for people ends the session as anonymous and clears the cookie.

## What a guest cannot do

Every non-GET request is refused with 403 except the two handoff legs, GraphQL (where every mutation but `signOut`, and every subscription, is refused with the guest error: a stream keeps the session it started with, so it would outlive a revocation or an expiry) and `POST /_system/federation/leave`. MCP answers 403. A guest is not handed further: the hub refuses to sign for one, and the neighbour refuses a hop other than 1.

## Protections

| Wall | Where |
|---|---|
| A token is a handoff, not a tool call, and the reverse | `use` claim, `mcp.VerifyGuestHandoff` |
| A form posted from another site opens nothing | signed origin against `Origin` |
| A copied ticket opens nothing in another browser | nonce cookie, `__Host-` prefix on HTTPS with `Path=/` |
| Single use | spent handoff ids; a ticket is removed when opened |
| Target stays on the site | `appresp.SamePath` on the hub and again on the neighbour |
| A note cannot choose the destination | the site address comes from the key |
| Note pages cannot be framed | `Content-Security-Policy: frame-ancestors 'self'` |
| Key identity | the session carries the key's row id; creating an inbound key under a name that existed is refused |

The nonce cookie takes the `__Host-` prefix whenever it is Secure. A `__Host-` cookie must have `Path=/`, so it cannot also be limited to the guest endpoint's path; the cookie path was never the boundary. Its name is unique to the guest endpoint, it lives 30 seconds and is cleared by the ticket leg.

## Settings and where they live

- **Accept people** is a column on the inbound key (`federation_secrets.accept_guests`, off by default), switched by the admin mutation `setFederationSecretAcceptGuests`. There is no instance-wide flag.
- **Landing page.** `mcp_federation_kb_path` in the KB-note's frontmatter names the page a guest lands on.
- **Open link.** `GET /_system/federation/open?note=<KB-note path>&path=<page on the neighbour>` shows the neighbour and a button that posts to the handoff endpoint. An anonymous reader gets the hub's sign-in wall at the same address. Federated search results from a direct neighbour carry `open_url`.
- **The bar.** The default template renders it on every page for a guest. A custom layout calls `{{ defaultTemplate.GuestBar() }}`; the method returns an empty string for anyone else.
- **Dead ends.** A browser that had a guest session and has none now is told once, on the sign-in or paywall page, to open the base again from its own knowledge base (`trip2g_was_guest` cookie, cleared on first read). The refusal page links back to the hub's origin only when the handoff's signature was valid.

## Known leaks, pinned by tests

- A closed note's title is shown on its closed page and in search results.
- Subgraph names are in the markup of links to closed notes (`class="paywall paywall-<subgraph>"`).
- A refused page still counts as a view.

## Tests

`cmd/server/federation_guest_security_test.go` holds the attack tests of the plan's section 8.1, in process against real handlers; the list of mutations is read from the GraphQL schema and the list of non-GET endpoints from the router, so one added later is covered. `e2e/federation-guest.spec.js` walks the flow in Chromium across three instances.
