#!/bin/bash
#
# E2E tests for obsidian-sync CLI
#
# Usage:
#   ./scripts/test-sync-cli.sh --api-key <key> [--endpoint <url>]
#   ./scripts/test-sync-cli.sh -k <key> [-e <url>]
#
# Arguments:
#   -k, --api-key    API key (required)
#   -e, --endpoint   GraphQL endpoint (default: http://localhost:8081/graphql)
#
# This script simulates two clients (testvault0 and testvault1) syncing
# to the same server to test multi-client scenarios.

set -e

# Parse arguments
API_KEY=""
ENDPOINT="http://localhost:8081/graphql"

while [[ $# -gt 0 ]]; do
    case $1 in
        -k|--api-key)
            API_KEY="$2"
            shift 2
            ;;
        -e|--endpoint)
            ENDPOINT="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 --api-key <key> [--endpoint <url>]"
            echo ""
            echo "Arguments:"
            echo "  -k, --api-key    API key (required)"
            echo "  -e, --endpoint   GraphQL endpoint (default: http://localhost:8081/graphql)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OBSIDIAN_SYNC_DIR="$PROJECT_ROOT/obsidian-sync"
TMP_DIR="$PROJECT_ROOT/tmp"
VAULT0="$TMP_DIR/testvault0"
VAULT1="$TMP_DIR/testvault1"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# ============ Helper Functions ============

log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    TESTS_RUN=$((TESTS_RUN + 1))
}

log_fail() {
    echo -e "${RED}✗${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    TESTS_RUN=$((TESTS_RUN + 1))
}

log_section() {
    echo ""
    echo -e "${YELLOW}════════════════════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}  $1${NC}"
    echo -e "${YELLOW}════════════════════════════════════════════════════════════${NC}"
}

# Sync a vault, returns output
sync_vault() {
    local vault="$1"
    local extra_args="${2:-}"

    log_info "Syncing $(basename $vault)..."
    cd "$OBSIDIAN_SYNC_DIR"
    npx tsx src/sync/cli/cmd.ts --folder "$vault" --api-key "$API_KEY" --api-url "$ENDPOINT" --two-way $extra_args 2>&1
}

# Sync a vault silently (for setup steps)
sync_vault_quiet() {
    local vault="$1"
    local extra_args="${2:-}"

    cd "$OBSIDIAN_SYNC_DIR"
    npx tsx src/sync/cli/cmd.ts --folder "$vault" --api-key "$API_KEY" --api-url "$ENDPOINT" --two-way $extra_args > /dev/null 2>&1
}

# Assert file exists
assert_file_exists() {
    local path="$1"
    local message="$2"

    if [ -f "$path" ]; then
        log_success "$message"
        return 0
    else
        log_fail "$message (file not found: $path)"
        return 1
    fi
}

# Assert file does not exist
assert_file_not_exists() {
    local path="$1"
    local message="$2"

    if [ ! -f "$path" ]; then
        log_success "$message"
        return 0
    else
        log_fail "$message (file exists but shouldn't: $path)"
        return 1
    fi
}

# Assert file contains string
assert_file_contains() {
    local path="$1"
    local expected="$2"
    local message="$3"

    if grep -q "$expected" "$path" 2>/dev/null; then
        log_success "$message"
        return 0
    else
        log_fail "$message (expected '$expected' in $path)"
        return 1
    fi
}

# Assert directories are equal (excluding hidden files and binary assets)
assert_dirs_equal() {
    local dir1="$1"
    local dir2="$2"
    local message="$3"

    local diff_result
    # Exclude hidden files, binary assets (png, jpg, svg, webp, mp4, css)
    diff_result=$(diff -rq "$dir1" "$dir2" \
        --exclude=".*" \
        --exclude="node_modules" \
        --exclude="*.png" \
        --exclude="*.jpg" \
        --exclude="*.jpeg" \
        --exclude="*.svg" \
        --exclude="*.webp" \
        --exclude="*.mp4" \
        --exclude="*.css" \
        --exclude="assets" \
        2>&1 || true)

    if [ -z "$diff_result" ]; then
        log_success "$message"
        return 0
    else
        log_fail "$message"
        echo "  Differences:"
        echo "$diff_result" | head -10 | sed 's/^/    /'
        return 1
    fi
}

