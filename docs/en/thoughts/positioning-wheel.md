---
title: "One folder, nine doors: how the positioning wheel is built"
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
order: 2
motion_visualization: channel
link: "[[en/user/telegram-blog-from-obsidian]]"
related:
  - "[[en/user/telegram]]"
lang: en
lang_redirect: "[[ru/circle/channel]]"
```

The body is the two to four sentences shown in the center. `link` is the main article, `related` the small links under it. `lang` and `lang_redirect` pair the twins, like every other bilingual note on this site.

Adding a point is adding a note. Removing one is deleting a file. The polygon takes as many sides as there are notes. Going from eight points to nine needed no change to the wheel; only the new point's animation needed its own script line (more on that below).

## A layout that lists its own folder

The page note `en/circle/_index.md` says `layout: mesh/circle`. The layout finds its folder from the note's own path and asks for every note in it:

```jet
{{ points := nvs.ByGlob(dir + "*.md").SortByMeta("order").SortBy("Title").All() }}
```

Then it skips anything without a `label`, so the page note itself is not a vertex. Links are resolved by path, and each one prints with the target note's title, so retitling an article retitles its link on the wheel.

## Text for search engines, script for the wheel

All nine descriptions are in the HTML the server sends. The script only toggles which one is visible and moves the vertices. With JavaScript off the page is a plain stack of sections, which is also what a search engine reads. The point notes themselves are `noindex` and point their canonical URL at the wheel. Opening one, say `/en/circle/channel`, shows the wheel turned to that point.

## Visualizations are layout assets

Each point can name a small looping animation. Every animation is one file next to the layout, `circle/viz_channel.js` and so on, and each is a pure function of time:

```js
window.trip2gCircleViz.channel = function (ctx, t, w, h, colors) { … }
```

Because the drawing depends only on `t`, "reduced motion" is free: draw one frame and stop. The files ship with the vault sync like any other layout asset.

## Where it pushed back

- **Scripts can't be chosen by name at runtime.** `asset()` only registers static string literals. So the layout has one `<script>` line per animation. A new point that reuses an animation is just a note. A new animation means a new file and a new line in the layout.
- **No `dirname`.** I got the folder by splitting the path on `/` and slicing off the last piece. It works. It's three lines that a helper should be.
- **Queries can't filter by a frontmatter field.** I filter inside the loop with `if`. `.Public()` would have been the natural filter, but it drops `noindex` notes, and the point notes are exactly that.
- **The landing's page shell didn't fit.** The mesh theme's shared shell hardcodes its language links to the home page, so this page writes its own `<head>`.
- **Captions inside animations are in the script.** The three-act company animation labels its acts in English or Russian by reading the page language. Those strings live in JavaScript, not in a note, which is the one place the "everything is a note" rule bends.
- **One CSS trap took a while to find.** A fade-in on the text panel used `transform`. With `animation-fill-mode: both`, the transform stayed applied and quietly became the positioning parent of the canvas inside it, so on phones the animation jumped into the middle of the text. Animating only opacity fixed it.

## Why bother

The page is a demonstration of the thing it describes. The positioning lives in notes I can edit in Obsidian, the server renders them, and the layout is a few hundred lines in the same vault. If I want a tenth door tomorrow, it's one more note.
