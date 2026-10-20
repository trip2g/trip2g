---
title: "One folder, ten doors: how the positioning wheel is built"
description: "The trip2g positioning wheel is a folder of ordinary markdown notes and one layout. How it works, what the engine made easy, and where it pushed back."
free: true
lang: en
lang_redirect: "[[ru/thoughts/positioning-wheel]]"
---

*What this is about: the [[en/circle/_index|positioning wheel]] is a full-screen page where every vertex is one way to use trip2g. It has no special backend and no build step. It is a folder of notes and one layout file. Here is how it is put together, and where the layout system ran out.*

trip2g has no one-line pitch, [[en/thoughts/universal-tools|and I've stopped looking for one]]. So instead of a pitch I wanted a page that shows the doors side by side: a site from Obsidian, a Telegram blog, a paid expert base, memory for agents and the rest. You pick a vertex, the wheel turns it to the top, the center explains it.

## One note is one point

Every vertex is a markdown note in `en/circle/`, with its Russian twin in `ru/circle/`:

```yaml
title: "A Telegram blog from Obsidian"
label: "telegram blog"
order: 3
motion_visualization: channel
link: "[[en/user/telegram-blog-from-obsidian]]"
related:
  - "[[en/user/telegram]]"
lang: en
lang_redirect: "[[ru/circle/channel]]"
```

The body is the two to four sentences shown in the center. `link` is the main article, `related` the small links under it. `lang` and `lang_redirect` pair the twins, like every other bilingual note on this site.

Adding a point is adding a note. Removing one is deleting a file. The polygon takes as many sides as there are notes. Going from eight points to nine, and then to ten, needed no change to the wheel; only each new point's animation needed its own script line (more on that below).

## A layout that lists its own folder

The page note `en/circle/_index.md` says `layout: mesh/circle`. The layout takes its folder from the note itself and asks a note query for the points in it:

```jet
{{ points := noteQueryRows("queries/circle_points.md", map("dir", note.Dir())) }}
```

The query is a few lines of Jsonnet in a note: the notes directly in that folder that have a `label`, sorted by `order`, each returned as just its `version_id`. The page note has no `label`, so it is not a vertex. For every row the layout calls `row.Note()` and gets the point note back, with its title, text and frontmatter. Links are resolved by path, and each one prints with the target note's title, so retitling an article retitles its link on the wheel.

## Text for search engines, script for the wheel

All ten descriptions are in the HTML the server sends. The script only toggles which one is visible and moves the vertices. With JavaScript off the page is a plain stack of sections, which is also what a search engine reads. The point notes themselves are `noindex` and point their canonical URL at the wheel. Opening one, say `/en/circle/channel`, shows the wheel turned to that point.

## Visualizations are layout assets

Each point can name a small looping animation. Every animation is one file next to the layout, `circle/viz_channel.js` and so on, and each is a pure function of time:

```js
window.trip2gCircleViz.channel = function (ctx, t, w, h, colors) { … }
```

Because the drawing depends only on `t`, "reduced motion" is free: draw one frame and stop. The files ship with the vault sync like any other layout asset.

## Where it pushed back

- **Scripts can't be chosen by name at runtime.** `asset()` only registers static string literals. So the layout has one `<script>` line per animation. A new point that reuses an animation is just a note. A new animation means a new file and a new line in the layout.
- **No `dirname`.** At first I got the folder by splitting the path on `/` and slicing off the last piece. It worked, but it was three lines that a helper should be. Now there is one: `note.Dir()` gives `en/circle/`.
- **Template queries couldn't filter by a frontmatter field.** So I filtered inside the loop with `if`. `.Public()` would have been the natural filter, but it drops `noindex` notes, and the point notes are exactly that. We hadn't found `noteQuery` at first: a query note filters on any field. Even then it only gave back JSON, so the layout would have had to copy every field it shows into the query, and the rendered text can't be copied at all. Now a query can return only ids, and `noteQueryRows` hands each one back as a note through `.Note()`, with the reader's access checked again. The loop has no `if` left.
- **The landing's page shell didn't fit.** The mesh theme's shared shell hardcoded its language links to the home page, so this page wrote its own `<head>`. Now the shell builds canonical, hreflang and `og:url` from the page it is given and that page's language twins (`note.LangAlternative()`), and the wheel uses it. Its slim top bar is part of the page, so it stayed.
- **Captions inside animations are in the script.** The three-act company animation labels its acts in English or Russian by reading the page language. Those strings live in JavaScript, not in a note, which is the one place the "everything is a note" rule bends.
- **One CSS trap took a while to find.** A fade-in on the text panel used `transform`. With `animation-fill-mode: both`, the transform stayed applied and quietly became the positioning parent of the canvas inside it, so on phones the animation jumped into the middle of the text. Animating only opacity fixed it.

## Why bother

The page is a demonstration of the thing it describes. The positioning lives in notes I can edit in Obsidian, the server renders them, and the layout is a few hundred lines in the same vault. If I want an eleventh door tomorrow, it's one more note.
