# Subgraph payment and access — reference

How a user gains access to paid notes in trip2g. This document is generated for AI agents to skip the research step next time. Verify against current code if anything looks stale.

---

## Core concept

**Subgraph** = named subset of notes (`subgraphs` table). Each note declares its subgraphs in frontmatter (`subgraph: x` or `subgraphs: [a, b]`); one note can belong to many. Access is granted at the subgraph level, not the note level. The same subgraph primitive is used for paid content, auth-walled content, and Telegram-chat-gated content.

**Offer** = a sellable bundle of subgraphs (`offers` + `offer_subgraphs`). Has price, optional lifetime, optional active window. Multiple offers can include the same subgraph.

**Purchase** = a payment record (`purchases`). Created on payment-link click, updated by IPN to `confirmed`. Triggers grant.

**Grant** = a row in `user_subgraph_accesses` linking user to subgraph with optional `expires_at` and `purchase_id`. This is the durable access record.

**Access check** = `canreadnote.Resolve` compares note's subgraphs against the union of the user's active subgraphs from all sources.

---

## Schema (relevant tables only)

```sql
subgraphs (
  id, name, color, hidden,
  show_unsubgraph_notes_for_paid_users,
  require_signin                            -- auth-only access, no payment needed
)

offers (
  id, public_id, lifetime,                  -- e.g. "+600 days"
  price_usd, starts_at, ends_at
)

offer_subgraphs (offer_id, subgraph_id)     -- which subgraphs an offer unlocks

purchases (
  id, offer_id, user_id, email,
  status, payment_provider, payment_data, price_usd
)

user_subgraph_accesses (
  id, user_id, subgraph_id,
  expires_at,                                -- nullable = no expiration
  revoke_id,                                 -- non-null = revoked
  purchase_id, created_by                    -- one of: purchase, admin
)
```

External grant sources (each has its own tier table → subgraph mapping):

```sql
patreon_tier_subgraphs (tier_id, subgraph_id, ...)
boosty_tier_subgraphs  (tier_id, subgraph_id, ...)
tg_chat_subgraph_accesses (chat_id, subgraph_id, ...)
tg_bot_chat_subgraph_accesses (chat_id, user_id, subgraph_id, ...)
```

---

## Payment flow (NOWPayments / crypto)

The reference path. Patreon and Boosty follow the same idea but enter via webhook.

### 1. Create payment link

`internal/case/createpaymentlink/resolve.go`

```
GraphQL mutation CreatePaymentLink(offer_id, return_path, email?)
  → Resolve:
    1. Find user via session OR via email (anonymous flow).
    2. Load offer by public_id (must be active and have price_usd).
    3. INSERT INTO purchases (status=pending, offer_id, email, price_usd).
       Generate unique purchase id, retry on collision (16 tries).
    4. Build callback URLs:
         IPN → /_system/nowpayments-ipn
         success/cancel → return_path with payment_result query param
         If anonymous: success URL also carries a HAT (signs user in after pay).
    5. nowpayments.CreateInvoice(price, order_id=purchase.id, callback URLs).
    6. If anonymous: also issue PurchaseToken (used to claim purchase later).
    7. Return invoice URL (and optional purchase token) to client.
```

Anonymous purchases: user doesn't have an account yet. They get a `PurchaseToken` (cookie-stored). After payment they hit `signinbypurchasetoken` with the token, which finds the purchase by id, looks up the user that was created/linked at IPN time, and sets up a session.

### 2. IPN delivers confirmation

`internal/case/processnowpaymentsipn/resolve.go`

```
POST /_system/nowpayments-ipn (signed by NOWPayments)
  → Resolve:
    1. PurchaseByID(req.OrderID).
    2. Append IPN payload to purchases.payment_data (audit trail).
    3. Update purchases.status = req.PaymentStatus.
    4. If status == "confirmed":
       a. CountUserSubgraphAccessByPurchaseID — idempotency guard.
       b. If 0 grants exist for this purchase:
            grantAccesses(purchase):
              - UserByEmail(purchase.email) OR InsertUserWithEmail.
              - ListSubgraphsByOfferID(purchase.offer_id).
              - OfferByID for lifetime → expires_at = now + lifetime (or NULL).
              - For each subgraph: CreateUserSubgraphAccess(user, subgraph, purchase, expires_at).
       c. Else: log warn "access already granted".
    5. NotifyPuchaseUpdated(email).
```

