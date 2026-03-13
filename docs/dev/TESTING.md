# Testing Guide

## E2E Testing with Playwright

### Quick Start

```bash
# Install dependencies (first time only)
npm install

# Install Playwright browsers (first time only)
npx playwright install chromium

# Run E2E tests
npm run test:e2e
```

### Development

```bash
# Run with UI (best for development)
npm run test:e2e:ui

# Run in headed mode (see browser)
npm run test:e2e:headed

# Run specific test file
npx playwright test e2e/smoke.spec.js

# Debug mode
npx playwright test --debug
```

### What Gets Tested

The E2E tests run against the test vault (`testdata/vault/`) and verify:

- ✅ All pages load correctly (200 status)
- ✅ Link resolution (unique, duplicates, relative paths)
- ✅ Markdown embeds (`![[file]]`)
- ✅ Custom layouts work
- ✅ Free content preview (free_paragraphs, free_cut)
- ✅ Subgraphs and premium content
- ✅ Table of contents rendering
- ✅ Cyrillic URLs and special characters
- ✅ Code blocks and media embeds
- ✅ Redirects
- ✅ Navigation between pages

### Test Environment

Tests run in Docker Compose with:

- **App**: http://localhost:20080
- **MinIO**: http://localhost:20000 (storage)
- **Health**: http://localhost:20082/health

The `test-e2e.sh` script automatically:
1. Builds and starts services
2. Waits for them to be healthy
3. Runs setup test (creates API key via UI)
4. Pushes test vault data via `push_notes.py`
5. Runs main Playwright tests
6. Cleans up containers

### Architecture Support

- **ARM Linux**: Builds natively for arm64
- **x64 CI**: Builds natively for amd64
- Docker automatically selects the correct architecture

### Manual Testing

If you need more control:

```bash
# Start services
docker-compose -f docker-compose.test.yml up -d --build

# Wait for ready
./scripts/waitfor.sh

# Run tests
npx playwright test

# View report
npx playwright show-report

# Cleanup
docker-compose -f docker-compose.test.yml down -v
```

### CI/CD

Tests run automatically in GitHub Actions on every push and PR.

See `.github/workflows/e2e.yml` (TODO: create this file)

### Troubleshooting

**Services won't start:**
```bash
# Check logs
docker-compose -f docker-compose.test.yml logs

# Rebuild from scratch
docker-compose -f docker-compose.test.yml down -v
docker-compose -f docker-compose.test.yml up -d --build --force-recreate
```

**Tests fail locally but pass in CI:**
- Check you're running the same version of Playwright
- Try `npm ci` instead of `npm install`
- Clear Playwright cache: `npx playwright install --force`

**Ports already in use:**
```bash
# Check what's using port 20080
lsof -i :20080

# Kill the process or change ports in docker-compose.test.yml
```

### Writing New Tests

Create a new file in `e2e/` directory:

```javascript
// e2e/my-feature.spec.js
import { test, expect } from '@playwright/test';

test('my feature works', async ({ page }) => {
  await page.goto('/my-page');
  await expect(page.locator('h1')).toContainText('Expected');
});
```

See [Playwright docs](https://playwright.dev/docs/writing-tests) for more examples.
