---
title: Markdown
free: true
lang_redirect: "[[ru/Markdown]]"
---

Markdown is a lightweight way to format text using plain symbols. Every note in trip2g is a `.md` file. It opens in Obsidian, VS Code, Notepad, or any text editor. No app owns the format.

### Summary

Markdown uses `#` for headings, `**` for bold, `*` for italic, `-` for lists, and triple backticks for code blocks. Wikilinks (`[[Note Name]]`) are an Obsidian extension for connecting notes. Frontmatter — a YAML block between `---` markers at the top of a file — controls how trip2g publishes the note: its URL, visibility, and position in navigation.

### Headings

```
# Heading 1
## Heading 2
### Heading 3
```

Use one `#` for the page title. Subsections get `##` and `###`. Going deeper than three levels is usually a sign the content should be split into separate notes.

### Text formatting

```
**bold**
*italic*
~~strikethrough~~
`inline code`
```

### Links

Standard Markdown link:

```
[link text](https://example.com)
```

### Wikilinks

Wikilinks are an extension to standard Markdown, introduced by Obsidian and supported natively by trip2g. They're not part of the Markdown spec — they're a convention for linking notes inside a vault.

```
[[Note Name]]
[[Note Name|Display Text]]
```

The text inside `[[` and `]]` is the filename of the target note (without `.md`). No path needed — trip2g finds the file anywhere in the vault.

**Backlinks.** When note A links to note B, note B automatically knows about it. Open any note and see every note that references it. Connections you didn't plan become visible.

**Embed.** Prefix with `!` to embed a note or image inline:

```
![[image.png]]
![[Another Note]]
```

Standard Markdown doesn't have backlinks or vault-aware resolution. That's the gap wikilinks fill.

### Lists

Unordered:

```
- First item
- Second item
  - Nested item
```

Ordered:

```
1. First step
2. Second step
3. Third step
```

Task list:

```
- [ ] To do
- [x] Done
```

### Images

Standard Markdown:

```
![alt text](path/to/image.png)
```

Obsidian-style embed (recommended in Obsidian vaults):

```
![[image.png]]
```

Both work in trip2g. The Obsidian embed resolves the file from anywhere in your vault without a path.

### Code

Inline code uses single backticks:

```
Run `npm install` to install dependencies.
```

Fenced code block with optional language hint for syntax highlighting:

````
```javascript
function greet(name) {
  return `Hello, ${name}`;
}
```
````

### Blockquotes

```
> This is a blockquote.
> It can span multiple lines.
```

### Tables

```
| Column A | Column B | Column C |
| -------- | :------: | -------: |
| left     | center   | right    |
| aligned  | aligned  | aligned  |
```

Colons in the separator row control alignment: left (default), center (`:---:`), right (`---:`).

### Frontmatter

Frontmatter is a YAML block at the very top of a Markdown file, enclosed by `---` markers:

```yaml
---
title: "My Note"
slug: my-note
free: true
---
```

Everything between the two `---` lines is metadata. It does not appear as visible content on the page. trip2g reads this block to know how to publish the note.

Common frontmatter fields trip2g recognizes:

| Field | Purpose |
| ----- | ------- |
| `title` | Page title shown in browser tab and site navigation |
| `slug` | URL path override — e.g. `slug: /docs/intro` |
| `free: true` | Makes the note publicly visible without a subscription |
| `home_position` | Controls sort order in magazine-style index pages |
| `lang_redirect` | Links to the equivalent page in another language |

If `free` is absent, the note is subscriber-only by default. If `slug` is absent, trip2g derives the URL from the file path.

### Further reading

[Complete Markdown cheatsheet](https://learnxinyminutes.com/markdown/) — learnxinyminutes.com
