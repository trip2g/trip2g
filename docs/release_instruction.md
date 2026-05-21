# Release Instructions

How to cut a new trip2g release. Every release gets an annotated git tag and a
matching entry in [`docs/changelog.md`](./changelog.md).

## Versioning

[Semantic versioning](https://semver.org/) — `vMAJOR.MINOR.PATCH`.

| Bump | When |
|------|------|
| **MAJOR** | Breaking change for users, API consumers, or DB schema with no auto-migration. |
| **MINOR** | New user-visible feature (`feat:` commits) or notable behavior addition that is backward-compatible. |
| **PATCH** | Bug fixes only (`fix:`, `chore:`, lint, internal refactors). |

If the diff contains **any** `feat:` since the last tag → MINOR at minimum.

## Steps

### 1. Review what's in the release

```bash
LAST=$(git tag --sort=-v:refname | head -1)
git log "$LAST"..HEAD --oneline
```

Decide the next tag (e.g. `v0.4.0`). Group commits by `feat / fix / docs / chore`.

### 2. Update the changelog

Add a new section at the **top** of [`docs/changelog.md`](./changelog.md):

```markdown
## vX.Y.Z — YYYY-MM-DD

### Features
- ...

### Fixes
- ...

### Docs & chore
- ...
```

Keep entries user-facing: explain *what changed* and *why it matters*, not commit hashes.
Skip pure internal noise (CI tweaks, typo fixes) unless they affect a user.

### 3. Commit the changelog

```bash
git add docs/changelog.md
git commit -m "docs(changelog): vX.Y.Z"
```

### 4. Create an annotated tag

Annotated tags carry release notes inside the tag object itself.

```bash
git tag -a vX.Y.Z -m "$(cat <<'EOF'
vX.Y.Z

<paste the changelog section body here>
EOF
)"
```

Verify:

```bash
git show vX.Y.Z --stat | head -40
```

### 5. Push (manual)

Push is a deliberate, separate step — done by a human, never automatically.

```bash
git push origin main
git push origin vX.Y.Z
```

If you need to undo a local tag before pushing:

```bash
git tag -d vX.Y.Z
```

## Conventions

- Tag format: `vMAJOR.MINOR.PATCH`, lowercase `v`, no suffix.
- Pre-releases: `vX.Y.Z-rc.N` (rarely used).
- Hotfixes off a release: branch `hotfix/vX.Y.Z+1`, tag once merged.
- Never re-tag a published version. Bump PATCH instead.
- Don't squash the changelog commit into the feature commits — keep it as its own atomic commit so the tag points at it.
