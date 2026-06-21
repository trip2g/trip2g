---
title: "Markdown syntax"
free: true
lang_redirect: "[[ru/user/markdown]]"
home_position: 20
---

Notes in trip2g are Markdown files. Write them in Obsidian, sync, and they appear on your site with formatting intact. This page covers the syntax trip2g renders and a few rendering behaviors specific to the platform.

### Headings

```markdown
# Heading 1
## Heading 2
### Heading 3
#### Heading 4
```

More `#` symbols mean a smaller heading. Use `##` and `###` for most sections — `#` is reserved for the page title.

### Text emphasis

```markdown
**bold**
*italic*
***bold and italic***
~~strikethrough~~
`inline code`
```

Result: **bold**, *italic*, ***bold and italic***, ~~strikethrough~~, `inline code`

### Lists

Unordered:

```markdown
- first item
- second item
  - nested item
```

Ordered:

```markdown
1. first item
2. second item
3. third item
```

### Task lists

```markdown
- [x] done
- [ ] not done
```

Rendered as checkboxes on the published page. The state is visual only — readers cannot check or uncheck them.

### Links

External link:

```markdown
[link text](https://example.com)
```

Wikilink to another note:

```markdown
[[Note name]]
[[Note name|Custom display text]]
```

Wikilinks work the same way as in Obsidian. trip2g resolves them globally across the vault: `[[note]]` finds the file by name regardless of where in the vault the linking note lives. If two files share the same name, the one closer to the vault root takes priority — use `[[folder/note]]` to be explicit.

A wikilink to a note that exists on the site becomes a live `<a>` tag. A wikilink to a note that does not exist on the site renders as underlined text (no link).

### Images

From the vault (wikilink embed):

```markdown
![[assets/photo.png]]
![[assets/photo.png|alt text]]
![[assets/photo.png|alt text|300x200]]
![[assets/photo.png|300]]
```

Size can be specified after a `|` as `widthxheight` or just `width`. Height is optional.

External image:

```markdown
![alt text](https://example.com/image.png)
```

Standard Markdown image syntax also works with local vault files:

```markdown
![](photo.png)
```

### Video files

Embed a video file from the vault with `![[...]]`:

```markdown
![[demo.mp4]]
```

Supported formats: `.mp4`, `.avi`, `.mov`, `.mkv`, `.webm`, `.m4v`. The file renders as a `<video>` player with controls.

### Audio files

Embed an audio file the same way:

```markdown
![[recording.mp3]]
```

Supported formats: `.mp3`, `.wav`, `.ogg`, `.flac`, `.m4a`, `.aac`. The file renders as an `<audio>` player with controls. Readers can play, pause, and scrub without leaving the page.

Standard Markdown image syntax also triggers the audio player for audio files:

```markdown
![](recording.mp3)
```

### Document files (download links)

When you embed a document file, trip2g renders it as a download link rather than trying to display it inline:

```markdown
![[report.pdf]]
![[data.xlsx]]
```

Supported as download links: `.pdf`, `.doc`, `.docx`, `.xls`, `.xlsx`, `.ppt`, `.pptx`, `.txt`, `.rtf`, `.odt`, `.ods`, `.odp`, `.csv`, `.zip`, `.rar`, `.7z`.

The link shows the filename and downloads the file when clicked.

### Blockquotes

```markdown
> A blockquote.
> This continues on the next line.
>
> A new paragraph within the same quote.
```

### Code

Inline: `` `code` ``

Fenced block with syntax highlighting:

````markdown
```python
def greet(name):
    return f"Hello, {name}!"
```
````

Specify the language after the opening fence. Supported languages include `python`, `javascript`, `typescript`, `go`, `bash`, `json`, `yaml`, `sql`, and many others.

A block without a language tag renders as plain monospace text.

### Tables

```markdown
| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| A        | B        | C        |
| D        | E        | F        |
```

Column alignment:

```markdown
| Left     | Center   | Right    |
|:---------|:--------:|---------:|
| text     | text     | text     |
```

### Horizontal rule

```markdown
---
```

Three dashes produce a horizontal divider.

### Escaping

Prefix a special character with `\` to render it literally:

```markdown
\*not italic\*
\`not code\`
\[[not a wikilink\]]
```

### Frontmatter

A YAML block at the top of the file sets metadata for the note:

```markdown
---
title: "My note title"
free: true
---

Note body starts here.
```

Frontmatter controls how trip2g publishes the note: its URL, access level, scheduled Telegram posts, and more. See [[en/user/publishing]] for the full property reference.

---

## Rendering specifics

### What trip2g renders

The renderer is built on [Goldmark](https://github.com/yuin/goldmark) with the GitHub Flavored Markdown (GFM) extension. This means the following work out of the box:

- Standard Markdown: headings, bold, italic, blockquotes, fenced code blocks, horizontal rules
- GFM additions: tables, strikethrough (`~~`), task lists (`- [x]`), autolinks (bare URLs become clickable links)
- Wikilinks and embed syntax (`![[...]]`)
- Syntax highlighting in code blocks (via the Chroma library, server-side — no client-side JS needed)
- YouTube URLs in image syntax render as embedded players: `![](https://youtube.com/watch?v=...)`
- Audio files embed as `<audio controls>` players
- Video files embed as `<video controls>` players
- Document files embed as download links

### What is not rendered

- **Footnotes** — the `[^1]` / `[^1]: ...` syntax is not processed. The markers appear as literal text.
- **Math / LaTeX** — `$...$` and `$$...$$` blocks are not rendered. The content appears as plain text.
- **Smart quotes** — straight quotes `"..."` and `'...'` are not converted to curly quotes.
- **Raw HTML** — arbitrary inline HTML is sanitized. Only a small set of safe tags passes through (`<u>`, `<mark>`, `<sup>`, `<sub>`, `<strong>`, `<em>`).

### Wikilink resolution

trip2g resolves wikilinks the same way Obsidian does: globally, by filename, not by relative path. `[[note]]` matches any file named `note.md` anywhere in the vault. When names conflict, the file with the shorter path from the vault root wins.

Use an explicit path to target a specific file: `[[folder/note]]`.

### Asset URLs

Files you attach to a note (images, audio, video, documents) are stored in the cloud and served from a signed URL. URLs are refreshed automatically — you do not need to re-sync files to keep them accessible.

---

## Related

- [[en/user/publishing]] — frontmatter properties, access control, URL configuration
- [[en/user/mermaid]] — Mermaid diagram blocks
- [[en/user/youtube]] — YouTube embed syntax
- [[en/typography]] — full rendering demo page
