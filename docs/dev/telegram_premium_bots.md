# Telegram Premium Business Bots: Reply on Behalf

## Overview

Telegram Business (launched April 2024, bundled into Telegram Premium) lets a Premium-subscribed user connect a bot that acts on their behalf in personal chats. Incoming messages from a customer are delivered to the bot as `business_message` updates; the bot replies with the same `business_connection_id` and the message appears in the chat as if the owner sent it manually — owner's name, owner's avatar, no `@bot` suffix.

For trip2g this flips the publishing model: instead of subscribers chatting with `@trip2g_publisher_bot`, they DM the publisher's **personal account** (e.g. `@alex`) and trip2g handles the inbox behind the scenes. Same wikilink browser, same forms, same payments — but the conversation feels personal.

## Wikilink Browser in DMs

Yes, the [navigation handler](./../../.omc/plans/tg-handler-routing.md) works in business chats unchanged. Business connections support:

- `sendMessage` with `reply_markup` (inline keyboards) — buttons render normally
- `updateBusinessBotCallbackQuery` — button taps reach the bot as callback queries with `business_connection_id` attached
- `editMessageText` — in-place note rendering still works
- All the methods our handler uses (`SendMessage`, `AnswerCallbackQuery`, `EditMessageText`)

The only routing change: dispatcher keys become `(bot_id, business_connection_id, user_id)` instead of `(bot_id, user_id)`. Empty `business_connection_id` means "regular bot mode" (legacy behavior preserved).

## How It Works

### Setup

| Actor | Action |
|-------|--------|
| Bot owner | `@BotFather` → Bot Settings → **Business Mode** → enable, configure default rights |
| End user (Premium) | Settings → **Telegram Business → Chatbots** → enter bot `@username`, grant `BusinessBotRights` |
| Telegram → bot | Sends `updateBotBusinessConnect` with `connection_id` |

The `connection_id` persists until the user revokes, disables, or changes the bot's rights — at which point a new `updateBotBusinessConnect` arrives (possibly with `disabled: true`).

### Update Stream

In addition to the regular `Update` types, business-connected bots receive:

| Update | When | Key fields |
|--------|------|-----------|
| `business_connection` | Connected / rights changed / disabled | `id`, `user` (owner), `can_reply`, `is_enabled` |
| `business_message` | Customer wrote to owner | `business_connection_id`, regular `Message` fields |
| `edited_business_message` | Customer edited a message | same as above |
| `deleted_business_messages` | Customer deleted messages | `business_connection_id`, `chat`, `message_ids` |

### Sending on Behalf

Most send/edit methods accept an optional `business_connection_id` parameter. When set, the API routes the action through the owner's account:

```
sendMessage(business_connection_id="abc123", chat_id=..., text=..., reply_markup=...)
```

The resulting `Message` has `via_business_connection: true` and `via_bot_id` set. To the customer, the indicator is barely visible (small "via bot" hint on long-press); functionally it looks like a 1:1 chat with the owner.

## What Bots Can Do

Methods callable with `business_connection_id` (subject to granted `BusinessBotRights`):

| Category | Methods |
|----------|---------|
| Messaging | `sendMessage`, `editMessageText`, `editMessageReplyMarkup`, `deleteMessages`, `readHistory`, `setTyping` |
| Media | `sendPhoto`, `sendDocument`, `sendVideo`, `sendMediaGroup`, `uploadMedia` |
| Stories | `sendStory`, `editStory`, `deleteStories` |
| Payments | `sendInvoice`, Stars transfers |
| Profile | `updateProfile`, `setGlobalPrivacySettings` |
| Polls | `sendPoll`, `stopPoll` |

## Limitations

| Limit | Detail |
|-------|--------|
| **Premium required** | Business is part of Telegram Premium; the owner's account must have an active subscription |
| **No forwarding** | Bots cannot `forwardMessage` or `copyMessage` from a business chat outward — content stays inside |
| **24h reply window** | `can_reply` only applies in chats active within the last 24 hours (per `BusinessBotRights`) |
| **Granular permissions** | Each right (`can_reply`, `can_delete_sent_messages`, `can_delete_all_messages`, `can_edit_name`, `can_edit_bio`, `can_view_gifts`, `can_sell_gifts`, `can_transfer_stars`, etc.) is opt-in per user |
| **`BOT_ACCESS_FORBIDDEN`** | Returned when an operation exceeds granted rights |
| **One bot per user** | A Premium user can connect only one business bot at a time |
| **Bot serves many owners** | A single bot can be connected by many Premium users; `connection_id` discriminates them |

## Impact on trip2g Architecture

### Schema Changes (incorporate into Phase 1)

The [handler routing plan](./../../.omc/plans/tg-handler-routing.md) introduces `tg_user_current_handlers` and `tg_user_navigation_states`. Both should include `business_connection_id` from the start to avoid a future migration:

