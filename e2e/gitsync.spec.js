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

  test('plugin push -> git clone sees it', async ({ request }) => {
    await gql(request, apiKey, PUSH_NOTES, {
      input: { updates: [{ path: 'from-plugin.md', content: '# plugin' }] },
    });
    const dir = path.join(workdir, 'clone1');
    git(workdir, {}, 'clone', authedRemote(token), 'clone1');
    expect(fs.readFileSync(path.join(dir, 'from-plugin.md'), 'utf8')).toContain('# plugin');
  });

  test('git push -> plugin/db sees it', async ({ request }) => {
    const dir = path.join(workdir, 'clone2');
    git(workdir, {}, 'clone', authedRemote(token), 'clone2');
    fs.writeFileSync(path.join(dir, 'from-git.md'), '# git');
    git(dir, GIT_ENV, 'add', 'from-git.md');
    git(dir, GIT_ENV, 'commit', '-m', 'add');
    git(dir, GIT_ENV, 'push', 'origin', 'HEAD:master');
    const data = await gql(request, apiKey, NOTE_PATHS);
    expect(data.notePaths.map((n) => n.value)).toContain('from-git.md');
  });

  test('stale git push is rejected, succeeds after pull', async ({ request }) => {
    const dir = path.join(workdir, 'clone3');
    git(workdir, {}, 'clone', authedRemote(token), 'clone3');
    await gql(request, apiKey, PUSH_NOTES, {
      input: { updates: [{ path: 'shared.md', content: '# v-plugin' }] },
    });
    fs.writeFileSync(path.join(dir, 'shared.md'), '# v-git');
    git(dir, GIT_ENV, 'add', 'shared.md');
    git(dir, GIT_ENV, 'commit', '-m', 'git edit');
    let rejected = false;
    try {
      git(dir, GIT_ENV, 'push', 'origin', 'HEAD:master');
    } catch {
      rejected = true;
    }
    expect(rejected).toBeTruthy();
    git(dir, GIT_ENV, 'pull', '--no-edit', 'origin', 'master');
    git(dir, GIT_ENV, 'push', 'origin', 'HEAD:master');
  });

  test('git deletion hides the note', async ({ request }) => {
    await gql(request, apiKey, PUSH_NOTES, {
      input: { updates: [{ path: 'to-delete.md', content: '# x' }] },
    });
    const dir = path.join(workdir, 'clone4');
    git(workdir, {}, 'clone', authedRemote(token), 'clone4');
    git(dir, GIT_ENV, 'rm', 'to-delete.md');
    git(dir, GIT_ENV, 'commit', '-m', 'rm');
    git(dir, GIT_ENV, 'push', 'origin', 'HEAD:master');
    const data = await gql(request, apiKey, NOTE_PATHS);
    expect(data.notePaths.map((n) => n.value)).not.toContain('to-delete.md');
  });
});
