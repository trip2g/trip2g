// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn } from './helpers/auth.js';
import { createApiKey, disableApiKey, fedGql, hideNotes, revokeFederationSecret } from './helpers/federation.js';

/**
 * Federation guest handoff E2E Tests
 *
 * A reader signed in on one base presses "Open knowledge base" on a KB-note and
 * is let into the neighbouring base as a read-only guest, then goes one hop
 * further the same way. See docs/dev/federation_guest_handoff.md.
 *
 * The chain is peer → peer2 → peer3, not hub → peer: federation.spec.js and
 * federation-acl.spec.js own the hub's outbound keys to the app-peer* URLs and
 * revoke whatever else they find there. Nobody else pairs peer with peer2 or
 * peer2 with peer3.
 *
 * A base signs its --public-url into the handoff and the neighbour accepts the
 * form only from that origin, so the browser has to be on the peers' docker
 * names. Chromium maps them to the published ports below; the admin setup runs
 * from node, which cannot resolve them, and goes through localhost.
 *
 * Each run leaves a reader on peer and a guest user on peer2 and peer3: there
 * is no mutation to delete a user.
 */

const SALES = { api: 'http://localhost:20091', site: 'http://app-peer:20091', cookie: 'trip2g_e2e_peer' };
const ENG = { api: 'http://localhost:20093', site: 'http://app-peer2:20093', cookie: 'trip2g_e2e_peer2' };
const DESIGN = { api: 'http://localhost:20095', site: 'http://app-peer3:20095', cookie: 'trip2g_e2e_peer3' };

test.use({
  launchOptions: {
    args: ['--host-resolver-rules=MAP app-peer 127.0.0.1, MAP app-peer2 127.0.0.1, MAP app-peer3 127.0.0.1'],
  },
});

const RUN = crypto.randomBytes(4).toString('hex');
const KID_SALES_ENG = `e2e-guest-a-${RUN}`;
const KID_ENG_DESIGN = `e2e-guest-b-${RUN}`;
const READER_EMAIL = `e2e-guest-${RUN}@example.com`;

const SG_READERS = 'e2e-guest-readers'; // on Sales: who may read the KB-note
const SG_SHARED = 'e2e-guest-shared'; // on Engineering: granted to the Sales edge
const SG_PARTNERS = 'e2e-guest-partners'; // on Design: granted to the Engineering edge
const SG_CLOSED = 'e2e-guest-closed'; // granted to no edge

const NOTE = {
  salesKB: `e2e_guest_${RUN}_eng_kb.md`,
  engShared: `e2e_guest_${RUN}_shared.md`,
  engClosed: `e2e_guest_${RUN}_closed.md`,
  engKB: `e2e_guest_${RUN}_design_kb.md`,
  designShared: `e2e_guest_${RUN}_partners.md`,
  designClosed: `e2e_guest_${RUN}_drafts.md`,
};
const BODY = {
  engShared: 'E2E-GUEST-SHARED-BODY',
  engClosed: 'E2E-GUEST-CLOSED-BODY',
  designShared: 'E2E-GUEST-PARTNERS-BODY',
  designClosed: 'E2E-GUEST-DRAFTS-BODY',
};

const HANDOFF_PATH = '/_system/federation/handoff';
const GUEST_PATH = '/_system/federation/guest';

const pageURL = (base, notePath) => `${base.site}/${notePath.replace(/\.md$/, '')}`;

function note(frontmatter, body) {
  return ['---', ...frontmatter, '---', body, ''].join('\n');
}

/** GraphQL from the page, so the request carries the browser's own cookies. */
function pageGql(page, query) {
  return page.evaluate(async (q) => {
    const response = await fetch('/_system/graphql', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: q }),
    });
    return response.json();
  }, query);
}

async function signInOnPage(page, base, email) {
  await page.goto(`${base.site}/`);
  await pageGql(page, `mutation { requestEmailSignInCode(input: { email: "${email}" }) { __typename } }`);
  const result = await pageGql(page, `mutation { signInByEmail(input: { email: "${email}", code: "111111" }) {
    __typename ... on ErrorPayload { message }
  } }`);
  expect(result.data.signInByEmail, `sign in ${email} on ${base.site}`).toEqual({ __typename: 'SignInPayload' });
}

/** Presses the KB-note button and returns the statuses of the handoff requests. */
async function pressKBButton(page, from, kbNote, to) {
  const trail = [];
  const record = (response) => {
    const url = new URL(response.url());
    if (url.pathname.startsWith('/_system/federation/')) {
      trail.push(`${response.request().method()} ${url.host}${url.pathname} ${response.status()}`);
    }
  };
  page.on('response', record);
  await page.goto(pageURL(from, kbNote));
  const button = page.locator(`form.kb-open[action="${HANDOFF_PATH}"] button`);
  await expect(button).toBeVisible();
  await button.click();
  await page.waitForURL(`${to.site}/**`);
  page.off('response', record);
  return trail;
}

