# Vault Access Grants for Telegram Customers

**Status:** design draft, no implementation. Captures options surfaced during the Business Connection demo on 2026-05-24.

## Problem

In the Business Connection demo, the bot replies on behalf of the owner (`@rnbwkpr`) in business chats and also exposes a `/browse` wikilink navigator in its DM. Right now navigation is unrestricted: anyone who can DM the bot can walk the full vault. We want the owner to **grant a customer access to a specific subgraph** of the vault, and the customer to **pin that grant** in their DM with the bot for self-service.

## Use Case

1. Owner @rnbwkpr is selling consulting. Vault contains `pricing/`, `legal/`, `clients/private/`, `marketing/`.
2. Customer @jrpcd asks about pricing in a business chat.
3. Owner taps "Share Pricing pack with @jrpcd" in their own bot DM.
4. Bot generates a grant token, sends a deep link to @jrpcd from owner's account (via business connection): "Self-serve pricing here: t.me/trip2g_dev4_bot?start=grant_abc".
5. Customer taps → opens DM with the bot → bot accepts the grant, persists it, and pins a "📁 Pricing pack" entry in their personal menu.
6. Customer can later return to the DM anytime, tap the pinned grant, navigate within the `pricing/` subgraph only. Wikilinks crossing the boundary appear as 🔒 or are hidden.

## Public Opt-In via Frontmatter (Tier 1)

The simplest layer, predating customer-specific grants: a note opts itself into the Telegram navigator with a frontmatter attribute.

```yaml
---
telegram_navigation: allow
---
```

Rules:

- `/browse` lists only notes that carry this flag (or their reachable subgraph filtered through it)
- Wikilinks pointing to notes **without** the flag stay as plain text in the rendered message — no button is generated and the callback is rejected if forged
- Applies in any chat where the bot is invoked (DM, business chat)
- **Does NOT affect site rendering.** The Obsidian publisher pipeline ignores this attribute; `[[Plans]]` remains a normal wikilink on the website regardless. The flag is metadata consumed only by the Telegram navigation handler.

Bulk application is handled by the existing [frontmatter patch system](./frontmatter_patches.md):

```yaml
# patch: telegram-pricing.yaml
match:
  path_prefix: pricing/
set:
  telegram_navigation: allow
```

One patch makes a whole folder publicly browsable from Telegram. Removing the patch removes the access.

This is the default public tier. The grant mechanics below (Tier 2) layer on top for customer-specific scopes that don't belong on the public surface.

## Grant Scope Options (Tier 2)

| Option | Definition | Pros | Cons |
|--------|------------|------|------|
| **A. Root + transitive closure** | Pick one note, grant access to it plus everything reachable via wikilinks | Zero curation. Matches "this whole topic" intuition | Leaks through unintended links; scope drifts as vault edits link new notes in |
| **B. Explicit allowlist** | Owner ticks specific note IDs | Maximum precision, audit-friendly | Tedious per grant; doesn't scale beyond a few notes |
| **C. Folder / path prefix** | Grant = "notes under `pricing/**`" | Matches Obsidian's folder mental model; clean revocation; deterministic | Requires publishers to organize by folder (most do anyway) |
| **D. Tag / frontmatter property** | Grant = "notes with `access: customer` in frontmatter" | Cross-folder, works for tag-based vaults | Less intuitive; requires editorial discipline |
| **E. Hybrid (C + B overrides)** | Folder + explicit excludes/includes | Best of both | More complex UI to express |

**Recommendation:** (C) as the default with (E) as the power-user escape. Folders are how publishers already think; "share `pricing/`" is a one-tap operation.

## Boundary Behavior for Wikilinks

When the renderer encounters a `[[Target]]` whose target lies outside the grant scope:

| Strategy | UX | Notes |
|----------|----|-------|
| **Hide** the link entirely from text and keyboard | Cleanest, no visual hint of locked content | Owner-side renderings differ from customer-side; might be confusing for the owner inspecting |
| **Show as 🔒 in text, no button** | Customer sees there's more, can't access | Honest, but invites "give me access too" pressure |
| **Show 🔒 button → tap = "Request access" message to owner** | Built-in access request funnel | Best for sales flow; needs owner-side UI to approve |

The third option doubles as a lead-generation mechanic, which fits the publishing platform's purpose.

## Schema Sketch

```sql
create table tg_vault_grants (
  id              integer primary key autoincrement,
  bot_id          int  not null references tg_bots(id) on delete cascade,
  granted_by      int  not null,                       -- owner's Telegram user id
  granted_to      int  not null,                       -- customer's Telegram user id
  scope_kind      text not null,                       -- 'folder' | 'note_ids' | 'tag'
  scope_value     text not null,                       -- folder path, JSON array of ids, or tag name
  label           text not null default '',            -- human-friendly name shown to customer
  token           text not null unique,                -- random token for deep link
  expires_at      datetime,                            -- nullable for indefinite grants
  revoked_at      datetime,                            -- nullable
  created_at      datetime not null default current_timestamp,
  last_used_at    datetime
);

create index tg_vault_grants_lookup on tg_vault_grants(bot_id, granted_to) where revoked_at is null;
```

A "pin" in the customer's DM is just a row with `granted_to = <customer>`. Their `/menu` (or implicit "no active session") screen lists all active grants for that `(bot_id, granted_to)` tuple.

## Granting Flow (Owner Side)

1. Owner is in their bot DM. From `/manage` or via a deep link from a business chat (`/start bizChat<chat_id>`), bot shows:
   - "Send grant to @jrpcd" button
