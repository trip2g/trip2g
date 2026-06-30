#!/bin/bash
# End-to-end test runner
# Usage: ./scripts/test-e2e.sh [--headed|--debug|--ui] [--update-snapshots]
# By default the CLI sync golden snapshots are verified; pass --update-snapshots
# (or set UPDATE_SNAPSHOTS=1) to overwrite them.
# Set ENABLE_TG=1 to enable Telegram tests (disabled by default)
#
# Test Flow Overview:
# ===================
# 1. Setup (e2e/setup.spec.js)
#    - Sign in as admin (hello@example.com with dev code 111111)
#    - Create API key via UI, save to .test-api-key
#    - This runs FIRST via Playwright before sync tests
#
# 2. CLI Sync Tests (scripts/test-sync-cli.sh)
#    - Upload test vault via obsidian-sync CLI
#    - Verify files are synced to backend
#    - NOTE: Some notes have scheduled telegram posts, creating background jobs
#
# 3. Telegram Cron
#    - Schedules telegram publish posts job
#    - Background jobs start processing immediately
#
# 4. Main Playwright Tests
#    - Browser-based UI tests (smoke, vault, rss, etc.)
#    - Excludes Setup, Layout CSS, and Webhook tests (--grep-invert)
#
# 5. Layout CSS Tests
#    - Test CSS hot-reload functionality
#
# 6. Telegram Update Tests
#    - Update telegram posts and verify changes
#    - Wait for ALL background jobs to complete (wait_all_jobs)
#
# 7. Webhook E2E Tests (e2e/webhooks.spec.js)
#    - Test webhook delivery, agent responses, depth protection
#    - CRITICAL: Must run after all telegram jobs complete (empty job queue)
#    - Uses /debug/wait_all_jobs which waits for ALL background jobs
#    - Running before telegram drains would cause timeouts due to pending jobs
#    - Must run BEFORE the release/draft specs: it pushes notes and serves them,
#      which needs the "serve latest committed" fallback those specs disable
#
# 8. Unreleased-changes E2E Tests (e2e/unreleased-changes.spec.js)
#    - Pins a static release as live and leaves it pinned (no unset-live mutation),
#      so it must come after any spec that pushes-and-serves notes.
#
# 9. show_draft_versions E2E Tests (e2e/show-draft-versions.spec.js) - RUNS LAST
#    - Flips the global show_draft_versions config, switching public serving from
#      "latest committed" to the live-release snapshot. Nothing must run after it.
#
# Why This Order?
# ===============
# - Setup FIRST: Creates admin session and API key needed by all other tests
# - Sync BEFORE Browser: Ensures test data is loaded before UI tests
# - Webhooks after telegram: Ensures job queue is empty (wait_all_jobs won't block)
# - Release/draft specs LAST: they poison global serving state (live release + config)
#
# Environment:
# ============
# - Docker Compose test environment (docker-compose.test.yml)
# - App runs on port 20081 with DEV=true (allows dev code 111111)
# - USER_TOKEN_COOKIE_NAME=trip2g_e2e (avoids conflicts with dev cookies)

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

export APP_URL="${APP_URL:-http://localhost:20081}"
export GIT_API_BASE_PATH="${GIT_API_BASE_PATH:-/git}"
export USER_TOKEN_COOKIE_NAME=trip2g_e2e
export ENDPOINT="${APP_URL}/graphql" # for push_notes.py

# Success flag - set to 1 at the very end if all tests pass
SUCCESS=0

# Snapshot mode for the CLI sync test (scripts/test-sync-cli.sh):
# VERIFY by default; only overwrite golden snapshots in testdata/sync-updates/
# when --update-snapshots is passed (or UPDATE_SNAPSHOTS=1 is set).
UPDATE_SNAPSHOTS="${UPDATE_SNAPSHOTS:-0}"
_filtered_args=()
for _arg in "$@"; do
  case "$_arg" in
    --update-snapshots) UPDATE_SNAPSHOTS=1 ;;
    *) _filtered_args+=("$_arg") ;;
  esac
done
set -- "${_filtered_args[@]}"

