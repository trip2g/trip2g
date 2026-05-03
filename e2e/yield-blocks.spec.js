// @ts-check
import { test, expect } from '@playwright/test';

test.describe('yield_blocks', () => {
  test('renders component HTML', async ({ page }) => {
    await page.goto('/yield_blocks_demo');
    await expect(page.locator('.yb-hero__title')).toContainText('yield_blocks Demo');
    await expect(page.locator('.yb-card').first()).toBeVisible();
    await expect(page.locator('.yb-button').first()).toBeVisible();
  });

  test('injects CSS for used components', async ({ page }) => {
    await page.goto('/yield_blocks_demo');
    const styleContent = await page.locator('#yield-blocks-styles').textContent();
    expect(styleContent).toContain('.yb-hero');
    expect(styleContent).toContain('.yb-card');
    expect(styleContent).toContain('.yb-button');
  });

  test('applies styles correctly', async ({ page }) => {
    await page.goto('/yield_blocks_demo');
    const hero = page.locator('.yb-hero');
    await expect(hero).toBeVisible();
    await expect(page.locator('.yb-button--primary')).toHaveCSS('background-color', 'rgb(0, 112, 243)');
  });
});
