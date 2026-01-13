// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Test Vault', () => {
  test('home page renders and shows all sections', async ({ page }) => {
    await page.goto('/');

    // Check main heading - use first() to avoid strict mode violation
    await expect(page.locator('h1').first()).toContainText('Test Vault');

    // Check main sections exist - use getByRole to avoid TOC link duplicates
    await expect(page.getByRole('heading', { name: 'Link Resolution Tests' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Publishing Features Tests' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Subgraph (Premium Course) Tests' })).toBeVisible();
  });

  test('public page is accessible without auth', async ({ page }) => {
    await page.goto('/public');

    await expect(page.locator('h1').first()).toContainText('Public Content');
    await expect(page.getByText('publicly accessible')).toBeVisible();
  });

  test('custom layout page uses correct template', async ({ page }) => {
    await page.goto('/with_layout');

    // Check page title
    await expect(page.locator('h1').first()).toContainText('Custom Layout');

    // Check custom layout elements
    await expect(page.locator('header nav')).toBeVisible();
    await expect(page.locator('footer')).toContainText('2025 Test Vault');

    // Check navigation links in custom layout
    await expect(page.locator('a[href="/"]')).toContainText('Home');

    // Check that custom CSS is loaded (text should be green)
    await expect(page.locator('body')).toHaveCSS('color', 'rgb(0, 255, 0)');
  });

  test('premium content preview works', async ({ page }) => {
    await page.goto('/paid_with_preview');

    // Check title is visible
    await expect(page.locator('#noteview-content h1').first()).toContainText('Premium Content with Preview');

    // Page without free flag should show subscription message
    await expect(page.getByText('Эта страница доступна только для подписчиков')).toBeVisible();
  });

  test('table of contents page renders', async ({ page }) => {
    await page.goto('/toc_test');

    // No title set, so it uses filename in lowercase
    await expect(page.locator('h1').first()).toContainText('toc_test');

    // Check sections exist
    // TODO: check better
    // await expect(page.getByRole('heading', { name: 'Section 1' })).toBeVisible();
    // await expect(page.getByRole('heading', { name: 'Section 2' })).toBeVisible();
    // await expect(page.getByRole('heading', { name: 'Section 3' })).toBeVisible();
  });

  test('cyrillic URLs work correctly', async ({ page }) => {
    await page.goto('/cyrillic_названия');

    // Title is set in frontmatter
    await expect(page.locator('h1').first()).toContainText('Проверка кириллицы');
    await expect(page.getByText('кириллическими ссылками')).toBeVisible();
  });

  test('files with spaces in names work', async ({ page }) => {
    await page.goto('/file_with_spaces');

    await expect(page.locator('h1').first()).toContainText('File Name With Spaces');
    await expect(page.getByText('URL normalization')).toBeVisible();
  });

  test('code and media page renders', async ({ page }) => {
    await page.goto('/code_and_media');

    await expect(page.locator('h1').first()).toContainText('Code and Media');

    // Check content exists
    await expect(page.getByText('code blocks and media embeds')).toBeVisible();
  });

  test('complex content with markdown features', async ({ page }) => {
    await page.goto('/complex_content');

    // Check title is visible
    await expect(page.locator('#noteview-content h1').first()).toContainText('Complex Content Example');

    // Page without free flag should show subscription message
    await expect(page.getByText('Эта страница доступна только для подписчиков')).toBeVisible();
  });

  test('navigation between pages works', async ({ page }) => {
    await page.goto('/');

    // Click on a link
    await page.click('text=public');

    // Verify navigation happened
    await expect(page).toHaveURL(/\/public/);
    await expect(page.locator('h1').first()).toContainText('Public Content');
  });

  test('premium course home page', async ({ page }) => {
    await page.goto('/premium');

    // Check title is visible
    await expect(page.locator('.sidebar__homepage a').first()).toContainText('Premium Course Home');

    // Page without free flag should show subscription message
    await expect(page.getByText('Эта страница доступна только для подписчиков')).toBeVisible();
  });
});

