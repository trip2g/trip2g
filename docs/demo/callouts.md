---
free: true
title: Obsidian Callouts
---

# Callouts

Obsidian-style callouts (`> [!type]`) are rendered as styled blocks with an icon and title.

## Standard Types

> [!note]
> A plain note callout. Uses the default capitalized type name as the title.

> [!tip]
> A helpful tip. Great for highlighting advice or shortcuts.

> [!warning]
> Something to watch out for. Proceed with caution.

> [!danger]
> A critical warning. This action is destructive.

## Custom Title

> [!info] Getting Started
> When you provide a custom title after the type, it overrides the default.

## Foldable: Collapsed

> [!tip]- How to fold a callout
> Add a `-` after the type to make the callout collapsed by default. Click to expand.

## Foldable: Expanded

> [!faq]+ Is this open by default?
> Yes! A `+` marker means the callout starts expanded but can be collapsed.

## Unknown / Custom Type

> [!my-custom-type]
> Unrecognised types still render as a callout block with a fallback icon.

## Nested Markdown

> [!note]
> Callout bodies support full Markdown:
>
> - First item
> - **Bold text** in a list
> - Another item