```sql
create table tg_user_current_handlers (
  bot_id                 int  not null references tg_bots(id) on delete cascade,
  business_connection_id text not null default '',  -- '' = regular bot mode
  user_id                int  not null,
  value                  text not null default '',
  updated_at             datetime not null default current_timestamp,
  primary key (bot_id, business_connection_id, user_id)
);

create table tg_user_navigation_states (
  bot_id                 int  not null references tg_bots(id) on delete cascade,
  business_connection_id text not null default '',
  user_id                int  not null,
  value                  text not null default '{}',
  updated_at             datetime not null default current_timestamp,
  primary key (bot_id, business_connection_id, user_id)
);
```

New table for tracking active business connections per bot:

```sql
create table tg_business_connections (
  connection_id text primary key,                    -- from updateBotBusinessConnect
  bot_id        int  not null references tg_bots(id) on delete cascade,
  owner_user_id int  not null,                       -- Telegram user ID of Premium owner
  rights        text not null default '{}',          -- JSON snapshot of BusinessBotRights
  is_enabled    boolean not null default true,
  connected_at  datetime not null default current_timestamp,
  updated_at    datetime not null default current_timestamp
);
```

Optional bot-level flag:

```sql
alter table tg_bots add column business_mode boolean not null default false;
```

### Dispatcher Routing

`cmd/server/telegram.go` extracts `(bot_id, business_connection_id, user_id)` from any update type:

```go
func extractBusinessConnectionID(update tgbotapi.Update) string {
    switch {
    case update.BusinessMessage != nil:
        return update.BusinessMessage.BusinessConnectionID
    case update.EditedBusinessMessage != nil:
        return update.EditedBusinessMessage.BusinessConnectionID
    case update.DeletedBusinessMessages != nil:
        return update.DeletedBusinessMessages.BusinessConnectionID
    case update.BusinessConnection != nil:
        return update.BusinessConnection.ID
    case update.CallbackQuery != nil && update.CallbackQuery.Message != nil:
        return update.CallbackQuery.Message.BusinessConnectionID
    }
    return ""
}
```

Handlers (`handletgupdate`, `handletgnavigationupdate`) receive `business_connection_id` via the request context and pass it through every `Send` / `Edit` call.

### Reply Helper

A thin wrapper on `Send` that auto-injects `BusinessConnectionID`:

```go
func (a *app) SendForUser(ctx context.Context, businessConnectionID string, msg tgbotapi.Chattable) (tgbotapi.Message, error) {
    if businessConnectionID != "" {
        if base, ok := msg.(tgbotapi.BaseChatMessage); ok {
            base.BusinessConnectionID = businessConnectionID
        }
    }
    return a.tg.Send(msg)
}
```

## Critical: Go Library Status

The project uses `github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.2-0.20221020003552` (a community pin from Oct 2022). The upstream library's last tagged release is **v5.5.1 (Dec 2021)** — predates Business API by years. The installed module contains **no** `BusinessConnection`, `BusinessMessage`, or `business_connection_id` symbols.

Options:

| Option | Effort | Risk |
|--------|--------|------|
| **Migrate to `go-telegram/bot`** | Medium — rewrite Telegram layer | Lower long-term; library is actively maintained and supports Business |
| **Fork `tgbotapi` and add Business types** | Small — ~5 new structs + parameter on existing methods | Become maintainer; future Telegram updates need follow-up |
| **Manual HTTP calls for Business endpoints** | Smallest diff today | Two parallel code paths; ugly |

**Recommendation:** evaluate migration to `github.com/go-telegram/bot` once Business is committed to the roadmap. For Phase 1 (handler routing without Business), the current library is fine and the schema accommodates Business when we're ready.

## Comparison with Userbot Approach

