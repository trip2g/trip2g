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

echo "🧪 Starting E2E tests..."
echo ""

# Cleanup function
cleanup() {
  echo ""
  echo "🧹 Cleaning up..."
  docker compose -f docker-compose.test.yml down -v

  # Remove temporary files
  rm -f .test-api-key
  rm -f testdata/vault/.sync-state.json

  echo -e "${GREEN}✓ Cleanup complete${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Clean up any existing test containers
echo "🧹 Cleaning up existing test containers..."
docker compose -f docker-compose.test.yml down -v 2>/dev/null || true

# Start services
echo "🚀 Starting services..."
docker compose -f docker-compose.test.yml up -d --build

# Wait for services
./scripts/waitfor localhost:20081 || {
  echo -e "${RED}✗ Services failed to start${NC}"
  docker compose -f docker-compose.test.yml logs
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

# Push test vault data
echo "📤 Pushing test vault data..."
echo ""

API_KEY="$API_KEY" npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder testdata/vault || {
  echo -e "${RED}✗ Failed to push test data${NC}"
  exit 1
}

echo -e "${GREEN}✓ Test data pushed successfully${NC}"
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

if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo ""
  echo -e "${GREEN}✅ All E2E tests passed!${NC}"
else
  echo ""
  echo -e "${RED}✗ Some tests failed${NC}"
  echo "Run with --ui for interactive debugging: ./scripts/test-e2e.sh --ui"
fi

exit $TEST_EXIT_CODE
