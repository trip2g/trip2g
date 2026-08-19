// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Header user-space slot', () => {
  // The slot is a $mol view root. $mol's base stylesheet sets
  // [mol_view_root] { width: 100%; height: 100% }, and the component's own
  // stylesheet brings it back to auto — both are injected at runtime by web.js.
  // While only the base rule is in effect, the slot claims the whole header row
  // and .site-header__nav collapses to min-content, stacking the links into a
  // column. Disabling the component's stylesheets reproduces that window.
  test('keeps its width when the component stylesheet is missing', async ({ page }) => {
    await page.goto('/');

    const space = page.locator('.site-header__space');
    await expect(space).toBeVisible();
    await expect
      .poll(() => space.evaluate((el) => el.getBoundingClientRect().width))
      .toBeGreaterThan(0);

    const size = await page.evaluate(() => {
      const slot = document.querySelector('.site-header__space');
      const header = document.querySelector('.site-header');
      const measure = () => ({
        slot: Math.round(slot.getBoundingClientRect().width),
        header: Math.round(header.getBoundingClientRect().height),
      });

      const before = measure();
      for (const sheet of document.styleSheets) {
        let rules = [];
        try {
          rules = [...sheet.cssRules];
        } catch {
          continue;
        }
        if (rules.some((r) => r.selectorText?.includes('trip2g_user_space'))) {
          sheet.disabled = true;
        }
      }
      return { before, after: measure() };
    });

    expect(size.after.slot).toBe(size.before.slot);
    expect(size.after.header).toBe(size.before.header);
  });
});
