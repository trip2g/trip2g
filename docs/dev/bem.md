# BEM Reference

Source: https://getbem.com/

## What is BEM?

BEM (Block, Element, Modifier) is a CSS naming methodology for organizing styles in larger projects. Key goals: development speed, maintainable code, and predictable specificity.

## Core Concepts

### Block
A standalone, self-meaningful entity that can exist independently.

- Lowercase Latin letters, digits, and dashes only
- Long names use dashes: `site-header`, `user-card`

```html
<div class="menu">...</div>
```
```css
.menu { }
```

### Element
A part of a block with no standalone meaning; semantically tied to its block.

- Separated from block by two underscores: `block__elem`
- Flat structure — never nest: `block__elem1` not `block__elem1__elem2`

```html
<div class="menu">
  <span class="menu__item">...</span>
</div>
```
```css
.menu__item { }   /* correct */
.menu .menu__item { }  /* WRONG — no descendant selectors */
div.menu__item { }     /* WRONG — no tag selectors */
```

### Modifier
A flag that changes appearance, behavior, or state of a block or element.

- Separated from block/element by two dashes: `block--mod`, `block__elem--mod`
- Always used together with the base class — never alone

```html
<div class="menu menu--hidden">...</div>
<div class="menu menu--theme-dark">...</div>
<span class="menu__item menu__item--active">...</span>
```
```css
.menu--hidden { display: none; }
.menu__item--active { font-weight: bold; }
```

## Rules

| Rule | Correct | Wrong |
|------|---------|-------|
| Class selectors only | `.block { }` | `div.block { }`, `#block { }` |
| No descendant selectors | `.block__elem { }` | `.block .block__elem { }` |
| Flat element nesting | `block__elem2` | `block__elem1__elem2` |
| Modifier requires base class | `class="b b--mod"` | `class="b--mod"` |
| Semantic modifier names | `--theme-dark` | `--border-bottom-5px` |

## Modifier affecting elements (ok)

When a block modifier should change an element:

```css
.block--xmas .block__elem { color: red; }
```

This is acceptable because elements don't make sense outside their block.

## Example

```html
<form class="form form--theme-xmas form--simple">
  <input class="form__input" type="text" />
  <input class="form__submit form__submit--disabled" type="submit" />
</form>
```

## In this project

Blocks used in `defaulttemplate`:

| Block | Elements | Modifiers |
|-------|----------|-----------|
| `layout` | `__main`, `__sidebar` | `--left`, `--right` (on sidebar) |
| `widget` | `__title`, `__content`, `__list`, `__data` | `--toc`, `--inlinks`, `--outlinks` |
| `magazine` | `__grid`, `__list` | — |
| `magazine-item` | `__title`, `__excerpt`, `__date`, `__image` | `--featured`, `--grid`, `--small`, `--list` |
| `content` | `__title`, `__body` | — |
| `prose` | — | — (typography wrapper) |
| `site-header` | `__content` | — |
| `site-footer` | `__content` | — |
