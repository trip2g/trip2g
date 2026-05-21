# Release Instructions

How to cut a new trip2g release. Every release gets:

1. A bilingual entry in **both** [`docs/en/changelog.md`](./en/changelog.md) and [`docs/ru/changelog.md`](./ru/changelog.md) with **What / Why / How to use** for each user-facing change.
2. A separate atomic commit `docs(changelog): vX.Y.Z`.
3. An **annotated** git tag that carries the same release notes (English).

## Versioning

[Semantic versioning](https://semver.org/) — `vMAJOR.MINOR.PATCH`.

| Bump | When |
|------|------|
| **MAJOR** | Breaking change for users, API consumers, or DB schema without auto-migration. |
| **MINOR** | New user-visible feature (`feat:` commits) or notable backward-compatible behavior. |
| **PATCH** | Bug fixes only (`fix:`, `chore:`, lint, internal refactors). |

If the diff contains **any** `feat:` since the last tag → MINOR at minimum.

## Algorithm

### 1. List what's in the release

```bash
LAST=$(git tag --sort=-v:refname | head -1)
git log "$LAST"..HEAD --oneline
```

Group the commits mentally by `feat / fix / docs / chore`. Decide the next tag (`vX.Y.Z`).

### 2. For each user-facing item, dig up evidence

Don't trust commit subjects alone — find the actual artifact so the changelog entry is accurate and the reader can verify:

- **User docs**: `docs/en/user/*.md` and `docs/ru/user/*.md` — link the page.
- **Dev docs**: `docs/dev/*.md` — link if the feature has an internal spec or pattern doc.
- **Tests / e2e**: `internal/**/<feature>_test.go`, `e2e/<feature>/*` — gives you a real example payload or invocation.
- **Specs / plans**: `docs/superpowers/specs/`, `docs/superpowers/plans/` — useful as authoritative description.
- **CLI / scripts**: `cmd/<x>/main.go`, `scripts/<x>.py` — gives you the exact invocation.

Use `grep -l <symbol> docs/ -r` and `find -name '<feature>*'` to locate them.

### 3. Write the changelog entry (EN + RU)

Add a new section at the **top** of **both** [`docs/en/changelog.md`](./en/changelog.md) and [`docs/ru/changelog.md`](./ru/changelog.md). The Russian version is a straight translation of the English one — same structure, same links (adjusted to relative paths), same set of features. Per the project rule on bilingual docs, neither file is allowed to drift.

```markdown
## vX.Y.Z — YYYY-MM-DD

### <Feature name> — <one-line value prop>

- **What.** <One sentence describing the change.>
- **Why.** <Concrete reason the user should care.>
- **How.** <Doc links, mutation/CLI example, frontmatter key, admin path — enough to start using it.>
```

Rules of thumb:

- Order: features first (most impactful on top), then **Fixes**, then **Docs & chore**.
- Skip pure internal noise (CI tweaks, single-typo fixes) unless they affect a user.
- A backward-compatible automatic improvement (no flag, no config) still gets a section — explicitly state "automatic, no flags".
- Keep the prose user-facing. No commit hashes in the changelog; that's what git is for.
- One feature = one block. Don't bundle unrelated changes.

### 4. Commit the changelog

```bash
git add docs/en/changelog.md docs/ru/changelog.md
git commit -m "docs(changelog): vX.Y.Z"
```

Use specific paths in `git add` — never `git add -A` on a release commit (you'll sweep up unrelated untracked files).

### 5. Create an annotated tag

Annotated tags carry the release notes inside the tag object so `git show vX.Y.Z` produces a self-contained summary.

```bash
git tag -a vX.Y.Z -m "$(cat <<'EOF'
vX.Y.Z

<paste a compact version of the changelog section here —
 What/Why/How bullets collapsed to one line each is fine>
EOF
)"
```

Verify:

```bash
git show vX.Y.Z --stat | head -40
```

### 6. Push (manual)

Push is a deliberate, human-controlled step — never automated.

```bash
git push origin main
git push origin vX.Y.Z
```

Undo a local tag before pushing if you change your mind:

```bash
git tag -d vX.Y.Z
```

## Backfilling older releases

If a tag predates this changelog convention and you want to record it:

1. `git log <prev-tag>..<tag> --oneline` and look at the user-facing diff.
2. Add a `## vX.Y.Z — historical (backfill)` section in both `docs/en/changelog.md` and `docs/ru/changelog.md`.
3. Don't fabricate dates or sub-bullets you can't verify — when in doubt say "historical".
4. Backfill is one commit per batch, not one per tag.

## Conventions

- Tag format: `vMAJOR.MINOR.PATCH`, lowercase `v`, no suffix.
- Pre-releases: `vX.Y.Z-rc.N` (rarely used).
- Hotfixes off a release: branch `hotfix/vX.Y.Z+1`, tag once merged.
- Never re-tag a published version. Bump PATCH instead.
- Don't squash the changelog commit into feature commits — keep it atomic so the tag points at it.
