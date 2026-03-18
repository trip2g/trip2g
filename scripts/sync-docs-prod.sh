#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SYNC_VERSION="0.3.5"
SYNC_BIN="$PROJECT_ROOT/tmp/trip2g-sync.mjs"

# Load PROD_API_KEY from .env
if [ -f "$PROJECT_ROOT/.env" ]; then
    PROD_API_KEY=$(grep '^PROD_API_KEY=' "$PROJECT_ROOT/.env" | head -1 | cut -d= -f2-)
fi

if [ -z "$PROD_API_KEY" ]; then
    echo "Error: PROD_API_KEY not set in .env"
    exit 1
fi

# Download sync CLI if not cached
if [ ! -f "$SYNC_BIN" ]; then
    mkdir -p "$PROJECT_ROOT/tmp"
    echo "Downloading trip2g-sync v$SYNC_VERSION..."
    curl -sL -o "$SYNC_BIN" \
        "https://github.com/trip2g/obsidian-sync/releases/download/$SYNC_VERSION/trip2g-sync.mjs"
fi

export API_KEY="$PROD_API_KEY"

node "$SYNC_BIN" \
    --folder "$PROJECT_ROOT/docs" \
    --api-url https://trip2g.com/graphql \
    --conflict-resolution local \
    --verbose