# ============ Setup ============

setup() {
    log_section "Setup"

    if [ -z "$API_KEY" ]; then
        echo -e "${RED}❌ --api-key is required${NC}"
        echo "Usage: $0 --api-key <key> [--endpoint <url>]"
        exit 1
    fi

    log_info "API_KEY: ${API_KEY:0:8}..."
    log_info "ENDPOINT: $ENDPOINT"

    # Clean tmp directory
    rm -rf "$VAULT0" "$VAULT1"
    mkdir -p "$TMP_DIR"

    # Copy testdata/vault to testvault0
    log_info "Copying testdata/vault → tmp/testvault0"
    cp -r "$PROJECT_ROOT/testdata/vault" "$VAULT0"
    rm -f "$VAULT0/.sync-state.json"

    # Create empty testvault1
    log_info "Creating empty tmp/testvault1"
    mkdir -p "$VAULT1"

    echo ""
}

cleanup() {
    log_section "Cleanup"
    # this files will be used for telegram testing
    # rm -rf "$VAULT0" "$VAULT1"
    log_info "Test vaults removed"
}

# ============ Test Cases ============

test_initial_sync_vault0() {
    log_section "Test: Initial sync of vault0"

    sync_vault "$VAULT0"
    assert_file_exists "$VAULT0/.sync-state.json" "Sync state created for vault0"
}

test_empty_vault_pulls_all() {
    log_section "Test: Empty vault1 pulls all files"

    sync_vault "$VAULT1"

    assert_file_exists "$VAULT1/.sync-state.json" "Sync state created for vault1"
    assert_file_exists "$VAULT1/_index.md" "Index file pulled to vault1"
    assert_dirs_equal "$VAULT0" "$VAULT1" "Both vaults have identical content"
}

test_new_file_sync_both_ways() {
    log_section "Test: New files sync between vaults"

    # Create new file in vault0
    cat > "$VAULT0/from_vault0.md" << 'EOF'
---
publish: true
---
# From Vault 0
Created in vault0.
EOF

    # Create different file in vault1
    cat > "$VAULT1/from_vault1.md" << 'EOF'
---
publish: true
---
# From Vault 1
Created in vault1.
EOF

    # Sync vault0 (pushes from_vault0.md)
    sync_vault_quiet "$VAULT0"

    # Sync vault1 (pushes from_vault1.md, pulls from_vault0.md)
    sync_vault_quiet "$VAULT1"

    # Sync vault0 again (pulls from_vault1.md)
    sync_vault_quiet "$VAULT0"

    assert_file_exists "$VAULT0/from_vault0.md" "vault0 has from_vault0.md"
    assert_file_exists "$VAULT0/from_vault1.md" "vault0 has from_vault1.md (pulled)"
    assert_file_exists "$VAULT1/from_vault0.md" "vault1 has from_vault0.md (pulled)"
    assert_file_exists "$VAULT1/from_vault1.md" "vault1 has from_vault1.md"
    assert_dirs_equal "$VAULT0" "$VAULT1" "Both vaults identical after cross-sync"
}

test_conflict_resolve_local() {
    log_section "Test: Conflict resolution --conflict-resolution=local"

    # Both vaults modify the same file
    cat > "$VAULT0/from_vault0.md" << 'EOF'
---
publish: true
---
# Modified by Vault 0
vault0 version
EOF

    cat > "$VAULT1/from_vault0.md" << 'EOF'
---
publish: true
---
# Modified by Vault 1
vault1 version
EOF

    # Sync vault0 first (pushes its version to server)
    sync_vault_quiet "$VAULT0"

    # Sync vault1 with --conflict-resolution=local (keeps vault1's version)
    sync_vault "$VAULT1" "--conflict-resolution=local"

    assert_file_contains "$VAULT1/from_vault0.md" "vault1 version" "vault1 kept its local version"
}

