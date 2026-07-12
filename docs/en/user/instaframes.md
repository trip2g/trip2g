---
free: true
title: Instagram carousels
home_position: 71
lang_redirect: "[[ru/user/instaframes]]"
---

`layout: trip2g/instaframe` turns a markdown file into a carousel of ready-to-download PNGs for Instagram and other social networks — no Canva, no designer, no export.

See carousels in action: [[instaframes/_index|Example gallery]].

## How it works

1. Place a file anywhere in your vault, add `layout: trip2g/instaframe` to its frontmatter.
2. Write an intro and slides, separated by `###` headings.
3. Open the page — the layout renders your markdown as slides and generates PNGs via `html2canvas`.
4. Download PNGs with the button on the page.

Edit the note and the carousel rebuilds on the next page load.

## Quick start

Create a file, for example `instaframes/my-carousel.md`:

```markdown
---
title: My carousel
layout: trip2g/instaframe
---

Carousel description — context for yourself, does not appear in slides.

### WHY DOES HOME COFFEE [TASTE BITTER]?

Slide text. 2–3 sentences maximum.

### NEXT [SLIDE]

Each `###` section becomes a separate slide.
```

Open the page in your vault — the carousel appears immediately.

## File structure

A carousel file has two parts:

**Intro** — everything before the first `###` heading. Rendered below the carousel as context; not included in slides.

**Slides** — each `###` section becomes one slide. The heading goes to the top of the slide; the body fills the main text area.

## Word highlight

`[WORD]` in a `###` heading gets a border. One highlighted word per heading.

Example:

```markdown
### WHY DOES HOME COFFEE [TASTE BITTER]?
```

`TASTE BITTER` renders inside a border; the rest of the heading stays plain.

## Downloading

Three buttons appear on the carousel page:

- Left — download the current slide
- Center — download all slides
- TXT — export the text of all slides

Files are named: `1-HEADING.png`, `2-HEADING.png`, and so on.

## Rules for good carousels

### First slide — the hook

The first slide stops the scroll. 3–5 words, a question or intrigue.

**Works:**
- Short question: `WHY DOES HOME COFFEE [TASTE BITTER]?`
- Unexpected claim: `IT'S NOT THE BEANS`

**Doesn't work:**
- Abstractions without a hook
- Long headings: 8+ words nobody reads while scrolling
- Clichés everyone already says

### Slide text

One idea per slide. Concrete examples, not abstractions. 2–3 sentences.

### What to avoid

1. Em-dashes (—) — use an en-dash (–) instead
2. Formulaic contrasts "not X, but Y"
3. Parallel constructions "X is more, Y is less"
4. Clipped sentences that say nothing: "Useless." "That's the signal."
5. Lecturing tone — share experience, don't instruct

### Carousel structure

- 8–12 slides
- Slide 1: hook (question or story)
- Slides 2–10: topic development with examples
- Second-to-last: what to do (a concrete step)
- Last: call to action

### Heading format

- 3–5 words in ALL CAPS
- One word in a border `[WORD]`
- A question or short statement

## Technical details

- Slide format: 4:5, export width 1080 px
- Fonts: Inter, Montserrat
- PNG rendering: html2canvas

## Branding

The `@trip2gcom` signature is hardcoded in `_layouts/trip2g/instaframe.html`.

To change the signature, edit that file in your vault. If a file `instaframes/_footer.md` exists alongside the carousel, the template renders its content below the carousel.

## After creating the carousel

1. Check the first slide — it should stop the scroll.
2. Read it aloud — does it sound like a real person?
3. Show it to someone — ask "would you swipe to the next one?"
