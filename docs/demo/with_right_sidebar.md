---
title: "All sidebar widgets demo"
left_sidebar:
  - TOC
  - inlinks
right_sidebar:
  - similar
  - outlinks
content:
  - selfcontent
free: true
---

## Introduction

This page demonstrates all available sidebar widgets:

- **Left sidebar**: TOC + Backlinks (inlinks)
- **Right sidebar**: Similar notes + Outlinks

## Widget reference

### TOC

Renders an interactive table of contents from the headings in the current note.

```yaml
left_sidebar:
  - TOC
```

### Backlinks (inlinks)

Shows notes that link to this page.

```yaml
left_sidebar:
  - inlinks
```

Alias: `backlinks`.

### Similar notes

Shows semantically similar notes using vector (chunk-level) embeddings. Requires vector search to be enabled in site config.

```yaml
right_sidebar:
  - similar
```

### Outlinks

Reserved — shows links from this note to other notes. Not yet implemented.

```yaml
right_sidebar:
  - outlinks
```

## Content widgets

Sidebars can also embed arbitrary notes by wikilink or file path:

```yaml
left_sidebar:
  - "[[My Navigation]]"
  - docs/nav.md
```

## Combining widgets

Any widget can appear in either sidebar, and multiple widgets can be listed:

```yaml
left_sidebar:
  - TOC
  - inlinks
right_sidebar:
  - similar
```

## Disabling sidebars

```yaml
left_sidebar: false
right_sidebar: false
```

## Section 1

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.

### Subsection 1.1

Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Subsection 1.2

Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.

## Section 2

Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.

### Subsection 2.1

Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium.

## Conclusion

Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit.
