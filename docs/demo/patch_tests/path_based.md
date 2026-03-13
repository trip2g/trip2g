---
# Expected patches: "Path-based logic" (priority 0)
# Expected meta: { patch_applied: true, free: true }
# Tests: if std.startsWith(path, "patch_tests/") then { patch_applied: true } else {}
free: true
layout: meta_inspector
title: Path-Based Logic Test
---

This page tests jsonnet logic based on the path variable.

Since path is "patch_tests/path_based.md", patch should add patch_applied: true.
