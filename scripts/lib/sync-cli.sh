# Shared sync-CLI invocation for scripts/test-sync-cli.sh and scripts/test-e2e.sh.
# Callers set SYNC_CLI_ENTRY to the path of obsidian-sync's cmd.ts.

# docs/demo ships _layouts/broken-layout.html with a deliberate Jet parse error.
# It has to stay in the vault: e2e/vault.spec.js asserts both the guest fallback
# and the admin "Layout Error: broken-layout" message. The server reports layout
# warnings for the whole knowledge base rather than just the files being pushed,
# so once that layout is on the server EVERY later sync reports it — even a
# one-file push from an unrelated directory. Since #260 the CLI exits non-zero
# on any CRITICAL, so that one fixture is tolerated below and nothing else is.
#
# Only layouts are listed: CRITICAL is produced solely by the layout loader, and
# layouts never register as notes, so broken_layout_test.md itself can never own
# one.
EXPECTED_CRITICAL_PATHS='_layouts/broken-layout.html'

# Reads CLI output on stdin. Succeeds only when at least one CRITICAL was
# reported and every one of them belongs to an expected fixture path.
only_expected_criticals() {
    sed 's/\x1b\[[0-9;]*m//g' | awk -v expected="$EXPECTED_CRITICAL_PATHS" '
        BEGIN { split(expected, e, " "); for (i in e) ok[e[i]] = 1 }
        /^  [^ ]/ { path = $0; sub(/^  /, "", path) }
        /^    \[CRITICAL\]/ { seen++; if (!(path in ok)) unexpected++ }
        END { exit (seen > 0 && unexpected == 0) ? 0 : 1 }
    '
}

# Every sync of the demo vault goes through here. Runs the CLI, prints its output
# and returns its exit code — except when the only CRITICAL warnings are the
# expected fixture. Calling cmd.ts directly bypasses that and reintroduces the
# abort, so don't.
run_sync_cli() {
    local output exit_code=0
    output=$(npx tsx "${SYNC_CLI_ENTRY:?SYNC_CLI_ENTRY is not set}" "$@" 2>&1) || exit_code=$?
    printf '%s\n' "$output"
    if [ "$exit_code" -ne 0 ] && printf '%s\n' "$output" | only_expected_criticals; then
        echo "ℹ Tolerated expected CRITICAL from the deliberately broken demo layout"
        exit_code=0
    fi
    return $exit_code
}

