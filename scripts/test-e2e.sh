#!/bin/bash
# End-to-end test runner
# Usage: ./scripts/test-e2e.sh

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

export ENDPOINT="http://localhost:20080/graphql"

echo "🧪 Starting E2E tests..."
echo ""

# Cleanup function
cleanup() {
  echo ""
  echo "🧹 Cleaning up..."
  docker-compose -f docker-compose.test.yml down -v

  # Remove temporary API key file
  rm -f .test-api-key

  echo -e "${GREEN}✓ Cleanup complete${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Start services
echo "🚀 Starting services..."
docker-compose -f docker-compose.test.yml up -d --build

# Wait for services
./scripts/waitfor.sh || {
  echo -e "${RED}✗ Services failed to start${NC}"
  docker-compose -f docker-compose.test.yml logs
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

API_KEY="$API_KEY" python3 scripts/push_notes.py testdata/vault || {
  echo -e "${RED}✗ Failed to push test data${NC}"
  exit 1
}

echo -e "${GREEN}✓ Test data pushed successfully${NC}"
echo ""

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
