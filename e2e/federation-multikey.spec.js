// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn } from './helpers/auth.js';
import {
  createApiKey,
  disableApiKey,
  fedGql,
  hideNotes,
  revokeFederationSecret,
  upsertKBNote,
} from './helpers/federation.js';

/**
 * Two live outbound keys to the same peer URL, each granted a different
 * subgraph on the peer. Every call must be signed with the key its KB-note
 * names: a call through A reads only A's subgraph, through B only B's, and once
 * A is revoked a call through A fails instead of borrowing B.
 *
 * The hub's KB-notes name the kids, so the other specs' keys to this peer URL
 * do not interfere, and both KB-notes are admin-only.
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER_URL = 'http://localhost:20095';
const HUB_MCP = `${HUB_URL}/_system/mcp`;
const PEER_KB_URL = 'http://trip2g-test-app-peer3:20095/_system/mcp';

const RUN = crypto.randomBytes(4).toString('hex');
const KID_A = `e2e-mk-a-${RUN}`;
const KID_B = `e2e-mk-b-${RUN}`;
const SECRET_A = crypto.randomBytes(32).toString('hex');
const SECRET_B = crypto.randomBytes(32).toString('hex');

const SUBGRAPH_A = `mk-a-${RUN}`;
const SUBGRAPH_B = `mk-b-${RUN}`;
const NOTE_A = `e2e_multikey_${RUN}/a.md`;
const NOTE_B = `e2e_multikey_${RUN}/b.md`;
const TEXT_A = `multikey-sentinel-a-${RUN}`;
const TEXT_B = `multikey-sentinel-b-${RUN}`;
const HUB_KB_NOTE_A = `e2e_multikey_${RUN}_a_kb.md`;
const HUB_KB_NOTE_B = `e2e_multikey_${RUN}_b_kb.md`;

const CREATE_INBOUND = `mutation($input: CreateInboundFederationSecretInput!) {
  admin { createInboundFederationSecret(input: $input) {
    ... on CreateInboundFederationSecretPayload { id kid }
    ... on ErrorPayload { message }
  } }
}`;

const GRANT = `mutation($input: AddFederationSecretSubgraphInput!) {
  admin { addFederationSecretSubgraph(input: $input) {
    ... on AddFederationSecretSubgraphPayload { success }
    ... on ErrorPayload { message }
  } }
}`;

const CREATE_OUTBOUND = `mutation($input: CreateOutboundFederationSecretInput!) {
  admin { createOutboundFederationSecret(input: $input) {
    ... on CreateOutboundFederationSecretPayload { id kid }
    ... on ErrorPayload { message }
  } }
}`;

test.describe.serial('Federation with two keys to one peer', () => {
  let hubRequest;
  let peerRequest;
  let hubCookie;
  let peerCookie;
  let hubApiKey;
  let hubApiKeyId;
  let peerApiKey;
  let peerApiKeyId;
  let inboundAId;
  let inboundBId;
  let outboundAId;
  let outboundBId;

  async function noteHTML(kbID, path) {
    const response = await hubRequest.post(HUB_MCP, {
      headers: { 'Content-Type': 'application/json', Cookie: hubCookie },
      data: {
        jsonrpc: '2.0',
        id: 1,
        method: 'tools/call',
        params: { name: 'federated_note_html', arguments: { kb_id: kbID, path } },
      },
    });
    expect(response.ok(), `federated_note_html status ${response.status()}`).toBeTruthy();
    return response.json();
  }

  async function expectReads(kbID, path, text) {
    const result = await noteHTML(kbID, path);
    expect(result.error, `${kbID} reading ${path}`).toBeUndefined();
    expect(result.result.isError ?? false, `${kbID} reading ${path}: ${JSON.stringify(result)}`).toBe(false);
    expect(result.result.content[0].text).toContain(text);
  }

  async function expectRefused(kbID, path, text, reason) {
    const result = await noteHTML(kbID, path);
    const body = JSON.stringify(result);
    expect(body, `${kbID} must not read ${path}`).not.toContain(text);
    expect(body.toLowerCase()).toMatch(reason);
  }

  async function grant(kid, subgraph) {
    const subgraphs = await fedGql(peerRequest, PEER_URL, { Cookie: peerCookie }, `
      query { admin { allSubgraphs { nodes { id name } } } }
    `);
    const id = subgraphs.admin.allSubgraphs.nodes.find((n) => n.name === subgraph)?.id;
    expect(id, `${subgraph} subgraph must exist on the peer`).toBeTruthy();
    const data = await fedGql(peerRequest, PEER_URL, { Cookie: peerCookie }, GRANT, {
      input: { kid, subgraphID: id },
    });
    expect(data.admin.addFederationSecretSubgraph.success, `grant ${subgraph} to ${kid}`).toBe(true);
  }

  async function createInbound(kid, secretHex) {
    const data = await fedGql(peerRequest, PEER_URL, { Cookie: peerCookie }, CREATE_INBOUND, {
      input: { kid, secretHex },
    });
    const id = data.admin.createInboundFederationSecret.id;
    expect(id, `inbound ${kid}: ${data.admin.createInboundFederationSecret.message}`).toBeTruthy();
    return id;
  }

  async function createOutbound(kid, secretHex) {
    const data = await fedGql(hubRequest, HUB_URL, { Cookie: hubCookie }, CREATE_OUTBOUND, {
      input: { kid, secretHex, kbURL: PEER_KB_URL },
    });
    const id = data.admin.createOutboundFederationSecret.id;
    expect(id, `outbound ${kid}: ${data.admin.createOutboundFederationSecret.message}`).toBeTruthy();
    return id;
  }

  test.beforeAll(async ({ playwright }) => {
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    hubCookie = `trip2g_e2e=${await graphqlSignIn(hubRequest)}`;

    peerRequest = await playwright.request.newContext({ baseURL: PEER_URL });
    const peerJwt = await graphqlSignIn(peerRequest, 'hello@example.com', '111111', { useCache: false });
    peerCookie = `trip2g_e2e_peer3=${peerJwt}`;

    ({ value: hubApiKey, id: hubApiKeyId } = await createApiKey(hubRequest, HUB_URL, hubCookie, `e2e multikey ${RUN}`));
    ({ value: peerApiKey, id: peerApiKeyId } = await createApiKey(peerRequest, PEER_URL, peerCookie, `e2e multikey ${RUN}`));

    const pushed = await fedGql(peerRequest, PEER_URL, { 'X-Api-Key': peerApiKey }, `
      mutation($input: PushNotesInput!) { pushNotes(input: $input) {
        ... on PushNotesPayload { updated { path } }
        ... on ErrorPayload { message }
      } }
    `, {
      input: {
        updates: [
          { path: NOTE_A, content: `---\nsubgraph: ${SUBGRAPH_A}\n---\nNote A says ${TEXT_A}.\n` },
          { path: NOTE_B, content: `---\nsubgraph: ${SUBGRAPH_B}\n---\nNote B says ${TEXT_B}.\n` },
        ],
      },
    });
    expect(pushed.pushNotes.updated, `pushNotes: ${pushed.pushNotes.message}`).toHaveLength(2);

    inboundAId = await createInbound(KID_A, SECRET_A);
    inboundBId = await createInbound(KID_B, SECRET_B);
    await grant(KID_A, SUBGRAPH_A);
    await grant(KID_B, SUBGRAPH_B);

    outboundAId = await createOutbound(KID_A, SECRET_A);
    outboundBId = await createOutbound(KID_B, SECRET_B);

    await upsertKBNote(hubRequest, HUB_URL, hubApiKey, { path: HUB_KB_NOTE_A, kbURL: PEER_KB_URL, kbID: KID_A });
    await upsertKBNote(hubRequest, HUB_URL, hubApiKey, { path: HUB_KB_NOTE_B, kbURL: PEER_KB_URL, kbID: KID_B });
  });

  test.afterAll(async () => {
    if (hubApiKey) {
      await hideNotes(hubRequest, HUB_URL, hubApiKey, [HUB_KB_NOTE_A, HUB_KB_NOTE_B]).catch(() => {});
    }
    if (peerApiKey) {
      await hideNotes(peerRequest, PEER_URL, peerApiKey, [NOTE_A, NOTE_B]).catch(() => {});
    }
    for (const id of [outboundAId, outboundBId]) {
      if (id) await revokeFederationSecret(hubRequest, HUB_URL, hubCookie, id).catch(() => {});
    }
    for (const id of [inboundAId, inboundBId]) {
      if (id) await revokeFederationSecret(peerRequest, PEER_URL, peerCookie, id).catch(() => {});
    }
    if (hubApiKeyId) await disableApiKey(hubRequest, HUB_URL, hubCookie, hubApiKeyId).catch(() => {});
    if (peerApiKeyId) await disableApiKey(peerRequest, PEER_URL, peerCookie, peerApiKeyId).catch(() => {});
    await hubRequest?.dispose();
    await peerRequest?.dispose();
  });

  test('kb_id A reads with key A: its own subgraph only', async () => {
    await expectReads(KID_A, NOTE_A, TEXT_A);
    await expectRefused(KID_A, NOTE_B, TEXT_B, /not found/);
  });

  test('kb_id B reads with key B: its own subgraph only', async () => {
    await expectReads(KID_B, NOTE_B, TEXT_B);
    await expectRefused(KID_B, NOTE_A, TEXT_A, /not found/);
  });

  test('a revoked key A fails the call instead of borrowing key B', async () => {
    await revokeFederationSecret(hubRequest, HUB_URL, hubCookie, outboundAId);
    outboundAId = null;

    await expectRefused(KID_A, NOTE_A, TEXT_A, /revoked/);
    await expectRefused(KID_A, NOTE_B, TEXT_B, /revoked/);
    await expectReads(KID_B, NOTE_B, TEXT_B);
  });
});
