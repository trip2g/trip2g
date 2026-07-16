// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

test.describe('frontmatter note path filters', () => {
  test('filters demo notes by effective frontmatter key and value', async ({ request }) => {
    const token = await graphqlSignIn(request);
    const response = await request.post('/_system/graphql', {
      headers: {
        'Content-Type': 'application/json',
        Cookie: `${USER_TOKEN_COOKIE_NAME}=${token}`,
      },
      data: {
        query: `
          query {
            notePaths(filter: {
              frontmatter: [{ key: "fleet_id", equals: "codellm" }]
            }) {
              value
            }
          }
        `,
      },
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.errors).toBeUndefined();
    const values = body.data.notePaths.map((note) => note.value);
    expect(values).toEqual(expect.arrayContaining([
      'fleet_llmcode_russian.md',
      'fleet_llmcode_english.md',
      'fleet_llmcode_unlabelled.md',
    ]));
  });

  test('combines frontmatter predicates', async ({ request }) => {
    const token = await graphqlSignIn(request);
    const response = await request.post('/_system/graphql', {
      headers: {
        'Content-Type': 'application/json',
        Cookie: `${USER_TOKEN_COOKIE_NAME}=${token}`,
      },
      data: {
        query: `
          query {
            notePaths(filter: {
              frontmatter: [
                { key: "fleet_id", equals: "codellm" }
                { key: "lang", equals: "ru" }
              ]
            }) {
              value
            }
          }
        `,
      },
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.errors).toBeUndefined();
    expect(body.data.notePaths.map((note) => note.value)).toEqual([
      'fleet_llmcode_russian.md',
    ]);
  });
});