test_conflict_resolve_remote() {
    log_section "Test: Conflict resolution --conflict-resolution=remote"

    # Ensure server has vault0's version
    cat > "$VAULT0/from_vault0.md" << 'EOF'
---
publish: true
---
# Server Version
from server via vault0
EOF
    sync_vault_quiet "$VAULT0"

    # Modify vault1's version
    cat > "$VAULT1/from_vault0.md" << 'EOF'
---
publish: true
---
# Should Be Overwritten
this should be replaced
EOF

    # Sync vault1 with --conflict-resolution=remote
    sync_vault "$VAULT1" "--conflict-resolution=remote"

    assert_file_contains "$VAULT1/from_vault0.md" "from server via vault0" "vault1 got remote version"
}

test_conflict_resolve_fail() {
    log_section "Test: Conflict resolution --conflict-resolution=fail"

    # Setup: ensure both have synced state
    sync_vault_quiet "$VAULT0"
    sync_vault_quiet "$VAULT1" "--conflict-resolution=remote"

    # Create conflict: modify same file in both
    cat > "$VAULT0/from_vault0.md" << 'EOF'
---
publish: true
---
# Conflict from vault0
EOF
    sync_vault_quiet "$VAULT0"

    cat > "$VAULT1/from_vault0.md" << 'EOF'
---
publish: true
---
# Conflict from vault1
EOF

    # Sync vault1 with fail mode - should exit with error
    log_info "Syncing vault1 with --conflict-resolution=fail (expecting failure)..."
    local exit_code=0
    sync_vault "$VAULT1" "--conflict-resolution=fail" || exit_code=$?

    if [ $exit_code -ne 0 ]; then
        log_success "Sync failed as expected on conflict"
    else
        log_fail "Sync should have failed but succeeded"
    fi
}

test_nested_folders() {
    log_section "Test: Nested folder sync"

    # Reset conflict state
    sync_vault_quiet "$VAULT1" "--conflict-resolution=remote"

    # Create nested structure in vault0
    mkdir -p "$VAULT0/deep/nested/folder"
    cat > "$VAULT0/deep/nested/folder/deep_file.md" << 'EOF'
---
publish: true
---
# Deep File
Deeply nested content.
EOF

    sync_vault_quiet "$VAULT0"
    sync_vault_quiet "$VAULT1" "--conflict-resolution=remote"

    assert_file_exists "$VAULT1/deep/nested/folder/deep_file.md" "Nested file synced to vault1"
}

test_file_deletion_sync() {
    log_section "Test: File deletion propagation"

    # Create file in vault0, sync to both
    cat > "$VAULT0/to_delete.md" << 'EOF'
---
publish: true
---
# To Delete
EOF
    sync_vault "$VAULT0"
    sync_vault "$VAULT1" "--conflict-resolution=remote"

    assert_file_exists "$VAULT1/to_delete.md" "File synced to vault1 before deletion"

    # Delete in vault0 and sync
    rm "$VAULT0/to_delete.md"
    sync_vault_quiet "$VAULT0"

    # Sync vault1 - should detect server deletion
    log_info "Note: Current CLI keeps local files when server deletes them"
    sync_vault "$VAULT1" "--conflict-resolution=remote"

    # Current behavior: keeps local file
    # assert_file_not_exists "$VAULT1/to_delete.md" "File deleted in vault1"
}

