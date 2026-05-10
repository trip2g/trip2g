# Sticky Sidebars Research

## Problem

Sidebar can be taller than the viewport. Need it to stay accessible (TOC always visible) without ugly scrollbars.

## Verdict

**Pure CSS bidirectional sticky (sidebar scrolls with page in both directions, no scrollbar) does not exist.** Choose between:

1. CSS with hidden scrollbar (de-facto standard)
2. JS bidirectional

## CSS Solutions

### Option 1: sticky + max-height + hidden scrollbar (recommended CSS-only)

Used by Tailwind docs, Next.js docs, MDN.

```css
.sidebar {
  position: sticky;
  top: calc(var(--header-h) + 1rem);
  align-self: start;              /* CRITICAL — prevents height stretch */
  max-height: calc(100dvh - var(--header-h) - 2rem);
  overflow-y: auto;
  scrollbar-width: none;          /* Firefox */
  &::-webkit-scrollbar { display: none; } /* Chrome/Safari */
}
```

The sidebar scrolls internally when hovered (mouse wheel). No visible scrollbar.

### Option 2: h-screen + overflow-y: auto (Tailwind canonical)

Both sidebar and main scroll independently. No whole-page scroll.

```html
<div class="flex">
  <aside class="sticky top-0 h-screen overflow-y-auto w-64 shrink-0">
  <main class="flex-1 overflow-y-auto">
</div>
```

Not suitable for trip2g — we need whole-page scroll for the main content.

### Why `align-self: start` is critical

Default `align-self: stretch` makes the sidebar as tall as the content column. A stretched element never triggers sticky because its bottom edge never leaves the viewport. `align-self: start` collapses sidebar to natural height.

### Common sticky failure: `overflow: hidden` on ancestor

`overflow: hidden` on any ancestor creates a scroll container. The sticky element then references that container (which may have no scroll) instead of the viewport. **Fix:** use `overflow: clip` instead — same visual effect, does not create scroll container. ~95% browser support (2024).

```css
/* Instead of: */
.parent { overflow-x: hidden; }  /* BREAKS sticky on children */

/* Use: */
.parent { overflow-x: clip; }    /* safe */
```

## JS Solutions (for true bidirectional, no scrollbar)

### sticky-sidebar-v2

- Zero dependency, vanilla JS
- Handles sidebars taller than viewport
- Uses `will-change: transform` + CSS transforms

```javascript
var sidebar = new StickySidebar('#sidebar', { topSpacing: 20, bottomSpacing: 20 });
```

Source: https://blixhavn.github.io/sticky-sidebar-v2/

### Minimal vanilla JS algorithm

```javascript
let lastScrollY = window.scrollY;
let sidebarTop = 0;

window.addEventListener('scroll', () => {
  const delta = window.scrollY - lastScrollY;
  const sidebarRect = sidebar.getBoundingClientRect();

  if (delta > 0) {
    // scrolling down: clamp at bottom of viewport
    sidebarTop = Math.min(sidebarTop + delta, window.innerHeight - sidebarRect.height);
  } else {
    // scrolling up: clamp at top offset
    sidebarTop = Math.max(sidebarTop + delta, 0);
  }

  sidebar.style.top = sidebarTop + 'px';
  lastScrollY = window.scrollY;
});
```

## Sources

- [A Dynamically-Sized Sticky Sidebar — CSS-Tricks](https://css-tricks.com/a-dynamically-sized-sticky-sidebar-with-html-and-css/)
- [Getting stuck: all ways position:sticky can fail — Polypane](https://polypane.app/blog/getting-stuck-all-the-ways-position-sticky-can-fail/)
- [position: sticky not working? Try overflow: clip](https://www.terluinwebdesign.nl/en/css/position-sticky-not-working-try-overflow-clip-not-overflow-hidden/)
- [Scroll/Follow Sidebar, Multiple Techniques — CSS-Tricks](https://css-tricks.com/scrollfollow-sidebar/)
- [sticky-sidebar-v2](https://blixhavn.github.io/sticky-sidebar-v2/)
- [Tailwind fixed sidebar gist](https://gist.github.com/BjornDCode/5cb836a6b23638d6d02f5cb6ed59a04a)
