#!/usr/bin/env bash
# check-doc-lang-links.sh — detect cross-language documentation link leaks.
#
# Scans docs/en/** and docs/ru/** for two kinds of leaks:
#   TYPE A  Explicit cross-language wikilinks or markdown links in the BODY
#           (e.g. [[en/user/foo]] in an ru/ file, or [[ru/user/foo]] in an en/ file).
#   TYPE B  Ambiguous bare wikilinks in ru/ files whose basename also exists under en/.
#           trip2g resolves [[basename]] by alphabetical full-path order, so en/ always
#           wins over ru/ — a bare [[foo]] in an ru page silently points to the EN page.
#
# EXCLUSIONS:
#   - The lang_redirect: frontmatter field (intentional cross-language pointer).
#   - Fenced code blocks (``` or ~~~).
#   - Inline code spans (`...`).
#   - External URLs (http:// / https://).
#
# OUTPUT:  file:line: <reason>  (one leak per line, to stdout)
# EXIT:    0 if no leaks found, 1 if any NEW (non-baselined) leaks found.
#
# BASELINE RATCHET:
#   If scripts/doc-lang-links-baseline.txt exists, any leak whose stable
#   signature appears in that file is "baselined" (grandfathered) and does NOT
#   cause a non-zero exit.  Only leaks absent from the baseline fail the check.
#
#   Stable signature format (one per line, tab-separated):
#     <relative-file-path>\t<leak-kind>\t<link-target>
#
#   Where:
#     <relative-file-path>  path relative to the repo root (docs/en/... or docs/ru/...)
#     <leak-kind>           the message category, one of:
#                             "explicit en->ru wikilink"
#                             "explicit en->ru markdown link"
#                             "explicit ru->en wikilink"
#                             "explicit ru->en markdown link"
#                             "ambiguous bare wikilink"
#     <link-target>         the raw link value as it appears in the source
#                           (wikilink including [[...]], markdown path, or bare basename)
#
#   Line numbers are intentionally excluded so that the signature remains stable
#   when surrounding content shifts.
#
# Usage: ./scripts/check-doc-lang-links.sh [docs_root]
#   docs_root defaults to docs/ relative to the repo root.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DOCS_ROOT="${1:-$REPO_ROOT/docs}"

EN_DIR="$DOCS_ROOT/en"
RU_DIR="$DOCS_ROOT/ru"

if [ ! -d "$EN_DIR" ] || [ ! -d "$RU_DIR" ]; then
  echo "error: expected $EN_DIR and $RU_DIR to exist" >&2
  exit 2
fi

BASELINE_FILE="$SCRIPT_DIR/doc-lang-links-baseline.txt"

# Build the set of basenames (without .md) that exist in BOTH en and ru trees.
# These are collision candidates — a bare wikilink to one of these in an ru file
# resolves alphabetically to the en/ copy.
COLL_FILE="$(mktemp)"
trap 'rm -f "$COLL_FILE"' EXIT

comm -12 \
  <(find "$EN_DIR" -name "*.md" | xargs -I{} basename {} .md | sort -u) \
  <(find "$RU_DIR" -name "*.md" | xargs -I{} basename {} .md | sort -u) \
  > "$COLL_FILE"

# AWK detector — processes one file at a time.
# Variables expected from the caller:
#   lang      = "en" or "ru"
#   coll_file = path to the collision-basename file
DETECTOR='
BEGIN {
    while ((getline line < coll_file) > 0) coll_map[line] = 1
    close(coll_file)
    other  = (lang == "en") ? "ru" : "en"
    in_front = 0
    in_fence = 0
    fence_re = ""
}

# --- Frontmatter handling ---
FNR == 1 {
    in_front = 0; in_fence = 0; fence_re = ""
    if (/^---[[:space:]]*$/) { in_front = 1; next }
}
in_front {
    if (/^---[[:space:]]*$/) in_front = 0
    next
}

