# E2E Tests

End-to-end tests for Trip2g using Playwright.

## Quick Start

```bash
# Run all E2E tests (setup + data push + tests)
./scripts/test-e2e.sh

# Run with UI (interactive mode)
./scripts/test-e2e.sh --ui

# Run in headed mode (see browser)
./scripts/test-e2e.sh --headed

# Run in debug mode
./scripts/test-e2e.sh --debug
```

## Test Flow

1. **Setup** (`e2e/setup.spec.js`):
   - Signs in via UI
   - Creates API key through admin panel
   - Saves API key to `.test-api-key`

2. **Data Push**:
   - Runs `scripts/push_notes.py testdata/vault`
   - Uses API key from setup step
   - Uploads all vault content to test instance

3. **Main Tests** (`e2e/*.spec.js`):
   - Tests all vault pages
   - Verifies link resolution
   - Checks layouts and navigation

## Manual Testing

If you need to test manually:

```bash
# 1. Start test environment
docker-compose -f docker-compose.test.yml up -d --build

# 2. Wait for services
./scripts/waitfor.sh

# 3. Run tests
npx playwright test

# 4. Cleanup
docker-compose -f docker-compose.test.yml down -v
```

## Services

When running, the following services are available:

- **App**: http://localhost:20080
- **App Health**: http://localhost:20082/health
- **MinIO Console**: http://localhost:20001 (user: testuser, password: testpassword)
- **MinIO API**: http://localhost:20000

## Architecture

- **ARM Linux**: Builds natively for arm64
- **x64 CI**: Builds natively for amd64
- Docker automatically selects the correct architecture

## Test Coverage

The vault tests cover:

✅ Link resolution (unique, duplicates, relative paths)
✅ Markdown embeds
✅ Free content (free, free_paragraphs, free_cut)
✅ Subgraphs and premium content
✅ Custom layouts and templates
✅ Table of contents
✅ Cyrillic and special characters
✅ Code blocks and media
✅ Redirects
✅ Headers and block references
✅ Navigation between pages

## Writing Tests

Add new test files to `e2e/*.spec.js`:

```javascript
import { test, expect } from '@playwright/test';

test('my test', async ({ page }) => {
  await page.goto('/my-page');
  await expect(page.locator('h1')).toContainText('Expected Text');
});
```

See [Playwright documentation](https://playwright.dev/docs/intro) for more examples.

## CI/CD

Tests run automatically in GitHub Actions on every push and PR.
See `.github/workflows/e2e.yml` for configuration.
