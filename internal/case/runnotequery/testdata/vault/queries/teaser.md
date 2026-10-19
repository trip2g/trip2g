---
type: note-query
free: true
all_notes: true
---
Open posts by title, and only the number of members-only ones.

```jsonnet
local blog = [n for n in notes if std.startsWith(n.path, 'blog/')];

{
  open: [n.title for n in blog if n.readable],
  locked: std.length([n for n in blog if !n.readable]),
  signedIn: currentUser.signedIn,
}
```