| Aspect | Bot API (regular) | Business Bot | Userbot (MTProto, `gotd/td`) |
|--------|------------------|--------------|------------------------------|
| Identity in chat | `@bot_name` | Owner (Premium user) | Owner (any user) |
| Subscription required | No | Premium ($5/mo) | None (or Premium for emoji/2048-char captions) |
| Auth | BotFather token | BotFather token + user opt-in | Phone + SMS code, session |
| Custom emoji | Fragment $2k connection | Inherited from Premium owner | With Premium |
| Inline keyboards | Yes | Yes | No (regular users can't send) |
| Setup friction | Low | Low (1 setting toggle for user) | High (SMS auth flow) |
| ToS risk | Low | Low (official Telegram feature) | Medium (automation discouraged) |
| Multi-tenant | One bot serves many | One bot serves many Premium owners | One session per owner |

For DM-style support / inbox flows with rich UI (buttons, callbacks), **Business bots are the clear winner** over userbot. Userbot remains the answer when publishing to channels with custom emoji and 2048-char media captions ([see `telegram_bot_vs_userbot.md`](./telegram_bot_vs_userbot.md)).

## Owner-Private Control Surface

A business chat is a regular 1-to-1 private chat — every message in it is visible to both the owner and the customer. There is no native way to render a message visible only to the owner in the same chat. To give the owner a private control surface (search, inbox, admin UI), use the bot's **own DM with the owner** as a parallel channel.

### Two Contexts, One Bot

| Context | Visible to | Purpose |
|---------|-----------|---------|
| Business chat (`business_connection_id` set) | Owner + customer | Customer-facing UI: navigation, forms, payments, replies |
| Bot DM (`@trip2g_dev4_bot` ↔ owner) | Owner only | Owner-facing admin UI: search, full inbox, agent suggestions, escalation controls |

The same person (`@rnbwkpr`) lives in both at once. In `tg_user_current_handlers`, this is naturally `(bot_id, business_connection_id, user_id)` — empty `business_connection_id` for the DM, the connection id for business chats.

### Bridging the Two: `/start bizChat<chat_id>` Deep-Link

Telegram itself generates `t.me/<bot>?start=bizChat<chat_id>` when the owner taps a "Manage" button inside a business chat. The owner's client switches to the bot DM and the bot receives a `/start bizChat<chat_id>` payload — letting the bot show owner-only controls for that specific customer chat.

The same mechanism works in reverse: the bot can attach buttons with `url=t.me/<bot>?start=<context>` to a message in the business chat. Tapping moves the owner into the DM with full context; the customer just sees the owner tapped something. No alert, no leakage.

### Options for "Owner-Only" Hints

| Mechanism | Owner-only? | Notes |
|-----------|-------------|-------|
| Send message to bot DM with owner | Yes | Standard pattern. Use for search results, suggested replies, agent activity |
| `url=t.me/bot?start=<ctx>` button in business chat | Tap routes only the owner | Clean handoff; customer sees a button press but no content |
| `answerCallbackQuery(show_alert=true)` | Only the tapper | Anyone in the chat could tap; not strictly owner-only |
| `editMessageReplyMarkup` (change buttons) | All participants see the change | Not private |
| Telegram "Saved Messages" of the owner | Yes (visible only to owner) | Bot API cannot write there |
| Telegram drafts | Yes (visible only to owner) | Bot API does not expose drafts |

### Recommended Pattern

- Keep customer-facing surfaces (nav, forms, payments) inside the business chat
- Move all owner controls (search, inbox triage, agent override, analytics) into the bot DM
- Add at most one **bridge button** in business chats (`url=t.me/bot?start=...`) when the owner needs to invoke something private from a customer thread; everything else lives in the DM permanently

This also matches Telegram's own UX: their built-in chatbot integrations expose a small "Manage" affordance in business chats that funnels everything into the bot DM.

## Phasing into trip2g

| Phase | Deliverable |
|-------|-------------|
| **Phase 1** (current plan) | Handler routing + nav browser, regular bots only. Tables include `business_connection_id` column with default `''` |
| **Phase 2** | Inbox table also carries `business_connection_id`; admin UI shows business chats separately |
| **Phase 3 (Business support)** | Migrate Telegram client lib, handle `business_connection` updates, store connections, propagate `business_connection_id` through every send |
| **Phase 4** | Admin UI: connect-your-account flow, per-publisher dashboard of active business connections, rights inspector |

## References

- [Telegram Bot API — Available Updates](https://core.telegram.org/bots/api#update) — `business_connection`, `business_message`, `edited_business_message`, `deleted_business_messages` fields
- [Telegram API — Business overview](https://core.telegram.org/api/business)
- [Telegram API — Connected Business Bots](https://core.telegram.org/api/bots/connected-business-bots) — method list, `BusinessBotRights`
- [Telegram API — botBusinessConnection constructor](https://core.telegram.org/constructor/botBusinessConnection)
- [grammY — Telegram Business guide](https://grammy.dev/advanced/business) — concrete code examples
- [Telegram.Bot .NET — Business Features](https://telegrambots.github.io/book/4/business.html)
- [`go-telegram/bot`](https://pkg.go.dev/github.com/go-telegram/bot) — actively maintained Go library with Business support
- [`go-telegram-bot-api/telegram-bot-api` releases](https://github.com/go-telegram-bot-api/telegram-bot-api/releases) — last stable v5.5.1 (Dec 2021)
- Related internal docs: [`telegram.md`](./telegram.md), [`telegram_bot_vs_userbot.md`](./telegram_bot_vs_userbot.md), [`telegram_publish_through_accounts.md`](./telegram_publish_through_accounts.md), [`telegram_inbox_agent.md`](./telegram_inbox_agent.md)
