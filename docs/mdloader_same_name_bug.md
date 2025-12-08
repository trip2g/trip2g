# Bug: Image with same name as note breaks page rendering

## Summary

When an image embed `![[name.png]]` has the same basename as an existing note `name.md`, the page fails to render with empty HTML.

## Root Cause

The `extractInLinks` function in `internal/mdloader/loader.go` was resolving image embeds as note links when their basenames matched.

**Example:**
- `software.md` contains `![[software.png]]`
- `extractInLinks` finds `software.md` by basename `software`
- Sets `link.Target = "/software"` (the note's permalink)
- During render, `renderEmbed` tries to embed `/software` (the note itself)
- The note has no HTML yet → `errNoHTML` → render fails

## Symptoms

- Page HTML is empty despite having content
- Debug logs show:
  ```
  DEBUG extractInLinks BEFORE: page=software.md target="software.png" embed=true
  DEBUG extractInLinks AFTER:  page=software.md target="/software" embed=true
  ```
- Error: `note has no HTML content`

## Fix

Added check at the start of `extractInLinks` to skip image/video links:

```go
// Skip image/video links - they should not be resolved as note links
if resolveAsImage(link) {
    return ast.WalkContinue, nil
}
```

## Test

`TestImageWithSameNameAsNote` in `loader_test.go` verifies that:
1. `![[software.png]]` renders as `<img src="software.png">`
2. `software.md` has non-empty HTML
3. Links to `[[software]]` from other pages work correctly

## Why the bug surfaced now

The bug in `extractInLinks` (resolving images as notes) existed for a long time. But it only broke rendering after commit `6baec8d`.

**Before commit `6baec8d`:**
- `extractInLinks` changed `![[software.png]]` → `/software`
- `ResolveWikilink` returned `/software` as-is (no version parameter for "live")
- `renderEmbed` found the note and embedded its content
- Result: Wrong behavior (showed note content instead of image), but page rendered

**After commit `6baec8d`:**
- Same transformation: `![[software.png]]` → `/software`
- `ResolveWikilink` checks `!resolveAsImage(n)` on current `n.Target` (which is `/software`, no `.png`)
- `resolveAsImage` returns `false` → version parameter added → `/software?version=latest`
- `renderEmbed` calls `removeVersion("/software?version=latest")` → `/software`
- But the note's HTML is empty at this point → `errNoHTML` → render fails

The key insight: `resolveAsImage(n)` checks the **current** `n.Target`, not the original. After `extractInLinks` corrupts the target, the image check no longer works.

## Timeline

- **Root cause introduced**: Unknown (extractInLinks always had this bug)
- **Triggered by**: Commit `6baec8d` (fix(mdloader): preserve slashes in versioned URL paths)
- **Found**: 2025-12-08, page `/software` was empty
- **Fixed**: Same day, added `resolveAsImage` check in `extractInLinks`

## Debugging Steps

1. Confirmed page exists in DB with content but HTML is empty
2. Used git bisect to find problematic commit
3. Added debug logging to `extractInLinks` (BEFORE/AFTER target)
4. Found `software.png` → `/software` transformation
5. Added fix and wrote regression test
