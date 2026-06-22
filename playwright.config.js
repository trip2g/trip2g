// @ts-check
import { defineConfig, devices } from '@playwright/test';

/**
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './e2e',

  /* Specs that modify live releases and must run in isolation (via test-e2e.sh).
     testIgnore applies even when the file is named on the CLI, so the isolated
     run must set RUN_ISOLATED_SPECS=1 to be able to select it. */
  testIgnore: process.env.RUN_ISOLATED_SPECS ? [] : ['**/unreleased-changes.spec.js', '**/show-draft-versions.spec.js'],

  /* Run tests in files in parallel */
  fullyParallel: true,

  /* Fail the build on CI if you accidentally left test.only in the source code */
  forbidOnly: !!process.env.CI,

  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,

  /* Opt out of parallel tests on CI */
  workers: process.env.CI ? 1 : undefined,

  /* Reporter to use. open:'never' so a failing local run exits instead of
     blocking on the served HTML report. */
  reporter: process.env.CI ? 'github' : [['list'], ['html', { open: 'never' }]],

  /* Shared settings for all the projects below */
  use: {
    /* Base URL for tests */
    baseURL: process.env.APP_URL || 'http://localhost:8081',

    /* Collect trace when retrying the failed test */
    trace: 'on-first-retry',

    /* Screenshot on failure */
    screenshot: 'only-on-failure',

    /* Video on failure */
    video: 'retain-on-failure',
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  /* Run local dev server before starting the tests */
  // webServer: {
  //   command: './scripts/start-test-env.sh',
  //   url: 'http://localhost:20080',
  //   reuseExistingServer: !process.env.CI,
  //   timeout: 120 * 1000,
  // },
});