test.describe('Link Resolution', () => {
  test('unique filename resolution', async ({ page }) => {
    await page.goto('/unique');

    // No title set, uses filename
    await expect(page.locator('h1').first()).toContainText('unique');
    await expect(page.getByText('should find /folder/deep.md')).toBeVisible();
  });

  test('duplicate filename priority (root wins)', async ({ page }) => {
    await page.goto('/folder/source');

    // No title set, uses filename
    await expect(page.locator('h1').first()).toContainText('source');

    // Check that link to 'dup' resolves to root /dup, not local /folder/dup
    // This verifies Obsidian's global resolution with root priority
    await expect(page.locator('a[href="/dup?version=latest"]')).toBeVisible();
  });

  test('headers and block references', async ({ page }) => {
    await page.goto('/headers');

    // No title set, uses filename
    await expect(page.locator('h1').first()).toContainText('headers');
    await expect(page.getByRole('heading', { name: 'Section One' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Section Two' })).toBeVisible();
  });
});

test.describe('Special Features', () => {
  test('redirect works', async ({ page }) => {
    await page.goto('/redirect_test');

    // Should redirect to /public
    await page.waitForURL(/\/public/);
    await expect(page.locator('h1').first()).toContainText('Public Content');
  });

  test('embedding markdown works', async ({ page }) => {
    await page.goto('/embedding');

    // No title set, uses filename
    await expect(page.locator('h1').first()).toContainText('embedding');

    // Check that embedded banners appear exactly once
    await expect(page.getByText("I'm the ROOT banner at /_banner.md")).toHaveCount(1);
    await expect(page.getByText("I'm the banner at /projectA/_banner.md")).toHaveCount(1);
    await expect(page.getByText("I'm the banner at /projectB/_banner.md")).toHaveCount(1);
  });
});

test.describe('Custom Slug URL Override', () => {
  test('home page has no WIP links', async ({ page }) => {
    await page.goto('/');

    // All links on home page should resolve - no WIP links
    const wipLinks = await page.locator('a.wip').count();
    expect(wipLinks).toBe(0);
  });

  test('relative slug replaces filename only', async ({ page }) => {
    await page.goto('/custom-name');

    await expect(page.locator('h1').first()).toContainText('Relative Slug Test');
    await expect(page.getByText('Expected URL: /custom-name')).toBeVisible();
  });

  test('relative slug in nested folder preserves directory', async ({ page }) => {
    await page.goto('/folder/my-custom-page');

    await expect(page.locator('h1').first()).toContainText('Nested Relative Slug');
    await expect(page.getByText('Expected URL: /folder/my-custom-page')).toBeVisible();
  });

  test('absolute slug overrides full path', async ({ page }) => {
    await page.goto('/archive/old-post');

    await expect(page.locator('h1').first()).toContainText('Absolute Slug Test');
    await expect(page.getByText('Expected URL: /archive/old-post')).toBeVisible();
  });

  test('relative slug with subdirectory creates nested path', async ({ page }) => {
    await page.goto('/sub/nested/page');

    await expect(page.locator('h1').first()).toContainText('Slug with Subdirectory');
    await expect(page.getByText('Expected URL: /sub/nested/page')).toBeVisible();
  });

  test('cyrillic slug without transliteration', async ({ page }) => {
    // URL-encoded cyrillic
    await page.goto('/%D0%BC%D0%BE%D1%8F-%D1%81%D1%82%D1%80%D0%B0%D0%BD%D0%B8%D1%86%D0%B0');

    await expect(page.locator('h1').first()).toContainText('Cyrillic Slug');
    await expect(page.getByText('no transliteration')).toBeVisible();
  });

  test('slug with spaces is URL encoded', async ({ page }) => {
    await page.goto('/page%20with%20spaces');

    await expect(page.locator('h1').first()).toContainText('Slug with Spaces');
    await expect(page.getByText('URL encoded')).toBeVisible();
  });
});

test.describe('Regression Tests', () => {
  test('image with same name as note renders correctly', async ({ page }) => {
    // Bug: ![[software.png]] was resolved as /software (the note) when software.md exists
    await page.goto('/software');

    await expect(page.locator('h1').first()).toContainText('Software Page');

    // The image should be rendered as <img>, not cause a render error
    const image = page.locator('#noteview-content img');
    await expect(image).toHaveCount(1);
    await expect(image).toBeVisible();

    // Image src should be software.png, not /software
    const src = await image.getAttribute('src');
    expect(src).toContain('software.png');
    expect(src).not.toContain('?version'); // Images don't get version parameter
  });

  test('links with dots in filenames resolve correctly in embeds', async ({ page }) => {
    // Bug: filepath.Ext("Сценарий. Ютубер") returns ". Ютубер" as extension
    await page.goto('/scenarios_test');

    await expect(page.locator('h1').first()).toContainText('Scenarios Test');

    // Check that embedded content is present
    await expect(page.getByText('Ютубер')).toBeVisible();
    await expect(page.getByText('Курсы')).toBeVisible();

    // Links should NOT be marked as wip since target pages exist
    const wipLinks = page.locator('#noteview-content a.wip');
    await expect(wipLinks).toHaveCount(0);

    // Links should have data-pid (proving pages were found)
    const links = page.locator('#noteview-content a[data-pid]');
    await expect(links).toHaveCount(2);
  });

  test('pages with dots in names are accessible directly', async ({ page }) => {
    // Navigate to page with dot in name
    await page.goto('/scenarij_yutuber');

    await expect(page.locator('h1').first()).toContainText('Сценарий Ютубер');
    await expect(page.getByText('сценарием для ютуберов')).toBeVisible();
  });
});

test.describe('CLI Meta Injection', () => {
  test('meta injection adds title to page without frontmatter', async ({ page }) => {
    // This page was synced from $VAULT0/cli_meta with --meta title=FromCLI
    // Path is relative to sync folder, so it's /cli_test not /cli_meta/cli_test
    await page.goto('/cli_test');

    // Title should be "FromCLI" (from injected frontmatter title field)
    await expect(page.locator('h1').first()).toContainText('FromCLI');

    // Content should be present
    await expect(page.getByText('This page was synced with --meta')).toBeVisible();
  });
});

test.describe('Image Resolution', () => {
  test('image resolution with duplicates - root priority', async ({ page }) => {
    await page.goto('/img_test');

    // No h1 title in markdown, uses filename
    await expect(page.locator('h1').first()).toContainText('img-test');

    // Check all images are loaded
    const images = page.locator('#noteview-content img');
    await expect(images).toHaveCount(3);

    // Wait for images to load
    await images.first().waitFor({ state: 'visible' });

    // Get src for all images
    const src1 = await images.nth(0).getAttribute('src'); // test.png -> /test.png
    const src2 = await images.nth(1).getAttribute('src'); // assets/test.png -> /assets/test.png
    const src3 = await images.nth(2).getAttribute('src'); // folder/test.png -> /folder/test.png

    // All src should be different (different files were resolved)
    expect(src1).not.toBe(src2);
    expect(src1).not.toBe(src3);
    expect(src2).not.toBe(src3);

    // All should be valid URLs
    expect(src1).toBeTruthy();
    expect(src2).toBeTruthy();
    expect(src3).toBeTruthy();
  });

  test('image formats - png, jpg, webp, svg', async ({ page }) => {
    await page.goto('/img_formats');

    // No h1 title in markdown, uses filename
    await expect(page.locator('h1').first()).toContainText('img-formats');

    // Check all 4 format images are loaded
    const images = page.locator('#noteview-content img');
    await expect(images).toHaveCount(4);

    // Wait for images to load
    await images.first().waitFor({ state: 'visible' });

    // Get all src
    const srcs = [];
    for (let i = 0; i < 4; i++) {
      const src = await images.nth(i).getAttribute('src');
      expect(src).toBeTruthy();
      srcs.push(src);
    }

    // All src should be different (different formats)
    const uniqueSrcs = new Set(srcs);
    expect(uniqueSrcs.size).toBe(4);
  });

  test('image resolution from folder - verifies root priority', async ({ page }) => {
    await page.goto('/folder/imgs');

    // No h1 title in markdown, uses filename
    await expect(page.locator('h1').first()).toContainText('imgs');

    // Check all images are loaded (4 root + 2 folder = 6)
    const images = page.locator('#noteview-content img');
    await expect(images).toHaveCount(6);

    await images.first().waitFor({ state: 'visible' });

    // Get all src
    const srcs = [];
    for (let i = 0; i < 6; i++) {
      const src = await images.nth(i).getAttribute('src');
      expect(src).toBeTruthy();
      srcs.push(src);
    }

    // All 6 src should be different
    // This proves:
    // - First 4 resolved to ROOT (different from folder versions)
    // - Last 2 resolved to folder/ (explicit paths)
    const uniqueSrcs = new Set(srcs);
    expect(uniqueSrcs.size).toBe(6);
  });

  test('image resolution from projectA', async ({ page }) => {
    await page.goto('/projecta/imgs');

    // No h1 title in markdown, uses filename
    await expect(page.locator('h1').first()).toContainText('imgs');

    // Check both images are loaded
    const images = page.locator('#noteview-content img');
    await expect(images).toHaveCount(2);

    await images.first().waitFor({ state: 'visible' });

    const src0 = await images.nth(0).getAttribute('src'); // format.jpg -> root
    const src1 = await images.nth(1).getAttribute('src'); // projectA/format.jpg

    // Both should be valid and different
    expect(src0).toBeTruthy();
    expect(src1).toBeTruthy();
    expect(src0).not.toBe(src1); // Root vs projectA file
  });
});
