# Telegram Bot vs Userbot: Custom Emoji and Media Group Limits

## Overview

This document explains the differences between Telegram Bot API and userbot (MTProto API) approaches, particularly for custom emoji support and media group caption limits.

## Telegram Bot API (Standard Approach)

### What It Is
- Official Telegram Bot API
- Uses bot token from @BotFather
- HTTP-based API

### Limitations

#### Custom Emoji
- **Requirement**: Bot must purchase Fragment username AND connect it to the bot
- **Cost**:
  - Fragment username: ~$12 USD (one-time)
  - **Connection fee: 1,000 TON (~$2,000 USD)** ⚠️
- **Conclusion**: Not economically viable for most use cases

#### Media Group Captions
- **Limit**: 1,024 characters maximum
- Cannot be increased

### Syntax for Custom Emoji (Bot API)
When bot has Fragment username connected:
```html
<tg-emoji emoji-id="5368324170671202286">👍</tg-emoji>
```

**Official Documentation:**
- https://core.telegram.org/bots/api#formatting-options
- https://core.telegram.org/bots/api-changelog (April 21, 2023 - Bot API 6.7)

## Userbot Approach (MTProto API)

### What It Is
- Operates as a regular Telegram user account
- Uses MTProto API (not Bot API)
- Requires phone number authentication

### Advantages

#### Custom Emoji
- **Requirement**: Account must have Telegram Premium subscription
- **Cost**: ~$5 USD/month (Telegram Premium)
- **Conclusion**: Much more affordable than Bot API approach

#### Media Group Captions
- **Standard account**: 1,024 characters
- **Premium account**: 2,048 characters ✅

### Implementation
- **Go Library**: [`github.com/gotd/td`](https://github.com/gotd/td) - official MTProto implementation
- **Authentication**: Phone number + SMS code (one-time), then session-based
- **API**: Different from Bot API - uses MTProto protocol

### Legal/TOS Considerations
- Telegram discourages automation for spam/mass messaging
- **Acceptable use cases**: Publishing own content to own channels
- **Unacceptable**: Mass spam, aggressive automation
- Always respect rate limits and Telegram's terms of service

## Recommended Strategy

### For trip2g Service

**Option 1: Standard Bot (Default)**
- Client creates bot via @BotFather
- No custom emoji support
- Media group captions limited to 1,024 characters
- **Cost**: Free

**Option 2: Premium Userbot (Optional)**
- Client connects their own Telegram account with Premium subscription
- Custom emoji support ✅
- Media group captions up to 2,048 characters ✅
- **Cost**: ~$5 USD/month (client pays directly to Telegram)

### Implementation Approach

1. **Default**: Use Bot API for standard publishing
2. **Premium Feature**: Allow clients to optionally connect their Premium account via userbot
3. **Client Responsibility**: Each client manages their own Premium subscription if they need these features

## Custom Emoji ID Extraction Tool

### cmd/emojibot

A simple utility bot to help clients get custom emoji IDs for use in Obsidian notes.

**How it works:**
1. User sends message with custom emoji to the bot
2. Bot extracts `custom_emoji_id` from message entities
3. Bot returns markdown code: `![emoji](tg://emoji?id=5368324170671202286)`

**Dependencies**: None (uses standard library only)

**Usage:**
```bash
export TELEGRAM_BOT_TOKEN="your_bot_token"
go run cmd/emojibot/main.go
```

See `cmd/emojibot/README.md` for details.

## Markdown Syntax for Custom Emoji

In Obsidian notes, use this syntax:
```markdown
![emoji](tg://emoji?id=5368324170671202286)
```

When publishing:
- **Via Bot API** (with Fragment): Converts to `<tg-emoji>` HTML tag
- **Via Userbot** (with Premium): Uses native custom emoji entities

## Cost Comparison

| Approach | One-time Cost | Monthly Cost | Custom Emoji | Media 2048 chars |
|----------|---------------|--------------|--------------|------------------|
| Standard Bot | $0 | $0 | ❌ | ❌ |
| Bot + Fragment | ~$2,012 | $0 | ✅ | ❌ |
| Userbot + Premium | $0 | ~$5 | ✅ | ✅ |

**Winner**: Userbot with Premium subscription

## Technical Notes

### Bot API Message Entities
```go
type MessageEntity struct {
    Type          string `json:"type"`           // "custom_emoji"
    Offset        int    `json:"offset"`
    Length        int    `json:"length"`
    CustomEmojiID string `json:"custom_emoji_id"` // The ID we need
}
```

### MTProto Implementation (Future)
When implementing userbot support:
1. Use `gotd/td` library
2. Authenticate with phone number
3. Store session for reuse
4. Implement rate limiting to avoid Telegram restrictions
5. Handle Premium status check before using Premium features

## References

- [Telegram Bot API Documentation](https://core.telegram.org/bots/api)
- [Telegram Bot API Changelog](https://core.telegram.org/bots/api-changelog)
- [Fragment Marketplace](https://fragment.com)
- [gotd/td - Go MTProto Client](https://github.com/gotd/td)
- [Custom Emoji Documentation](https://core.telegram.org/api/custom-emoji)
