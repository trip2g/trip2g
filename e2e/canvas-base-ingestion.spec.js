// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Canvas and Base file ingestion', () => {
  test('.canvas URL returns 200 with unsupported page', async ({ request }) => {
    const response = await request.get('/telegramnavigation/demo.canvas');
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('Canvas files are not supported yet.');
  });

  test('.base URL returns 200 with unsupported page', async ({ request }) => {
    const response = await request.get('/example_base.base');
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('Bases are not supported yet.');
  });

  test('.excalidraw URL returns 200 with unsupported page', async ({ request }) => {
    const response = await request.get('/example.excalidraw');
    expect(response.status()).toBe(200);
    const body = await response.text();
    expect(body).toContain('Excalidraw files are not supported yet.');
  });

  test('.md URL still returns normal note page', async ({ request }) => {
    const response = await request.get('/public');
    expect(response.status()).toBe(200);
    const body = await response.text();
    // Normal note page: no unsupported-file message
    expect(body).not.toContain('Canvas files are not supported yet.');
    expect(body).not.toContain('Bases are not supported yet.');
    expect(body).not.toContain('Excalidraw files are not supported yet.');
  });
});
