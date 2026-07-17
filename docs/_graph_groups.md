---
type: frontmatter-patch
include:
  - demo/**
  - en/user/**
  - ru/user/**
  - en/hub/**
  - ru/hub/**
---

Assigns a `subgraph` to notes that do not declare one, so the admin note graph
clusters the vault by section: demo, en_user, ru_user, hub.

```jsonnet
if std.objectHas(meta, "subgraph") || std.objectHas(meta, "subgraphs") then {}
else if std.startsWith(path, "demo/") then { subgraph: "demo" }
else if std.startsWith(path, "en/user/") then { subgraph: "en_user" }
else if std.startsWith(path, "ru/user/") then { subgraph: "ru_user" }
else if std.startsWith(path, "en/hub/") || std.startsWith(path, "ru/hub/") then { subgraph: "hub" }
else {}
```
