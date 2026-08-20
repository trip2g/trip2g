#!/bin/bash
# Update e2e screenshot baselines. Needs the seeded e2e instance up (./scripts/test-e2e.sh).
APP_URL="${APP_URL:-http://localhost:20081}" npx playwright test e2e/screenshots.spec.js --update-snapshots