# Helper function to run cron job
run_telegram_cron() {
  echo "⏳ Running Telegram publish posts cron job..."
  curl -f "$APP_URL/debug/run_cron_job?name=send_scheduled_telegram_publishposts"
}

# Helper function to wait for all background jobs
wait_all_jobs() {
  echo "⏳ Waiting for all background jobs to complete..."
  curl -s --max-time 300 "$APP_URL/debug/wait_all_jobs" | tee /dev/stderr | grep -q "^ok:" || exit 1

  sleep 2 # wait a bit for consistency
}

# Helper function to sync vault
sync_vault() {
  echo "🔄 Syncing vault..."
  npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder tmp/testvault0 --api-key "$API_KEY" --api-url "$ENDPOINT"
}

# Helper: sign in to peer and push seedvault content
sync_seedvault_to_peer() {
  local PEER_URL="http://localhost:20091"
  local PEER_GRAPHQL="$PEER_URL/graphql"

  echo "🔄 Setting up peer instance (seedvault push)..."

  # 1. Request sign-in code on peer
  curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { requestEmailSignInCode(input: { email: \"hello@example.com\" }) { ... on RequestEmailSignInCodePayload { success } } }"}' > /dev/null

  # 2. Sign in to peer (dev code 111111)
  local PEER_TOKEN
  PEER_TOKEN=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { signInByEmail(input: { email: \"hello@example.com\", code: \"111111\" }) { ... on SignInPayload { token } } }"}' \
    | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_TOKEN" ]; then
    echo -e "${RED}✗ Failed to sign in to peer${NC}"
    return 1
  fi

  # 3. Create API key on peer
  local PEER_API_KEY
  PEER_API_KEY=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -H "Cookie: trip2g_e2e_peer=$PEER_TOKEN" \
    -d '{"query":"mutation AdminCreateApiKey($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) { ... on ErrorPayload { message } ... on CreateApiKeyPayload { value } } } }","variables":{"input":{"description":"demo"}}}' \
    | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_API_KEY" ]; then
    echo -e "${RED}✗ Failed to create peer API key${NC}"
    return 1
  fi

  echo -e "${GREEN}✓ Peer API key created: ${PEER_API_KEY:0:20}...${NC}"

  # 4. Push seedvault to peer via obsidian-sync CLI
  npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder testdata/seedvault --api-key "$PEER_API_KEY" --api-url "$PEER_GRAPHQL"

  echo -e "${GREEN}✓ Seedvault pushed to peer${NC}"
}

sync_seedvault2_to_peer2() {
  local PEER_URL="http://localhost:20093"
  local PEER_GRAPHQL="$PEER_URL/graphql"

  echo "🔄 Setting up peer2 instance (seedvault2 push)..."

  curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { requestEmailSignInCode(input: { email: \"hello@example.com\" }) { ... on RequestEmailSignInCodePayload { success } } }"}' > /dev/null

  local PEER_TOKEN
  PEER_TOKEN=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { signInByEmail(input: { email: \"hello@example.com\", code: \"111111\" }) { ... on SignInPayload { token } } }"}' \
    | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_TOKEN" ]; then
    echo -e "${RED}✗ Failed to sign in to peer2${NC}"
    return 1
  fi

  local PEER_API_KEY
  PEER_API_KEY=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -H "Cookie: trip2g_e2e_peer2=$PEER_TOKEN" \
    -d '{"query":"mutation AdminCreateApiKey($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) { ... on ErrorPayload { message } ... on CreateApiKeyPayload { value } } } }","variables":{"input":{"description":"demo"}}}' \
    | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_API_KEY" ]; then
    echo -e "${RED}✗ Failed to create peer2 API key${NC}"
    return 1
  fi

  npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder testdata/seedvault2 --api-key "$PEER_API_KEY" --api-url "$PEER_GRAPHQL"
  echo -e "${GREEN}✓ Seedvault2 pushed to peer2${NC}"
}

sync_seedvault3_to_peer3() {
  local PEER_URL="http://localhost:20095"
  local PEER_GRAPHQL="$PEER_URL/graphql"

  echo "🔄 Setting up peer3 instance (seedvault3 push)..."

  curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { requestEmailSignInCode(input: { email: \"hello@example.com\" }) { ... on RequestEmailSignInCodePayload { success } } }"}' > /dev/null

  local PEER_TOKEN
  PEER_TOKEN=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { signInByEmail(input: { email: \"hello@example.com\", code: \"111111\" }) { ... on SignInPayload { token } } }"}' \
    | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_TOKEN" ]; then
    echo -e "${RED}✗ Failed to sign in to peer3${NC}"
    return 1
  fi

  local PEER_API_KEY
  PEER_API_KEY=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -H "Cookie: trip2g_e2e_peer3=$PEER_TOKEN" \
    -d '{"query":"mutation AdminCreateApiKey($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) { ... on ErrorPayload { message } ... on CreateApiKeyPayload { value } } } }","variables":{"input":{"description":"demo"}}}' \
    | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_API_KEY" ]; then
    echo -e "${RED}✗ Failed to create peer3 API key${NC}"
    return 1
  fi

  npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder testdata/seedvault3 --api-key "$PEER_API_KEY" --api-url "$PEER_GRAPHQL"
  echo -e "${GREEN}✓ Seedvault3 pushed to peer3${NC}"
}

# NOTE: presigned MinIO URLs contain "minio:29000" hostname.
# Add "127.0.0.1 minio" to /etc/hosts for local testing.
# In CI this is done in the workflow file.

echo "🧪 Starting E2E tests..."
echo ""

# Cleanup function
cleanup() {
  echo ""

  # Show logs if tests didn't complete successfully
  # if [ $SUCCESS -eq 0 ]; then
  #   echo "📋 Container logs (due to error):"
  #   echo "================================="
  #   docker compose -f docker-compose.test.yml logs
  #   echo "================================="
  #   echo ""
  # fi

  echo "🧹 Cleaning up..."
  #docker compose -f docker-compose.test.yml down -v

  # Remove temporary files
  # rm -f .test-api-key tmp/.test-admin-jwt
  # rm -rf tmp/testvault0 tmp/testvault1

  echo -e "${GREEN}✓ Cleanup complete${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Clean up any existing test containers (keep embedding running to avoid slow model reload)
echo "🧹 Cleaning up existing test containers..."
docker compose -f docker-compose.test.yml stop app app-replica app-replica2 app-peer app-peer2 app-peer3 minio fleet llm-mock krisp-mock test-data 2>/dev/null || true
docker compose -f docker-compose.test.yml rm -f app app-replica app-replica2 app-peer app-peer2 app-peer3 fleet llm-mock krisp-mock test-data 2>/dev/null || true
# MinIO needs no explicit volume drop: docker-compose.test.yml mounts its /data as
# tmpfs, so every container (re)create starts with an empty backup store — no stale
# backup can be restored over the freshly-seeded DB (keeps the onboarding spec valid).

# Prepare database
export DB_PATH="tmp/data/test.sqlite3"
echo "🗄️  Preparing test database $DB_PATH"

mkdir -p tmp/data
rm -f "$DB_PATH" "$DB_PATH-shm" "$DB_PATH-wal"
# Reproducibility note: the federation e2e spec assumes a clean DB. Without this
# reset, multiple active outbound rows for the same kbURL accumulate across runs
# and make revoke ineffective (FederationSecretByKBURL still finds an active row).
sqlite3 "$DB_PATH" < testdata/e2e_seed.sql

# Cold idempotency for the federation specs: reset the peer DBs and the
# obsidian-sync CLI state files together. Peers are seeded by the sync CLI,
# which records what it pushed in a per-vault .sync-state.json. If the peer DBs
# are wiped (or otherwise out of sync) while those state files persist, the CLI
# assumes the peers are already seeded and pushes nothing — leaving peers empty
# and federation/search specs failing with "No results found". Resetting both
# keeps a from-cold run reproducible. Set KEEP_PEER_DATA=1 to skip (faster local
# iteration when the peers are already warm and consistent).
if [ "${KEEP_PEER_DATA}" != "1" ]; then
  rm -f tmp/data/peer.sqlite3* tmp/data/peer2.sqlite3* tmp/data/peer3.sqlite3*
  find testdata/seedvault testdata/seedvault2 testdata/seedvault3 \
    -name ".sync-state*.json" -delete 2>/dev/null || true
fi

# Cleanup telegram channels (only if ENABLE_TG=1)
if [ "${ENABLE_TG}" = "1" ]; then
  go run ./cmd/tge2e -db "$DB_PATH" patch-db
  echo "🧹 Cleaning up Telegram channels..."
  go run ./cmd/tge2e -db "$DB_PATH" cleanup
else
  echo "🧹 Removing Telegram accounts and bots (ENABLE_TG not set)..."
  sqlite3 "$DB_PATH" "
    PRAGMA foreign_keys = OFF;
    delete from telegram_publish_sent_messages;
    delete from telegram_publish_sent_account_messages;
    delete from telegram_publish_note_tags;
    delete from telegram_publish_notes;
    delete from telegram_publish_chats;
    delete from telegram_publish_instant_chats;
    delete from telegram_publish_account_chats;
    delete from telegram_publish_account_instant_chats;
    delete from telegram_publish_tags;
    delete from telegram_accounts;
    delete from tg_user_states;
    delete from tg_user_profiles;
    delete from wait_list_tg_bot_requests;
    delete from tg_attach_codes;
    delete from tg_bot_chat_subgraph_accesses;
    delete from tg_bot_chat_subgraph_invites;
    delete from tg_bot_chats;
    delete from tg_bots;
    PRAGMA foreign_keys = ON;
  "
fi

# Start services (embedding is kept alive between runs; start it without recreate if not running)
echo "🚀 Starting services..."
docker compose -f docker-compose.test.yml up -d --no-recreate embedding 2>/dev/null || true
docker compose -f docker-compose.test.yml up -d --build --force-recreate app app-replica app-replica2 app-peer app-peer2 app-peer3 minio llm-mock krisp-mock fleet

# Wait for services
./scripts/waitfor localhost:20081 || {
  echo -e "${RED}✗ Services failed to start${NC}"
  exit 1
}

# Wait for the read replica (read-replica.spec.js)
./scripts/waitfor localhost:20071 || {
  echo -e "${RED}✗ Read replica failed to start${NC}"
  exit 1
}

./scripts/waitfor localhost:20073 || {
  echo -e "${RED}✗ Read replica2 failed to start${NC}"
  exit 1
}

# Wait for peer instance (federation e2e)
./scripts/waitfor localhost:20091 || {
  echo -e "${RED}✗ Peer service failed to start${NC}"
  exit 1
}

# Wait for peer2 and peer3 instances (federation ACL e2e)
./scripts/waitfor localhost:20093 || {
  echo -e "${RED}✗ Peer2 service failed to start${NC}"
  exit 1
}

./scripts/waitfor localhost:20095 || {
  echo -e "${RED}✗ Peer3 service failed to start${NC}"
  exit 1
}

# Wait for fleet services (needed by e2e/fleet.spec.js and e2e/krisp-ingest.spec.js)
./scripts/waitfor localhost:29091 -t 30 || {
  echo -e "${RED}✗ llm-mock failed to start${NC}"
  exit 1
}
./scripts/waitfor localhost:29092 -t 30 || {
  echo -e "${RED}✗ krisp-mock failed to start${NC}"
  exit 1
}
./scripts/waitfor localhost:29090 -t 60 || {
  echo -e "${RED}✗ fleet failed to start${NC}"
  exit 1
}

# Run setup test to create API key
echo "🔑 Running setup test (create API key)..."
echo ""

npx playwright test e2e/setup.spec.js || {
  echo -e "${RED}✗ Setup test failed${NC}"
  exit 1
}

# Check if API key was created
if [ ! -f .test-api-key ]; then
  echo -e "${RED}✗ API key file not found${NC}"
  exit 1
fi

API_KEY=$(cat .test-api-key)
echo -e "${GREEN}✓ API key created: ${API_KEY:0:20}...${NC}"
echo ""

# Run CLI sync E2E tests (also pushes test data)
echo "🔄 Running CLI sync E2E tests..."
echo ""

SYNC_SNAPSHOT_ARGS=()
if [ "$UPDATE_SNAPSHOTS" = "1" ]; then
  echo -e "${YELLOW}↻ Updating CLI sync golden snapshots${NC}"
  SYNC_SNAPSHOT_ARGS+=(--update-snapshots)
fi
./scripts/test-sync-cli.sh --api-key "$API_KEY" --endpoint "$ENDPOINT" "${SYNC_SNAPSHOT_ARGS[@]}" || {
  echo -e "${RED}✗ CLI sync tests failed${NC}"
  exit 1
}

echo ""
echo -e "${GREEN}✓ CLI sync tests passed${NC}"
echo ""

# Push seedvault to peer instance for federation tests
sync_seedvault_to_peer || {
  echo -e "${RED}✗ Peer seedvault sync failed${NC}"
  exit 1
}
echo ""

sync_seedvault2_to_peer2 || {
  echo -e "${RED}✗ Peer2 seedvault sync failed${NC}"
  exit 1
}
echo ""

sync_seedvault3_to_peer3 || {
  echo -e "${RED}✗ Peer3 seedvault sync failed${NC}"
  exit 1
}
echo ""

# Schedule send_scheduled_telegram_publishposts job (only if ENABLE_TG=1)
if [ "${ENABLE_TG}" = "1" ]; then
  run_telegram_cron
fi

# Check for MANUAL mode
if [ "$MANUAL" = "1" ] || [ "$MANUAL" = "true" ]; then
  echo -e "${YELLOW}🔧 Manual testing mode${NC}"
  echo ""
  echo "Services are running:"
  echo "  App: ${APP_URL}"
  echo "  GraphQL: ${ENDPOINT}"
  echo "  MinIO: http://localhost:29000 (console: http://localhost:29001)"
  echo ""
  echo "API Key: ${API_KEY}"
  echo ""
  echo "Push notes command:"
  echo "  ENDPOINT=\"${ENDPOINT}\" API_KEY=\"${API_KEY}\" npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder docs/demo"
  echo ""
  echo "Press ENTER to stop services and cleanup..."
  read -r
  exit 0
fi

# Drain background jobs (embeddings from the vault syncs above) so browser
# tests see a stable search index and an empty queue.
wait_all_jobs

# Run main Playwright tests
echo "🎭 Running main Playwright tests..."
echo ""

if [ "$1" = "--headed" ]; then
  npx playwright test --grep-invert "Setup|Layout CSS|Webhook|Screenshot|Bidirectional Federation|coexistence" --headed
elif [ "$1" = "--debug" ]; then
  npx playwright test --grep-invert "Setup|Layout CSS|Webhook|Screenshot|Bidirectional Federation|coexistence" --debug
elif [ "$1" = "--ui" ]; then
  npx playwright test --grep-invert "Setup|Layout CSS|Webhook|Screenshot|Bidirectional Federation|coexistence" --ui
else
  npx playwright test --grep-invert "Setup|Layout CSS|Webhook|Screenshot|Bidirectional Federation|coexistence"
fi

TEST_EXIT_CODE=$?

if [ $TEST_EXIT_CODE -ne 0 ]; then
  echo ""
  echo -e "${RED}✗ Playwright tests failed${NC}"
  echo "Run with --ui for interactive debugging: ./scripts/test-e2e.sh --ui"
  exit $TEST_EXIT_CODE
fi

# Screenshot regression tests
echo ""
echo "📸 Running screenshot tests..."
npx playwright test e2e/screenshots.spec.js || {
  echo -e "${RED}✗ Screenshot tests failed${NC}"
  exit 1
}
echo -e "${GREEN}✓ Screenshot tests passed${NC}"

# Test CSS hot-reload: update layout CSS and verify change
echo ""
echo "🎨 Testing CSS hot-reload..."
echo "body { color: #f00; }" >> tmp/testvault0/_layouts/custom/styles.css
sync_vault

echo "🎭 Running layout CSS tests..."
npx playwright test e2e/layoutcss.spec.js || {
  echo -e "${RED}✗ Layout CSS tests failed${NC}"
  exit 1
}
echo -e "${GREEN}✓ Layout CSS tests passed${NC}"
echo ""

if [ "${ENABLE_TG}" = "1" ]; then
  # Wait for telegram messages to be sent
  wait_all_jobs

  # Check channel snapshots
  echo "📷 Checking Telegram channel snapshots..."
  go run ./cmd/tge2e -db tmp/data/test.sqlite3 -snapshots testdata/telegram/step0 check

  # Update posts and re-sync
  echo "📝 Updating Telegram posts..."

  # Add "Updated!" to all telegram posts
  for f in tmp/testvault0/telegram_*.md; do
    echo -e "\n\nUpdated!" >> "$f"
  done

  # Add photo embed to text post (account posting adds image)
  echo -e "\n![[telegram_photo.png]]" >> tmp/testvault0/telegram_text.md

  # Replace photo in one_photo post to test photo replacement
  sed -i 's/telegram_photo\.png/test.png/' tmp/testvault0/telegram_one_photo.md

  # Sync updated files
  sync_vault

  # Wait for telegram messages to be sent
  wait_all_jobs

  # Check channel snapshots after update
  echo "📷 Checking Telegram channel snapshots after update..."
  go run ./cmd/tge2e -db tmp/data/test.sqlite3 -snapshots testdata/telegram/step1 check
fi

# Run bidirectional federation E2E tests
echo ""
echo "🔗 Running bidirectional federation E2E tests..."
npx playwright test e2e/federation-bidir.spec.js || {
  echo -e "${RED}✗ Bidirectional federation E2E tests failed${NC}"
  exit 1
}
echo -e "${GREEN}✓ Bidirectional federation E2E tests passed${NC}"

# Run git<->plugin coexistence E2E tests in isolation. gitsync shares the single
# instance's DB-canonical git mirror; under fullyParallel it races note-mutating
# specs (e.g. updatenotes) whose materialize advances master between gitsync's
# pull and push -> spurious non-fast-forward. Run it alone so nothing mutates the
# mirror concurrently. Must precede unreleased-changes/show-draft (serving-state poison).
echo ""
echo "🔁 Running git<->plugin coexistence E2E tests..."
npx playwright test e2e/gitsync.spec.js || {
  echo -e "${RED}✗ gitsync E2E tests failed${NC}"
  exit 1
}
echo -e "${GREEN}✓ gitsync E2E tests passed${NC}"

# Run webhook E2E tests (when job queue is empty). Must run BEFORE
# unreleased-changes: webhooks push notes and serve them at their public URL,
# which only works while serving falls back to "latest committed". The
# unreleased-changes spec pins a static release as live and cannot revert that
# (no unset-live mutation exists), so anything that pushes-and-serves after it
# 404s. Keep it last instead.
echo ""
echo "🔗 Running webhook E2E tests..."
npx playwright test e2e/webhooks.spec.js || {
  echo -e "${RED}✗ Webhook E2E tests failed${NC}"
  exit 1
}
echo -e "${GREEN}✓ Webhook E2E tests passed${NC}"

# Run unreleased-changes E2E tests: this spec pins a live release globally and
# leaves it pinned, so it must come after any spec that pushes-and-serves notes.
echo ""
echo "📋 Running unreleased-changes E2E tests..."
RUN_ISOLATED_SPECS=1 npx playwright test e2e/unreleased-changes.spec.js || {
  echo -e "${RED}✗ Unreleased changes E2E tests failed${NC}"
  exit 1
}
echo -e "${GREEN}✓ Unreleased changes E2E tests passed${NC}"

# Run show_draft_versions E2E tests DEAD LAST: this spec flips the global
# show_draft_versions config (switching public serving from "latest committed" to
# the live-release snapshot). Any later spec that pushes notes and expects them
# served would 404, so nothing must run after it.
echo ""
echo "📋 Running show_draft_versions E2E tests..."
RUN_ISOLATED_SPECS=1 npx playwright test e2e/show-draft-versions.spec.js || {
  echo -e "${RED}✗ show_draft_versions E2E tests failed${NC}"
  exit 1
}
echo -e "${GREEN}✓ show_draft_versions E2E tests passed${NC}"

echo ""
echo -e "${GREEN}✅ All E2E tests passed!${NC}"

SUCCESS=1

exit 0
