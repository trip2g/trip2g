// @ts-check
import { test, expect } from '@playwright/test';
import { graphqlSignIn } from './helpers/auth.js';

const GRAPHQL_URL = '/_system/graphql';

async function gql(request, baseURL, query, variables, cookie = '') {
  const headers = { 'Content-Type': 'application/json' };
  if (cookie) headers['Cookie'] = cookie;
  const res = await request.post(`${baseURL}${GRAPHQL_URL}`, {
    headers,
    data: { query, variables },
  });
  return res.json();
}

test.describe('Forms in Notes', () => {
  test('note with form: frontmatter embeds form-spec script tag', async ({ page, baseURL }) => {
    await page.goto('/form_test_note');
    const el = page.locator('#form-spec');
    await expect(el).toBeAttached();
    const text = await el.textContent();
    const spec = JSON.parse(text);
    expect(spec).toHaveProperty('note_version_id');
    expect(spec).toHaveProperty('forms');
    expect(spec.forms['']).toHaveProperty('fields');
    expect(spec.forms[''].fields[0].name).toBe('email');
  });

  test('submitForm mutation accepts a valid submission', async ({ request, baseURL }) => {
    const pageRes = await request.get(`${baseURL}/form_test_note`);
    const html = await pageRes.text();
    const match = html.match(/<script id="form-spec"[^>]*type="application\/json">(.*?)<\/script>/s);
    expect(match).toBeTruthy();
    const spec = JSON.parse(match[1]);
    const noteVersionId = spec.note_version_id;

    const res = await gql(request, baseURL, `
      mutation SubmitForm($input: SubmitFormInput!) {
        submitForm(input: $input) {
          ... on SubmitFormPayload { submitId }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: {
        noteVersionId,
        fields: [
          { name: 'email', stringValue: 'test@example.com' },
          { name: 'message', stringValue: 'Hello from e2e test' },
        ],
      },
    });
    expect(res.data.submitForm).toHaveProperty('submitId');
    expect(res.data.submitForm.submitId).toBeGreaterThan(0);
  });

  test('admin-only form: anon submit is denied with admin_required', async ({ request, baseURL }) => {
    const pageRes = await request.get(`${baseURL}/form_admin_test_note`);
    const html = await pageRes.text();
    const match = html.match(/<script id="form-spec"[^>]*type="application\/json">(.*?)<\/script>/s);
    expect(match, 'form-spec script must be embedded on the admin-only note').toBeTruthy();
    const spec = JSON.parse(match[1]);
    expect(spec.forms[''].can_submit).toBe('admin');

    const res = await gql(request, baseURL, `
      mutation SubmitForm($input: SubmitFormInput!) {
        submitForm(input: $input) {
          __typename
          ... on SubmitFormPayload { submitId }
          ... on FormSubmitDeniedPayload { reason }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: {
        noteVersionId: spec.note_version_id,
        fields: [
          { name: 'email', stringValue: 'anon@example.com' },
          { name: 'message', stringValue: 'should be denied' },
        ],
      },
    });
    expect(res.data.submitForm.__typename).toBe('FormSubmitDeniedPayload');
    expect(res.data.submitForm.reason).toBe('admin_required');
  });

  test('admin-only form: admin submit succeeds', async ({ request, baseURL }) => {
    const token = await graphqlSignIn(request);
    const cookie = `trip2g_e2e=${token}`;

    const pageRes = await request.get(`${baseURL}/form_admin_test_note`);
    const html = await pageRes.text();
    const match = html.match(/<script id="form-spec"[^>]*type="application\/json">(.*?)<\/script>/s);
    expect(match).toBeTruthy();
    const spec = JSON.parse(match[1]);

    const res = await gql(request, baseURL, `
      mutation SubmitForm($input: SubmitFormInput!) {
        submitForm(input: $input) {
          __typename
          ... on SubmitFormPayload { submitId }
          ... on FormSubmitDeniedPayload { reason }
          ... on ErrorPayload { message }
        }
      }
    `, {
      input: {
        noteVersionId: spec.note_version_id,
        fields: [
          { name: 'email', stringValue: 'admin@example.com' },
          { name: 'message', stringValue: 'admin submission' },
        ],
      },
    }, cookie);
    expect(res.data.submitForm.__typename).toBe('SubmitFormPayload');
    expect(res.data.submitForm.submitId).toBeGreaterThan(0);
  });

  test('form-spec exposes success_url', async ({ request, baseURL }) => {
    const pageRes = await request.get(`${baseURL}/form_admin_test_note`);
    const html = await pageRes.text();
    const match = html.match(/<script id="form-spec"[^>]*type="application\/json">(.*?)<\/script>/s);
    expect(match).toBeTruthy();
    const spec = JSON.parse(match[1]);
    expect(spec.forms[''].success_url).toBe('/demo/form_admin_test_note?submitted=1');
  });

  test('admin can query form submits', async ({ request, baseURL }) => {
    const token = await graphqlSignIn(request);
    const cookie = `trip2g_e2e=${token}`;

    const pathRes = await gql(request, baseURL, `
      query { admin { allLatestNoteViews(filter: {}) { nodes { pathId path } } } }
    `, {}, cookie);
    const nodes = pathRes.data?.admin?.allLatestNoteViews?.nodes ?? [];
    const formNote = nodes.find(n => n.path === 'form_test_note');
    if (!formNote) {
      test.skip(true, 'form_test_note not found in vault');
      return;
    }

    const res = await gql(request, baseURL, `
      query FormSubmits($notePathId: Int64!) {
        admin {
          formSubmits(notePathId: $notePathId) {
            nodes {
              id noteVersionId formId ip status createdAt
              fields { name stringValue }
            }
          }
        }
      }
    `, { notePathId: formNote.pathId }, cookie);

    const submits = res.data?.admin?.formSubmits?.nodes ?? [];
    expect(submits.length).toBeGreaterThan(0);
    const lastSubmit = submits[0];
    expect(lastSubmit.fields.find(f => f.name === 'email')?.stringValue).toBe('test@example.com');
  });
});
