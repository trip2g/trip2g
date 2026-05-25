#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
# Use the CLI built from local obsidian-sync/ source (has --exclude support).
SYNC_BIN="$PROJECT_ROOT/obsidian-sync/dist/trip2g-sync.mjs"

# Load PROD_API_KEY from .env
if [ -f "$PROJECT_ROOT/.env" ]; then
    PROD_API_KEY=$(grep '^PROD_API_KEY=' "$PROJECT_ROOT/.env" | head -1 | cut -d= -f2-)
fi

if [ -z "$PROD_API_KEY" ]; then
    echo "Error: PROD_API_KEY not set in .env"
    exit 1
fi

# Build the CLI from local source so it always reflects current obsidian-sync code.
echo "Building trip2g-sync from obsidian-sync/src..."
( cd "$PROJECT_ROOT/obsidian-sync" && npm run build:cli >/dev/null )

export API_KEY="$PROD_API_KEY"

node "$SYNC_BIN" \
    --folder "$PROJECT_ROOT/docs" \
    --api-url https://trip2g.com/graphql \
    --conflict-resolution local \
    --exclude demo --exclude dev \
    --verbose
