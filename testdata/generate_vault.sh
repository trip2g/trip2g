#!/bin/bash

# Obsidian Link Resolution Test Vault Generator
# Creates minimal test structure for link resolution testing

VAULT="vault"

# Helper function to download placeholder image
# Usage: download_placeholder "path/to/file.png" "color"
download_placeholder() {
  local path="$1"
  local color="${2:-gray}"
  local file="$VAULT/$path"
  local ext="${path##*.}"

  echo "Downloading $path..."
  local url="https://placehold.co/400x300/${color}/white/${ext}?text=${path}"
  curl -sL "$url" -o "$file"
}

echo "Creating test vault: $VAULT"
# rm -rf "$VAULT"
mkdir -p "$VAULT"/folder/folder
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

Should resolve to ROOT:

![[_banner]]

Should resolve to local folder:

![[./_banner]]
EOF

cat > "$VAULT/folder/_banner.md" << 'EOF'
A'm at folder/_banner.md
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

cat > "$VAULT/telegram_text.md" << 'EOF'
---
telegram_publish_disable_web_page_preview: false
telegram_publish_at: 2025-11-18T09:36:00
telegram_publish_tags:
  - test_channel
free: true
title: Text-only Telegram Post
---
id: telegram_text

This is a **text-only** Telegram post with no media attachments. It will be sent using `sendMessage` API method.

**Text formatting examples:**

**Bold text**, *italic text*, and ***bold italic*** formatting.

~~Strikethrough~~ is also supported.

`Inline code` and <u>underlined text</u>.

Code block with syntax highlighting:

```python
def hello():
    print("Hello, world!")
```

**Links to other posts:**

Check out these posts:
- [[telegram_media_group|Cool media group]] with photos and video
- [[telegram_one_photo]] single photo example
- [[_index|Главная]] link to website (not published to Telegram)

**Lists:**

Unordered list:
- First item
- Second item
- Third item

Numbered list:
1. First step
2. Second step
3. Third step

**Blockquote:**

> This is a blockquote
> Multiple lines supported

