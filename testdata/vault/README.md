# Test Vault

## Tests
1. [[unique]] - unique filename
2. [[folder/source]] - duplicate filename priority
3. [[projectA/README]] - multiple conflicts
4. [[img-test]] - image resolution
5. [[headers]] - headers/blocks

## Key Test: Duplicate Priority
From [[folder/source]]:
- `[[dup]]` → /dup.md (root!) ⚠️
- `[[folder/dup]]` → /folder/dup.md ✅

## Expected Behavior
Global resolution, root priority on conflicts.
