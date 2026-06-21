#!/usr/bin/env node
// @ts-check
//
// Bring up the syncperf stack and prepare everything for benchmarking.
//
//   node scripts/syncperf/setup.mjs [--count 2000] [--no-build] [--rebuild]
//
// Steps:
//   1. build the plugin (main.js) so the live plugin in the vault is current
//   2. seed a fresh DB (tmp/data/syncperf.sqlite3 from testdata/e2e_seed.sql)
//   3. docker compose up (app + minio, vector search OFF)
//   4. wait for health
//   5. mint an API key (requestEmailSignInCode -> signInByEmail 111111 -> createApiKey)
//   6. generate the vault and write the key into its data.json + .syncperf-api-key
//
// After it prints "READY", open Obsidian on tmp/syncperf-vault and run the bench.

import { execFileSync, spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const REPO = path.resolve(HERE, "..", "..");
const COMPOSE = ["compose", "-f", "docker-compose.syncperf.yml"];
const DB = "tmp/data/syncperf.sqlite3";
const APP = "http://localhost:20071";
const GQL = `${APP}/_system/graphql`;
const COOKIE = "trip2g_syncperf";
const OWNER = "hello@example.com";
const VAULT = path.join(REPO, "tmp", "syncperf-vault");
const KEY_FILE = path.join(HERE, ".syncperf-api-key");

const args = process.argv.slice(2);
const has = (f) => args.includes(f);
const argVal = (f, d) => { const i = args.indexOf(f); return i >= 0 ? args[i + 1] : d; };
const count = argVal("--count", "2000");

const sh = (cmd, a, opts = {}) =>
	execFileSync(cmd, a, { cwd: REPO, stdio: "inherit", ...opts });
const log = (m) => console.log(`\x1b[36m${m}\x1b[0m`);
const ok = (m) => console.log(`\x1b[32m✓ ${m}\x1b[0m`);

async function gql(query, variables, cookie) {
	const headers = { "Content-Type": "application/json" };
	if (cookie) headers["Cookie"] = cookie;
	const res = await fetch(GQL, { method: "POST", headers, body: JSON.stringify({ query, variables }) });
	const setCookie = res.headers.getSetCookie?.() ?? [];
	const json = await res.json();
	if (json.errors) throw new Error(`GraphQL: ${JSON.stringify(json.errors)}`);
	return { data: json.data, setCookie };
}

async function waitHealthy(timeoutMs = 180000) {
	const start = Date.now();
	log("⏳ waiting for app health...");
	while (Date.now() - start < timeoutMs) {
		try {
			const r = await fetch(`${APP}/`, { signal: AbortSignal.timeout(3000) });
			if (r.status > 0) { ok("app is up"); return; }
		} catch { /* not ready */ }
		await new Promise((r) => setTimeout(r, 2000));
	}
	throw new Error("app did not become healthy in time");
}

async function mintApiKey() {
	log("🔑 minting API key...");
	await gql(
		`mutation($i: RequestEmailSignInCodeInput!){ requestEmailSignInCode(input:$i){ __typename
			... on RequestEmailSignInCodePayload { success }
			... on ErrorPayload { message }
			... on RequestCaptchaPayload { siteKey } } }`,
		{ i: { email: OWNER } },
	);
	const signIn = await gql(
		`mutation($i: SignInByEmailInput!){ signInByEmail(input:$i){ __typename
			... on SignInPayload { token }
			... on ErrorPayload { message } } }`,
		{ i: { email: OWNER, code: "111111" } },
	);
	const token = signIn.data?.signInByEmail?.token;
	if (!token) throw new Error(`sign-in failed: ${JSON.stringify(signIn.data)}`);
	// Prefer the Set-Cookie the server issued; fall back to the returned token.
	const fromHeader = signIn.setCookie.map((c) => c.split(";")[0]).find((c) => c.startsWith(`${COOKIE}=`));
	const cookie = fromHeader ?? `${COOKIE}=${token}`;
	const created = await gql(
		`mutation($i: CreateApiKeyInput!){ admin { createApiKey(input:$i){ __typename
			... on CreateApiKeyPayload { value }
			... on ErrorPayload { message } } } }`,
		{ i: { description: "syncperf benchmark" } },
		cookie,
	);
	const value = created.data?.admin?.createApiKey?.value;
	if (!value) throw new Error(`createApiKey failed: ${JSON.stringify(created.data)}`);
	fs.writeFileSync(KEY_FILE, value);
	ok(`API key minted (${value.slice(0, 8)}...) -> ${path.relative(REPO, KEY_FILE)}`);
	return value;
}

async function main() {
	// --mint-only: app already up + DB seeded; just (re)mint the key and vault.
	const mintOnly = has("--mint-only");

	if (!mintOnly) {
		// 1. build plugin via esbuild directly (the `npm run build` tsc gate is
		// currently broken by a parse error in node_modules/@types/d3-dispatch,
		// unrelated to sync; esbuild bundles main.js without typechecking).
		if (!has("--no-build")) {
			log("🔨 building plugin (main.js via esbuild)...");
			sh("node", ["esbuild.config.mjs", "production"], { cwd: path.join(REPO, "obsidian-sync") });
		}

		// 2. seed DB
		log(`🗄️  seeding ${DB}`);
		fs.mkdirSync(path.join(REPO, "tmp", "data"), { recursive: true });
		for (const f of [DB, `${DB}-shm`, `${DB}-wal`]) fs.rmSync(path.join(REPO, f), { force: true });
		const seed = spawnSync("sh", ["-c", `sqlite3 ${DB} < testdata/e2e_seed.sql`], { cwd: REPO, stdio: "inherit" });
		if (seed.status !== 0) throw new Error("DB seed failed");

		// 3. up — compose builds the image if missing; --rebuild forces a rebuild.
		log("🚀 docker compose up (app + minio, no vector search)...");
		sh("docker", [...COMPOSE, "up", "-d", ...(has("--rebuild") ? ["--build"] : []), "app", "minio"]);
	}

	// 4. health
	await waitHealthy();

	// 5. mint key
	await mintApiKey();

	// 6. generate vault with the key baked in
	const assetMode = argVal("--asset-mode", "unique");
	log(`📝 generating vault (${count} notes, assets=${assetMode})...`);
	sh("node", [path.join(HERE, "gen-vault.mjs"), "--count", count, "--out", VAULT, "--api-url", APP, "--asset-mode", assetMode]);

	console.log("\n\x1b[1m\x1b[32mREADY\x1b[0m");
	console.log(`  Server:   ${APP}   (GraphQL ${GQL})`);
	console.log(`  Vault:    ${VAULT}`);
	console.log(`  API key:  ${path.relative(REPO, KEY_FILE)}`);
	console.log(`\n  → Open Obsidian on the vault:  obsidian "${VAULT}"  (or File → Open vault)`);
	console.log(`  → Then run the benchmark:      node scripts/syncperf/bench.mjs`);
}

main().catch((e) => { console.error(`\x1b[31m✗ ${e.message}\x1b[0m`); process.exit(1); });