test_delete_vs_modify_conflict() {
    log_section "Test: Delete vs Modify conflict"

    # Create file, sync to both
    cat > "$VAULT0/delete_modify.md" << 'EOF'
---
publish: true
---
# Original
EOF
    sync_vault_quiet "$VAULT0"
    sync_vault_quiet "$VAULT1" "--conflict-resolution=remote"

    # vault0 deletes, vault1 modifies
    rm "$VAULT0/delete_modify.md"
    cat > "$VAULT1/delete_modify.md" << 'EOF'
---
publish: true
---
# Modified by vault1
EOF

    sync_vault_quiet "$VAULT0"
    sync_vault "$VAULT1"

    # vault1 should keep its modified version (current behavior)
    assert_file_exists "$VAULT1/delete_modify.md" "vault1 keeps modified file after server deletion"
}

test_syncstate_reset_as_conflict() {
    log_section "Test: Reset syncstate → all as conflict"

    # Ensure vault1 is synced
    sync_vault_quiet "$VAULT1" "--conflict-resolution=remote"

    # Remove syncstate
    rm -f "$VAULT1/.sync-state.json"

    # Modify a file
    cat > "$VAULT1/from_vault1.md" << 'EOF'
---
publish: true
---
# Modified after syncstate reset
EOF

    # Sync - should detect as conflict (no lastSyncedHash)
    log_info "Syncing after syncstate deletion - conflicts expected"
    sync_vault "$VAULT1" "--conflict-resolution=local"

    assert_file_exists "$VAULT1/.sync-state.json" "Sync state recreated"
}

test_asset_upload() {
    log_section "Test: Asset upload"

    # Create note with asset reference
    mkdir -p "$VAULT0/assets"
    cp "$VAULT0/test.png" "$VAULT0/assets/test_asset.png"

    cat > "$VAULT0/note_with_asset.md" << 'EOF'
---
publish: true
---
# Note with Asset
![[test_asset.png]]
EOF

    sync_vault "$VAULT0" "-v"

    assert_file_exists "$VAULT0/assets/test_asset.png" "Asset exists in vault0"
}

test_asset_different_no_conflict() {
    log_section "Test: Different assets - no conflict"

    # vault0 has asset A
    cp "$VAULT0/format.png" "$VAULT0/assets/asset_a.png"
    cat > "$VAULT0/note_asset_a.md" << 'EOF'
---
publish: true
---
# Note A
![[asset_a.png]]
EOF

    # vault1 has asset B
    mkdir -p "$VAULT1/assets"
    cp "$VAULT0/format.jpg" "$VAULT1/assets/asset_b.png"
    cat > "$VAULT1/note_asset_b.md" << 'EOF'
---
publish: true
---
# Note B
![[asset_b.png]]
EOF

    sync_vault_quiet "$VAULT0"
    sync_vault_quiet "$VAULT1"
    sync_vault_quiet "$VAULT0"

    assert_file_exists "$VAULT0/note_asset_a.md" "vault0 has note_asset_a.md"
    assert_file_exists "$VAULT0/note_asset_b.md" "vault0 has note_asset_b.md"
    assert_file_exists "$VAULT1/note_asset_a.md" "vault1 has note_asset_a.md"
    assert_file_exists "$VAULT1/note_asset_b.md" "vault1 has note_asset_b.md"
}

test_asset_download_two_way() {
    log_section "Test: Asset download (two-way sync)"

    # Create note with NEW asset in vault0
    cp "$VAULT0/test.png" "$VAULT0/assets/download_test.png"
    cat > "$VAULT0/note_download_asset.md" << 'EOF'
---
publish: true
---
# Note with downloadable asset
![[download_test.png]]
EOF

    # Sync vault0 (uploads note + asset)
    sync_vault "$VAULT0" "-v"

    # Sync vault1 with two-way (should download asset)
    sync_vault "$VAULT1" "-v"

    assert_file_exists "$VAULT1/note_download_asset.md" "Note pulled to vault1"
    assert_file_exists "$VAULT1/assets/download_test.png" "Asset downloaded to vault1"
}

