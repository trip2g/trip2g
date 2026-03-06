### 2026-03-06

04:24 The frontend is built inside the container. Locally I just cd ../mam && ln -s ../trip2g/ui trip2g, but for CI and other developers I automated this. More to come - still need to solve a caching issue.

06:00 Add scripts/gen_mol_deps.sh to generate mol deps component for faster Docker image builds.

08:22 More CI fixes.

12:57 Investigate a bug in multidomain logic. testdata/vault/multidomain/root.md should be rendered with full links to the main domain, but it doesn't. So hard do it without CC.
