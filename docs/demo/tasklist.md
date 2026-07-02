---
free: true
title: Task List Checkboxes
description: Interactive GFM task checkboxes — admins can toggle them right on the page
---

# Task List Checkboxes

Admins see these checkboxes as clickable. A click flips the marker in the note
source (`[ ]` ↔ `[x]`) and saves through `updateNotes`. Non-admins see the same
list read-only. Open this page while logged in as an admin to try it.

## Unique lines — patched in place

Each of these lines is unique in the note, so a toggle rewrites just that line
via the `patch` (find/replace) path.

- [ ] Draft the launch announcement
- [x] Set up the demo vault
- [ ] Record the walkthrough video
- [ ] Review the pricing page copy
- [ ] Schedule the newsletter

## Ordered task list

1. [x] Open the repo
2. [ ] Read the contributing guide
3. [ ] Build the project locally

## Nested and quoted tasks

Marker toggling ignores list/quote prefixes, so indented and quoted tasks work
the same way.

- [ ] Parent task
  - [x] Nested done
  - [ ] Nested pending

> [!note]
> - [ ] A task inside a callout

## Duplicate lines — whole-note compare-and-swap

These two items are byte-identical. A duplicate line can't be found uniquely,
so toggling either one falls back to rewriting the whole note under an
`expectedHash` guard (optimistic concurrency). Use these to exercise that path.

- [ ] Follow up with the customer
- [ ] Follow up with the customer