`payment_data` is a JSON array of all received IPN payloads — useful for debugging and dispute resolution.

`expires_at` semantics: NULL = no expiration. Lifetime is set on the offer (`+600 days`, etc). Recurring billing is not modelled at trip2g level — recurring offers re-grant on each successful purchase.

### 3. External provider grants (no purchase row)

Patreon and Boosty grants don't create `purchases`. Instead, they create membership rows that join to subgraphs at query time:

- **Patreon:** `processpatreonwebhook` updates `patreon_members` (status, current_tier_id). Tier → subgraphs via `patreon_tier_subgraphs`. Match user by email join `users.email = patreon_members.email`.
- **Boosty:** `refreshboostydata` polls Boosty API, updates `boosty_members`. Same email-join pattern via `boosty_tier_subgraphs`.
- **Telegram chats:** admin maps chats to subgraphs (`tg_chat_subgraph_accesses` or `tg_bot_chat_subgraph_accesses`). User joining a chat (tracked in `tg_chat_members`/`tg_bot_chat_members`) implicitly gets the chat's subgraphs. Match by `users.tg_user_id = members.user_id`.

External grants are computed live, not denormalised into `user_subgraph_accesses`. That's why `listactiveusersubgraphs` aggregates four sources separately.

### 4. Manual admin grant

`AdminCreateUserSubgraphAccess` (write query): admin inserts directly into `user_subgraph_accesses` with `created_by=admin_id`, optional `expires_at`. No purchase row.

---

## Access aggregation

`internal/case/listactiveusersubgraphs/resolve.go`

The single source of truth for "what subgraphs is this user currently allowed to see".

```
ListActiveUserSubgraphs(user_id):
  1. UserBanByUserID — banned users get nothing (return nil).
  2. Admin check (AdminByUserID):
     - If admin → return ALL subgraph names from ListAllSubgraphs.
  3. Otherwise, union of:
     - ListActiveSubgraphNamesByUserID         (from purchases / admin grants)
     - ListActiveTgChatSubgraphNamesByUserID   (Telegram chat memberships)
     - ListActivePatreonSubgraphNamesByUserID  (active Patreon members)
     - ListActiveBoostySubgraphNamesByUserID   (active Boosty members)
  4. Dedupe via map → return []string.
```

The four queries (read.sql.go):

```sql
-- Direct grants (purchases, admin):
select distinct s.name
  from user_subgraph_accesses a
  join subgraphs s on a.subgraph_id = s.id
 where user_id = ?
   and (expires_at > datetime('now') or expires_at is null)
   and revoke_id is null;

-- Telegram chat membership:
select distinct s.name
  from users u
  join tg_chat_members m on u.tg_user_id = m.user_id
  join tg_bot_chats bc on bc.id = m.chat_id
  join tg_chat_subgraph_accesses a on a.chat_id = bc.id
  join subgraphs s on s.id = a.subgraph_id
 where u.id = ? and bc.removed_at is null;

-- Patreon active membership:
select distinct s.name
  from users u
  join patreon_members pm on u.email = pm.email
  join patreon_tier_subgraphs pts on pm.current_tier_id = pts.tier_id
  join subgraphs s on pts.subgraph_id = s.id
 where u.id = ? and pm.status = 'active_patron';

-- Boosty active membership: same shape with boosty_members + boosty_tier_subgraphs.
```

Note the result is `[]string` (subgraph names), not ids. Names are stable across sites and used in note frontmatter.

---

## Note → subgraph membership

Frontmatter:

```yaml
---
subgraph: course1               # single
# or
subgraphs: [course1, premium]   # multiple
---
```

Parsed in `model/note.go`:
- `(*NoteView).ExtractSubgraphs` reads both `subgraph` and `subgraphs` from `RawMeta`, dedupes, normalises into `NoteView.SubgraphNames []string`.
- `(*NoteViews).ExtractSubgraphs` builds the reverse index: `NoteViews.Subgraphs[name] = *NoteSubgraph` (with sidebar/home note pointers).
- Each `NoteView.Subgraphs` map is later attached so a note knows its subgraph metadata.

Names are validated by `internal/validator/subgraph.go` (`^[a-zA-Z0-9_]+$`, joined by `|` for multi-name fields where applicable).

---

## Access check (read path)

`internal/case/canreadnote/resolve.go`

