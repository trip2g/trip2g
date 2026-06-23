// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Obsidian %% comment stripping', () => {
  test('inline comment: surrounding text is visible, comment text is absent', async ({ page }) => {
    await page.goto('/obsidian-comments');

    // The text on both sides of the inline comment must be present
    await expect(page.locator('body')).toContainText('BEFORE');
    await expect(page.locator('body')).toContainText('AFTER');

    // The comment content itself must not appear in the rendered page
    const bodyText = await page.locator('body').textContent();
    expect(bodyText).not.toContain('this comment must not appear');
  });

  test('block comment: surrounding paragraphs are visible, block content is absent', async ({ page }) => {
    await page.goto('/obsidian-comments');

    // Paragraphs surrounding the block comment must be present
    await expect(page.locator('body')).toContainText('Text before the block comment.');
    await expect(page.locator('body')).toContainText('Text after the block comment.');

    // The block comment content must not appear
    const bodyText = await page.locator('body').textContent();
    expect(bodyText).not.toContain('This entire block comment must not appear in output.');
    expect(bodyText).not.toContain('Multiple lines are hidden.');
  });

  test('code block: literal %% inside fenced code is preserved', async ({ page }) => {
    await page.goto('/obsidian-comments');

    // The literal %% inside a fenced code block must survive
    await expect(page.locator('pre code')).toContainText('%%');
    await expect(page.locator('pre code')).toContainText('this literal percent-percent inside a code block must be preserved');
  });
});
