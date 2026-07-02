---
title: "Wikilink resolution"
description: "By default, a bare [[Name]] resolves to the shallowest match in the vault — Obsidian-compatible. Opt in to scoped mode for per-language resolution on multilingual sites."
free: true
slug: wikilink-resolution
lang_redirect: "[[ru/user/wikilink-resolution]]"
---

A bare `[[Name]]` resolves to the **shallowest match in the vault** — the file closest to the root, regardless of where the link appears. This matches Obsidian's own behavior: the same link always lands on the same file, no matter which note it comes from.

### Default behavior: global

trip2g picks the file whose path has the fewest segments. When two files share a name, the one closer to the root wins. Ties are broken alphabetically by full path.

Example:

```
en/user/templates.md
ru/user/templates.md
```

A bare `[[Templates]]` written anywhere in the vault resolves to whichever of these two sits shallowest. If both are at the same depth, the alphabetically earlier path wins — `en/user/templates.md` in this case. The result is the same regardless of which note contains the link.

This is what Obsidian does. Vaults that were built and linked in Obsidian work out of the box.

### Opt-in: scoped mode

For multilingual sites, you can switch to language-aware resolution. Set this in your site settings:

```
wikilink_resolution: scoped
```

In scoped mode, trip2g walks a three-step ladder and stops at the first match:

1. **Same folder.** A note in the same folder as the linking note wins.
2. **Same language.** Otherwise a note whose `lang` frontmatter (or top-level folder, `en/`, `ru/`) matches the linking note wins.
3. **Global shallowest.** Otherwise the note closest to the vault root wins.

Ties at any step are broken alphabetically by path.

With scoped mode, a bare `[[Templates]]` written in `en/user/getting-started.md` resolves to `en/user/templates.md`, and the same link written in `ru/user/Начало работы.md` resolves to `ru/user/templates.md`. You can write bare links without thinking about cross-language collisions.

Use scoped mode when:
- Your vault has parallel note structures in two or more languages.
- You want bare links to stay within the reader's language by default.
- You only need explicit paths when crossing languages on purpose.

### Forcing a target

Add a path to point at an exact file. This works the same in both modes.

```markdown
[[folder/Name]]
[[ru/user/Name]]
[[./Name]]
```

- `[[folder/Name]]` and `[[ru/user/Name]]` — path from the vault root.
- `[[./Name]]` — path relative to the linking note.

Example. From an English note you want to send the reader to the Russian version:

```markdown
See the [[ru/user/templates|Russian version]].
```

### Related

- [[en/user/markdown|Markdown syntax]] — wikilink and embed syntax
- [[en/user/multilingual|Multilingual sites]] — one vault, many languages
- [[en/user/publishing|Publishing notes]] — frontmatter properties reference
