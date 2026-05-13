// @ts-check
import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';
import fs from 'fs';
import os from 'os';
import path from 'path';

const APP_URL = process.env.APP_URL || 'http://localhost:8081';
const GRAPHQL_URL = `${APP_URL}/_system/graphql`;

test.describe('Body size limit', () => {
  test('GraphQL request exceeding body limit returns GraphQL-formatted 413 error', async () => {
    // Generate a payload larger than the server body limit (10 MB default)
    const tmpFile = path.join(os.tmpdir(), `body-limit-test-${Date.now()}.json`);
    const padding = 'x'.repeat(11 * 1024 * 1024);
    fs.writeFileSync(tmpFile, JSON.stringify({
      query: 'query { __typename }',
      variables: { padding },
    }));

    // We use curl instead of Playwright's request API because fasthttp closes
    // the connection (TCP RST) while the client is still uploading the body.
    // Higher-level HTTP clients (Playwright, Node http) get ECONNRESET before
    // reading the response. curl handles early server responses gracefully.
    try {
      const result = execSync(
        `curl -s -w '\\n%{http_code}' -X POST '${GRAPHQL_URL}' ` +
        `-H 'Content-Type: application/json' -d @${tmpFile}`,
        { encoding: 'utf-8', timeout: 30_000 },
      );

      const lines = result.trimEnd().split('\n');
      const statusCode = parseInt(lines.pop(), 10);
      const body = lines.join('\n');

      expect(statusCode).toBe(413);

      const json = JSON.parse(body);
      expect(json).toHaveProperty('errors');
      expect(json.errors).toHaveLength(1);
      expect(json.errors[0].message).toContain('body size exceeds');
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });
});