# --- Fenced code block handling (``` or ~~~, 3+ chars) ---
/^(`{3,}|~{3,})/ {
    if (!in_fence) {
        in_fence = 1
        match($0, /^(`{3,}|~{3,})/)
        fence_re = "^" substr($0, 1, RLENGTH) "[[:space:]]*$"
    } else if ($0 ~ fence_re) {
        in_fence = 0
    }
    next
}
in_fence { next }

# --- Body line processing ---
{
    line = $0
    # Strip inline code spans so backtick-quoted wikilinks are ignored.
    while (match(line, /`+[^`]*`+/)) {
        line = substr(line, 1, RSTART-1) substr(line, RSTART+RLENGTH)
    }

    # TYPE A — explicit cross-language wikilinks: [[other_lang/...]]
    rest = line
    while (match(rest, /\[\[[^\]]+\]\]/)) {
        lnk   = substr(rest, RSTART, RLENGTH)
        inner = substr(lnk, 3, length(lnk)-4)
        # Remove alias suffix
        ai = index(inner, "|"); if (ai > 0) inner = substr(inner, 1, ai-1)
        if (substr(inner, 1, length(other)+1) == other "/") {
            printf "%s:%d: explicit %s->%s wikilink: %s\n", FILENAME, FNR, lang, other, lnk
        }
        rest = substr(rest, RSTART+RLENGTH)
    }

    # TYPE A — explicit cross-language markdown links: [text](path) with /en/ or /ru/
    rest = line
    while (match(rest, /\]\([^)]+\)/)) {
        url = substr(rest, RSTART+2, RLENGTH-3)
        if (url !~ /^https?:\/\//) {
            if (lang == "ru" && url ~ /(\/en\/|^\.\.\/en\/|^en\/)/) {
                printf "%s:%d: explicit ru->en markdown link: ](%s)\n", FILENAME, FNR, url
            } else if (lang == "en" && url ~ /(\/ru\/|^\.\.\/ru\/|^ru\/)/) {
                printf "%s:%d: explicit en->ru markdown link: ](%s)\n", FILENAME, FNR, url
            }
        }
        rest = substr(rest, RSTART+RLENGTH)
    }

    # TYPE B — ambiguous bare wikilinks (only in ru files; en < ru alphabetically)
    if (lang == "ru") {
        rest = line
        while (match(rest, /\[\[[^\]]+\]\]/)) {
            lnk   = substr(rest, RSTART, RLENGTH)
            inner = substr(lnk, 3, length(lnk)-4)
            ai = index(inner, "|"); if (ai > 0) inner = substr(inner, 1, ai-1)
            # Only bare names — no path separator
            if (index(inner, "/") == 0 && inner in coll_map) {
                printf "%s:%d: ambiguous bare wikilink %s (en/%s exists; en/ sorts before ru/)\n",
                    FILENAME, FNR, lnk, inner
            }
            rest = substr(rest, RSTART+RLENGTH)
        }
    }
}
'

# Collect all leak lines from both language trees.
ALL_LEAKS_FILE="$(mktemp)"
trap 'rm -f "$COLL_FILE" "$ALL_LEAKS_FILE"' EXIT

# Scan en/ files
while IFS= read -r -d '' f; do
    out=$(awk -v lang=en -v coll_file="$COLL_FILE" "$DETECTOR" "$f")
    if [ -n "$out" ]; then
        printf '%s\n' "$out" >> "$ALL_LEAKS_FILE"
    fi
done < <(find "$EN_DIR" -name "*.md" -print0 | sort -z)

# Scan ru/ files
while IFS= read -r -d '' f; do
    out=$(awk -v lang=ru -v coll_file="$COLL_FILE" "$DETECTOR" "$f")
    if [ -n "$out" ]; then
        printf '%s\n' "$out" >> "$ALL_LEAKS_FILE"
    fi
done < <(find "$RU_DIR" -name "*.md" -print0 | sort -z)

# If no leaks at all, exit clean.
if [ ! -s "$ALL_LEAKS_FILE" ]; then
    echo "check-doc-lang-links: no cross-language link leaks found."
    exit 0
fi

# Convert a raw leak line into a stable tab-separated signature:
#   <relative-file-path>\t<leak-kind>\t<link-target>
# Input format (from awk printf above):
#   /abs/path/to/file:LINE: explicit en->ru wikilink: [[...]]
#   /abs/path/to/file:LINE: explicit en->ru markdown link: ](...)
#   /abs/path/to/file:LINE: explicit ru->en wikilink: [[...]]
#   /abs/path/to/file:LINE: explicit ru->en markdown link: ](...)
#   /abs/path/to/file:LINE: ambiguous bare wikilink [[...]] (en/X exists; en/ sorts before ru/)
leak_to_sig() {
    local raw="$1"
    # Strip absolute path prefix to get relative path; strip :LINE: portion.
    # raw = /repo/root/docs/en/foo.md:42: <message>
    local relpath msg kind target
    # Remove repo root prefix (including trailing slash).
    relpath="${raw#"$REPO_ROOT/"}"
    # relpath = docs/en/foo.md:42: <message>
    # Strip line number: remove everything up to and including the first ': '
    # after the path+colon+digits portion.
    msg="${relpath#*:[0-9]*: }"
    # msg = explicit en->ru wikilink: [[...]]  OR  ambiguous bare wikilink [[...]] (...)
    # Strip the file:line prefix to get just the path.
    relpath="${relpath%%:*}"
    # relpath = docs/en/foo.md  (clean, no line number)

    # Parse kind and target from msg.
    if [[ "$msg" == "explicit "* ]]; then
        # "explicit en->ru wikilink: [[...]]"  or  "explicit ru->en markdown link: ](...)"
        # kind is everything before the last ': '
        kind="${msg%%: *([^\[]]*}"
        # Actually: kind = "explicit en->ru wikilink"  target = "[[...]]"
        # Simpler: split on ': ' — first segment is kind, rest is target.
        kind="${msg%%: *}"
        target="${msg#*: }"
    elif [[ "$msg" == "ambiguous bare wikilink "* ]]; then
        kind="ambiguous bare wikilink"
        # target: the [[...]] link, everything before the ' (' explanation.
        target="${msg#ambiguous bare wikilink }"
        target="${target%% (*}"
    else
        # Fallback: use full message as target.
        kind="unknown"
        target="$msg"
    fi

    printf '%s\t%s\t%s\n' "$relpath" "$kind" "$target"
}

# If no baseline file, behave as original: print all leaks, exit 1.
if [ ! -f "$BASELINE_FILE" ]; then
    cat "$ALL_LEAKS_FILE"
    echo "" >&2
    echo "check-doc-lang-links: leaks found (see above). Fix or add lang_redirect where intentional." >&2
    exit 1
fi

# Load baseline signatures into an associative array.
declare -A BASELINE
while IFS= read -r sig; do
    # Skip comment lines and blank lines.
    [[ "$sig" == "#"* ]] && continue
    [[ -z "$sig" ]] && continue
    BASELINE["$sig"]=1
done < "$BASELINE_FILE"

# Partition leaks into baselined vs new.
baselined=0
new_leaks=0
new_leak_lines=()

while IFS= read -r raw; do
    sig=$(leak_to_sig "$raw")
    if [[ -v BASELINE["$sig"] ]]; then
        (( baselined++ )) || true
    else
        (( new_leaks++ )) || true
        new_leak_lines+=("$raw")
    fi
done < "$ALL_LEAKS_FILE"

# Print new leaks (if any) to stdout.
if [ ${#new_leak_lines[@]} -gt 0 ]; then
    printf '%s\n' "${new_leak_lines[@]}"
fi

echo ""
echo "check-doc-lang-links: $baselined baselined, $new_leaks new."

if [ "$new_leaks" -gt 0 ]; then
    echo "Fix new leaks above, or add their signatures to scripts/doc-lang-links-baseline.txt if intentional." >&2
    exit 1
fi

exit 0
