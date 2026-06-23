// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Obsidian Callouts', () => {
  test('standard callout renders container and type modifier class', async ({ page }) => {
    await page.goto('/callouts');

    // A note callout: <div class="callout callout--note">
    const noteCallout = page.locator('.callout--note').first();
    await expect(noteCallout).toBeVisible();
    await expect(noteCallout).toHaveClass(/callout/);
  });

  test('warning and danger callouts have type modifier classes', async ({ page }) => {
    await page.goto('/callouts');

    await expect(page.locator('.callout--warning')).toBeVisible();
    await expect(page.locator('.callout--danger')).toBeVisible();
  });

  test('custom title renders inside callout__title span', async ({ page }) => {
    await page.goto('/callouts');

    // The info callout has custom title "Getting Started"
    const infoCallout = page.locator('.callout--info');
    await expect(infoCallout.locator('.callout__title')).toContainText('Getting Started');
  });

  test('collapsed foldable is a <details> without open attribute', async ({ page }) => {
    await page.goto('/callouts');

    // [!tip]- renders as <details class="callout callout--tip"> (no open)
    const collapsed = page.locator('details.callout--tip');
    await expect(collapsed).toBeVisible();
    await expect(collapsed).not.toHaveAttribute('open');
  });

  test('expanded foldable is a <details open>', async ({ page }) => {
    await page.goto('/callouts');

    // [!faq]+ renders as <details open class="callout callout--faq">
    const expanded = page.locator('details.callout--faq');
    await expect(expanded).toBeVisible();
    await expect(expanded).toHaveAttribute('open');
  });

  test('foldable callout uses <summary> for the header', async ({ page }) => {
    await page.goto('/callouts');

    // Both foldable callouts use <summary class="callout__header">
    const summaries = page.locator('details .callout__header');
    await expect(summaries).toHaveCount(2);
  });

  test('nested markdown inside callout body renders correctly', async ({ page }) => {
    await page.goto('/callouts');

    // The nested-markdown note contains a list and bold text
    const noteCallouts = page.locator('.callout--note');
    const lastNote = noteCallouts.last();
    await expect(lastNote.locator('.callout__body li')).toHaveCount(3);
    await expect(lastNote.locator('.callout__body strong')).toContainText('Bold text');
  });

  test('unknown callout type still renders as a callout', async ({ page }) => {
    await page.goto('/callouts');

    // [!my-custom-type] → <div class="callout callout--my-custom-type">
    const customCallout = page.locator('.callout--my-custom-type');
    await expect(customCallout).toBeVisible();
    await expect(customCallout).toHaveClass(/callout/);
  });
});