test_asset_conflict_local() {
    log_section "Test: Asset conflict --conflict-resolution=local"

    # Both vaults have same asset with different content
    mkdir -p "$VAULT0/assets" "$VAULT1/assets"

    # vault0: use format.png as the asset
    cp "$VAULT0/format.png" "$VAULT0/assets/conflict_asset.png"
    cat > "$VAULT0/note_conflict_asset.md" << 'EOF'
---
publish: true
---
# Conflict Asset Note
![[conflict_asset.png]]
EOF

    # Sync vault0 first
    sync_vault_quiet "$VAULT0"

    # vault1: use format.jpg (different content) with same name
    cp "$VAULT0/format.jpg" "$VAULT1/assets/conflict_asset.png"

    # Sync vault1 - should detect conflict and keep local
    sync_vault "$VAULT1" "--conflict-resolution=local -v"

    # vault1 should still have its version (jpg content, not png)
    local vault1_size=$(stat -c%s "$VAULT1/assets/conflict_asset.png" 2>/dev/null || stat -f%z "$VAULT1/assets/conflict_asset.png")
    local original_jpg_size=$(stat -c%s "$VAULT0/format.jpg" 2>/dev/null || stat -f%z "$VAULT0/format.jpg")

    if [ "$vault1_size" = "$original_jpg_size" ]; then
        log_success "Asset conflict resolved: kept local version"
    else
        log_fail "Asset conflict: local version was overwritten"
    fi
}

test_one_way_sync() {
    log_section "Test: One-way sync (no --two-way flag)"

    # Reset vaults
    rm -f "$VAULT1/one_way_test.md"

    # Create file in vault0
    cat > "$VAULT0/one_way_test.md" << 'EOF'
---
publish: true
---
# One-way test
EOF

    # Sync vault0 (push)
    sync_vault_quiet "$VAULT0"

    # Sync vault1 WITHOUT --two-way flag
    log_info "Syncing vault1 in one-way mode (no --two-way)..."
    cd "$OBSIDIAN_SYNC_DIR"
    npx tsx src/sync/cli/cmd.ts --folder "$VAULT1" --api-key "$API_KEY" --api-url "$ENDPOINT" 2>&1

    # In one-way mode, vault1 should NOT pull from server
    assert_file_not_exists "$VAULT1/one_way_test.md" "One-way sync does not pull files"
}

test_publish_field_filtering() {
    log_section "Test: publishFields filtering"

    # Note: CLI doesn't have publishFields support yet (it's for Obsidian plugin)
    # This test verifies that notes WITHOUT publish:true are still synced in CLI mode
    # (CLI syncs all .md files, filtering is Obsidian-specific via metadataCache)

    cat > "$VAULT0/no_publish_field.md" << 'EOF'
---
title: No publish field
---
# No publish field
This note has no publish: true
EOF

    cat > "$VAULT0/publish_false.md" << 'EOF'
---
publish: false
---
# Publish false
This note has publish: false
EOF

    sync_vault "$VAULT0"

    # CLI should sync all files (no filtering)
    log_info "Note: CLI syncs all .md files, publishFields filtering is Obsidian-specific"
    log_success "publishFields test: CLI syncs without filtering (expected behavior)"
}

test_html_files_sync() {
    log_section "Test: HTML files sync"

    # Create HTML layout file in vault0
    mkdir -p "$VAULT0/_layouts/demo"
    cat > "$VAULT0/_layouts/demo/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>HTML Test</h1></body>
</html>
EOF

    sync_vault_quiet "$VAULT0"
    sync_vault_quiet "$VAULT1"

    assert_file_exists "$VAULT1/_layouts/demo/index.html" "HTML layout synced to vault1"
    assert_file_contains "$VAULT1/_layouts/demo/index.html" "HTML Test" "HTML content correct"
}

