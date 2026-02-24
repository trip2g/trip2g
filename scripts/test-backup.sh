#!/bin/bash
# Backup/restore E2E test
# Requires: containers already running after scripts/test-e2e.sh
#
# Test flow:
#   1. Check app container is running
#   2. Stop app gracefully (SIGTERM → waitForShutdown → PerformBackup → MinIO)
#   3. Delete database
#   4. Start app (restoreBackup() → RestoreOnStartup → download from MinIO)
#   5. Wait for healthy
#   6. Run Playwright tests to verify data is intact

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

export APP_URL="${APP_URL:-http://localhost:20081}"

# 1. Check container is running
if ! docker compose -f docker-compose.test.yml ps app --status running 2>/dev/null | grep -q "app"; then
  echo -e "${RED}ERROR: app container is not running.${NC}"
  echo "Run scripts/test-e2e.sh first to start the test environment."
  exit 1
fi
echo -e "${GREEN}✓ Container is running${NC}"

# 2. Graceful stop → triggers shutdown backup
echo "Stopping app (triggers shutdown backup to MinIO)..."
docker compose -f docker-compose.test.yml stop app
echo -e "${GREEN}✓ App stopped, backup should be in MinIO${NC}"

# 3. Delete database
echo "Deleting database..."
rm -f ./tmp/data/test.sqlite3
echo -e "${GREEN}✓ Database deleted${NC}"

# 4. Start app → triggers restore from MinIO
echo "Starting app (triggers restore from MinIO)..."
docker compose -f docker-compose.test.yml start app

# 5. Wait for healthy
echo "Waiting for app to be healthy..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:20082/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ App is healthy after restore${NC}"
    break
  fi
  sleep 2
  if [ "$i" -eq 30 ]; then
    echo -e "${RED}ERROR: App failed to become healthy after restore${NC}"
    echo "Recent logs:"
    docker compose -f docker-compose.test.yml logs app --tail=50
    exit 1
  fi
done

# 6. Verify data integrity
echo "Verifying data integrity after restore..."
npx playwright test \
  --grep-invert "Setup|Layout CSS|Webhook" \
  --reporter=line

echo -e "${GREEN}✓ Backup/restore test passed!${NC}"