2. Tap → bot asks: which folder? (or: paste path)
3. Bot generates token, inserts row, returns a deep link:
   ```
   t.me/trip2g_dev4_bot?start=grant_<token>
   ```
4. Bot sends the deep link to @jrpcd *via the business connection* (so the customer sees it as if owner messaged them). Optionally adds a one-line pitch: "Pricing pack — tap to open."

## Redeeming Flow (Customer Side)

1. Customer taps deep link → Telegram opens the bot DM with `/start grant_<token>`.
2. Bot validates token (exists, not expired, not revoked, `granted_to` matches `from.id`).
3. Bot persists the customer's redemption (mark `last_used_at`), shows:
   - "✅ Access granted: 📁 Pricing pack — granted by @rnbwkpr"
   - Inline buttons: "Open now" / "Pin to menu"
4. Subsequent `/start` or `/menu` in the DM lists pinned grants and lets the customer enter any of them. Navigation is constrained to the grant scope.

## Permission Checks in `renderNote`

Today (demo): every wikilink resolves freely. With grants:

```go
func renderNote(noteID int64, grant *VaultGrant) (text string, buttons []Button) {
    // ... existing rendering ...
    for _, wl := range parseWikilinks(body) {
        targetID := resolve(wl.Target)
        if !grant.Allows(targetID) {
            // hide, lock, or convert to "request access" button
            continue
        }
        buttons = append(buttons, navButton(targetID, wl.Display))
    }
}
```

The grant struct carries `scope_kind` + `scope_value`; `Allows(noteID)` returns `bool` based on the note's path/tags/id.

## Revocation & Visibility

- Owner sees "Active grants" in `/manage`: customer name, scope, last accessed, button to revoke.
- Revoke = `update tg_vault_grants set revoked_at = current_timestamp where id = ?`. Next browse attempt returns "access revoked".
- Optional: notify customer on revoke ("Access to Pricing pack has ended").

## Open Questions

1. **Multi-grant per customer**: how many active grants per customer is reasonable to surface in their DM menu? Probably show all, but cap practical limit somewhere.
2. **Anonymous redemption**: should grants be redeemable by anyone with the token, or strictly by the pre-named `granted_to`? Pre-named is safer; anonymous is more shareable (good for "magic link" flows).
3. **Notification on first access**: ping the owner when a customer first redeems? Probably yes — sales signal.
4. **Bundles**: should "pack" be a first-class entity ("Pricing Pack" composed of folder + 3 individual notes), or always resolve via single `scope_value`? Bundles fit reality (sales kits include miscellaneous notes) but add a layer.
5. **Discoverability**: should owners be able to publish "self-service" grants on a public landing page (no per-customer ask), where any visitor can redeem? Yes, this becomes a marketing surface.
6. **Cross-bot grants**: scope tied to `bot_id`. If publisher has multiple bots (rare today), grants don't cross. Acceptable.

## Relationship to Existing Plans

- Builds directly on [`tg-handler-routing.md`](../../.omc/plans/tg-handler-routing.md) Phase 0/1 — the navigation handler is the consumer of grants.
- Pairs with [`telegram_premium_bots.md`](./telegram_premium_bots.md) (Owner-Private Control Surface section) — grants are created from the owner's bot DM, redeemed in customer's bot DM, bridged by deep links.
- Pairs with the future agent (`telegram_inbox_agent.md`) — agent can auto-suggest "share this folder" when classifying inbound questions.

## Future: GraphQL API for Inbox Replies with Navigation

Operators (and the inbox agent) will need a way to programmatically send a navigation-anchored reply into a specific business chat from outside the bot's update loop — e.g. from the admin UI's inbox view or from an autonomous agent processing leads.

Sketch:

```graphql
type Mutation {
  """
  Reply to an inbox message in a Telegram business chat, optionally attaching
  a wikilink navigation rooted at the given note path.
  """
  tgReplyWithNavigation(input: TgReplyWithNavigationInput!): TgReplyResult!
}

input TgReplyWithNavigationInput {
  botID: ID!
  businessConnectionID: String!     # which connected business owner
  chatID: ID!                       # target customer chat
  replyToMessageID: Int             # optional: thread the reply
  text: String                      # optional: prefix text above the note
  navRootPath: String               # vault path to open as the starting note
  navMode: TgNavMode                # SHOW_BUTTONS | TEXT_ONLY (no keyboard)
}

enum TgNavMode {
  SHOW_BUTTONS    # render note with wikilink inline keyboard
  TEXT_ONLY       # render rendered note text without nav buttons
}

type TgReplyResult {
  messageID: Int!
  errors: [Error!]
}
```

Under the hood the resolver:

1. Validates the caller owns `botID` and that `businessConnectionID` is active for them
2. Resolves `navRootPath` → note ID; rejects if note lacks `telegram_navigation: allow` (or operator overrides via separate permission)
3. Reuses the same `renderNote` path as the live handler
4. Calls `sendMessage` with `business_connection_id`, persists nav state in `tg_user_navigation_states` so the customer's taps continue from this seed
5. Returns the message id so the admin UI can show "sent at HH:MM" and offer "withdraw" via `deleteMessage`

This unlocks: agent classifies inbound → picks a relevant note → fires `tgReplyWithNavigation` → customer continues self-serve from there. Operator can do the same manually from the inbox view.

Pairs with the inbox table from [`telegram_inbox_agent.md`](./telegram_inbox_agent.md) — that's where the source `replyToMessageID` comes from.

## When to Pick This Up

After Phase 1 of `tg-handler-routing.md` lands (navigation handler + dispatcher + nav state table). Grants are an extension layer, not a foundation. Don't build them until the nav handler is stable.
