// @ts-check
import { expect } from '@playwright/test';

/**
 * Federation pairing helpers for the e2e specs.
 *
 * A base picks the outbound key for a KB-note by kid: the live key whose kid
 * equals the note's kb_id wins, a lone live key to the peer serves any kb_id,
 * and several live keys with none named by the kb_id is an error. The specs run
 * in parallel against the same peers, so each one either gives the hub a KB-note
 * of its own (kb_id = its kid) or owns a peer URL outright.
 */

/**
 * @param {import('@playwright/test').APIRequestContext} request
 * @param {string} baseURL
 * @param {Record<string, string>} headers
 * @param {string} query
 * @param {object} [variables]
 */
export async function fedGql(request, baseURL, headers, query, variables = {}) {
  const response = await request.post(`${baseURL}/_system/graphql`, {
    headers: { 'Content-Type': 'application/json', ...headers },
    data: { query, variables },
  });
  if (!response.ok()) throw new Error(`GraphQL ${baseURL} failed: ${await response.text()}`);
  const body = await response.json();
  if (body.errors) throw new Error(`GraphQL errors (${baseURL}): ${JSON.stringify(body.errors)}`);
  return body.data;
}

const REVOKE = `mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
  ... on RevokeFederationSecretPayload { revokedId }
  ... on ErrorPayload { message }
} } }`;

/**
 * Revokes every live outbound key to kbURL. A spec that owns a peer URL calls it
 * before pairing, so a key left by a run that was killed before its afterAll
 * cannot sit next to the new one.
 */
export async function revokeLiveOutboundSecrets(request, baseURL, cookie, kbURL) {
  const data = await fedGql(request, baseURL, { Cookie: cookie }, `
    query { admin { federationSecrets { id kbUrl revokedAt } } }
  `);
  const live = data.admin.federationSecrets.filter((s) => s.kbUrl === kbURL && !s.revokedAt);
  for (const secret of live) {
    await fedGql(request, baseURL, { Cookie: cookie }, REVOKE, { id: secret.id });
  }
}

export async function revokeFederationSecret(request, baseURL, cookie, id) {
  await fedGql(request, baseURL, { Cookie: cookie }, REVOKE, { id });
}

/** @returns {Promise<{ value: string, id: string }>} */
export async function createApiKey(request, baseURL, cookie, description) {
  const data = await fedGql(request, baseURL, { Cookie: cookie }, `
    mutation($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) {
      ... on CreateApiKeyPayload { value apiKey { id } }
      ... on ErrorPayload { message }
    } } }
  `, { input: { description } });
  const payload = data.admin.createApiKey;
  expect(payload.value, `createApiKey on ${baseURL}: ${payload.message}`).toBeTruthy();
  return { value: payload.value, id: payload.apiKey.id };
}

export async function disableApiKey(request, baseURL, cookie, id) {
  await fedGql(request, baseURL, { Cookie: cookie }, `
    mutation($input: DisableApiKeyInput!) { admin { disableApiKey(input: $input) {
      ... on ErrorPayload { message }
    } } }
  `, { input: { id } });
}

const UPDATE_NOTES = `mutation($input: UpdateNotesInput!) { updateNotes(input: $input) {
  ... on UpdateNotesSuccessPayload { paths }
  ... on ErrorPayload { message }
} }`;

/**
 * Upserts a KB-note on the hub. frontmatter holds the access keys (free,
 * subgraphs) besides the two federation keys.
 */
export async function upsertKBNote(request, baseURL, apiKey, { path, kbURL, kbID, frontmatter = [] }) {
  const content = [
    '---',
    ...frontmatter,
    `mcp_federation_kb_url: ${kbURL}`,
    `mcp_federation_kb_id: ${kbID}`,
    '---',
    `Pairing ${kbID} for an e2e spec.`,
    '',
  ].join('\n');
  const data = await fedGql(request, baseURL, { 'X-Api-Key': apiKey }, UPDATE_NOTES, {
    input: { changes: [{ upsert: { path, content } }] },
  });
  expect(data.updateNotes.paths, `updateNotes ${path}: ${data.updateNotes.message}`).toContain(path);
}

export async function hideNotes(request, baseURL, apiKey, paths) {
  await fedGql(request, baseURL, { 'X-Api-Key': apiKey }, UPDATE_NOTES, {
    input: { changes: paths.map((path) => ({ hide: { path } })) },
  });
}
