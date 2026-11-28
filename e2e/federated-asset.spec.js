// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import http from 'http';
import { graphqlSignIn, USER_TOKEN_COOKIE_NAME } from './helpers/auth.js';

/**
 * federated_asset E2E: the hub hands out a download link for an attachment of a
 * peer note, and fetching the link streams the file from the peer, once.
 *
 * The hub gets its own KB-note with kb_id equal to the pairing's kid, so the
 * call is signed with this spec's key whatever other keys to the peer are live.
 * It pairs through the peer's container name, since the service name belongs to
 * federation.spec.js, and its KB-note is admin-only, so anonymous callers of the
 * other specs do not see an extra base.
 *
 * The loader counts an attachment only when the note embeds it (![[…]]) and its
 * extension is a media or document one, so both notes embed a .zip.
 *
 * A second pairing reaches the same peer through a proxy this spec runs, which
 * answers without Content-Length the way a reverse proxy in front of a peer
 * does once it has unpacked a gzipped body. The hub reaches the proxy on
 * PROXY_HOST, the docker host by default.
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER_URL = process.env.PEER_URL || 'http://localhost:20091';
const HUB_MCP = `${HUB_URL}/_system/mcp`;
const PEER_KB_URL = process.env.PEER_KB_URL || 'http://trip2g-test-app-peer:20091/_system/mcp';

const SECRET_HEX = 'd3e3f3a3b3c3d3e3f30405060708090a0b0c0d0e0f101112131415161718d3e3';
const KID = `e2e-asset-${crypto.randomBytes(4).toString('hex')}`;
const PROXIED_KID = `${KID}-proxied`;
const PROXY_HOST = process.env.PROXY_HOST || 'host.docker.internal';

const GRANTED_SUBGRAPH = 'e2e-asset-granted';
const OTHER_SUBGRAPH = 'e2e-asset-other';
const GRANTED_NOTE = 'e2e_federated_asset/granted.md';
const OTHER_NOTE = 'e2e_federated_asset/other.md';
const GRANTED_ASSET = 'files/granted.zip';
const OTHER_ASSET = 'files/other.zip';
const HUB_KB_NOTE = 'e2e_federated_asset_kb.md';
const HUB_PROXIED_KB_NOTE = 'e2e_federated_asset_proxied_kb.md';

// Larger than the relay client's 64 KiB buffer, so the download streams.
const GRANTED_BYTES = Buffer.concat([Buffer.from('PK\x03\x04'), crypto.randomBytes(160 * 1024)]);
const OTHER_BYTES = crypto.randomBytes(70 * 1024);

const sha256 = (buf) => crypto.createHash('sha256').update(buf).digest('hex');

function startChunkingProxy(target) {
  const upstream = new URL(target);
  const server = http.createServer((req, res) => {
    const forwarded = http.request({
      host: upstream.hostname,
      port: upstream.port,
      method: req.method,
      path: req.url,
      headers: { ...req.headers, host: upstream.host },
    }, (peerResponse) => {
      const headers = { ...peerResponse.headers };
      delete headers['content-length'];
      res.writeHead(peerResponse.statusCode ?? 502, headers);
      peerResponse.pipe(res);
    });
    forwarded.on('error', () => {
      res.writeHead(502);
      res.end();
    });
    req.pipe(forwarded);
  });
  return new Promise((resolve) => {
    server.listen(0, '0.0.0.0', () => resolve(server));
  });
}

async function gql(request, baseURL, headers, query, variables = {}) {
  const response = await request.post(`${baseURL}/_system/graphql`, {
    headers: { 'Content-Type': 'application/json', ...headers },
    data: { query, variables },
  });
  if (!response.ok()) throw new Error(`GraphQL ${baseURL} failed: ${await response.text()}`);
  const body = await response.json();
  if (body.errors) throw new Error(`GraphQL errors (${baseURL}): ${JSON.stringify(body.errors)}`);
  return body.data;
}

async function uploadAsset(request, baseURL, apiKey, noteId, asset, bytes) {
  const query = `mutation($input: UploadNoteAssetInput!) { uploadNoteAsset(input: $input) {
    ... on UploadNoteAssetPayload { uploadSkipped }
    ... on ErrorPayload { message }
  } }`;
  const operations = {
    query,
    variables: {
      input: {
        file: null,
        noteId,
        sha256Hash: sha256(bytes),
        path: asset.path,
        absolutePath: asset.path,
      },
    },
  };
  // A FormData keeps the parts in order: the server wants operations first.
  const form = new FormData();
  form.append('operations', JSON.stringify(operations));
  form.append('map', JSON.stringify({ 0: ['variables.input.file'] }));
  form.append('0', new Blob([bytes], { type: 'application/octet-stream' }), asset.path.split('/').pop());
  const response = await request.post(`${baseURL}/_system/graphql`, {
    headers: { 'X-Api-Key': apiKey },
    multipart: form,
  });
  if (!response.ok()) throw new Error(`uploadNoteAsset failed: ${await response.text()}`);
  const body = await response.json();
  if (body.errors) throw new Error(`uploadNoteAsset errors: ${JSON.stringify(body.errors)}`);
  expect(body.data.uploadNoteAsset.message).toBeUndefined();
}

test.describe.serial('Federated asset', () => {
  let hubRequest;
  let peerRequest;
  let hubCookie;
  let peerCookie;
  let hubApiKey;
  let hubApiKeyId;
  let peerApiKey;
  let peerApiKeyId;
  const pairings = [];
  let grantedSubgraphId;
  let proxy;

  async function mcpTool(name, args) {
    const response = await hubRequest.post(HUB_MCP, {
      headers: { 'Content-Type': 'application/json', Cookie: hubCookie },
      data: { jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name, arguments: args } },
    });
    expect(response.ok(), `MCP ${name} status ${response.status()}`).toBeTruthy();
    return response.json();
  }

  async function createApiKey(request, baseURL, cookie) {
    const data = await gql(request, baseURL, { Cookie: cookie }, `
      mutation($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) {
        ... on CreateApiKeyPayload { value apiKey { id } }
        ... on ErrorPayload { message }
      } } }
    `, { input: { description: `e2e federated asset ${KID}` } });
    const payload = data.admin.createApiKey;
    expect(payload.value, `createApiKey on ${baseURL}: ${payload.message}`).toBeTruthy();
    return { value: payload.value, id: payload.apiKey.id };
  }

  async function pair(kid, kbURL, kbNotePath) {
    const inbound = await gql(peerRequest, PEER_URL, { Cookie: peerCookie }, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid, secretHex: SECRET_HEX } });
    const inboundId = inbound.admin.createInboundFederationSecret.id;
    expect(inboundId, `inbound secret ${kid} must be created on the peer`).toBeTruthy();
    const pairing = { inboundId, kbNotePath };
    pairings.push(pairing);

    const grant = await gql(peerRequest, PEER_URL, { Cookie: peerCookie }, `
      mutation($input: AddFederationSecretSubgraphInput!) {
        admin { addFederationSecretSubgraph(input: $input) {
          ... on AddFederationSecretSubgraphPayload { success }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid, subgraphID: grantedSubgraphId } });
    expect(grant.admin.addFederationSecretSubgraph.success).toBe(true);

    const outbound = await gql(hubRequest, HUB_URL, { Cookie: hubCookie }, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid, secretHex: SECRET_HEX, kbURL } });
    pairing.outboundId = outbound.admin.createOutboundFederationSecret.id;
    expect(pairing.outboundId, `outbound secret ${kid} must be created on the hub`).toBeTruthy();

    const kbNote = await gql(hubRequest, HUB_URL, { 'X-Api-Key': hubApiKey }, `
      mutation($input: UpdateNotesInput!) { updateNotes(input: $input) {
        ... on UpdateNotesSuccessPayload { paths }
        ... on ErrorPayload { message }
      } }
    `, {
      input: {
        changes: [{
          upsert: {
            path: kbNotePath,
            content: `---\nmcp_federation_kb_url: ${kbURL}\nmcp_federation_kb_id: ${kid}\n---\nThe peer, paired for the federated_asset e2e.\n`,
          },
        }],
      },
    });
    expect(kbNote.updateNotes.paths, `updateNotes: ${kbNote.updateNotes.message}`).toContain(kbNotePath);
  }

  test.beforeAll(async ({ playwright }) => {
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    hubCookie = `${USER_TOKEN_COOKIE_NAME}=${await graphqlSignIn(hubRequest)}`;

    peerRequest = await playwright.request.newContext({ baseURL: PEER_URL });
    const peerJwt = await graphqlSignIn(peerRequest, 'hello@example.com', '111111', { useCache: false });
    peerCookie = `${process.env.PEER_COOKIE_NAME || 'trip2g_e2e_peer'}=${peerJwt}`;

    ({ value: hubApiKey, id: hubApiKeyId } = await createApiKey(hubRequest, HUB_URL, hubCookie));
    ({ value: peerApiKey, id: peerApiKeyId } = await createApiKey(peerRequest, PEER_URL, peerCookie));

    const pushed = await gql(peerRequest, PEER_URL, { 'X-Api-Key': peerApiKey }, `
      mutation($input: PushNotesInput!) { pushNotes(input: $input) {
        ... on PushNotesPayload { updated { id path assets { path absolutePath } } }
        ... on ErrorPayload { message }
      } }
    `, {
      input: {
        updates: [
          {
            path: GRANTED_NOTE,
            content: `---\nsubgraph: ${GRANTED_SUBGRAPH}\n---\nThe granted note.\n\n![[${GRANTED_ASSET}]]\n`,
          },
          {
            path: OTHER_NOTE,
            content: `---\nsubgraph: ${OTHER_SUBGRAPH}\n---\nThe note outside the grant.\n\n![[${OTHER_ASSET}]]\n`,
          },
        ],
      },
    });
    const updated = pushed.pushNotes.updated;
    expect(updated, `pushNotes: ${pushed.pushNotes.message}`).toBeTruthy();
    const byPath = Object.fromEntries(updated.map((n) => [n.path, n]));
    for (const [notePath, bytes] of [[GRANTED_NOTE, GRANTED_BYTES], [OTHER_NOTE, OTHER_BYTES]]) {
      const note = byPath[notePath];
      expect(note?.assets, `${notePath} must list its attachment`).toHaveLength(1);
      await uploadAsset(peerRequest, PEER_URL, peerApiKey, note.id, note.assets[0], bytes);
    }

    const subgraphs = await gql(peerRequest, PEER_URL, { Cookie: peerCookie }, `
      query { admin { allSubgraphs { nodes { id name } } } }
    `);
    grantedSubgraphId = subgraphs.admin.allSubgraphs.nodes.find((n) => n.name === GRANTED_SUBGRAPH)?.id;
    expect(grantedSubgraphId, `${GRANTED_SUBGRAPH} subgraph must exist on the peer`).toBeTruthy();

    await pair(KID, PEER_KB_URL, HUB_KB_NOTE);

    proxy = await startChunkingProxy(PEER_URL);
    const proxiedKBURL = `http://${PROXY_HOST}:${proxy.address().port}/_system/mcp`;
    await pair(PROXIED_KID, proxiedKBURL, HUB_PROXIED_KB_NOTE);
  });

  test.afterAll(async () => {
    const revoke = `mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
      ... on RevokeFederationSecretPayload { revokedId }
      ... on ErrorPayload { message }
    } } }`;
    const hide = `mutation($input: UpdateNotesInput!) { updateNotes(input: $input) {
      ... on UpdateNotesSuccessPayload { paths }
      ... on ErrorPayload { message }
    } }`;
    const disable = `mutation($input: DisableApiKeyInput!) { admin { disableApiKey(input: $input) {
      ... on ErrorPayload { message }
    } } }`;
    const hideAll = (paths) => ({ input: { changes: paths.map((p) => ({ hide: { path: p } })) } });

    if (hubApiKey) {
      const kbNotes = pairings.map((p) => p.kbNotePath);
      await gql(hubRequest, HUB_URL, { 'X-Api-Key': hubApiKey }, hide, hideAll(kbNotes)).catch(() => {});
    }
    if (peerApiKey) {
      await gql(peerRequest, PEER_URL, { 'X-Api-Key': peerApiKey }, hide, hideAll([GRANTED_NOTE, OTHER_NOTE])).catch(() => {});
    }
    for (const { inboundId, outboundId } of pairings) {
      if (outboundId) await gql(hubRequest, HUB_URL, { Cookie: hubCookie }, revoke, { id: outboundId }).catch(() => {});
      if (inboundId) await gql(peerRequest, PEER_URL, { Cookie: peerCookie }, revoke, { id: inboundId }).catch(() => {});
    }
    await new Promise((resolve) => (proxy ? proxy.close(resolve) : resolve()));
    if (hubApiKeyId) await gql(hubRequest, HUB_URL, { Cookie: hubCookie }, disable, { input: { id: hubApiKeyId } }).catch(() => {});
    if (peerApiKeyId) await gql(peerRequest, PEER_URL, { Cookie: peerCookie }, disable, { input: { id: peerApiKeyId } }).catch(() => {});
    await hubRequest?.dispose();
    await peerRequest?.dispose();
  });

  test('tools/list includes federated_asset', async () => {
    const response = await hubRequest.post(HUB_MCP, {
      headers: { 'Content-Type': 'application/json', Cookie: hubCookie },
      data: { jsonrpc: '2.0', id: 1, method: 'tools/list', params: {} },
    });
    const body = await response.json();
    expect(body.result.tools.map((t) => t.name)).toContain('federated_asset');
  });

  test('federated_note_html lists the granted note assets', async () => {
    const result = await mcpTool('federated_note_html', { kb_id: KID, path: GRANTED_NOTE });
    expect(result.error).toBeUndefined();
    expect(result.result.structuredContent).toMatchObject({
      note_path: GRANTED_NOTE,
      assets: [{ name: GRANTED_ASSET, size: GRANTED_BYTES.length, sha256: sha256(GRANTED_BYTES) }],
    });
  });

  test('federated_asset returns a link that downloads the same bytes', async ({ playwright }) => {
    const result = await mcpTool('federated_asset', { kb_id: KID, note_path: GRANTED_NOTE, name: GRANTED_ASSET });
    expect(result.error).toBeUndefined();
    const payload = result.result.structuredContent;
    expect(payload).toMatchObject({ name: GRANTED_ASSET, size: GRANTED_BYTES.length, sha256: sha256(GRANTED_BYTES) });
    expect(payload.url).toMatch(new RegExp(`^${HUB_URL}/_system/mcp/federated_asset\\?link=`));
    const expiresIn = Date.parse(payload.expires_at) - Date.now();
    expect(expiresIn).toBeGreaterThan(0);
    expect(expiresIn).toBeLessThanOrEqual(5 * 60 * 1000);

    const anonymous = await playwright.request.newContext({});
    try {
      const download = await anonymous.get(payload.url);
      expect(download.status()).toBe(200);
      const body = await download.body();
      expect(body.length).toBe(GRANTED_BYTES.length);
      expect(sha256(body)).toBe(payload.sha256);
      const headers = download.headers();
      expect(headers['content-disposition']).toMatch(/^attachment;/);
      expect(headers['content-disposition']).toContain('granted.zip');
      expect(headers['x-content-type-options']).toBe('nosniff');

      const again = await anonymous.get(payload.url);
      expect(again.status(), 'a link downloads once').toBe(410);
      expect(await again.text()).toContain('call federated_asset again');

      const link = new URL(payload.url);
      const token = link.searchParams.get('link') ?? '';
      const [header, claims, signature] = token.split('.');
      const i = Math.floor(claims.length / 2);
      const flipped = claims.slice(0, i) + (claims[i] === 'A' ? 'B' : 'A') + claims.slice(i + 1);
      link.searchParams.set('link', [header, flipped, signature].join('.'));
      const tampered = await anonymous.get(link.toString());
      expect(tampered.status()).toBe(404);
    } finally {
      await anonymous.dispose();
    }
  });

  test('federated_asset refuses an attachment of a note outside the grant', async () => {
    const result = await mcpTool('federated_asset', { kb_id: KID, note_path: OTHER_NOTE, name: OTHER_ASSET });
    expect(result.result).toBeUndefined();
    expect(result.error.message).toContain(`no asset "${OTHER_ASSET}"`);
    expect(JSON.stringify(result)).not.toContain('federated_asset?link=');
  });

  test('federated_asset refuses a name the note does not have', async () => {
    const result = await mcpTool('federated_asset', { kb_id: KID, note_path: GRANTED_NOTE, name: OTHER_ASSET });
    expect(result.result).toBeUndefined();
    expect(result.error.message).toContain(`no asset "${OTHER_ASSET}"`);
    expect(JSON.stringify(result)).not.toContain('federated_asset?link=');
  });

  test('federated_asset refuses crafted names and note paths', async () => {
    const crafted = [
      { note_path: 'e2e_federated_asset/../e2e_federated_asset/other.md', name: OTHER_ASSET },
      { note_path: `/${OTHER_NOTE}`, name: OTHER_ASSET },
      { note_path: GRANTED_NOTE, name: `files/../${OTHER_ASSET}` },
      { note_path: GRANTED_NOTE, name: `../${OTHER_ASSET}` },
      { note_path: GRANTED_NOTE, name: sha256(OTHER_BYTES) },
    ];
    for (const args of crafted) {
      const result = await mcpTool('federated_asset', { kb_id: KID, ...args });
      expect(result.result, JSON.stringify(args)).toBeUndefined();
      expect(result.error, JSON.stringify(args)).toBeTruthy();
      expect(JSON.stringify(result)).not.toContain('federated_asset?link=');
    }
  });

  test('through a proxy that drops Content-Length the link downloads the same bytes', async ({ playwright }) => {
    const result = await mcpTool('federated_asset', { kb_id: PROXIED_KID, note_path: GRANTED_NOTE, name: GRANTED_ASSET });
    expect(result.error).toBeUndefined();
    const payload = result.result.structuredContent;
    expect(payload).toMatchObject({ name: GRANTED_ASSET, size: GRANTED_BYTES.length, sha256: sha256(GRANTED_BYTES) });

    const anonymous = await playwright.request.newContext({});
    try {
      const download = await anonymous.get(payload.url);
      expect(download.status()).toBe(200);
      const body = await download.body();
      expect(body.length).toBe(GRANTED_BYTES.length);
      expect(sha256(body)).toBe(sha256(GRANTED_BYTES));
    } finally {
      await anonymous.dispose();
    }
  });

  test('through the proxy a note outside the grant still gives no link', async () => {
    const result = await mcpTool('federated_asset', { kb_id: PROXIED_KID, note_path: OTHER_NOTE, name: OTHER_ASSET });
    expect(result.result).toBeUndefined();
    expect(result.error.message).toContain(`no asset "${OTHER_ASSET}"`);
    expect(JSON.stringify(result)).not.toContain('federated_asset?link=');
  });
});
