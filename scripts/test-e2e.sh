#!/bin/bash
# End-to-end test runner
# Usage: ./scripts/test-e2e.sh

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

export APP_URL="${APP_URL:-http://localhost:20081}"
export ENDPOINT="${APP_URL}/graphql" # for push_notes.py

# Success flag - set to 1 at the very end if all tests pass
SUCCESS=0

echo "🧪 Starting E2E tests..."
echo ""

# Cleanup function
cleanup() {
  echo ""

  # Show logs if tests didn't complete successfully
  if [ $SUCCESS -eq 0 ]; then
    echo "📋 Container logs (due to error):"
    echo "================================="
    docker compose -f docker-compose.test.yml logs
    echo "================================="
    echo ""
  fi

  echo "🧹 Cleaning up..."
  #docker compose -f docker-compose.test.yml down -v

  # Remove temporary files
  rm -f .test-api-key
  rm -rf tmp/testvault0 tmp/testvault1

  echo -e "${GREEN}✓ Cleanup complete${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Clean up any existing test containers
echo "🧹 Cleaning up existing test containers..."
docker compose -f docker-compose.test.yml down -v 2>/dev/null || true

# Prepare database
export DB_PATH="tmp/data/test.sqlite3"
echo "🗄️  Preparing test database $DB_PATH"

mkdir -p tmp/data
rm -f "$DB_PATH"
sqlite3 "$DB_PATH" < testdata/e2e_seed.sql
go run ./cmd/tge2e -db "$DB_PATH" patch-db

# Cleanup telegram channels
echo "🧹 Cleaning up Telegram channels..."
go run ./cmd/tge2e -db "$DB_PATH" cleanup

# Start services
echo "🚀 Starting services..."
docker compose -f docker-compose.test.yml up -d --build

# Wait for services
./scripts/waitfor localhost:20081 || {
  echo -e "${RED}✗ Services failed to start${NC}"
  exit 1
}

# Wait send_scheduled_telegram_publishposts job
echo "⏳ Waiting for scheduled Telegram publish posts job to complete..."
curl -f "$APP_URL/debug/run_cron_job?name=send_scheduled_telegram_publishposts"

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

./scripts/test-sync-cli.sh --api-key "$API_KEY" --endpoint "$ENDPOINT" || {
  echo -e "${RED}✗ CLI sync tests failed${NC}"
  exit 1
}

echo ""
echo -e "${GREEN}✓ CLI sync tests passed${NC}"
echo ""

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
  echo "  ENDPOINT=\"${ENDPOINT}\" API_KEY=\"${API_KEY}\" npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder testdata/vault"
  echo ""
  echo "Press ENTER to stop services and cleanup..."
  read -r
  exit 0
fi

# Run main Playwright tests
echo "🎭 Running main Playwright tests..."
echo ""

if [ "$1" = "--headed" ]; then
  npx playwright test --grep-invert "Setup" --headed
elif [ "$1" = "--debug" ]; then
  npx playwright test --grep-invert "Setup" --debug
elif [ "$1" = "--ui" ]; then
  npx playwright test --grep-invert "Setup" --ui
else
  npx playwright test --grep-invert "Setup"
fi

TEST_EXIT_CODE=$?

if [ $TEST_EXIT_CODE -ne 0 ]; then
  echo ""
  echo -e "${RED}✗ Playwright tests failed${NC}"
  echo "Run with --ui for interactive debugging: ./scripts/test-e2e.sh --ui"
  exit $TEST_EXIT_CODE
fi

# Wait for telegram messages to be sent
curl -s "$APP_URL/debug/wait_all_jobs" | tee /dev/stderr | grep -q "^ok:" || exit 1

echo ""
echo -e "${GREEN}✅ All E2E tests passed!${NC}"

SUCCESS=1

exit 0
