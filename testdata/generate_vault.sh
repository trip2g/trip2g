#!/bin/bash

# Obsidian Link Resolution Test Vault Generator
# Creates minimal test structure for link resolution testing

VAULT="vault"

echo "Creating test vault: $VAULT"
rm -rf "$VAULT"
mkdir -p "$VAULT"/folder
mkdir -p "$VAULT"/assets
mkdir -p "$VAULT"/projectA
mkdir -p "$VAULT"/projectB
mkdir -p "$VAULT"/_layouts/custom

# ============================================================================
# Test 1: Unique filenames - simple case
# ============================================================================

cat > "$VAULT/unique.md" << 'EOF'
---
free: true
---
Link: [[deep]] - should find /folder/deep.md
EOF

cat > "$VAULT/folder/deep.md" << 'EOF'
---
free: true
---
Found me! Path: /folder/deep.md
EOF

# ============================================================================
# Test 2: Duplicate filenames - priority test (CRITICAL)
# ============================================================================

cat > "$VAULT/dup.md" << 'EOF'
---
free: true
---
I'm at /dup.md
EOF

cat > "$VAULT/folder/dup.md" << 'EOF'
---
free: true
---
I'm at /folder/dup.md
EOF

cat > "$VAULT/folder/source.md" << 'EOF'
---
free: true
---
Test: [[dup]] - goes to ROOT, not local! ⚠️
Local: [[./dup]] - this one stays local ✅
Explicit: [[folder/dup]] - also local ✅
EOF

# ============================================================================
# Test 3: Multiple conflicts across subfolders
# ============================================================================

cat > "$VAULT/projectA/README.md" << 'EOF'
---
free: true
---
Testing link resolution with ambiguous filenames.

Link: [[guide]] - ambiguous!
Explicit: [[projectA/guide]] - clear
EOF

cat > "$VAULT/projectA/guide.md" << 'EOF'
---
free: true
---
Guide A file located at /projectA/guide.md
EOF

cat > "$VAULT/projectA/_index.md" << 'EOF'
---
free: true
---
This is the index page for Project A
EOF

cat > "$VAULT/projectB/README.md" << 'EOF'
---
free: true
---
Project B testing duplicate README files.

Link: [[_index]] - to vault home
EOF

cat > "$VAULT/projectB/guide.md" << 'EOF'
---
free: true
---
Guide B file located at /projectB/guide.md
EOF

cat > "$VAULT/projectB/_index.md" << 'EOF'
---
free: true
---
This is the index page for Project B
EOF

cat > "$VAULT/public.md" << 'EOF'
---
free: true
title: Public Content Page
description: This is a public page available to everyone
---

This is publicly accessible content without paywall. Anyone can read this page without authentication or subscription.
EOF

cat > "$VAULT/telegram_post.md" << 'EOF'
---
telegram_publish_at: 2025-10-28T10:01:00
telegram_publish_tags:
  - my_group
---

This content will be published to Telegram at the scheduled time. It demonstrates the telegram_publish_at and telegram_publish_tags frontmatter fields for automated posting to Telegram groups.
EOF

cat > "$VAULT/paid_with_preview.md" << 'EOF'
---
free_paragraphs: 2
subgraphs: premium
title: Premium Content with Preview
description: Paid content with 2 free preview paragraphs
---

This is the first paragraph that everyone can read.

This is the second free paragraph with more information.

This is paid content. You need a subscription to read this.

More exclusive content here that requires payment.
EOF

cat > "$VAULT/paid_with_cut.md" << 'EOF'
---
free_cut: true
subgraphs: premium
---

This is the free preview section.

You can read this part without subscription.

---

This is the paid section after the cut.

Premium content continues here.
EOF

cat > "$VAULT/with_layout.md" << 'EOF'
---
free: true
layout: custom/page
title: Custom Layout Page
---

This page uses a custom layout from _layouts/custom/page.html.

The layout includes:
- Simple header with navigation
- Content area with article wrapper
- Footer with copyright
EOF

cat > "$VAULT/toc_test.md" << 'EOF'
---
free: true
toc: show
complexity: medium
reading_time: 5
---

