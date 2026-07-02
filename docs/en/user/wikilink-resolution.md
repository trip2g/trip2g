---
title: "Wikilink resolution"
description: "How a bare [[Name]] link finds its target on a multilingual site: same folder, then same language, then the shortest path. Plus how to force an exact target."
free: true
slug: wikilink-resolution
lang_redirect: "[[ru/user/wikilink-resolution]]"
---

A bare `[[Name]]` link finds its target in this order: **the same folder as the linking note, then the same language, then the shortest path from the vault root.** The choice is deterministic, so the same link always lands on the same page.

This matters when two notes share a name. On a bilingual site you often have `en/user/templates.md` and `ru/user/templates.md`. A bare `[[Templates]]` should go to the reader's language, not jump across it.

### How resolution works

trip2g walks three steps and stops at the first match:

1. **Same folder.** A note in the same folder as the linking note wins.
2. **Same language.** Otherwise a note in the same language wins. Language comes from the `lang` frontmatter property, or from the top-level folder (`en/`, `ru/`).
3. **Shortest path.** Otherwise the note closest to the vault root wins.

When several notes tie on a step, the one whose path comes first alphabetically wins. Nothing is left to chance.

Example. Both pages exist:

```
en/user/templates.md
ru/user/templates.md
```

A bare `[[Templates]]` written in `en/user/getting-started.md` resolves to `en/user/templates.md`. The same bare `[[Templates]]` written in `ru/user/Начало работы.md` resolves to `ru/user/templates.md`. Each link stays in its language.

### Forcing a target

Add a path to point at an exact file. This overrides the three steps above.

```markdown
[[folder/Name]]
[[ru/user/Name]]
[[./Name]]
```

- `[[folder/Name]]` and `[[ru/user/Name]]` — a path from the vault root.
- `[[./Name]]` — a path relative to the linking note.

Use this when you deliberately want to link across languages, or when a bare name is too ambiguous to trust.

Example. From an English note you want to send the reader to the Russian original:

```markdown
See the [[ru/user/templates|Russian version]].
```

### Multilingual tip

You no longer need to prefix every bare link. Same language wins on its own.

Before, `[[Templates]]` was ambiguous. Both `en/user/templates.md` and `ru/user/templates.md` sat at the same depth, so the link landed on whichever one the system happened to pick. The result could differ between page loads.

Now the linking note's language decides. A bare `[[Templates]]` in an English note goes to the English page; the same link in a Russian note goes to the Russian page. Write bare links and let the folder do the work.

Reach for an explicit path only when you want to cross languages on purpose, like `[[ru/user/templates]]`.

### Legacy behavior

Some vaults were built around the old rule, where a bare link always went to the shortest path anywhere in the vault, regardless of language. To keep that behavior, set the site setting:

```
wikilink_resolution: global
```

With `global`, trip2g skips the same-folder and same-language steps and picks the shortest path from the root, exactly as before. Leave it unset (or set it to `scoped`) for the language-aware resolution described above.

### Related

- [[en/user/markdown|Markdown syntax]] — wikilink and embed syntax
- [[en/user/multilingual|Multilingual sites]] — one vault, many languages
- [[en/user/publishing|Publishing notes]] — frontmatter properties reference