```
canreadnote(note):
  1. CurrentUserToken → user identity (or nil for guest).
  2. Admin → true.
  3. Guest:
     - If note.Free → true.
     - Else → false.
  4. Authenticated user:
     a. Compute userSubgraphs = ListActiveUserSubgraphs(user).
     b. If any of note.Subgraphs has require_signin=true → true.
        (sign-in wall already passed in rendernotepage.)
     c. If userSubgraphs is empty → false.
     d. If note.SubgraphNames is empty (general knowledge):
          → true (any active subscriber sees general notes).
     e. Else: any intersection of note.SubgraphNames ∩ userSubgraphs → true.
        Otherwise → false.
```

Notes:
- `require_signin` is the "auth-walled but free" mode: any logged-in user qualifies, no subscription needed.
- Notes without subgraphs are considered general knowledge and visible to any active paid user. The `show_unsubgraph_notes_for_paid_users` flag on a subgraph affects rendering; default true.
- The check is intentionally simple and not optimised (see comment in source). Per-request cost is bounded because `ListActiveUserSubgraphs` runs once per page render.

---

## Sign-in flows that interact with payments

| Flow | Code | When |
|------|------|------|
| Email | `signinbyemail` | User clicks magic link from email. |
| HAT (Hot Auth Token) | `signinbyhat` | External system (e.g. payment success URL) hands a JWT. Used to bridge anonymous purchase → authenticated session. |
| Telegram | `signinbytgauthtoken` | Telegram login widget. |
| Purchase token | `signinbypurchasetoken` | Anonymous purchase: cookie-stored token, claimed after payment. Sets up session for the user that was created/linked during IPN processing. |

`requestemailsignin` issues sign-in codes; `createpaymentlink` reuses its `Env` for the anonymous-flow email signup.

---

## Useful entry points (file:symbol)

- Subgraphs schema: `db/schema.sql:CREATE TABLE subgraphs`
- Note → subgraphs: `internal/model/note.go:NoteView.ExtractSubgraphs`
- Active accesses union: `internal/case/listactiveusersubgraphs/resolve.go:Resolve`
- Direct DB query: `internal/db/queries.read.sql.go:ListActiveSubgraphNamesByUserID`
- Read check: `internal/case/canreadnote/resolve.go:Resolve`
- Payment link: `internal/case/createpaymentlink/resolve.go:Resolve`
- IPN → grant: `internal/case/processnowpaymentsipn/resolve.go:grantAccesses`
- Anonymous purchase claim: `internal/case/signinbypurchasetoken/resolve.go:Resolve`
- Patreon webhook: `internal/case/processpatreonwebhook/...`
- Boosty refresh: `internal/case/refreshboostydata/...`
- TG chat access table: `db/schema.sql:CREATE TABLE tg_chat_subgraph_accesses`
- Subgraph name validator: `internal/validator/subgraph.go`

---

## Common gotchas

- **Multiple subgraphs per note:** the access check is a set intersection, not equality. Don't accidentally use the first subgraph only.
- **External grants are live-joined**, not denormalised. If Patreon/Boosty data is stale, access can be stale too — the refresh jobs (`refreshpatreondata`, `refreshboostydata`) are critical to keep accesses current.
- **Email is the join key for Patreon/Boosty.** Users with mismatched emails between trip2g and the external provider will silently lose access.
- **`expires_at IS NULL` means forever.** Don't treat NULL as "expired".
- **`revoke_id` is the soft-delete mechanism for grants.** A non-NULL `revoke_id` means revoked; the row is preserved for audit. Revocation is administrative, separate from expiration.
- **Admin shortcut:** admins always pass the access check via `AdminByUserID` (returns ALL subgraphs). When testing access logic, log out of admin or test with a non-admin user.
- **Free notes:** `note.Free` (set from frontmatter, see model package) bypasses the subscription check entirely for guests. Don't confuse with "no subgraphs assigned".
- **Anonymous purchases are real:** the user row is created during IPN processing if the email is new. Don't assume `users` row exists at payment-link creation time.
- **Idempotency on IPN:** the same IPN can fire multiple times. `CountUserSubgraphAccessByPurchaseID` is the guard against double-grants.
- **HAT vs purchase token:** HAT is a generic auth bridge (email-based JWT). Purchase token is purchase-id-specific and only works after the IPN created the user. They serve different roles in the anonymous flow.
