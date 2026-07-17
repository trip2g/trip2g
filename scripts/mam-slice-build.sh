#!/usr/bin/env sh
# Build one or more mam module slices (e.g. trip2g/user/favoritenote) against
# THIS checkout's assets/ui - a full compile + type check of the module and its
# dependency closure. Uses a throwaway workspace of symlinks so the shared
# ~/projects2/mam workspace (whose trip2g symlink points at the main checkout)
# is never touched. Builder artifact dirs (-*/) land next to the sources and
# are gitignored (assets/ui/.gitignore).
#
# Usage: ./scripts/mam-slice-build.sh trip2g/user/favoritenote [more paths...]
# Env:   MAM_WORKSPACE - mam workspace to borrow (default ~/projects2/mam)
set -e

[ $# -ge 1 ] || { echo "usage: $0 <mam-module-path>..." >&2; exit 2; }

MAM=${MAM_WORKSPACE:-$HOME/projects2/mam}
REPO=$(cd "$(dirname "$0")/.." && pwd)
WS=$(mktemp -d)
trap 'rm -rf "$WS"' EXIT

for f in "$MAM"/* "$MAM"/.*; do
	b=$(basename "$f")
	case "$b" in trip2g|-|.|..|.git) continue;; esac
	ln -s "$f" "$WS/$b"
done
ln -s "$REPO/assets/ui" "$WS/trip2g"

# mam self-updates with `git pull origin master` on start; give the throwaway
# workspace a COPY of the real .git (1.5M) so the pull no-ops instead of trying
# a from-scratch merge over the untracked symlinks (which aborts the build)
cp -a "$MAM/.git" "$WS/.git"

cd "$WS"
node node_modules/.bin/mam "$@"
