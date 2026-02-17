// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

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
    await expect(page.locator('a[href="/dup"]')).toBeVisible();
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
  test('meta injection adds title to page with prefix', async ({ page }) => {
    // This page was synced with prefix "cli_meta" and --meta title=FromCLI
    // So it's uploaded as cli_meta/cli_test.md
    await page.goto('/cli_meta/cli_test');

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

test.describe('Broken Layout Handling', () => {
  test('guest sees default rendering for page with broken layout', async ({ request }) => {
    // Use raw HTTP request without any cookies (guest mode)
    const response = await request.fetch('/broken_layout_test', {
      headers: {
        'Cookie': '', // Explicitly clear cookies
      },
    });

    expect(response.ok()).toBeTruthy();

    const html = await response.text();

    // Guest should see the page content
    expect(html).toContain('Broken Layout Test Page');
    expect(html).toContain('This page uses a broken layout');

    // No error message for guests (should render with default layout)
    expect(html).not.toContain('Layout Error');
  });

  test('admin sees error message for page with broken layout', async ({ page }) => {
    // 1. Sign in as admin
    await page.goto('/');

    await page.locator('mol_button_minor[trip2g_user_space_signinbutton]').click();
    await page.locator('input[trip2g_auth_email_form_email_control]').fill('hello@example.com');
    await page.locator('button[trip2g_auth_email_form_requestcode]').click();
    await page.locator('input[trip2g_auth_code_form_code_control]').waitFor({ state: 'visible' });
    await page.locator('input[trip2g_auth_code_form_code_control]').clear();
    await page.locator('input[trip2g_auth_code_form_code_control]').fill('111111');
    await page.locator('mol_button_major[trip2g_auth_code_form_signup]').click();
    await page.waitForTimeout(500);

    // 2. Navigate to page with broken layout
    await page.goto('/broken_layout_test');

    // Admin should see error message
    await expect(page.getByText('Layout Error: broken-layout')).toBeVisible();

    // Error details should be visible
    await expect(page.getByText('parsing')).toBeVisible();

    // Content should still be visible below the error
    await expect(page.getByText('This page uses a broken layout')).toBeVisible();
  });
});

test.describe('Custom Telegram Emoji', () => {
  test('custom emoji images render inline at 20x20', async ({ page }) => {
    await page.goto('/telegram_image_with_emoji');

    await expect(page.locator('h1').first()).toContainText('Image with Custom Emoji');

    // Find all custom-emoji images
    const emojiImgs = page.locator('img.custom-emoji');
    await expect(emojiImgs).toHaveCount(2);

    // Both should have 20x20 size attributes
    for (let i = 0; i < 2; i++) {
      const img = emojiImgs.nth(i);
      await expect(img).toHaveAttribute('width', '20');
      await expect(img).toHaveAttribute('height', '20');
    }

    // Verify actual rendered size is small (not block-level)
    const box = await emojiImgs.first().boundingBox();
    expect(box.width).toBeLessThanOrEqual(25);
    expect(box.height).toBeLessThanOrEqual(25);
  });

  test('regular images do NOT have custom-emoji class', async ({ page }) => {
    await page.goto('/telegram_image_with_emoji');

    // The main photo should not have custom-emoji class
    const allImgs = page.locator('#noteview-content img');
    const regularImgs = page.locator('#noteview-content img:not(.custom-emoji)');

    // Total images: 1 regular photo + 2 custom emoji = 3
    await expect(allImgs).toHaveCount(3);
    await expect(regularImgs).toHaveCount(1);
  });
});

test.describe('YouTube Embed', () => {
  test('YouTube link renders as embedded iframe', async ({ page }) => {
    await page.goto('/code_and_media');

    await expect(page.locator('h1').first()).toContainText('Code and Media');

    // YouTube embed iframe should be visible
    const iframe = page.locator('iframe.youtube-enclave-object');
    await expect(iframe).toBeVisible();

    const src = await iframe.getAttribute('src');
    expect(src).toContain('youtube.com/embed/');
    expect(src).toContain('dQw4w9WgXcQ'); // Video ID from test page
  });

  test('YouTube embed is wrapped in proper container', async ({ page }) => {
    await page.goto('/code_and_media');

    // Check the wrapper structure
    const wrapper = page.locator('.enclave-object-wrapper');
    await expect(wrapper).toBeVisible();

    // Should have auto-resize class for responsive sizing
    await expect(wrapper).toHaveClass(/auto-resize/);
  });
});

test.describe('Frontmatter Patches', () => {
  test.beforeAll(async ({ request, browser }) => {
    // Sign in as admin via GraphQL API
    const token = await graphqlSignIn(request);

    // Create a new context with the auth cookie
    const context = await browser.newContext({
      baseURL: 'http://localhost:20081',
      extraHTTPHeaders: {
        'Cookie': `trip2g_e2e=${token}`
      }
    });

    // Create a request context from this context that will include cookies
    const authenticatedRequest = context.request;

    // Create test patches
    const patches = [
      {
        description: 'Make blog posts free',
        includePatterns: ['patch_tests/simple.md'],
        excludePatterns: [],
        jsonnet: '{ free: true }',
        priority: 0,
        enabled: true
      },
      {
        description: 'Add default layout',
        includePatterns: ['patch_tests/*'],
        excludePatterns: ['patch_tests/excluded.md'],
        jsonnet: 'if std.objectHas(meta, "layout") then {} else { layout: "default" }',
        priority: 0,
        enabled: true
      },
      {
        description: 'Override blog layout',
        includePatterns: ['patch_tests/chained.md'],
        excludePatterns: [],
        jsonnet: '{ layout: "blog_layout" }',
        priority: 10,
        enabled: true
      },
      {
        description: 'Set chained free',
        includePatterns: ['patch_tests/chained.md'],
        excludePatterns: [],
        jsonnet: '{ free: true }',
        priority: 20,
        enabled: true
      },
      {
        description: 'Site title suffix',
        includePatterns: ['patch_tests/title_template.md'],
        excludePatterns: [],
        jsonnet: 'meta + { title: meta.title + " — Test Site" }',
        priority: 100,
        enabled: true
      },
      {
        description: 'Path-based logic',
        includePatterns: ['patch_tests/*'],
        excludePatterns: [],
        jsonnet: 'if std.startsWith(path, "patch_tests/") then { patch_applied: true } else {}',
        priority: 0,
        enabled: true
      }
    ];

    for (const patch of patches) {
      const createResponse = await authenticatedRequest.post('/graphql', {
        data: {
          query: `
            mutation CreatePatch($input: CreateFrontmatterPatchInput!) {
              admin {
                data: createFrontmatterPatch(input: $input) {
                  __typename
                  ... on CreateFrontmatterPatchPayload {
                    frontmatterPatch {
                      id
                      description
                    }
                  }
                  ... on ErrorPayload {
                    message
                  }
                }
              }
            }
          `,
          variables: {
            input: patch
          }
        }
      });

      // Check response and show error details if failed
      if (!createResponse.ok()) {
        const errorText = await createResponse.text();
        throw new Error(`Create patch failed (${createResponse.status()}): ${errorText}`);
      }

      const data = await createResponse.json();
      if (data.data?.admin?.data?.__typename === 'ErrorPayload') {
        throw new Error(`Create patch GraphQL error: ${data.data.admin.data.message}`);
      }
    }

    // Clean up context after creating patches
    await context.close();

    // Notes are automatically reloaded after patch creation
  });

  test('simple patch applies free: true', async ({ page }) => {
    await page.goto('/patch_tests/simple');

    await expect(page.locator('h1').first()).toContainText('Simple Patch Test');

    // Page should be free (no subscription message)
    await expect(page.getByText('Эта страница доступна только для подписчиков')).not.toBeVisible();

    // Content should be visible
    await expect(page.getByText('simple frontmatter patch')).toBeVisible();
  });

  test('chained patches apply in priority order', async ({ page }) => {
    await page.goto('/patch_tests/chained');

    await expect(page.locator('h1').first()).toContainText('Chained Patch Test');

    // Page should be free (priority 20 patch)
    await expect(page.getByText('Эта страница доступна только для подписчиков')).not.toBeVisible();

    // Content should be visible
    await expect(page.getByText('patch chaining with different priorities')).toBeVisible();

    // Layout should be blog_layout (priority 10 overrides priority 0)
    // This would require checking metadata via GraphQL or DOM inspection
  });

  test('conditional patch adds layout only when missing', async ({ page }) => {
    await page.goto('/patch_tests/conditional');

    await expect(page.locator('h1').first()).toContainText('Conditional Patch Test');

    // Page should be free (from frontmatter)
    await expect(page.getByText('Эта страница доступна только для подписчиков')).not.toBeVisible();

    // Content should be visible
    await expect(page.getByText('conditional jsonnet logic')).toBeVisible();
  });

  test('conditional patch does not override existing layout', async ({ page }) => {
    await page.goto('/patch_tests/has_layout');

    await expect(page.locator('h1').first()).toContainText('Has Layout Test');

    // Page should be free
    await expect(page.getByText('Эта страница доступна только для подписчиков')).not.toBeVisible();

    // Content should be visible - use partial text match to avoid backtick issues
    await expect(page.getByText('already has', { exact: false })).toBeVisible();
    await expect(page.getByText('layout: custom')).toBeVisible();
  });

  test('excluded pattern prevents patch application', async ({ page }) => {
    await page.goto('/patch_tests/excluded');

    await expect(page.locator('h1').first()).toContainText('Excluded Patch Test');

    // Page should NOT be free (patch was excluded)
    await expect(page.getByText('Эта страница доступна только для подписчиков')).toBeVisible();

    // free field should remain false (original value)
  });

  test('title template patch merges with original title', async ({ page }) => {
    await page.goto('/patch_tests/title_template');

    // Title should have suffix appended by patch
    await expect(page.locator('h1').first()).toContainText('Title Template Test — Test Site');

    // Page should be free
    await expect(page.getByText('Эта страница доступна только для подписчиков')).not.toBeVisible();

    // Content should be visible
    await expect(page.getByText('title template patch')).toBeVisible();
  });

  test('path-based logic adds custom field', async ({ request }) => {
    // Page uses meta_inspector layout which outputs raw meta as JSON
    const response = await request.get('/patch_tests/path_based');
    expect(response.ok()).toBeTruthy();

    const meta = await response.json();

    // patch_applied should be true (added by path-based patch)
    expect(meta.patch_applied).toBe(true);

    // free should still be true (from frontmatter)
    expect(meta.free).toBe(true);
  });

  test('verify all patch test pages are accessible', async ({ page }) => {
    // Navigate to home page and verify patch test links exist
    await page.goto('/');

    // Check that all patch test links are present and not WIP
    const patchTestLinks = [
      'patch_tests/simple',
      'patch_tests/chained',
      'patch_tests/conditional',
      'patch_tests/has_layout',
      'patch_tests/excluded',
      'patch_tests/title_template',
      'patch_tests/path_based'
    ];

    for (const link of patchTestLinks) {
      const linkElement = page.locator(`a[href="/${link}"]`);
      await expect(linkElement).toBeVisible();

      // Link should not be marked as WIP (page exists)
      await expect(linkElement).not.toHaveClass(/wip/);
    }
  });
});
