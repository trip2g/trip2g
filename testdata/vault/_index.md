---
free: true
title: Test Vault Home
description: Comprehensive test vault for Obsidian publishing features
sidebar: true
---

Welcome to the comprehensive test vault for Obsidian publishing!

## Link Resolution Tests
1. [[unique]] - unique filename resolution
2. [[folder/source]] - duplicate filename priority
3. [[projectA/README]] - multiple conflicts across folders
4. [[img-test]] - image resolution
5. [[headers]] - headers and block references
6. [[embedding]] - markdown embeds with duplicates

## Publishing Features Tests
7. [[public]] - free public page (no paywall)
8. [[paid_with_preview]] - paid content with 2 paragraph preview
9. [[paid_with_cut]] - paid content with `---` cut marker
10. [[with_layout]] - custom layout test
11. [[toc_test]] - table of contents (auto/show/hide)
12. [[telegram_text]] - Telegram text post (no media, type: text)
13. [[telegram_one_photo]] - Telegram single photo post (type: photo)
14. [[telegram_media_group]] - Telegram media group (2+ media, type: media_group)
15. [[cyrillic_названия]] - Cyrillic in URLs and links
16. [[File with spaces]] - spaces in filenames
17. [[code_and_media]] - code blocks and media embeds
18. [[complex_content]] - comprehensive markdown features
19. [[redirect_test]] - page redirect functionality

## Subgraph (Premium Course) Tests
20. [[premium]] - premium subgraph home page
21. Check sidebar: should show premium sidebar for premium pages

## Special Files Tests
- `_banner.md` - banner embed (try ![[_banner]])
- `_sidebar.md` - global sidebar
- `_sidebar_premium.md` - subgraph-specific sidebar
- `_index.md` in projectA and projectB

## Key Test: Duplicate Priority
From [[folder/source]]:
- `[[dup]]` → /dup.md (root!) ⚠️
- `[[folder/dup]]` → /folder/dup.md ✅

From [[embedding]]:
- `![[_banner]]` → /_banner.md (root!) ⚠️
- `![[projectA/_banner]]` → /projectA/_banner.md ✅

## Expected Behavior
- Global link resolution with root directory priority
- Explicit paths (e.g., `folder/file`) always work
- Relative paths (`./file`) for local resolution
- Subgraphs create separate content spaces with their own sidebars
- Free content preview works with `free_paragraphs` and `free_cut`

## Frontmatter Fields Tested

| Field | Example | Purpose |
|-------|---------|---------|
| `free` | `true` | No paywall |
| `free_paragraphs` | `2` | Show N paragraphs free |
| `free_cut` | `true` | Cut at `---` marker |
| `title` | `"Page Title"` | Custom title |
| `description` | `"SEO text"` | Meta description |
| `subgraphs` | `premium` | Assign to course |
| `sidebar` | `false` / path | Show/hide/custom |
| `layout` | `custom/name` | Custom layout |
| `toc` | `auto/show/hide` | Table of contents |
| `complexity` | `0/1/2` or `easy/medium/hard` | Content difficulty |
| `reading_time` | `5` | Minutes to read |
| `telegram_publish_at` | datetime | Telegram post time |
| `telegram_publish_tags` | `[tag1]` | Telegram groups |
| `hidden` | `true` | Hide from listing |
| `embed_class` | `alert` | CSS class for embeds |

![[_banner]]
