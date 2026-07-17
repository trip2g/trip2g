---
title: Note graph
free: true
lang: en
lang_redirect: "[[ru/user/admin_graph]]"
---

**TL;DR.** Admin, Notes & Content, Note Graph shows your vault as a graph: notes are nodes, wikilinks are edges. The layout pulls notes of the same group together, and the `Group by:` field lets you cluster by subgraph (default) or by any frontmatter key, such as `lang` or `author`. To group whole folders without editing each note, add a frontmatter-patch file to the vault.

### What the graph shows

Every note is a node, every wikilink an edge with an arrow from the linking note to the linked one. Node styling encodes state at a glance:

| Marker | Meaning |
|--------|---------|
| Green square | Public note (`free: true`) |
| Diamond | Home page |
| Circle in a subgraph color | Note in that subgraph |
| Grey circle | No subgraph |

Colors come from the subgraph settings in Admin, Notes & Content, Subgraphs.

Drag a node and its new position is saved right away. That makes the graph a canvas you can arrange by hand, not just an auto layout.

### Header controls

| Control | What it does |
|---------|--------------|
| Auto layout | Recomputes all positions with the force layout. Notes of the same group attract each other, so clusters emerge. Not saved until you press Save positions |
| Save positions | Stores the current position of every node |
| Public outside | On the next Auto layout, pushes public (`free: true`) notes to an outer ring. Useful to see the public surface of the site around the private core |
| Group by: | The frontmatter field used for clustering. Default `subgraph` |

### Grouping by any frontmatter field

Type a frontmatter key into `Group by:` and press Auto layout. Notes with the same value of that key are pulled into one cluster:

- `subgraph` (the default) groups by [[en/user/subgraphs|subgraph]] labels, so access areas become visible clusters.
- `lang` splits the graph into language islands.
- Any custom key works: `author`, `status`, `type`, whatever your notes carry.

Notes without the field stay unclustered and settle by their links only. If no positions are saved yet, the graph re-clusters as soon as you change the field; with saved positions it keeps them until you press Auto layout, so an accidental edit never destroys a hand-made arrangement.

### Drive the grouping from the vault

Tagging hundreds of notes by hand is pointless. A [[en/user/frontmatter-patches|frontmatter patch]] note assigns the grouping key to whole folders at load time. The demo vault of this site does exactly that with a `_graph_groups.md` note at the vault root:

````markdown
---
type: frontmatter-patch
include:
  - demo/**
  - en/user/**
  - ru/user/**
  - en/hub/**
  - ru/hub/**
---

```jsonnet
if std.objectHas(meta, "subgraph") || std.objectHas(meta, "subgraphs") then {}
else if std.startsWith(path, "demo/") then { subgraph: "demo" }
else if std.startsWith(path, "en/user/") then { subgraph: "en_user" }
else if std.startsWith(path, "ru/user/") then { subgraph: "ru_user" }
else if std.startsWith(path, "en/hub/") || std.startsWith(path, "ru/hub/") then { subgraph: "hub" }
else {}
```
````

How it reads: `include` lists the folders the rule applies to, the jsonnet block computes the frontmatter to merge in. A note that already declares `subgraph` or `subgraphs` is left untouched, every other note gets one by its folder. Open the note graph on this site and you see four clusters: `demo`, `en_user`, `ru_user`, `hub`.

Two things worth knowing:

- A subgraph label by itself does not close a note. Access is a separate axis: `free: true` keeps the note public, and only a subgraph with configured access rules gates readers. See [[en/user/subgraphs]].
- The file name starts with `_`, so the patch note itself is hidden from search and listings.

The same pattern works for any grouping field, not just `subgraph`. Patch `{ team: "platform" }` onto a folder and group the graph by `team`.

### Related

- [[en/user/subgraphs]] tells what subgraphs are and how access works
- [[en/user/frontmatter-patches]] covers the full patch syntax: globs, exclude, priority, chaining
