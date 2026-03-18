---
title: Typography & Markdown
free: true
description: A demonstration of all supported Markdown elements
---

This page shows every Markdown element rendered by trip2g. Use it as a reference when writing notes.

---

## Headings

# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6

---

## Paragraphs

A paragraph is a block of text separated by blank lines. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.

A second paragraph follows after an empty line. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

---

## Emphasis

**Bold text** is wrapped in double asterisks.

*Italic text* is wrapped in single asterisks.

***Bold and italic*** uses triple asterisks.

~~Strikethrough~~ uses double tildes.

---

## Blockquotes

> A simple blockquote.

> Blockquotes can span multiple lines.
> This is the continuation of the same quote.
>
> A new paragraph inside the quote.

> **Nested quotes:**
>> A quote inside a quote.

---

## Lists

### Unordered

- Item one
- Item two
  - Nested item
  - Another nested item
    - Deeply nested
- Item three

### Ordered

1. First item
2. Second item
3. Third item
   1. Sub-item
   2. Sub-item

### Task list

- [x] Completed task
- [x] Also done
- [ ] Not yet done
- [ ] Another pending item

---

## Code

### Inline code

Use `inline code` for short snippets, variable names like `myVar`, or commands like `git status`.

### Code blocks

```python
def greet(name: str) -> str:
    return f"Hello, {name}!"

print(greet("world"))
```

```javascript
const add = (a, b) => a + b;

console.log(add(2, 3)); // 5
```

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```

```bash
# Install dependencies
npm install

# Start dev server
npm run dev
```

```json
{
  "name": "my-site",
  "version": "1.0.0",
  "description": "Published with trip2g"
}
```

```
Plain code block without syntax highlighting.
Useful for configuration files, logs, etc.
```

---

## Tables

| Name       | Type     | Required | Description              |
|------------|----------|----------|--------------------------|
| `title`    | string   | No       | Page title               |
| `free`     | boolean  | No       | Visible without login    |
| `date`     | date     | No       | Publication date         |
| `tags`     | list     | No       | Tags for filtering       |

Alignment:

| Left aligned | Centered | Right aligned |
|:-------------|:--------:|--------------:|
| Cell         |   Cell   |          Cell |
| Longer value |   Mid    |         12345 |

---

## Links

Internal wikilink: [[en/user/getting-started|Getting started]]

Internal with alias: [[en/user/publishing|How to publish notes]]

External link: [Obsidian](https://obsidian.md)

Bare URL: https://trip2g.com

---

## Video

YouTube videos are embedded with a standard image link:

![](https://www.youtube.com/watch?v=dQw4w9WgXcQ)

---

## Images

Local image from the vault:

![[demo/assets/asset0.png]]

Image with alt text:

![Format examples]([[demo/format.jpg]])

External image:

![Obsidian obsidian](https://upload.wikimedia.org/wikipedia/commons/1/17/Lipari-Obsidienne_%285%29.jpg)

---

## Horizontal rules

Three dashes produce a horizontal rule:

---

Three asterisks also work:

***

---

## Footnotes

Here is a sentence with a footnote.[^1]

Another sentence references a second note.[^note]

[^1]: This is the first footnote.
[^note]: Footnotes can have longer identifiers.

---

## HTML (inline)

Some <strong>bold</strong> and <em>italic</em> via raw HTML.

Text with a <mark>highlight</mark> using the mark element.

Superscript: H<sup>2</sup>O. Subscript: CO<sub>2</sub>.

---

## Escaping

Use a backslash to escape Markdown characters:

\*not italic\* \*\*not bold\*\* \`not code\`