**Custom emoji:** ![2️⃣](https://ce.trip2g.com/5307907239380528763.webp)

This tests comprehensive Telegram text formatting with all supported markdown features.

[[telegram_one_photo]] | [[telegram_media_group]]
EOF

cat > "$VAULT/telegram_one_photo.md" << 'EOF'
---
telegram_publish_at: 2025-11-18T09:37:00
telegram_publish_tags:
  - test_channel
free: true
title: Single Photo Telegram Post
---
id: telegram_one_photo

This post contains **one photo** and will be sent using `sendPhoto` API method.

The post type is: **photo**

![[telegram_photo.png]]

Caption features:
- Maximum 1024 characters
- HTML formatting
- Can be edited later with `editMessageCaption`

This tests single media attachment with caption.

[[telegram_text]] | [[telegram_one_video]] | [[telegram_media_group]]
EOF

cat > "$VAULT/telegram_one_video.md" << 'EOF'
---
telegram_publish_at: 2025-11-18T09:37:30
telegram_publish_tags:
  - test_channel
free: true
title: Single Video Telegram Post
---
id: telegram_one_video

This post contains **one video** and will be sent using `sendVideo` API method.

The post type is: **photo** (same as single photo internally)

![[telegram_single_video.mp4]]

Caption features:
- Maximum 1024 characters
- HTML formatting
- Can be edited later with `editMessageCaption` (not EditMessageWithPhoto!)

This tests single video attachment with caption.

[[telegram_text]] | [[telegram_one_photo]] | [[telegram_media_group]]
EOF

cat > "$VAULT/telegram_media_group.md" << 'EOF'
---
telegram_publish_at: 2025-11-18T09:38:00
telegram_publish_tags:
  - test_channel
free: true
title: Media Group Telegram Post
---
id: telegram_media_group

This post contains **multiple media files** (2-10) and will be sent using `sendMediaGroup` API method.

The post type is: **media_group**

![[telegram_photo.png]]
![[telegram_photo2.jpg]]
![[telegram_video.mp4]]

Features:
- Multiple photos and videos (up to 10)
- Only first media gets the caption
- Caption can be edited with `editMessageCaption`
- Media files cannot be changed after sending

This tests media group functionality with mixed photo and video content.

[[telegram_text]] | [[telegram_one_photo]]
EOF

cat > "$VAULT/telegram_image_with_emoji.md" << 'EOF'
---
telegram_publish_at: 2025-11-18T09:39:00
telegram_publish_tags:
  - test_channel
free: true
title: Image with Custom Emoji
---
id: telegram_image_with_emoji

This post has a **single photo** with custom emoji in caption.

![[telegram_photo.png]]

Custom emoji test: ![➡️|20x20](tg_ce_5974249837439224721.webp) and ![😅](https://ce.trip2g.com/5384209107215456745.webp).

The custom emoji files should NOT be included as media attachments!
Only `telegram_photo.png` should be the post media.

[[telegram_text]] | [[telegram_video_with_emoji]]
EOF

cat > "$VAULT/telegram_video_with_emoji.md" << 'EOF'
---
telegram_publish_at: 2025-11-18T09:40:00
telegram_publish_tags:
  - test_channel
free: true
title: Video with Custom Emoji
---
id: telegram_video_with_emoji

This post has a **single video** with custom emoji in caption.

![[telegram_single_video.mp4]]

Custom emoji test: ![➡️|20x20](tg_ce_5974249837439224721.webp) and ![😅](https://ce.trip2g.com/5384209107215456745.webp).

The custom emoji files should NOT be included as media attachments!
Only `telegram_single_video.mp4` should be the post media.

[[telegram_text]] | [[telegram_image_with_emoji]]
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
Should be test.png (red):
![[test.png]]

Should be assets/test.png (blue):
![[assets/test.png]]

Should be folder/test.png (green):
![[folder/test.png]]

---

Links: [[img-formats]] [[folder/imgs]]
EOF

cat > "$VAULT/img-formats.md" << 'EOF'
---
free: true
---
Should be format.png (orange):
![[format.png]]

Should be format.jpg (purple):
![[format.jpg]]

Should be format.webp (cyan):
![[format.webp]]

Should be format.svg (gold):
![[format.svg]]

---

Links: [[img-test]] [[folder/imgs]]
EOF

cat > "$VAULT/folder/imgs.md" << 'EOF'
---
free: true
---
Should be format.png (orange - ROOT):
![[format.png]]

Should be format.jpg (purple - ROOT):
![[format.jpg]]

Should be format.webp (cyan - ROOT):
![[format.webp]]

Should be format.svg (gold - ROOT):
![[format.svg]]

Should be folder/format.png (pink):
![[folder/format.png]]

Should be folder/format.jpg (lime):
![[folder/format.jpg]]

---

Links: [[img-test]] [[img-formats]]
EOF

cat > "$VAULT/projectA/imgs.md" << 'EOF'
---
free: true
---
Should be format.jpg (purple - ROOT):
![[format.jpg]]

Should be projectA/format.jpg (teal):
![[projectA/format.jpg]]
EOF

cat > "$VAULT/folder/folder/imgs.md" << 'EOF'
---
free: true
---
Should be test.png (red - ROOT):
![[test.png]]

Should be folder/folder/test.png (yellow - LOCAL):
![[./test.png]]

Should be assets/asset0.png (violet):
![[asset0.png]]
EOF

# Create test images with placeholders
echo "Creating test images..."

# test.png files
download_placeholder "test.png" "red"
download_placeholder "assets/test.png" "blue"
download_placeholder "folder/test.png" "green"
download_placeholder "folder/folder/test.png" "yellow"

# asset0.png
download_placeholder "assets/asset0.png" "violet"

# format.* files (root)
download_placeholder "format.png" "orange"
download_placeholder "format.jpg" "purple"
download_placeholder "format.webp" "cyan"
download_placeholder "format.svg" "FFD700"

# format.* files (folder)
download_placeholder "folder/format.png" "pink"
download_placeholder "folder/format.jpg" "lime"
download_placeholder "folder/format.webp" "navy"
download_placeholder "folder/format.svg" "8B4513"

# format.jpg (projectA)
download_placeholder "projectA/format.jpg" "teal"

# Telegram post media files
download_placeholder "telegram_photo.png" "3498db"
download_placeholder "telegram_photo2.jpg" "e74c3c"

# Custom emoji files (should be excluded from post media)
# Local tg_ce_* pattern
# download_placeholder "tg_ce_fire.webp" "ff4500"
# download_placeholder "tg_ce_thumbsup.webp" "1e90ff"
curl -sL "https://ce.trip2g.com/5974249837439224721.webp" -o "$VAULT/tg_ce_5974249837439224721.webp"

# Generate test videos (requires ffmpeg)
echo "Creating test videos..."
if ! command -v ffmpeg &> /dev/null; then
  echo "⚠️  ffmpeg not found. Install it with: sudo apt install ffmpeg"
  echo "Skipping video generation."
else
  # Video for media group (green)
  ffmpeg -f lavfi -i color=c=2ecc71:s=640x480:d=2 -f lavfi -i anullsrc=channel_layout=stereo:sample_rate=44100 \
    -c:v libx264 -preset ultrafast -crf 28 -t 2 -pix_fmt yuv420p \
    -c:a aac -b:a 64k -shortest \
    -y "$VAULT/telegram_video.mp4" 2>/dev/null

  if [ -f "$VAULT/telegram_video.mp4" ]; then
    file_size=$(du -h "$VAULT/telegram_video.mp4" | cut -f1)
    echo "✓ Created telegram_video.mp4 ($file_size)"
  else
    echo "⚠️  Failed to create telegram_video.mp4"
  fi

  # Video for single video post (blue)
  ffmpeg -f lavfi -i color=c=3498db:s=640x480:d=3 -f lavfi -i anullsrc=channel_layout=stereo:sample_rate=44100 \
    -c:v libx264 -preset ultrafast -crf 28 -t 3 -pix_fmt yuv420p \
    -c:a aac -b:a 64k -shortest \
    -y "$VAULT/telegram_single_video.mp4" 2>/dev/null

  if [ -f "$VAULT/telegram_single_video.mp4" ]; then
    file_size=$(du -h "$VAULT/telegram_single_video.mp4" | cut -f1)
    echo "✓ Created telegram_single_video.mp4 ($file_size)"
  else
    echo "⚠️  Failed to create telegram_single_video.mp4"
  fi
fi

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
# Test 6: Image with same name as note (regression test)
# Bug: ![[software.png]] was resolved as /software (the note) when software.md exists
# ============================================================================

cat > "$VAULT/software.md" << 'EOF'
---
free: true
title: Software Page
---
This page tests image with same basename as the note.

![[software.png]]

The image above should render as an image, not cause a render error.
EOF

download_placeholder "software.png" "2980b9"

# ============================================================================
# Test 7: Links with dots in filenames (regression test)
# Bug: filepath.Ext("Сценарий. Ютубер") returns ". Ютубер" as extension
# ============================================================================

cat > "$VAULT/_scenarios.md" << 'EOF'
Links to pages with dots in names:
- [[Сценарий. Ютубер|Ютубер]]
- [[Сценарий. Курсы|Курсы]]
EOF

cat > "$VAULT/Сценарий. Ютубер.md" << 'EOF'
---
free: true
title: Сценарий Ютубер
---
Страница со сценарием для ютуберов.
EOF

cat > "$VAULT/Сценарий. Курсы.md" << 'EOF'
---
free: true
title: Сценарий Курсы
---
Страница со сценарием для курсов.
EOF

cat > "$VAULT/scenarios_test.md" << 'EOF'
---
free: true
title: Scenarios Test
---
This page embeds _scenarios which has links with dots in names.

![[_scenarios]]

Links above should NOT be marked as "wip" since target pages exist.
EOF

# ============================================================================
# Test 8: Markdown embeds with duplicates
# ============================================================================

cat > "$VAULT/_banner.md" << 'EOF'
---
banner: true
---
I'm the ROOT banner at /_banner.md
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

Global embed. Should resolve to ROOT:

![[_banner]]

ProjectA banner:

![[projectA/_banner]]

ProductB banner:

![[projectB/_banner]]
EOF

# ============================================================================
# Test 7: Custom layouts
# ============================================================================

cat > "$VAULT/_layouts/custom/styles.css" << 'EOF'
body {
  font-family: system-ui, -apple-system, sans-serif;
  color: #0f0;
}

main {
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
}
EOF

cat > "$VAULT/_layouts/custom/blocks.html" << 'EOF'
{{ block main_layout() }}

<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ note.Title() }}</title>
    <link rel="stylesheet" href="{{ asset("styles.css") }}">
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

<!-- {{ asset("styles.css") }} -->

{{ yield main_layout() content }}

<article>
  <h1>{{ note.Title() }}</h1>

  <div>
    {{ note.HTMLString() | unsafe }}
  </div>
</article>

{{ end }}
EOF

# ============================================================================
# JSON Layout Tests (_layouts)
# ============================================================================

cat > "$VAULT/_layouts/json-test.html.json" << 'EOF'
{
  "meta": {},
  "body": [
    {"type": "html", "html": "<!DOCTYPE html>\n<html>\n<head>\n  <meta charset=\"UTF-8\">\n  <title>"},
    {"type": "expr", "expr": "note.Title()"},
    {"type": "html", "html": "</title>\n</head>\n<body>\n"},
    {"type": "html", "html": "<header id=\"json-layout-header\">\n  <h1>"},
    {"type": "expr", "expr": "note.Title()"},
    {"type": "html", "html": "</h1>\n</header>\n"},
    {
      "type": "if",
      "condition": "note.M().GetBool(\"show_sidebar\", false)",
      "content": [
        {"type": "html", "html": "<aside id=\"json-layout-sidebar\">"},
        {"type": "note_content", "path": "/_json_test_sidebar.md"},
        {"type": "html", "html": "</aside>\n"}
      ]
    },
    {"type": "html", "html": "<main id=\"json-layout-main\">\n"},
    {"type": "note_content"},
    {"type": "html", "html": "\n</main>\n"},
    {"type": "html", "html": "<footer id=\"json-layout-footer\">"},
    {"type": "html", "html": "<p>JSON Layout Footer</p>"},
    {"type": "html", "html": "</footer>\n"},
    {"type": "html", "html": "</body>\n</html>"}
  ]
}
EOF

cat > "$VAULT/_layouts/json-include-missing.html.json" << 'EOF'
{
  "meta": {},
  "body": [
    {"type": "html", "html": "<!DOCTYPE html>\n<html>\n<head>\n  <meta charset=\"UTF-8\">\n  <title>"},
    {"type": "expr", "expr": "note.Title()"},
    {"type": "html", "html": "</title>\n</head>\n<body>\n"},
    {"type": "html", "html": "<h1>"},
    {"type": "expr", "expr": "note.Title()"},
    {"type": "html", "html": "</h1>\n"},
    {"type": "html", "html": "<div id=\"include-missing-test\">"},
    {"type": "include_note", "path": "/_nonexistent_file.md"},
    {"type": "html", "html": "</div>\n"},
    {"type": "html", "html": "<main>"},
    {"type": "note_content"},
    {"type": "html", "html": "</main>\n"},
    {"type": "html", "html": "</body>\n</html>"}
  ]
}
EOF

cat > "$VAULT/_json_test_sidebar.md" << 'EOF'
---
title: JSON Test Sidebar
---

- [Home](/)
- [Public](/public)

Sidebar loaded via note_content.
EOF

cat > "$VAULT/json_layout_test.md" << 'EOF'
---
free: true
layout: json-test
title: JSON Layout Test Page
show_sidebar: true
---

This page uses a JSON layout file (.html.json) instead of a regular .html template.

The layout demonstrates:
- HTML blocks
- Expression blocks (title)
- Conditional rendering (sidebar)
- include_note with fallback
- note_content for main content
EOF

cat > "$VAULT/json_layout_no_sidebar.md" << 'EOF'
---
free: true
layout: json-test
title: JSON Layout No Sidebar
show_sidebar: false
---

This page uses the same JSON layout but with show_sidebar: false.

The sidebar should NOT be visible.
EOF

cat > "$VAULT/json_layout_missing_include.md" << 'EOF'
---
free: true
layout: json-include-missing
title: JSON Layout Missing Include
---

This page tests include_note with a missing file.

The include should show "Create file: /_nonexistent_file.md" message.
EOF

# ============================================================================
# Template Views Tests (_layouts)
# ============================================================================

cat > "$VAULT/_layouts/meta-test.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>{{ note.Title() }}</title>
</head>
<body>
  <h1>{{ note.Title() }}</h1>

  <div id="meta-test">
    <p id="meta-author">Author: {{ note.M().GetString("author", "Unknown Author") }}</p>
    <p id="meta-version">Version: {{ note.M().GetInt("version", 0) }}</p>
    <p id="meta-featured">Featured: {{ note.M().GetBool("featured", false) }}</p>
    <p id="meta-has-author">Has author: {{ note.M().Has("author") }}</p>
    <p id="meta-has-missing">Has missing: {{ note.M().Has("nonexistent") }}</p>
  </div>

  <div id="note-info">
    <p id="note-reading-time">Reading time: {{ note.ReadingTime() }} min</p>
    <p id="note-path-id">Path ID: {{ note.PathID() }}</p>
    <p id="note-permalink">Permalink: {{ note.Permalink() }}</p>
  </div>

  <div id="content">
    {{ note.HTMLString() | unsafe }}
  </div>
</body>
</html>
EOF

cat > "$VAULT/_layouts/with-sidebar.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>{{ note.Title() }}</title>
</head>
<body>
  <div id="layout-container">
    {{ sidebar := nvs.ByPath("/_test_sidebar.md") }}
    {{ if sidebar }}
    <aside id="custom-sidebar">
      <h2>{{ sidebar.Title() }}</h2>
      {{ sidebar.HTMLString() | unsafe }}
    </aside>
    {{ end }}

    <main id="main-content">
      <h1>{{ note.Title() }}</h1>
      {{ note.HTMLString() | unsafe }}
    </main>

    {{ footer := nvs.ByPath("/_test_footer.md") }}
    {{ if footer }}
    <footer id="custom-footer">
      {{ footer.HTMLString() | unsafe }}
    </footer>
    {{ end }}
  </div>
</body>
</html>
EOF

cat > "$VAULT/_layouts/with-backlinks.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>{{ note.Title() }}</title>
</head>
<body>
  <main>
    <h1>{{ note.Title() }}</h1>
    {{ note.HTMLString() | unsafe }}
  </main>

  <section id="backlinks">
    <h2>Backlinks</h2>
    {{ backlinks := nvs.BackLinks(note) }}
    {{ if len(backlinks) > 0 }}
    <ul id="backlinks-list">
      {{ range i, link := backlinks }}
      <li><a href="{{ nvs.ResolveURL(link) }}">{{ link.Title() }}</a></li>
      {{ end }}
    </ul>
    {{ else }}
    <p id="no-backlinks">No pages link to this one.</p>
    {{ end }}
  </section>
</body>
</html>
EOF

# Test notes for template views

cat > "$VAULT/template_meta_test.md" << 'EOF'
---
free: true
layout: meta-test
title: Meta Test Page
author: John Doe
version: 42
featured: true
---

This page tests the template Meta accessor methods.
EOF

cat > "$VAULT/template_meta_defaults.md" << 'EOF'
---
free: true
layout: meta-test
title: Meta Defaults Page
---

This page tests Meta accessor default values (no custom meta fields set).
EOF

cat > "$VAULT/_test_sidebar.md" << 'EOF'
---
title: Test Sidebar
---

- [Home](/)
- [Public](/public)
- [About](/about)

This is the sidebar content.
EOF

cat > "$VAULT/_test_footer.md" << 'EOF'
---
title: Test Footer
---

© 2025 Test Site. All rights reserved.
EOF

cat > "$VAULT/template_sidebar_test.md" << 'EOF'
---
free: true
layout: with-sidebar
title: Sidebar Test Page
---

This page tests nvs.ByPath() for loading sidebar and footer from separate notes.
EOF

cat > "$VAULT/template_backlinks_target.md" << 'EOF'
---
free: true
layout: with-backlinks
title: Backlinks Target
---

This page is linked from other pages. Check the backlinks section below.
EOF

cat > "$VAULT/template_backlinks_source1.md" << 'EOF'
---
free: true
title: Backlinks Source 1
---

This page links to [[template_backlinks_target]].
EOF

cat > "$VAULT/template_backlinks_source2.md" << 'EOF'
---
free: true
title: Backlinks Source 2
---

Another page linking to [[template_backlinks_target]].
EOF

# ============================================================================
# Test 8: Custom slug URL override
# ============================================================================

cat > "$VAULT/slug_relative.md" << 'EOF'
---
free: true
slug: custom-name
title: Relative Slug Test
---
File: slug_relative.md
Expected URL: /custom-name (relative, replaces filename only)
EOF

cat > "$VAULT/folder/slug_relative_nested.md" << 'EOF'
---
free: true
slug: my-custom-page
title: Nested Relative Slug
---
File: folder/slug_relative_nested.md
Expected URL: /folder/my-custom-page
EOF

cat > "$VAULT/slug_absolute.md" << 'EOF'
---
free: true
slug: /archive/old-post
title: Absolute Slug Test
---
File: slug_absolute.md
Expected URL: /archive/old-post (absolute, full path override)
EOF

cat > "$VAULT/slug_with_subdir.md" << 'EOF'
---
free: true
slug: sub/nested/page
title: Slug with Subdirectory
---
File: slug_with_subdir.md
Expected URL: /sub/nested/page (relative with subdirs)
EOF

cat > "$VAULT/slug_cyrillic.md" << 'EOF'
---
free: true
slug: моя-страница
title: Cyrillic Slug
---
File: slug_cyrillic.md
Expected URL: /моя-страница (no transliteration!)
EOF

cat > "$VAULT/slug_spaces.md" << 'EOF'
---
free: true
slug: page with spaces
title: Slug with Spaces
---
File: slug_spaces.md
Expected URL: /page%20with%20spaces (URL encoded)
EOF

# ============================================================================
# MCP (Model Context Protocol) test notes
# ============================================================================

cat > "$VAULT/_mcp_initialize.md" << 'EOF'
---
mcp_method: initialize
---

# Knowledge Base

Personal knowledge base with notes on programming, projects, and ideas.

## Tools

- **search** — keyword search across all notes
- **similar** — find semantically related notes
- **instructions** — detailed usage guide

## Tips

1. Use search for specific topics
2. Use similar to explore related content
3. Notes link to each other via [[wikilinks]]
EOF

cat > "$VAULT/_mcp_instructions.md" << 'EOF'
---
mcp_method: instructions
mcp_description: Detailed instructions for using this knowledge base
---

# Detailed Knowledge Base Instructions

This is a comprehensive guide for working with this knowledge base.

## Available Tools

### search
Search notes by keyword or phrase.

**When to use:**
- Looking for specific topic
- Finding notes by title or content
- Keyword-based queries

**Example queries:**
- "golang error handling"
- "project architecture"
- "API design"

### similar
Find semantically similar notes using vector search.

**When to use:**
- Exploring related topics
- Finding notes with similar concepts
- When keyword search returns too few results

## Content Organization

Notes are organized hierarchically:
- Top-level folders for major topics
- Subfolders for specific areas
- `_` prefix for system/hidden notes

## Best Practices

1. Start broad, then narrow down
2. Combine search + similar for comprehensive results
3. Follow [[wikilinks]] to discover connections
4. Check related notes for context
EOF

# ============================================================================
# Broken Layout Test (parse error handling)
# ============================================================================

cat > "$VAULT/_layouts/broken-layout.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
  <title>{{ note.Title() }}</title>
</head>
<body>
  {{ some_unclosed_tag
  <main>
    {{ note.HTMLString() | unsafe }}
  </main>
</body>
</html>
EOF

cat > "$VAULT/broken_layout_test.md" << 'EOF'
---
free: true
layout: broken-layout
title: Broken Layout Test Page
---

This page uses a broken layout that has a parse error.

For guests: should render with default layout (no error visible).
For admins: should show the layout error message.
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
7. [[software]] - image with same name as note (regression)
8. [[scenarios_test]] - links with dots in filenames (regression)

## Publishing Features Tests
7. [[public]] - free public page (no paywall)
8. [[paid_with_preview]] - paid content with 2 paragraph preview
9. [[paid_with_cut]] - paid content with `---` cut marker
10. [[with_layout]] - custom layout test
11. [[toc_test]] - table of contents (auto/show/hide)
12. [[telegram_text]] - Telegram text post (no media, type: text)
13. [[telegram_one_photo]] - Telegram single photo post (type: photo)
14. [[telegram_one_video]] - Telegram single video post (type: photo, uses sendVideo)
15. [[telegram_media_group]] - Telegram media group (2+ media, type: media_group)
16. [[telegram_image_with_emoji]] - Image with custom emoji (tg_ce_* excluded from media)
17. [[telegram_video_with_emoji]] - Video with custom emoji (ce.trip2g.com/* excluded from media)
18. [[cyrillic_названия]] - Cyrillic in URLs and links
19. [[File with spaces]] - spaces in filenames
20. [[code_and_media]] - code blocks and media embeds
21. [[complex_content]] - comprehensive markdown features
22. [[redirect_test]] - page redirect functionality
23. [[slug_relative]] - relative slug (replaces filename)
24. [[slug_absolute]] - absolute slug (full path override)
25. [[slug_with_subdir]] - slug with subdirectory
26. [[slug_cyrillic]] - cyrillic slug (no transliteration)
27. [[slug_spaces]] - slug with spaces (URL encoded)

## JSON Layout Tests
28. [[json_layout_test]] - JSON layout with sidebar (show_sidebar: true)
29. [[json_layout_no_sidebar]] - JSON layout without sidebar (show_sidebar: false)
30. [[json_layout_missing_include]] - JSON layout with missing include_note file

## Layout Error Handling
31. [[broken_layout_test]] - page with broken layout (parse error handling)

## Subgraph (Premium Course) Tests
28. [[premium]] - premium subgraph home page
29. Check sidebar: should show premium sidebar for premium pages

## Frontmatter Patches Tests
30. [[patch_tests/simple]] - simple patch (free: true)
31. [[patch_tests/chained]] - chained patches with priorities
32. [[patch_tests/conditional]] - conditional logic (layout only if missing)
33. [[patch_tests/has_layout]] - conditional no-op (layout exists)
34. [[patch_tests/excluded]] - excluded by exclude_patterns
35. [[patch_tests/title_template]] - title template with meta merge
36. [[patch_tests/path_based]] - jsonnet path-based logic

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
| `slug` | `custom-url` or `/full/path` | Custom URL (relative or absolute) |

![[_banner]]
EOF

# ============================================================================
# Test 9: Frontmatter Patches
# ============================================================================

mkdir -p "$VAULT"/patch_tests

cat > "$VAULT/patch_tests/simple.md" << 'EOF'
---
# Expected patches: "Make blog posts free" (priority 0)
# Expected meta after patches: { free: true }
# Original value BEFORE patch:
free: false
title: Simple Patch Test
---

This page tests a simple frontmatter patch that sets `free: true`.

The patch should match pattern `patch_tests/simple.md` and apply `{ free: true }`.
EOF

cat > "$VAULT/patch_tests/chained.md" << 'EOF'
---
# Expected patches: "Add default layout" (priority 0), "Override blog layout" (priority 10)
# Expected meta after patches: { layout: "blog_layout", free: true }
# Original frontmatter has no layout field
free: false
title: Chained Patch Test
---

This page tests patch chaining with different priorities.

Patch 1 (priority 0): Sets default layout
Patch 2 (priority 10): Overrides with blog_layout
Patch 3 (priority 20): Sets free: true

Final result should have layout="blog_layout" and free=true.
EOF

cat > "$VAULT/patch_tests/conditional.md" << 'EOF'
---
# Expected patches: "Conditional default layout" (priority 0)
# Expected meta after patches: { layout: "default", free: true }
# Tests: if std.objectHas(meta, "layout") then {} else { layout: "default" }
free: true
title: Conditional Patch Test
---

This page tests conditional jsonnet logic.

The patch checks `if std.objectHas(meta, "layout")` and only sets layout if missing.

Since this note has no layout in frontmatter, patch should add `layout: "default"`.
EOF

cat > "$VAULT/patch_tests/has_layout.md" << 'EOF'
---
# Expected patches: "Conditional default layout" (priority 0)
# Expected meta: { layout: "custom", free: true } - layout NOT overridden
# Tests: conditional should return {} (no-op) when layout already exists
layout: custom
free: true
title: Has Layout Test
---

This page already has `layout: custom` in frontmatter.

The conditional patch should detect this and return `{}` (no-op).

Final layout should remain "custom", not changed to "default".
EOF

cat > "$VAULT/patch_tests/excluded.md" << 'EOF'
---
# Expected patches: NONE (excluded by exclude_patterns)
# Expected meta: unchanged from frontmatter
# Tests: exclude patterns override include patterns
free: false
title: Excluded Patch Test
---

This page matches include pattern `patch_tests/*` BUT is excluded by `exclude_patterns: ["patch_tests/excluded.md"]`.

No patches should be applied. The `free` field should remain `false`.
EOF

cat > "$VAULT/patch_tests/title_template.md" << 'EOF'
---
# Expected patches: "Site title suffix" (priority 100)
# Expected meta: { title: "Title Template Test — Test Site", free: true }
# Tests: meta + { title: meta.title + " — Test Site" }
free: true
title: Title Template Test
---

This page tests title template patch using jsonnet expression:
`meta + { title: meta.title + " — Test Site" }`

The patch should append " — Test Site" to the original title.

Final title should be: "Title Template Test — Test Site"
EOF

cat > "$VAULT/patch_tests/path_based.md" << 'EOF'
---
# Expected patches: "Path-based logic" (priority 0)
# Expected meta: { patch_applied: true, free: true }
# Tests: if std.startsWith(path, "patch_tests/") then { patch_applied: true } else {}
free: true
layout: meta_inspector
title: Path-Based Logic Test
---

This page tests jsonnet logic based on the path variable.

Since path is "patch_tests/path_based.md", patch should add patch_applied: true.
EOF

# meta_inspector layout: outputs raw frontmatter as JSON (for e2e verification)
cat > "$VAULT/_layouts/meta_inspector.html" << 'EOF'
{{ note.M().Raw() | writeJson }}
EOF

# ============================================================================
# Multi-domain routing tests
# ============================================================================

mkdir -p "$VAULT/multidomain"

cat > "$VAULT/multidomain/root.md" << 'EOF'
---
free: true
title: Custom Domain Root
route: customdomain.test/
---
This is the root page on the custom domain.
EOF

cat > "$VAULT/multidomain/about.md" << 'EOF'
---
free: true
title: Custom Domain About
route: customdomain.test/about
---
About page on the custom domain.
EOF

cat > "$VAULT/multidomain/multi_route.md" << 'EOF'
---
free: true
title: Multi-Route Note
routes:
  - customdomain.test/multi
  - /multi-alias
---
This note is accessible via two routes.
EOF

cat > "$VAULT/multidomain/no_route.md" << 'EOF'
---
free: true
title: No Route — Patch Target
---
This note has no route in frontmatter.
A frontmatter patch will add: route customdomain.test/patch-target
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
echo "   ✓ Telegram publishing (text, photo, media_group)"
echo "   ✓ Cyrillic and special characters in filenames"
echo "   ✓ Code blocks and media embeds"
echo "   ✓ Redirects"
echo "   ✓ Headers and block references"
echo "   ✓ Complex markdown (tables, lists, quotes, tasks)"
echo "   ✓ Custom slug URL override (relative, absolute, cyrillic, spaces)"
echo "   ✓ Frontmatter patches (simple, chained, conditional, excluded, title template, path-based)"
echo "   ✓ Multi-domain routing (route/routes frontmatter, custom domain)"
echo ""
echo "📖 Open vault/_index.md to see all available tests"
echo ""
