# Obsidian Link Resolution Algorithm

## Core Principle

**Obsidian uses GLOBAL link resolution across the entire vault, not relative paths.**

`[[note]]` always points to the same file regardless of where the link is located.

## Resolution Algorithm

### For `[[wikilink]]` format:

1. **Index all files** in the vault
2. **Search by filename** (case-insensitive)
3. **Find longest match** for the link text
4. **Priority on conflict**: Files closer to vault root take precedence
5. **Extension optional**: `.md` can be omitted for markdown files

### Shortest Path Mode (Default)

```
If filename is UNIQUE:
  Use only filename: [[note]]
  
If filename has DUPLICATES:
  Use path from root: [[folder/note]]
  Priority: shortest path from root
```

## Key Behavior

### Example: Duplicate Filenames

```
Structure:
/note.md          ← File A (root)
/folder/note.md   ← File B (subfolder)
/folder/source.md

From /folder/source.md:
  [[note]]              → /note.md (root, NOT local!) ⚠️
  [[folder/note]]       → /folder/note.md ✅
  [[./note]]            → /folder/note.md (explicit relative) ✅
```

**Critical**: `[[note]]` resolves to root even when called from `/folder/`!

## Path Syntax

| Syntax | Resolution | Example |
|--------|-----------|---------|
| `[[name]]` | Global search | `[[note]]` → finds `/note.md` or `/path/note.md` |
| `[[path/name]]` | Explicit path | `[[folder/note]]` → `/folder/note.md` |
| `[[./name]]` | Current folder | `[[./note]]` → relative to current file |
| `[[../name]]` | Parent folder | `[[../note]]` → one level up |
| `[[/name]]` | Absolute from root | `[[/note]]` → `/note.md` |

## Edge Cases

### 1. Multiple files with same name

```
/A.md
/folder/A.md

Anywhere: [[A]] → /A.md (root wins)
```

### 2. Assets (images)

```
/assets/photo.png
/project/photo.png

[[photo.png]] → /assets/photo.png (alphabetically first full path)
![[photo.png]] → embeds the same
```

### 3. Headers and blocks

```markdown
[[note#Header]]        → Link to heading
[[note#^block-id]]     → Link to block
[[#Header]]            → Link within same file
```

## Implementation Pseudocode

```javascript
function resolveWikilink(linkText, currentFile, vault) {
  // Handle explicit paths
  if (linkText.includes('/')) {
    if (linkText.startsWith('./')) {
      return resolveRelative(linkText, currentFile);
    }
    if (linkText.startsWith('/')) {
      return resolveAbsolute(linkText.slice(1), vault);
    }
    return findByPath(linkText, vault);
  }
  
  // Global search
  const matches = vault.findAllFilesByName(linkText);
  
  if (matches.length === 0) return null;
  if (matches.length === 1) return matches[0];
  
  // Multiple matches: prioritize by shortest path from root
  return matches.sort((a, b) => 
    a.path.split('/').length - b.path.split('/').length
  )[0];
}
```

## Test Cases

### Test 1: Unique Names
```
/unique.md
/folder/deep.md

[[unique]] → /unique.md ✅
[[deep]] → /folder/deep.md ✅
```

### Test 2: Duplicates (Priority)
```
/dup.md
/folder/dup.md
/folder/source.md

From /folder/source.md:
[[dup]] → /dup.md (root priority!) ⚠️
```

### Test 3: Explicit Paths
```
/dup.md
/folder/dup.md

[[folder/dup]] → /folder/dup.md ✅
[[./dup]] → depends on context ✅
[[/dup]] → /dup.md ✅
```

### Test 4: Assets
```
/img.png
/assets/img.png

![[img.png]] → first alphabetically by full path
![[assets/img.png]] → explicit ✅
```

## Official Sources

- [Forum: Shortest Path Explanation](https://forum.obsidian.md/t/settings-new-link-format-what-is-shortest-path-when-possible/6748)
- [Forum: Link Resolution Behavior](https://forum.obsidian.md/t/absolute-link-path-has-higher-precedence-than-relative-path/69542)
- [GitHub: obsidian-help](https://github.com/obsidianmd/obsidian-help/blob/master/en/Linking%20notes%20and%20files/Internal%20links.md)

## Implementation Notes

### For vault-to-website rendering:

1. **Index all files** on initial load
2. **Build lookup map**: `filename → [fullPath1, fullPath2, ...]`
3. **Resolve during render**: Apply algorithm above
4. **Warn on ambiguity**: Multiple matches with same name
5. **Handle broken links**: Missing files gracefully

### Performance

- Cache index between builds
- Use hash map for O(1) lookup by name
- Sort once per filename, not per link

### Edge Cases to Handle

- Files without extension
- Case sensitivity (Obsidian is case-insensitive)
- URL encoding in paths
- Special characters in filenames
- Circular references (shouldn't break)

---

**Key Takeaway**: This is NOT standard Markdown behavior. Obsidian prioritizes global uniqueness over relative proximity.