const handoffTrail = (from, to) => [
  `POST ${new URL(from.site).host}${HANDOFF_PATH} 200`,
  `POST ${new URL(to.site).host}${GUEST_PATH} 303`,
  `GET ${new URL(to.site).host}${GUEST_PATH} 302`,
];

/** Posts a copied handoff form from whatever site the page is on. */
async function postHandoffForm(page, form) {
  const [response] = await Promise.all([
    page.waitForResponse((r) => r.url() === form.action && r.request().method() === 'POST'),
    page.evaluate(({ action, token, path }) => {
      const el = document.createElement('form');
      el.method = 'post';
      el.action = action;
      for (const [name, value] of Object.entries({ token, path })) {
        const input = document.createElement('input');
        input.type = 'hidden';
        input.name = name;
        input.value = value;
        el.appendChild(input);
      }
      document.body.appendChild(el);
      el.submit();
    }, form),
  ]);
  return response.status();
}

async function expectPage(page, url, status, { has, lacks } = {}) {
  const response = await page.goto(url);
  expect(response?.status(), url).toBe(status);
  const html = await page.content();
  if (has) expect(html, `${url} shows its body`).toContain(has);
  if (lacks) expect(html, `${url} hides its body`).not.toContain(lacks);
}

test.describe.serial('Federation guest handoff', () => {
  /** @type {Record<string, { request: import('@playwright/test').APIRequestContext, cookie: string, apiKey?: string, apiKeyId?: string }>} */
  const admin = {};
  const secrets = []; // { base, id } of every key this spec created
  let engInboundId;
  /** @type {import('@playwright/test').BrowserContext} */
  let reader;
  /** @type {import('@playwright/test').Page} */
  let page;

  const adminGql = (base, query, variables) => (
    fedGql(admin[base.api].request, base.api, { Cookie: admin[base.api].cookie }, query, variables)
  );

  async function pushNotes(base, notes) {
    const changes = Object.entries(notes).map(([path, content]) => ({ upsert: { path, content } }));
    const data = await fedGql(admin[base.api].request, base.api, { 'X-Api-Key': admin[base.api].apiKey }, `
      mutation($input: UpdateNotesInput!) { updateNotes(input: $input) {
        ... on UpdateNotesSuccessPayload { paths }
        ... on ErrorPayload { message }
      } }
    `, { input: { changes } });
    expect(data.updateNotes.paths, `updateNotes on ${base.api}: ${data.updateNotes.message}`).toEqual(Object.keys(notes));
  }

  async function subgraphID(base, name) {
    const data = await adminGql(base, 'query { admin { allSubgraphs { nodes { id name } } } }');
    const id = data.admin.allSubgraphs.nodes.find((node) => node.name === name)?.id;
    expect(id, `subgraph ${name} on ${base.api}`).toBeTruthy();
    return id;
  }

  /** Pairs from → to under kid and grants the edge one subgraph on `to`. */
  async function pair(from, to, kid, grant) {
    const secretHex = crypto.randomBytes(32).toString('hex');
    const inbound = await adminGql(to, `
      mutation($input: CreateInboundFederationSecretInput!) { admin { createInboundFederationSecret(input: $input) {
        ... on CreateInboundFederationSecretPayload { id }
        ... on ErrorPayload { message }
      } } }
    `, { input: { kid, secretHex } });
    const inboundId = inbound.admin.createInboundFederationSecret.id;
    expect(inboundId, `inbound ${kid}: ${inbound.admin.createInboundFederationSecret.message}`).toBeTruthy();
    secrets.push({ base: to, id: inboundId });

    const outbound = await adminGql(from, `
      mutation($input: CreateOutboundFederationSecretInput!) { admin { createOutboundFederationSecret(input: $input) {
        ... on CreateOutboundFederationSecretPayload { id }
        ... on ErrorPayload { message }
      } } }
    `, { input: { kid, secretHex, kbURL: `${to.site}/_system/mcp` } });
    const outboundId = outbound.admin.createOutboundFederationSecret.id;
    expect(outboundId, `outbound ${kid}: ${outbound.admin.createOutboundFederationSecret.message}`).toBeTruthy();
    secrets.push({ base: from, id: outboundId });

    const granted = await adminGql(to, `
      mutation($input: AddFederationSecretSubgraphInput!) { admin { addFederationSecretSubgraph(input: $input) {
        ... on AddFederationSecretSubgraphPayload { success }
        ... on ErrorPayload { message }
      } } }
    `, { input: { kid, subgraphID: await subgraphID(to, grant) } });
    expect(granted.admin.addFederationSecretSubgraph.success, `grant ${grant} to ${kid}`).toBe(true);
    return inboundId;
  }

  test.beforeAll(async ({ playwright, browser }) => {
    for (const base of [SALES, ENG, DESIGN]) {
      const request = await playwright.request.newContext({ baseURL: base.api });
      const jwt = await graphqlSignIn(request, 'hello@example.com', '111111', { useCache: false });
      admin[base.api] = { request, cookie: `${base.cookie}=${jwt}` };
      const key = await createApiKey(request, base.api, admin[base.api].cookie, `e2e federation guest ${RUN}`);
      admin[base.api].apiKey = key.value;
      admin[base.api].apiKeyId = key.id;
    }

    await pushNotes(SALES, {
      [NOTE.salesKB]: note([
        `subgraphs: ${SG_READERS}`,
        `mcp_federation_kb_url: ${ENG.site}/_system/mcp`,
        `mcp_federation_kb_id: ${KID_SALES_ENG}`,
      ], 'The Engineering knowledge base.'),
    });
    await pushNotes(ENG, {
      [NOTE.engShared]: note([`subgraphs: ${SG_SHARED}`], BODY.engShared),
      [NOTE.engClosed]: note([`subgraphs: ${SG_CLOSED}`], BODY.engClosed),
      [NOTE.engKB]: note([
        `subgraphs: ${SG_SHARED}`,
        `mcp_federation_kb_url: ${DESIGN.site}/_system/mcp`,
        `mcp_federation_kb_id: ${KID_ENG_DESIGN}`,
      ], 'The Design knowledge base.'),
    });
    await pushNotes(DESIGN, {
      [NOTE.designShared]: note([`subgraphs: ${SG_PARTNERS}`], BODY.designShared),
      [NOTE.designClosed]: note([`subgraphs: ${SG_CLOSED}`], BODY.designClosed),
    });

    engInboundId = await pair(SALES, ENG, KID_SALES_ENG, SG_SHARED);
    await pair(ENG, DESIGN, KID_ENG_DESIGN, SG_PARTNERS);

    // The reader has an account on Sales only, and may read the KB-note there.
    const created = await adminGql(SALES, `
      mutation($input: CreateUserInput!) { admin { createUser(input: $input) {
        ... on CreateUserPayload { user { id } }
        ... on ErrorPayload { message }
      } } }
    `, { input: { email: READER_EMAIL } });
    const readerId = created.admin.createUser.user?.id;
    expect(readerId, `create reader: ${created.admin.createUser.message}`).toBeTruthy();
    const access = await adminGql(SALES, `
      mutation($input: CreateUserSubgraphAccessInput!) { admin { createUserSubgraphAccess(input: $input) {
        ... on CreateUserSubgraphAccessPayload { accesses { id } }
        ... on ErrorPayload { message }
      } } }
    `, { input: { userId: readerId, subgraphIds: [await subgraphID(SALES, SG_READERS)] } });
    expect(access.admin.createUserSubgraphAccess.accesses, access.admin.createUserSubgraphAccess.message).toHaveLength(1);

    reader = await browser.newContext();
    page = await reader.newPage();
    await signInOnPage(page, SALES, READER_EMAIL);
  });

  test.afterAll(async () => {
    await reader?.close();
    for (const { base, id } of secrets) {
      await revokeFederationSecret(admin[base.api].request, base.api, admin[base.api].cookie, id).catch(() => {});
    }
    const notes = [
      [SALES, [NOTE.salesKB]],
      [ENG, [NOTE.engShared, NOTE.engClosed, NOTE.engKB]],
      [DESIGN, [NOTE.designShared, NOTE.designClosed]],
    ];
    for (const [base, paths] of notes) {
      const a = admin[base.api];
      if (!a?.apiKey) continue;
      await hideNotes(a.request, base.api, a.apiKey, paths).catch(() => {});
      await disableApiKey(a.request, base.api, a.cookie, a.apiKeyId).catch(() => {});
    }
    for (const a of Object.values(admin)) await a.request.dispose();
  });

  test('a visitor without a session is refused the shared note', async ({ browser }) => {
    const context = await browser.newContext();
    const anonymous = await context.newPage();
    await expectPage(anonymous, pageURL(ENG, NOTE.engShared), 401, { lacks: BODY.engShared });
    await context.close();
  });

  test('the button on the KB-note lets the reader into the neighbour as a guest', async () => {
    const trail = await pressKBButton(page, SALES, NOTE.salesKB, ENG);
    expect(trail).toEqual(handoffTrail(SALES, ENG));
    expect(page.url()).toBe(`${ENG.site}/`);

    await expectPage(page, pageURL(ENG, NOTE.engShared), 200, { has: BODY.engShared });
    await expectPage(page, pageURL(ENG, NOTE.engClosed), 403, { lacks: BODY.engClosed });

    const cookies = await reader.cookies([ENG.site, `${ENG.site}${GUEST_PATH}`]);
    expect(cookies.map((cookie) => cookie.name), 'session set, nonce cleared').toEqual([ENG.cookie]);

    // The reader is still signed in at home.
    await expectPage(page, pageURL(SALES, NOTE.salesKB), 200);
  });

  test('a guest has no email and can only read', async () => {
    await page.goto(`${ENG.site}/`);
    const viewer = await pageGql(page, '{ viewer { user { email } } }');
    expect(viewer.data.viewer.user).toEqual({ email: null });

    const mutation = await pageGql(page, `mutation { createUserToken(input: { name: "guest" }) { __typename } }`);
    expect(mutation.errors?.[0]?.message).toBe('a federation guest can only read');

    const mcpStatus = await page.evaluate(async () => {
      const response = await fetch('/_system/mcp', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'tools/list' }),
      });
      return response.status;
    });
    expect(mcpStatus).toBe(403);
  });

  test('the guest goes one hop further from the neighbour', async () => {
    const trail = await pressKBButton(page, ENG, NOTE.engKB, DESIGN);
    expect(trail).toEqual(handoffTrail(ENG, DESIGN));

    await expectPage(page, pageURL(DESIGN, NOTE.designShared), 200, { has: BODY.designShared });
    await expectPage(page, pageURL(DESIGN, NOTE.designClosed), 403, { lacks: BODY.designClosed });
  });

  test('a handoff form posted from another site is refused', async ({ browser }) => {
    // The reader's own signed form, as they could copy it out of their browser.
    await page.goto(pageURL(SALES, NOTE.salesKB));
    const form = await page.evaluate(async ({ path, notePath }) => {
      const response = await fetch(path, { method: 'POST', body: new URLSearchParams({ note: notePath }) });
      const doc = new DOMParser().parseFromString(await response.text(), 'text/html');
      const handoff = doc.querySelector('form#handoff');
      return {
        action: handoff?.getAttribute('action'),
        token: handoff?.querySelector('[name=token]')?.getAttribute('value'),
        path: handoff?.querySelector('[name=path]')?.getAttribute('value'),
      };
    }, { path: HANDOFF_PATH, notePath: NOTE.salesKB });
    expect(form).toEqual({ action: `${ENG.site}${GUEST_PATH}`, token: expect.any(String), path: '/' });

    // A colleague's browser is made to post it from a third site.
    const context = await browser.newContext();
    const victim = await context.newPage();
    await victim.goto(`${DESIGN.site}/`);
    expect(await postHandoffForm(victim, form)).toBe(403);
    await victim.waitForURL(form.action);
    expect(await context.cookies([ENG.site, form.action])).toEqual([]);
    await expectPage(victim, pageURL(ENG, NOTE.engShared), 401, { lacks: BODY.engShared });

    // The refusal was about the site: the same form is still good from Sales.
    await victim.goto(`${SALES.site}/`);
    expect(await postHandoffForm(victim, form)).toBe(303);
    await context.close();
  });

  test('the handoff leaves the reader\'s own session on the neighbour alone', async ({ browser }) => {
    const context = await browser.newContext();
    const own = await context.newPage();
    await signInOnPage(own, SALES, READER_EMAIL);
    await signInOnPage(own, ENG, 'hello@example.com');

    expect(await pressKBButton(own, SALES, NOTE.salesKB, ENG)).toEqual(handoffTrail(SALES, ENG));
    expect(own.url()).toBe(`${ENG.site}/`);

    const viewer = await pageGql(own, '{ viewer { role user { email } } }');
    expect(viewer.data.viewer).toEqual({ role: 'ADMIN', user: { email: 'hello@example.com' } });
    await expectPage(own, pageURL(ENG, NOTE.engClosed), 200, { has: BODY.engClosed });
    await context.close();
  });

  test('revoking the edge ends the guest session', async () => {
    await expectPage(page, pageURL(ENG, NOTE.engShared), 200, { has: BODY.engShared });

    await revokeFederationSecret(admin[ENG.api].request, ENG.api, admin[ENG.api].cookie, engInboundId);

    await expectPage(page, pageURL(ENG, NOTE.engShared), 401, { lacks: BODY.engShared });
    expect(await reader.cookies(ENG.site), 'the guest cookie is cleared').toEqual([]);

    // The button now leads to the neighbour's refusal page.
    await page.goto(pageURL(SALES, NOTE.salesKB));
    const [refusal] = await Promise.all([
      page.waitForResponse((r) => r.url() === `${ENG.site}${GUEST_PATH}` && r.request().method() === 'POST'),
      page.locator('form.kb-open button').click(),
    ]);
    expect(refusal.status()).toBe(403);
  });
});