This page demonstrates the table of contents feature with the 'toc: show' frontmatter field. It shows automatic navigation generation from heading structure.

## Section 1
Content for section 1

## Section 2
Content for section 2

### Subsection 2.1
Nested content

## Section 3
More content
EOF

cat > "$VAULT/cyrillic_названия.md" << 'EOF'
---
free: true
title: Проверка кириллицы
description: Тест русских символов в URL
---

Страница с [[Моя страница|кириллическими ссылками]] для проверки работы с русскими символами в URL и названиях файлов.
EOF

cat > "$VAULT/Моя страница.md" << 'EOF'
---
free: true
---
Контент с русским названием файла для проверки работы с кириллицей в именах файлов.
EOF

cat > "$VAULT/File with spaces.md" << 'EOF'
---
free: true
title: File Name With Spaces
---

This file has spaces in its name to test URL normalization.

Link back: [[_index]]
EOF

cat > "$VAULT/code_and_media.md" << 'EOF'
---
free: true
title: Code and Media Test
---

This page demonstrates various code blocks and media embeds including YouTube videos, external images, and local image references.

## Code Block

\`\`\`python
def hello_world():
    print("Hello, World!")
    return 42
\`\`\`

## Inline Code

This is \`inline code\` in text.

## YouTube Embed

![](https://www.youtube.com/watch?v=dQw4w9WgXcQ)

## External Image

![External](https://via.placeholder.com/300x200)

## Local Image

![[test.png]]
EOF

cat > "$VAULT/redirect_test.md" << 'EOF'
---
redirect: /public
title: This page redirects
---

You should not see this page. It redirects to [[public]].
EOF

cat > "$VAULT/complex_content.md" << 'EOF'
---
free_paragraphs: 3
subgraphs: premium
complexity: hard
reading_time: 10
toc: auto
title: Complex Content Example
description: Advanced content with various markdown features
---

This is a comprehensive example demonstrating all markdown features including lists, blockquotes, tables, links, emphasis, task lists, and horizontal rules.

## Lists

- Item 1
- Item 2
  - Nested item 2.1
  - Nested item 2.2
- Item 3

1. First
2. Second
3. Third

## Blockquote

> This is a blockquote
> with multiple lines
>
> And a new paragraph

## Tables

| Header 1 | Header 2 | Header 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |

## Links

- Internal: [[_index]]
- With alias: [[_index|Home Page]]
- External: [Google](https://google.com)
- Email: <test@example.com>

## Emphasis

**Bold text** and *italic text* and ***bold italic***.

~~Strikethrough~~ text.

## Task List

- [x] Completed task
- [ ] Pending task
- [ ] Another task

## Horizontal Rule

---

Content after the rule (this is paid).

More premium content here.
EOF

# ============================================================================
# Test 4: Assets (images) with duplicates
# ============================================================================

cat > "$VAULT/img-test.md" << 'EOF'
---
free: true
---
This page tests image resolution with duplicate filenames.

Global: ![[test.png]] - which one?
Explicit: ![[assets/test.png]] - clear
EOF

# Create test images (minimal 1x1 PNGs with different colors)
# Alternative with network: curl -s "https://placehold.co/600x200?text=/test.png" -o "$VAULT/test.png"
echo "Creating test images..."
# Red pixel (root image)
curl -s "https://placehold.co/600x200?text=/test.png" -o "$VAULT/test.png"
# Green pixel (assets image)
curl -s "https://placehold.co/600x200?text=/assets/test.png" -o "$VAULT/assets/test.png"
# Blue pixel (folder image)
curl -s "https://placehold.co/600x200?text=/folder/test.png" -o "$VAULT/folder/test.png"

# ============================================================================
# Test 5: Headers and blocks
# ============================================================================

cat > "$VAULT/headers.md" << 'EOF'
---
free: true
---
This page demonstrates header links and block references in Obsidian-style links.

## Section One
Content here.

## Section Two
More content. ^block-id

Link to header: [[headers#Section One]]
Link to block: [[headers#^block-id]]
EOF

# ============================================================================
# Test 6: Markdown embeds with duplicates
# ============================================================================

cat > "$VAULT/_banner.md" << 'EOF'
---
banner: true
---
I'm the banner at /_banner.md
EOF

cat > "$VAULT/projectA/_banner.md" << 'EOF'
---
banner: true
---
I'm the banner at /projectA/_banner.md
EOF

cat > "$VAULT/projectB/_banner.md" << 'EOF'
---
banner: true
---
I'm the banner at /projectB/_banner.md
EOF

cat > "$VAULT/embedding.md" << 'EOF'
---
free: true
---
This page demonstrates markdown embeds with duplicate filenames. Global embed should resolve to ROOT, while explicit paths are clear.

Global embed: ![[_banner]] - should resolve to ROOT
Explicit: ![[projectA/_banner]] - clear
Another: ![[projectB/_banner]] - also clear
EOF

# ============================================================================
# Test 7: Custom layouts
# ============================================================================

cat > "$VAULT/_layouts/custom/blocks.html" << 'EOF'
{{ block main_layout() }}

<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ note.Title }}</title>
  </head>
  <body>
    <header>
      <nav>
        <a href="/">Home</a>
        <a href="/about">About</a>
      </nav>
    </header>

    <main>
      {{ yield content }}
    </main>

    <footer>
      <p>&copy; 2025 Test Vault. All rights reserved.</p>
    </footer>
  </body>
</html>

{{ end }}
EOF

cat > "$VAULT/_layouts/custom/page.html" << 'EOF'
{{ import "blocks" }}

{{ yield main_layout() content }}

<article>
  <h1>{{ note.Title }}</h1>

  <div>
    {{ note.HTMLString() | unsafe }}
  </div>
</article>

{{ end }}
EOF

# ============================================================================
# Main page and sidebar
# ============================================================================

cat > "$VAULT/_sidebar.md" << 'EOF'
- [[_index|Home]]
- [[public]]
- [[paid_with_preview]]
- [[toc_test]]
EOF

cat > "$VAULT/_sidebar_premium.md" << 'EOF'
# Premium Course

- [[premium|Home]]
- [[paid_with_preview]]
- [[paid_with_cut]]
EOF

cat > "$VAULT/premium.md" << 'EOF'
---
subgraphs: premium
title: Premium Course Home
---

This is the home page for the premium subgraph.

Available lessons:
- [[paid_with_preview]]
- [[paid_with_cut]]
EOF

cat > "$VAULT/_index.md" << 'EOF'
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
12. [[telegram_post]] - Telegram publishing integration
13. [[cyrillic_названия]] - Cyrillic in URLs and links
14. [[File with spaces]] - spaces in filenames
15. [[code_and_media]] - code blocks and media embeds
16. [[complex_content]] - comprehensive markdown features
17. [[redirect_test]] - page redirect functionality

## Subgraph (Premium Course) Tests
18. [[premium]] - premium subgraph home page
19. Check sidebar: should show premium sidebar for premium pages

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
EOF

echo ""
echo "✅ Test vault created successfully!"
echo ""
echo "📁 Location: $VAULT/"
echo ""
echo "📝 Files created:"
find "$VAULT" -type f -name "*.md" | wc -l | xargs echo "   Markdown files:"
find "$VAULT" -type f -name "*.png" -o -name "*.jpg" -o -name "*.webp" | wc -l | xargs echo "   Images:"
echo ""
echo "🧪 Test coverage:"
echo "   ✓ Link resolution (unique, duplicates, relative, explicit paths)"
echo "   ✓ Markdown embeds (![[]])"
echo "   ✓ Free content (free, free_paragraphs, free_cut)"
echo "   ✓ Subgraphs and premium content"
echo "   ✓ Custom layouts and sidebars"
echo "   ✓ Table of contents (auto/show/hide)"
echo "   ✓ Telegram publishing"
echo "   ✓ Cyrillic and special characters in filenames"
echo "   ✓ Code blocks and media embeds"
echo "   ✓ Redirects"
echo "   ✓ Headers and block references"
echo "   ✓ Complex markdown (tables, lists, quotes, tasks)"
echo ""
echo "📖 Open vault/_index.md to see all available tests"
echo ""
