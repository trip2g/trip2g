---
# Expected patches: NONE (excluded by exclude_patterns)
# Expected meta: unchanged from frontmatter
# Tests: exclude patterns override include patterns
free: false
title: Excluded Patch Test
---

This page matches include pattern `patch_tests/*` BUT is excluded by `exclude_patterns: ["patch_tests/excluded.md"]`.

No patches should be applied. The `free` field should remain `false`.
