// Setup helper for the render benchmark.
// Pushes N synthetic notes (with and without custom Jet layout) via pushNotes.
//
// Env: APP_URL, N (note count).
// Run with LAYOUT=1 to push _layouts/bench.html and bench_layout/* notes.
import { performance } from 'node:perf_hooks';

const APP_URL = process.env.APP_URL || 'http://localhost:28081';
const N = parseInt(process.env.N || '100', 10);
const LAYOUT = process.env.LAYOUT === '1';

async function gql(query, variables, headers = {}) {
  const res = await fetch(`${APP_URL}/graphql`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...headers },
    body: JSON.stringify({ query, variables }),
  });
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

async function signIn() {
  await gql(`mutation { requestEmailSignInCode(input: { email: "hello@example.com" }) { ... on RequestEmailSignInCodePayload { success } } }`);
  for (const code of ['111111', '000000']) {
    try {
      const d = await gql(`mutation { signInByEmail(input: { email: "hello@example.com", code: "${code}" }) { ... on SignInPayload { token } ... on ErrorPayload { message } } }`);
      if (d.signInByEmail.token) return d.signInByEmail.token;
    } catch { /* try next */ }
  }
  throw new Error('sign-in failed');
}

async function createApiKey(token) {
  const d = await gql(
    `mutation { admin { createApiKey(input: { description: "bench-render" }) { ... on CreateApiKeyPayload { value } ... on ErrorPayload { message } } } }`,
    {},
    { Cookie: `trip2g_token=${token}` },
  );
  const v = d.admin.createApiKey.value;
  if (!v) throw new Error('createApiKey failed: ' + JSON.stringify(d));
  return v;
}

function noteContent(i, layout = false) {
  const fm = ['---', 'free: true', `title: Bench note ${i}`, layout ? 'layout: /bench' : ''].filter(Boolean);
  return [
    ...fm, '---', '',
    `# Bench note ${i}`, '',
    'This is a synthetic note for render benchmarking. It contains representative content:',
    'a paragraph, a list, and a code block.', '',
    '- item one', '- item two', '- item three', '',
    '```js', `const i = ${i};`, `console.log('note', i);`, '```', '',
    'A closing paragraph to give the renderer real work to do.',
  ].join('\n');
}

// Minimal Jet layout for the bench — no frontmatter, raw Jet template content only.
// (note.Content is used verbatim as the Jet template source; frontmatter breaks the parser.)
const layoutContent = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{ note.Title() }}</title>
  <style>body{font-family:system-ui,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem}</style>
</head>
<body>
  <article>
    <h1>{{ note.Title() }}</h1>
    {{ note.HTMLString() | unsafe }}
  </article>
</body>
</html>`;

async function push(apiKey, updates) {
  const d = await gql(
    `mutation Push($input: PushNotesInput!) { pushNotes(input: $input) { __typename ... on ErrorPayload { message } } }`,
    { input: { updates } },
    { 'X-API-Key': apiKey },
  );
  if (d.pushNotes.__typename === 'ErrorPayload') throw new Error('push failed: ' + d.pushNotes.message);
}

async function main() {
  process.stdout.write(`  pushing ${N} notes (layout=${LAYOUT})... `);
  const t0 = performance.now();

  const token = await signIn();
  const apiKey = await createApiKey(token);

  if (!LAYOUT) {
    // Default template notes: bench/note_0..N-1
    const updates = Array.from({ length: N }, (_, i) => ({
      path: `bench/note_${i}.md`,
      content: noteContent(i, false),
    }));
    await push(apiKey, updates);
  } else {
    // Layout file + bench_layout/note_0..N-1
    const updates = [
      { path: '_layouts/bench.html', content: layoutContent },
      ...Array.from({ length: N }, (_, i) => ({
        path: `bench_layout/note_${i}.md`,
        content: noteContent(i, true),
      })),
    ];
    await push(apiKey, updates);
  }

  process.stdout.write(`done in ${Math.round(performance.now() - t0)} ms\n`);
}

main().catch((e) => { console.error(e); process.exit(1); });
