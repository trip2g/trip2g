# AI Chat: Knowledge Base Assistant

Publishing platform's AI assistant that answers user questions using the site's knowledge base (notes/vault). Works via MCP protocol — both the web widget and Telegram bot are MCP clients.

## Architecture: Thin Adapter (Approach A)

The MCP endpoint is the universal API. All clients (web widget, Telegram bot, third-party) connect to the same `/_system/mcp` endpoint. The LLM tool-calling loop lives in a single `internal/aichat/` package. Platform-specific behavior is abstracted behind a `ChatTransport` interface:

```go
type ChatTransport interface {
    SendMessageStream(ctx context.Context, conversationID string, tokenChan <-chan string) error
    SendToolActivity(ctx context.Context, conversationID string, activity ToolActivity) error
    AskUser(ctx context.Context, conversationID string, question string) (string, error)
}
```

Two implementations: `WebTransport` (SSE streaming via widget) and `TelegramTransport` (Telegram streaming API).

## Phases

### Phase 1A: Sign-in Wall + Captcha

Foundation layer. Gates content behind sign-in, protects against spam.

#### Sign-in Wall (subgraph flag)

New subgraph-level flag `require_signin: true`. When a guest visits a page in that subgraph, they see a sign-in prompt instead of the content.

- Subgraph config field: `require_signin` (boolean)
- Default template: render sign-in wall (like paywall but for auth) when guest hits a gated page
- New `$mol` widget: sign-in wall widget in `assets/ui/` (similar to paywall widget but triggers sign-in flow)
- After sign-in, user sees the content normally
- Existing `signInByCode` mutation already creates users on first use — sign-in IS sign-up

#### Captcha (Cloudflare Turnstile)

Spam protection for `requestEmailSignInCode` mutation. Triggered when global sign-in attempts exceed a configurable threshold.

- Track all sign-in code requests globally (not per-IP)
- Threshold configurable via `internal/appconfig` (default: 5 per hour)
- When threshold exceeded, return `RequestCaptchaPayload` instead of sending the code
- GraphQL union extended: `union RequestEmailSignInCodeOrErrorPayload = RequestEmailSignInCodePayload | ErrorPayload | RequestCaptchaPayload`
- `RequestCaptchaPayload` contains a `siteKey` field for the Cloudflare Turnstile widget
- UI shows Turnstile captcha widget, user solves it, re-sends request with captcha token
- `requestEmailSignInCode` input extended with optional `captchaToken` field
- Backend verifies captcha token with Cloudflare API before sending email
- Reference implementation: `../trip2g_simplepanel` (same Cloudflare Turnstile pattern)

### Phase 1B: MCP Tokens

Authenticated MCP access. Depends on Phase 1A (users must be able to sign in).

#### MCP Tokens

New token type for authenticated MCP access. Available to all registered users.

**Table:**

```sql
CREATE TABLE mcp_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL DEFAULT '',
    token_hash TEXT NOT NULL,
    token_prefix TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    last_used_at DATETIME,
    revoked_at DATETIME
);
```

**Mechanics:**
- Token passed in URL: `https://site.com/_system/mcp?token=mcp_abc123...`
- Generated in user/space widget (new "MCP" tab)
- Shown once at creation (plaintext), stored as SHA256 hash
- Prefix stored for display (`mcp_abc1...xyz9`)
- User can create multiple tokens, revoke any
- Limit: 10 tokens per user (including revoked ones still in DB)
- Cronjob cleans revoked tokens older than 7 days: `DELETE FROM mcp_tokens WHERE revoked_at IS NOT NULL AND revoked_at < now() - 7 days`

**Auth resolution in MCP endpoint:**
- `?token=` param -> hash -> lookup in `mcp_tokens` -> resolve user + role
- No token -> guest (public tools only)
- Admin user -> full tool set including write operations

#### User/Space Widget: MCP Tab

New tab in the existing `assets/ui/user/space` widget:
- Table of tokens: name, prefix, created date, last used
- Each row has copyable full MCP URL: `https://site.com/_system/mcp?token=<full_token>`
- "Add" button -> dialog with name field -> shows generated URL once
- Revoke button per row

### Phase 2: MCP Improvements (separate detailed spec)

Richer MCP tools for AI clients. High-level list:

**Tools:**
- `search` — improved descriptions, includes paid/free status in results
- `show_note` — replaces `note_html` and `open`; returns note content + outlinks for graph traversal; for paid notes returns preview/summary + buy CTA
- `similar` — existing, improved descriptions
- `ask_answer` — fixed response with optional custom field (FAQ/canned responses)
- `schedule_consultation` — cal.com integration as an MCP method
- Admin-only: `edit_note` — find/replace pattern (like webhook MCP implementation), not full content replacement

**Per-user AI memory:**

```sql
CREATE TABLE ai_user_memory (
    user_id TEXT NOT NULL REFERENCES users(id),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, key)
);
```

AI can read/write per-user facts via `get_memory` / `set_memory` tools.

**Paid content handling:**
- Search results include paid/free status (already shown in search UI, same for AI)
- `show_note` returns preview + "this content is paid" indicator for locked notes
- New concept: **promo subgraph** — a note (or group of notes) attached to a paid note that acts as a banner/landing page explaining why the user should buy it
- (Future: time-gated content — notes with availability dates)
- Note frontmatter includes AI-facing meta: explains to the AI WHY to propose this content and WHEN (e.g. "propose when user asks about X topic", "suggest after user shows interest in Y")
- AI never reveals paid content directly — uses promo subgraph + meta to naturally recommend purchases
- Old "cut" feature (show partial content) exists but the new approach (promo subgraph + AI meta) is preferred going forward

### Phase 3: AI Chat Transport

The chat layer that sits on top of MCP.

**LLM Integration:**
- OpenAI-compatible API (covers OpenAI, OpenRouter, Gemini)
- Two config fields: `ai_provider` (base URL) + `ai_key`
- Site owner provides default key in site settings
- Users can override with their own key (stored in user settings)

**Conversations:**
- Persistent in DB, user can return and continue
- Site owner can view all conversations (admin panel)
- Streaming responses on both web (SSE) and Telegram (native streaming API)

**Rate Limiting:**
- Per-user daily message count limit (configurable by site owner)
- Per-user token budget (input + output tokens tracked)
- Users with own key bypass site owner's limits

**Web UI:**
- Floating widget (like existing user/space pattern)
- Hybrid tool activity display (collapsible: "Searching for X...", "Reading note Y...")
- Focus on the answer, tool calls are secondary

**Telegram:**
- Auth required (existing Telegram auth flow)
- Streaming via Telegram's new real-time streaming API
- Same conversation persistence as web

**Transport interface:**
- `WebTransport` — SSE via GraphQL subscription
- `TelegramTransport` — Telegram bot streaming API
- Core LLM loop in `internal/aichat/` — identical for both platforms

## Dependencies

```
Phase 1A (sign-in wall + captcha) -> Phase 1B (MCP tokens) -> Phase 2 (MCP tools) -> Phase 3 (AI chat)
```

Phase 1A unblocks user registration with spam protection. Phase 1B adds MCP token auth. Phase 2 makes the MCP good enough for AI clients. Phase 3 adds the transport layer.

## Out of Scope

- Payment integration (existing system handles purchases)
- Note editing UI (existing admin panel)
- MCP marketplace / discovery
- Multi-model routing (single provider per site for now)
