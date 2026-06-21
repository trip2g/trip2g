// @ts-check
import { test, expect } from '@playwright/test';
import { execFileSync } from 'child_process';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { graphqlSignIn } from './helpers/auth.js';

const GRAPHQL_URL = '/_system/graphql';
const APP_URL = process.env.APP_URL || 'http://localhost:8081';
// The git smart-HTTP base path is configurable; the e2e stack serves it at /git
// (docker-compose.test.yml sets GIT_API_BASE_PATH=/git), prod default is /_system/git.
const GIT_BASE_PATH = process.env.GIT_API_BASE_PATH || '/_system/git';

async function gql(request, apiKey, query, variables = {}) {
  const res = await request.post(GRAPHQL_URL, {
    headers: { 'Content-Type': 'application/json', 'X-Api-Key': apiKey },
    data: { query, variables },
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

// gqlAdmin runs a mutation in the admin namespace using an admin JWT (Bearer).
async function gqlAdmin(request, jwt, query, variables = {}) {
  const res = await request.post(GRAPHQL_URL, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${jwt}` },
    data: { query, variables },
  });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

function git(cwd, env, ...args) {
  return execFileSync('git', args, { cwd, env: { ...process.env, ...env }, encoding: 'utf8' });
}

function authedRemote(token) {
  const u = new URL(APP_URL);
  return `${u.protocol}//user:${encodeURIComponent(token)}@${u.host}${GIT_BASE_PATH}`;
}

const GIT_ENV = {
  GIT_AUTHOR_NAME: 't',
  GIT_AUTHOR_EMAIL: 't@t',
  GIT_COMMITTER_NAME: 't',
  GIT_COMMITTER_EMAIL: 't@t',
};

test.describe('git <-> plugin coexistence', () => {
  // These tests share one server-side git mirror (the DB-canonical repo) and
  // mutate global note state, so they must run in order on a single worker.
  // In parallel, one test's GraphQL push advances master (via materialize) and
  // turns another test's push into an unexpected non-fast-forward.
  test.describe.configure({ mode: 'serial' });

  let apiKey, token, workdir;

  test.beforeAll(async ({ request }) => {
    apiKey = fs.readFileSync(path.join(process.cwd(), '.test-api-key'), 'utf8').trim();

    // createGitToken lives under the admin namespace and needs an admin JWT.
    const jwt = await graphqlSignIn(request);
    const data = await gqlAdmin(
      request,
      jwt,
      `mutation($input: CreateGitTokenInput!) {
         admin {
           createGitToken(input: $input) {
             ... on CreateGitTokenPayload { value }
             ... on ErrorPayload { message }
           }
         }
       }`,
      { input: { description: 'e2e', canPull: true, canPush: true } },
    );
    token = data.admin.createGitToken.value;
    expect(token).toBeTruthy();

    workdir = fs.mkdtempSync(path.join(os.tmpdir(), 'gitsync-'));
  });

  const PUSH_NOTES = `mutation($input: PushNotesInput!) {
    pushNotes(input: $input) {
      ... on PushNotesPayload { notes { path } }
      ... on ErrorPayload { message }
    }
  }`;
  const NOTE_PATHS = `query { notePaths { value } }`;

  // Unique per run: these tests mutate the shared, persistent note DB + git
  // mirror, so fixed paths would collide with state left by earlier runs
  // (e.g. "nothing to commit" when the file already exists). A run-scoped
  // prefix keeps each execution independent.
  const RUN = `gitsync-${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
  const pluginNote = `${RUN}-from-plugin.md`;
  const gitNote = `${RUN}-from-git.md`;
  const staleGitNote = `${RUN}-git-side.md`;
  const stalePluginNote = `${RUN}-plugin-side.md`;
  const deleteNote = `${RUN}-to-delete.md`;

  test('plugin push -> git clone sees it', async ({ request }) => {
    await gql(request, apiKey, PUSH_NOTES, {
      input: { updates: [{ path: pluginNote, content: '# plugin' }] },
    });
    const dir = path.join(workdir, 'clone1');
    git(workdir, {}, 'clone', authedRemote(token), 'clone1');
    expect(fs.readFileSync(path.join(dir, pluginNote), 'utf8')).toContain('# plugin');
  });

  test('git push -> plugin/db sees it', async ({ request }) => {
    const dir = path.join(workdir, 'clone2');
    git(workdir, {}, 'clone', authedRemote(token), 'clone2');
    fs.writeFileSync(path.join(dir, gitNote), '# git');
    git(dir, GIT_ENV, 'add', gitNote);
    git(dir, GIT_ENV, 'commit', '-m', 'add');
    git(dir, GIT_ENV, 'push', 'origin', 'HEAD:master');
    const data = await gql(request, apiKey, NOTE_PATHS);
    expect(data.notePaths.map((n) => n.value)).toContain(gitNote);
  });

  test('stale git push is rejected, succeeds after pull', async ({ request }) => {
    const dir = path.join(workdir, 'clone3');
    git(workdir, {}, 'clone', authedRemote(token), 'clone3');
    // Plugin advances the mirror (a different note) AFTER the clone, so the
    // clone is now stale.
    await gql(request, apiKey, PUSH_NOTES, {
      input: { updates: [{ path: stalePluginNote, content: '# plugin side' }] },
    });
    fs.writeFileSync(path.join(dir, staleGitNote), '# git side');
    git(dir, GIT_ENV, 'add', staleGitNote);
    git(dir, GIT_ENV, 'commit', '-m', 'git edit');
    // Stale push is rejected (non-fast-forward).
    let rejected = false;
    try {
      git(dir, GIT_ENV, 'push', 'origin', 'HEAD:master');
    } catch {
      rejected = true;
    }
    expect(rejected).toBeTruthy();
    // Pull reconciles (the two sides touched different notes → clean merge),
    // then the push succeeds. git 2.x needs an explicit reconcile strategy.
    git(dir, GIT_ENV, '-c', 'pull.rebase=false', 'pull', '--no-edit', 'origin', 'master');
    git(dir, GIT_ENV, 'push', 'origin', 'HEAD:master');
    // Both notes now coexist in the DB.
    const data = await gql(request, apiKey, NOTE_PATHS);
    const paths = data.notePaths.map((n) => n.value);
    expect(paths).toContain(staleGitNote);
    expect(paths).toContain(stalePluginNote);
  });

  test('git deletion hides the note', async ({ request }) => {
    await gql(request, apiKey, PUSH_NOTES, {
      input: { updates: [{ path: deleteNote, content: '# x' }] },
    });
    const dir = path.join(workdir, 'clone4');
    git(workdir, {}, 'clone', authedRemote(token), 'clone4');
    git(dir, GIT_ENV, 'rm', deleteNote);
    git(dir, GIT_ENV, 'commit', '-m', 'rm');
    git(dir, GIT_ENV, 'push', 'origin', 'HEAD:master');
    const data = await gql(request, apiKey, NOTE_PATHS);
    expect(data.notePaths.map((n) => n.value)).not.toContain(deleteNote);
  });
});
