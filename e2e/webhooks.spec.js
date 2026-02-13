// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import fs from 'fs';
import path from 'path';
import { signInAsAdmin } from './helpers/auth.js';

/**
 * Webhook E2E Tests
 *
 * Tests the 4 essential webhook flows:
 * 1. Change webhook fires on commit with HMAC verification
 * 2. Agent response creates notes
 * 3. Depth protection prevents infinite loop
 * 4. Cron webhook fires with instruction and api_token
 */

const APP_URL = process.env.APP_URL || 'http://localhost:20081';
const GRAPHQL_URL = `${APP_URL}/graphql`;
const TEST_PATH_PREFIX = 'e2e_webhook_test';

// Helper: Execute GraphQL admin mutation
async function graphqlAdmin(request, cookie, query, variables = {}) {
  const response = await request.post(GRAPHQL_URL, {
    headers: {
      'Content-Type': 'application/json',
      'Cookie': cookie,
    },
    data: { query, variables },
  });

  expect(response.ok()).toBeTruthy();
  const result = await response.json();
  if (result.errors) {
    throw new Error(`GraphQL errors: ${JSON.stringify(result.errors)}`);
  }
  return result.data;
}

// Helper: Execute GraphQL with API key
async function graphqlApi(request, apiKey, query, variables = {}) {
  const response = await request.post(GRAPHQL_URL, {
    headers: {
      'Content-Type': 'application/json',
      'X-Api-Key': apiKey,
    },
    data: { query, variables },
  });

  expect(response.ok()).toBeTruthy();
  const result = await response.json();
  if (result.errors) {
    throw new Error(`GraphQL errors: ${JSON.stringify(result.errors)}`);
  }
  return result.data;
}

// Helper: Clear webhook calls
async function clearWebhookCalls(request) {
  const response = await request.delete(`${APP_URL}/debug/test_webhook_calls`);
  expect(response.ok()).toBeTruthy();
}

// Helper: Wait for all background jobs to complete
async function waitAllJobs(request) {
  const response = await request.get(`${APP_URL}/debug/wait_all_jobs`, {
    timeout: 60000,
  });
  expect(response.ok()).toBeTruthy();
}

// Helper: Get all webhook calls from debug endpoint
async function getWebhookCalls(request) {
  const response = await request.get(`${APP_URL}/debug/test_webhook_calls`);
  expect(response.ok()).toBeTruthy();
  return await response.json();
}

// Helper: Push notes and commit in one operation
async function pushAndCommit(request, apiKey, updates) {
  // Push notes
  const pushQuery = `
    mutation PushNotes($input: PushNotesInput!) {
      pushNotes(input: $input) {
        ... on PushNotesPayload {
          notes { id }
        }
      }
    }
  `;
  await graphqlApi(request, apiKey, pushQuery, { input: { updates } });

  // Commit notes
  const commitQuery = `
    mutation CommitNotes {
      commitNotes {
        ... on CommitNotesPayload {
          success
        }
      }
    }
  `;
  await graphqlApi(request, apiKey, commitQuery);
}

// Helper: Verify HMAC signature
function verifyHmac(secret, bodyStr, signatureHeader) {
  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(bodyStr);
  const expected = 'sha256=' + hmac.digest('hex');
  return signatureHeader === expected;
}

