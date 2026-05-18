// @ts-check
import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';

const GRAPHQL_URL = '/_system/graphql';

async function gqlApi(request, apiKey, query, variables = {}) {
  const res = await request.post(GRAPHQL_URL, {
    headers: { 'Content-Type': 'application/json', 'X-Api-Key': apiKey },
    data: { query, variables },
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

const UPDATE_NOTES = `
  mutation UpdateNotes($input: UpdateNotesInput!) {
    updateNotes(input: $input) {
      ... on UpdateNotesSuccessPayload { paths }
      ... on UpdateNotesHashMismatchPayload { path actualHash }
      ... on UpdateNotesPatchNotFoundPayload { path find }
      ... on ErrorPayload { message }
    }
  }
`;

test.describe('updateNotes mutation', () => {
  let apiKey;

  test.beforeAll(() => {
    const apiKeyPath = path.join(process.cwd(), '.test-api-key');
    apiKey = fs.readFileSync(apiKeyPath, 'utf8').trim();
  });

  test('upsert: creates a new note', async ({ request }) => {
    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: {
        changes: [{ upsert: { path: 'updatenotes_upsert_test.md', content: '# Hello\n\nCreated by e2e.' } }],
      },
    });
    const payload = data.updateNotes;
    expect(payload.paths).toContain('updatenotes_upsert_test.md');
  });

  test('upsert: updates existing note content', async ({ request }) => {
    const path1 = 'updatenotes_update_test.md';

    // Create note first
    await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ upsert: { path: path1, content: 'original content' } }] },
    });

    // Update it
    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ upsert: { path: path1, content: 'updated content' } }] },
    });
    expect(data.updateNotes.paths).toContain(path1);
  });

  test('patch: replaces find string in note', async ({ request }) => {
    const notePath = 'updatenotes_patch_test.md';

    // Create note with a checkbox
    await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ upsert: { path: notePath, content: '- [ ] buy milk\n- [x] sleep\n' } }] },
    });

    // Patch the checkbox
    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: {
        changes: [{ patch: { path: notePath, find: '- [ ] buy milk', replace: '- [x] buy milk' } }],
      },
    });
    expect(data.updateNotes.paths).toContain(notePath);
  });

  test('patch: returns UpdateNotesPatchNotFoundPayload when find string absent', async ({ request }) => {
    const notePath = 'updatenotes_patchnotfound_test.md';
    await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ upsert: { path: notePath, content: '- [x] done\n' } }] },
    });

    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ patch: { path: notePath, find: '- [ ] missing task', replace: '- [x] missing task' } }] },
    });
    expect(data.updateNotes).toMatchObject({ path: notePath, find: '- [ ] missing task' });
    expect(data.updateNotes.paths).toBeUndefined();
  });

  test('patch: returns UpdateNotesPatchNotFoundPayload on multiple occurrences', async ({ request }) => {
    const notePath = 'updatenotes_patchambig_test.md';
    await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ upsert: { path: notePath, content: '- [ ] task\n- [ ] task\n' } }] },
    });

    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ patch: { path: notePath, find: '- [ ] task', replace: '- [x] task' } }] },
    });
    expect(data.updateNotes).toMatchObject({ path: notePath, find: '- [ ] task' });
  });

  test('patch: returns ErrorPayload when note does not exist', async ({ request }) => {
    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: {
        changes: [{ patch: { path: 'nonexistent_note_xyz.md', find: 'x', replace: 'y' } }],
      },
    });
    expect(data.updateNotes.message).toMatch(/note not found/);
  });

  test('upsert: returns UpdateNotesHashMismatchPayload on wrong expectedHash', async ({ request }) => {
    const notePath = 'updatenotes_hashmismatch_test.md';
    await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ upsert: { path: notePath, content: 'current content' } }] },
    });

    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: {
        changes: [{ upsert: { path: notePath, content: 'new content', expectedHash: 'wronghash==' } }],
      },
    });
    expect(data.updateNotes.path).toBe(notePath);
    expect(data.updateNotes.actualHash).toBeTruthy();
    expect(data.updateNotes.paths).toBeUndefined();
  });

  test('hide: removes note from public view', async ({ request }) => {
    const notePath = 'updatenotes_hide_test.md';
    await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ upsert: { path: notePath, content: 'to be hidden' } }] },
    });

    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{ hide: { path: notePath } }] },
    });
    expect(data.updateNotes.paths).toContain(notePath);
  });

  test('mixed batch: upsert + patch + hide in one call', async ({ request }) => {
    const upsertPath = 'updatenotes_mixed_upsert.md';
    const patchPath = 'updatenotes_mixed_patch.md';
    const hidePath = 'updatenotes_mixed_hide.md';

    // Set up patch and hide targets
    await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: {
        changes: [
          { upsert: { path: patchPath, content: 'hello world' } },
          { upsert: { path: hidePath, content: 'to be hidden' } },
        ],
      },
    });

    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: {
        changes: [
          { upsert: { path: upsertPath, content: '# New note' } },
          { patch: { path: patchPath, find: 'world', replace: 'Go' } },
          { hide: { path: hidePath } },
        ],
      },
    });
    const { paths } = data.updateNotes;
    expect(paths).toContain(upsertPath);
    expect(paths).toContain(patchPath);
    expect(paths).toContain(hidePath);
  });

  test('empty change item is silently skipped', async ({ request }) => {
    const data = await gqlApi(request, apiKey, UPDATE_NOTES, {
      input: { changes: [{}] },
    });
    expect(data.updateNotes.paths).toEqual([]);
  });
});
