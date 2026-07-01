# How to cut a release

**TL;DR:** pick the next version **from the changelog line** (not the git tags, which lagged), add a bilingual changelog entry, then push a `vX.Y.Z` tag. The tag fires two workflows: `docker.yml` (pushes the image to ghcr) and `release-binaries.yml` (builds the cross-platform binaries and creates the GitHub Release). Finish by setting the release notes.

## Versioning: the changelog is the source of truth

Git tags fell behind the real version history (tags stopped at `v0.5.0` while the changelog moved on to `v0.7.1`). **Always read `docs/en/changelog.md` for the current version and bump from there**, then confirm the number is not already used:

```sh
grep -E '^## v' docs/en/changelog.md | head -3   # current changelog version
git tag -l 'v*' --sort=-v:refname | head -3       # existing git tags
```

Reusing a version that already has a changelog entry creates a tag/changelog mismatch. (This happened once: a `v0.6.0` tag was pushed while the changelog already had a `v0.6.0` from a different set of changes; it had to be deleted and re-cut.)

Patch (`x.y.Z`) for tooling and small operator-facing changes, minor (`x.Y.0`) for notable features.

## 1. Pre-flight

- `main` is green: the `lint`, `test`, and `Docker` CI runs on the latest `main` commit all pass. The `lint` job includes `trip2g lint docs`, so the docs must be clean.
- You are on an up-to-date `main` (`git checkout main && git pull`).

## 2. Changelog (bilingual)

Add a `## vX.Y.Z (YYYY-MM-DD)` entry to the TOP of both `docs/en/changelog.md` and `docs/ru/changelog.md`, above the previous entry. Match the existing format: one `### Section` per change, each with `- **What.** / **Why.** / **How.**` bullets (RU: `**Что.** / **Зачем.** / **Как.**`).

Scope the entry to everything merged since the last changelog version. List merges to check coverage:

```sh
git log --merges --since=<last-release-date> --pretty='%h %s' origin/main
```

Rules for the entry:
- Cover only real, user- or operator-facing changes. Verify each claim against the actual diff (do not describe what you assumed shipped).
- Run the humanizer over it: no em/en dashes, no AI-vocab, straight quotes. Keep it technical and neutral.
- Link guides with same-language wikilinks (`[[en/...]]` in the EN file, `[[ru/...]]` in the RU file). Never link across languages.

Land the changelog on `main` (branch, PR, merge) so the tag you cut includes it.

## 3. Release notes

Write a short GitHub Release description (condensed from the changelog: a one-line intro, one bullet per change, and an Install section). Keep it release-notes length, not the full changelog. Save it to a file for step 5, e.g. `release-vX.Y.Z.md`.

The binary asset names include the version: `trip2g_vX.Y.Z_<os>_<arch>.tar.gz` (`.zip` for Windows), each with a `.sha256`. Make the Install `curl` URLs match exactly.

## 4. Tag and push

```sh
git checkout main && git pull
git tag -a vX.Y.Z -m "vX.Y.Z: <one-line summary>"
git push origin vX.Y.Z
```

The tag push triggers, in parallel:

| Workflow | Trigger | Produces |
|----------|---------|----------|
| `.github/workflows/docker.yml` | `push` tags `v*` | multi-arch image to `ghcr.io/trip2g/trip2g` (tags: `X.Y.Z`, `X.Y`, and `latest` only on `main`) |
| `.github/workflows/release-binaries.yml` | `push` tags `v*` | GitHub Release with `trip2g-server` + `fleet` archives (linux amd64/arm64, darwin amd64/arm64, windows amd64) and a `.sha256` per archive |

The release-binaries workflow builds the `$mol` frontend by building the Dockerfile `frontend` stage and extracting the bundles (they are gitignored build artifacts the embed needs), then cross-compiles with `CGO_ENABLED=0` (the app already runs on pure-Go modernc SQLite, so cross-compile is clean). `softprops/action-gh-release` creates the Release on first upload.

## 5. Finish and verify

```sh
gh run watch --exit-status                              # or: gh run list --limit 5
gh release edit vX.Y.Z --notes-file release-vX.Y.Z.md   # set the description
gh release view vX.Y.Z                                   # confirm all assets + .sha256 attached
```

Smoke-check one binary and the image:

```sh
curl -LO https://github.com/trip2g/trip2g/releases/download/vX.Y.Z/trip2g_vX.Y.Z_linux_amd64.tar.gz
curl -LO https://github.com/trip2g/trip2g/releases/download/vX.Y.Z/trip2g_vX.Y.Z_linux_amd64.tar.gz.sha256
sha256sum -c trip2g_vX.Y.Z_linux_amd64.tar.gz.sha256
docker pull ghcr.io/trip2g/trip2g:vX.Y.Z
```

## If something goes wrong

A release is recoverable until people depend on it. To redo a mis-cut tag:

```sh
gh run cancel <run-id>                    # stop the in-flight workflow runs
gh release delete vX.Y.Z --yes            # if the Release was already created
git push origin :refs/tags/vX.Y.Z         # delete the remote tag
git tag -d vX.Y.Z                         # delete the local tag
```

Then fix the cause and re-cut.

## Gotchas

- **Pre-push hook.** `git push` runs `make test` (`go test ./...`, no `-tags dev`), which fails in a fresh worktree because the frontend embed files are not built (`pattern ui/admin/-/web.js: no matching files found`). Push from a full checkout with built assets, or use `git push --no-verify` when the change is unrelated to Go (docs, workflows) and CI covers the tests.
- **Docker workflow runs on `main` pushes too**, not only tags, so `latest` tracks `main`.
- **e2e is skipped in CI** by design; it is not a release gate.
