---
telegram_rich: on
telegram_publish_at: 2026-07-20T09:00:00
telegram_publish_tags:
  - test_channel
free: true
title: Rich Telegram Post
---
id: telegram_rich

# Rich Telegram Post

This post is delivered through `sendRichMessage` as typed blocks, not through
`parse_mode`. The **structure below survives** the trip: headings keep their
level, the table keeps its columns, and the callout stays collapsible.

## A second-level heading

Headings are real blocks here. The classic HTML path drops them entirely,
because Telegram's classic formatting has no heading at all.

### A third-level heading

Each level arrives as the level it was written at.

## A table

| Feature | Classic | Rich |
|---|---|---|
| Headings | dropped | kept |
| Tables | dropped | kept |
| Collapsible | flattened | kept |

## A collapsible section

> [!note]- Folded by default
> An Obsidian callout written with `-` arrives collapsed, and the reader
> expands it. Written with `+` it arrives expanded instead.

## Lists

Unordered:

- First item
- Second item
- Third item

Ordered:

1. First step
2. Second step
3. Third step

## Code

```python
def hello():
    print("Hello, rich world!")
```

Formatting still works inside a paragraph: **bold**, *italic*, ~~strikethrough~~,
`inline code` and <u>underline</u>.
