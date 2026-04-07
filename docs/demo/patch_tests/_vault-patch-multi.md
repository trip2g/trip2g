---
type: frontmatter-patch
include:
  - patch_tests/vault_target.md
priority: 10
---

Multiple patches in one file. Applied after _vault-patch-free.md (priority 10 > 0).

```jsonnet
{ multi_block: "first" }
```

```jsonnet
{ multi_block_second: true }
```
