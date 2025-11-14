#!/bin/bash
# Wait for services to be ready before running tests

set -e

echo "⏳ Waiting for services to start..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to wait for HTTP endpoint
wait_for_http() {
  local url=$1
  local service_name=$2
  local max_attempts=30
  local attempt=1

  echo -n "Waiting for ${service_name} (${url})... "

  while [ $attempt -le $max_attempts ]; do
    if curl -s -f -o /dev/null "${url}"; then
      echo -e "${GREEN}✓${NC}"
      return 0
    fi

    echo -n "."
    sleep 1
    attempt=$((attempt + 1))
  done

  echo -e "${RED}✗ Failed after ${max_attempts} attempts${NC}"
  return 1
}

# Function to wait for docker-compose health check
wait_for_healthy() {
  local service_name=$1
  local max_attempts=60
  local attempt=1

  echo -n "Waiting for ${service_name} to be healthy... "

  while [ $attempt -le $max_attempts ]; do
    health_status=$(docker-compose -f docker-compose.test.yml ps -q ${service_name} | xargs docker inspect -f '{{.State.Health.Status}}' 2>/dev/null || echo "starting")

    if [ "$health_status" = "healthy" ]; then
      echo -e "${GREEN}✓${NC}"
      return 0
    fi

    if [ "$health_status" = "unhealthy" ]; then
      echo -e "${RED}✗ Service is unhealthy${NC}"
      echo "Logs:"
      docker-compose -f docker-compose.test.yml logs --tail=50 ${service_name}
      return 1
    fi

    echo -n "."
    sleep 1
    attempt=$((attempt + 1))
  done

  echo -e "${RED}✗ Timeout${NC}"
  return 1
}

# Check if services are running
if ! docker-compose -f docker-compose.test.yml ps | grep -q "Up"; then
  echo -e "${RED}Error: No services are running. Did you run 'docker-compose -f docker-compose.test.yml up -d'?${NC}"
  exit 1
fi

# Wait for MinIO
echo ""
echo "📦 MinIO..."
wait_for_healthy minio || exit 1
wait_for_http "http://localhost:20000/minio/health/live" "MinIO API" || exit 1

# Wait for App
echo ""
echo "🚀 Application..."
wait_for_healthy app || exit 1
wait_for_http "http://localhost:20082/health" "App Health" || exit 1
wait_for_http "http://localhost:20080/" "App Frontend" || exit 1

echo ""
echo -e "${GREEN}✅ All services are ready!${NC}"
echo ""
echo "Services available at:"
echo "  - App:            http://localhost:20080"
echo "  - App Health:     http://localhost:20082/health"
echo "  - MinIO Console:  http://localhost:20001"
echo "  - MinIO API:      http://localhost:20000"
echo ""