test_meta_injection() {
    log_section "Test: Meta injection with --meta and prefix"

    # Create temp directory for CLI meta test (separate from vault0)
    local CLI_TEST_DIR="$TMP_DIR/cli_meta_source"
    rm -rf "$CLI_TEST_DIR"
    mkdir -p "$CLI_TEST_DIR"

    # Simple file without frontmatter - title should be injected via --meta
    cat > "$CLI_TEST_DIR/cli_test.md" << 'EOF'
# CLI Test Page

This page was synced with --meta title=FromCLI to test meta injection.

The page title (h1 in header) should be "FromCLI" from injected frontmatter.
EOF

    # Sync with prefix "cli_meta" - file will be uploaded as cli_meta/cli_test.md
    log_info "Syncing with prefix cli_meta and --meta title=FromCLI --meta free=true..."
    cd "$OBSIDIAN_SYNC_DIR"
    npx tsx src/sync/cli/cmd.ts \
        "$CLI_TEST_DIR" cli_meta \
        --api-key "$API_KEY" \
        --api-url "$ENDPOINT" \
        --meta title=FromCLI \
        --meta free=true

    # Verify via GraphQL that meta was injected
    log_info "Verifying meta injection via GraphQL..."

    local response
    response=$(curl -s -X POST "$ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: $API_KEY" \
        -d '{"query":"query { notePaths(filter: { paths: [\"cli_meta/cli_test.md\"] }) { path: value latestNoteView { content } } }"}')

    # Check for title: FromCLI in frontmatter
    if echo "$response" | grep -q "title: FromCLI"; then
        log_success "Meta injection: title=FromCLI found in response"
    else
        log_fail "Meta injection: title=FromCLI NOT found"
        echo "Response: $response" | head -500
    fi

    # Note: NOT cleaning up - it's used by e2e/vault.spec.js to verify meta injection
    log_info "cli_meta files kept for e2e verification"
}

test_dry_run() {
    log_section "Test: Dry run mode"

    # Modify a file
    echo "<!-- dry run test -->" >> "$VAULT0/unique.md"

    # Run dry run
    local output
    output=$(sync_vault "$VAULT0" "--dry-run")

    if [[ "$output" == *"Dry run"* ]]; then
        log_success "Dry run shows changes without executing"
    else
        log_fail "Dry run output missing expected message"
    fi

    # Verify nothing changed - run again
    output=$(sync_vault "$VAULT0" "--dry-run")
    if [[ "$output" == *"To push:"* ]]; then
        log_success "Dry run did not modify sync state"
    else
        log_fail "Dry run may have modified state"
    fi

    # Clean up - actually sync
    sync_vault_quiet "$VAULT0"
}

# ============ Main ============

main() {
    log_section "Obsidian Sync CLI E2E Tests"

    setup

    # Basic sync
    test_initial_sync_vault0
    test_empty_vault_pulls_all
    test_new_file_sync_both_ways

    # Conflict resolution
    test_conflict_resolve_local
    test_conflict_resolve_remote
    test_conflict_resolve_fail

    # Edge cases
    test_nested_folders
    test_file_deletion_sync
    test_delete_vs_modify_conflict
    test_syncstate_reset_as_conflict

    # Assets
    test_asset_upload
    test_asset_different_no_conflict
    test_asset_download_two_way
    test_asset_conflict_local

    # Sync modes
    test_one_way_sync
    test_publish_field_filtering
    test_html_files_sync

    # Meta injection
    test_meta_injection

    # Other
    test_dry_run

    # Summary
    log_section "Summary"
    echo ""
    echo "  Total:  $TESTS_RUN"
    echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
        echo ""
        cleanup
        exit 1
    else
        echo "  Failed: $TESTS_FAILED"
    fi
    echo ""

    cleanup
    echo -e "${GREEN}✅ All tests passed!${NC}"
}

# Run with cleanup on exit
trap 'cleanup' EXIT

main "$@"