test.describe.serial('Webhook E2E Tests', () => {
  let adminCookie;
  let apiKey;
  let changeWebhookIds = [];
  let cronWebhookIds = [];

  test.beforeAll(async ({ browser }) => {
    // Sign in as admin
    const page = await browser.newPage();
    adminCookie = await signInAsAdmin(page);
    await page.close();

    // Read API key from file
    const apiKeyPath = path.join(process.cwd(), '.test-api-key');
    apiKey = fs.readFileSync(apiKeyPath, 'utf8').trim();
  });

  test.afterAll(async ({ request }) => {
    // Clean up all created webhooks
    for (const webhookId of changeWebhookIds) {
      try {
        const query = `
          mutation ChangeWebhookDelete($input: ChangeWebhookDeleteInput!) {
            admin {
              changeWebhookDelete(input: $input) {
                ... on ChangeWebhookDeletePayload {
                  deletedId
                }
              }
            }
          }
        `;
        await graphqlAdmin(request, adminCookie, query, { input: { id: webhookId } });
      } catch (err) {
        console.warn(`Failed to delete change webhook ${webhookId}:`, err);
      }
    }

    for (const cronWebhookId of cronWebhookIds) {
      try {
        const query = `
          mutation DeleteCronWebhook($input: DeleteCronWebhookInput!) {
            admin {
              deleteCronWebhook(input: $input) {
                ... on DeleteCronWebhookPayload {
                  deletedId
                }
              }
            }
          }
        `;
        await graphqlAdmin(request, adminCookie, query, { input: { id: cronWebhookId } });
      } catch (err) {
        console.warn(`Failed to delete cron webhook ${cronWebhookId}:`, err);
      }
    }
  });

  test.beforeEach(async ({ request }) => {
    // Clear webhook calls before each test
    await clearWebhookCalls(request);
  });

  test('change webhook fires on commit', async ({ request }) => {
    // 1. Create change webhook
    const createQuery = `
      mutation ChangeWebhookCreate($input: ChangeWebhookCreateInput!) {
        admin {
          changeWebhookCreate(input: $input) {
            ... on ChangeWebhookCreatePayload {
              webhook {
                id
              }
              secret
            }
          }
        }
      }
    `;

    const createData = await graphqlAdmin(request, adminCookie, createQuery, {
      input: {
        url: `${APP_URL}/debug/test_webhook`,
        includePatterns: ['e2e_wh_test1/**'],
        passApiKey: false,
        includeContent: true,
        maxDepth: 1,
      },
    });

    const webhookId = createData.admin.changeWebhookCreate.webhook.id;
    const secret = createData.admin.changeWebhookCreate.secret;
    changeWebhookIds.push(webhookId);

    // 2. Push a test note
    await pushAndCommit(request, apiKey, [
      {
        path: 'e2e_wh_test1/test.md',
        content: '# Test 1\n\nWebhook test content',
      },
    ]);

    // 3. Wait for webhook job to complete
    await waitAllJobs(request);

    // 4. Verify webhook was called
    const calls = await getWebhookCalls(request);
    expect(calls.length).toBe(1);

    const call = calls[0];

    // Verify headers
    expect(call.headers['X-Webhook-Signature']).toBeTruthy();
    expect(call.headers['X-Webhook-Signature']).toMatch(/^sha256=/);
    expect(call.headers['X-Webhook-Attempt']).toBe('1');
    expect(call.headers['Content-Type']).toContain('application/json');
    expect(call.headers['User-Agent']).toBe('trip2g-webhooks/1.0');

    // Verify payload structure
    expect(call.body.version).toBe(1);
    expect(call.body.depth).toBe(0);
    expect(Array.isArray(call.body.changes)).toBe(true);
    expect(call.body.changes.length).toBe(1);

    const change = call.body.changes[0];
    expect(change.path).toBe('e2e_wh_test1/test.md');
    expect(change.content).toContain('# Test 1');
    expect(['create', 'update']).toContain(change.event);

    // Verify HMAC signature
    const bodyStr = JSON.stringify(call.body);
    const isValid = verifyHmac(secret, bodyStr, call.headers['X-Webhook-Signature']);
    expect(isValid).toBe(true);
  });

  test('agent response creates notes', async ({ request }) => {
    // 1. Create webhook that returns changes
    const agentResponse = JSON.stringify({
      status: 'ok',
      changes: [
        {
          path: 'e2e_wh_test2/auto_created.md',
          content: '---\nfree: true\n---\n\n# Auto Created\n\nThis was created by the agent response.',
        },
      ],
    });

    const createQuery = `
      mutation ChangeWebhookCreate($input: ChangeWebhookCreateInput!) {
        admin {
          changeWebhookCreate(input: $input) {
            ... on ChangeWebhookCreatePayload {
              webhook {
                id
              }
              secret
            }
          }
        }
      }
    `;

    const createData = await graphqlAdmin(request, adminCookie, createQuery, {
      input: {
        url: `${APP_URL}/debug/test_webhook?body=${encodeURIComponent(agentResponse)}`,
        includePatterns: ['e2e_wh_test2/**'],
        passApiKey: true,
        includeContent: true,
        maxDepth: 1,
        writePatterns: ['e2e_wh_test2/**'],
      },
    });

    const webhookId = createData.admin.changeWebhookCreate.webhook.id;
    changeWebhookIds.push(webhookId);

    // 2. Push a trigger note
    await pushAndCommit(request, apiKey, [
      {
        path: 'e2e_wh_test2/trigger.md',
        content: '---\nfree: true\n---\n\n# Trigger\n\nThis triggers the webhook.',
      },
    ]);

    // 3. Wait for webhook job to complete
    await waitAllJobs(request);

    // 4. Verify the agent-created note exists
    const noteResponse = await request.get(`${APP_URL}/e2e_wh_test2/auto_created`);
    expect(noteResponse.status()).toBe(200);
    const noteContent = await noteResponse.text();
    expect(noteContent).toContain('Auto Created');
  });

  test('depth protection prevents infinite loop', async ({ request }) => {
    // 1. Create webhook that modifies the same path
    const agentResponse = JSON.stringify({
      status: 'ok',
      changes: [
        {
          path: 'e2e_wh_test3/depth_test.md',
          content: '---\nfree: true\n---\n\n# Modified by agent\n\nThis is the agent\'s version.',
        },
      ],
    });

    const createQuery = `
      mutation ChangeWebhookCreate($input: ChangeWebhookCreateInput!) {
        admin {
          changeWebhookCreate(input: $input) {
            ... on ChangeWebhookCreatePayload {
              webhook {
                id
              }
              secret
            }
          }
        }
      }
    `;

    const createData = await graphqlAdmin(request, adminCookie, createQuery, {
      input: {
        url: `${APP_URL}/debug/test_webhook?body=${encodeURIComponent(agentResponse)}`,
        includePatterns: ['e2e_wh_test3/**'],
        passApiKey: true,
        maxDepth: 1,
        writePatterns: ['e2e_wh_test3/**'],
      },
    });

    const webhookId = createData.admin.changeWebhookCreate.webhook.id;
    changeWebhookIds.push(webhookId);

    // 2. Push the original note
    await pushAndCommit(request, apiKey, [
      {
        path: 'e2e_wh_test3/depth_test.md',
        content: '---\nfree: true\n---\n\n# Original\n\nThis is the original content.',
      },
    ]);

    // 3. Wait for all jobs to complete
    await waitAllJobs(request);

    // 4. Verify only 1 webhook call was made (no infinite loop)
    const calls = await getWebhookCalls(request);
    expect(calls.length).toBe(1);

    // 5. Verify the agent's changes were applied
    const noteResponse = await request.get(`${APP_URL}/e2e_wh_test3/depth_test`);
    expect(noteResponse.status()).toBe(200);
    const noteContent = await noteResponse.text();
    expect(noteContent).toContain('Modified by agent');
  });

  test('cron webhook fires with instruction', async ({ request }) => {
    // 1. Create cron webhook
    const createQuery = `
      mutation CreateCronWebhook($input: CreateCronWebhookInput!) {
        admin {
          createCronWebhook(input: $input) {
            ... on CreateCronWebhookPayload {
              cronWebhook {
                id
              }
              secret
            }
          }
        }
      }
    `;

    const createData = await graphqlAdmin(request, adminCookie, createQuery, {
      input: {
        url: `${APP_URL}/debug/test_webhook`,
        cronSchedule: '0 0 1 1 *', // Never auto-fires
        instruction: 'Generate E2E test digest',
        passApiKey: true,
        readPatterns: ['*'],
        writePatterns: [`${TEST_PATH_PREFIX}/**`],
      },
    });

    const cronWebhookId = createData.admin.createCronWebhook.cronWebhook.id;
    const secret = createData.admin.createCronWebhook.secret;
    cronWebhookIds.push(cronWebhookId);

    // 2. Trigger manually
    const triggerQuery = `
      mutation TriggerCronWebhook($input: TriggerCronWebhookInput!) {
        admin {
          triggerCronWebhook(input: $input) {
            ... on TriggerCronWebhookPayload {
              deliveryId
            }
          }
        }
      }
    `;

    await graphqlAdmin(request, adminCookie, triggerQuery, {
      input: { cronWebhookId },
    });

    // 3. Wait for webhook job to complete
    await waitAllJobs(request);

    // 4. Verify webhook was called
    const calls = await getWebhookCalls(request);
    expect(calls.length).toBe(1);

    const call = calls[0];

    // Verify payload
    expect(call.body.instruction).toBe('Generate E2E test digest');
    expect(call.body.api_token).toBeTruthy();
    expect(typeof call.body.api_token).toBe('string');
    expect(call.body.version).toBe(1);
    expect(call.body.attempt).toBe(1);

    // Verify HMAC header is present
    expect(call.headers['X-Webhook-Signature']).toBeTruthy();
    expect(call.headers['X-Webhook-Signature']).toMatch(/^sha256=/);
  });
});
