#!/usr/bin/env node
// @ts-check
//
// Benchmark obsidian-sync on the syncperf stack via the CLI (same classify/execute
// code as the plugin). Appends a labeled section to the report so before/after
// fixes compare cleanly.
//
//   node scripts/syncperf/bench.mjs [--label baseline] [--repeats 3] [--touch 5]
//
// Scenarios:
//   cold         first sync of all notes (empty .sync-state.json) — push everything
//   noop         re-sync, nothing changed (the idle case: dead cache re-hashes all)
//   dry-noop     --dry-run, nothing changed (classify only: local hash + getServerHashes)
//   small-change touch K notes, sync (incremental push)
//   twoway-noop  --two-way, nothing changed (adds pull-side asset handling)
//
// Decomposition hint in the report: (noop − dry-noop) ≈ execute/asset overhead.

import { spawnSync, execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const REPO = path.resolve(HERE, "..", "..");
const PLUGIN_DIR = path.join(REPO, "obsidian-sync");
const VAULT = path.join(REPO, "tmp", "syncperf-vault");
const GQL = "http://localhost:20071/_system/graphql";
const KEY_FILE = path.join(HERE, ".syncperf-api-key");
const REPORT = path.join(REPO, "docs", "dev", "obsidian_sync_bench_2026-06-21.md");

const args = process.argv.slice(2);
const argVal = (f, d) => { const i = args.indexOf(f); return i >= 0 ? args[i + 1] : d; };
const label = argVal("--label", "baseline");
const repeats = parseInt(argVal("--repeats", "3"), 10);
const touchK = parseInt(argVal("--touch", "5"), 10);

if (!fs.existsSync(KEY_FILE)) {
	console.error("✗ no API key — run: node scripts/syncperf/setup.mjs");
	process.exit(1);
}
const apiKey = fs.readFileSync(KEY_FILE, "utf8").trim();
const env = { ...process.env, TRIP2G_API_KEY: apiKey, TRIP2G_ENDPOINT: GQL };

const stateFile = path.join(VAULT, ".sync-state.json");
const note = (m) => console.log(`\x1b[36m${m}\x1b[0m`);

/** Run the CLI once, return { ms, out, parsed }. */
function runCli(extra = []) {
	const t0 = process.hrtime.bigint();
	const r = spawnSync("npx", ["tsx", "src/sync/cli/cmd.ts", "--folder", VAULT, ...extra], {
		cwd: PLUGIN_DIR, env, encoding: "utf8", maxBuffer: 64 * 1024 * 1024,
	});
	const ms = Number(process.hrtime.bigint() - t0) / 1e6;
	const out = (r.stdout || "") + (r.stderr || "");
	const num = (re) => { const m = out.match(re); return m ? parseInt(m[1], 10) : null; };
	const parsed = {
		toPush: num(/To push:\s+(\d+)/),
		unchanged: num(/Unchanged:\s+(\d+)/),
		pushed: num(/Pushed:\s+(\d+)/),
		pulled: num(/Pulled:\s+(\d+)/),
		assetsUp: num(/Assets uploaded:\s+(\d+)/),
		errors: num(/Errors:\s+(\d+)/),
	};
	if (r.status !== 0) console.error(out.slice(-2000));
	return { ms, out, parsed, status: r.status ?? -1 };
}

function median(xs) { const s = [...xs].sort((a, b) => a - b); const m = s.length >> 1; return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2; }
const fmt = (ms) => `${(ms / 1000).toFixed(2)}s`;

function repeat(name, extra, n) {
	const times = [];
	let last = null;
	for (let i = 0; i < n; i++) {
		if (name === "small-change") {
			spawnSync("node", [path.join(HERE, "gen-vault.mjs"), "--out", VAULT, "--touch", String(touchK), "--rev", String(Date.now() + i)], { cwd: REPO, stdio: "ignore" });
		}
		const r = runCli(extra);
		times.push(r.ms);
		last = r;
		note(`  ${name} run ${i + 1}/${n}: ${fmt(r.ms)}  (push=${r.parsed.toPush ?? r.parsed.pushed ?? "?"} err=${r.parsed.errors ?? 0})`);
	}
	return { name, times, median: median(times), min: Math.min(...times), max: Math.max(...times), last };
}

function main() {
	const gitHash = (() => { try { return execSync("git rev-parse --short HEAD", { cwd: REPO }).toString().trim(); } catch { return "?"; } })();
	const noteCount = (() => { try { return execSync(`find "${VAULT}" -name '*.md' | wc -l`).toString().trim(); } catch { return "?"; } })();

	console.log(`\n\x1b[1mobsidian-sync benchmark — label="${label}", repeats=${repeats}, notes=${noteCount}\x1b[0m\n`);

	const results = [];

	note("▶ cold (fresh state, push all)");
	fs.rmSync(stateFile, { force: true });
	const cold = runCli([]);
	note(`  cold: ${fmt(cold.ms)}  (pushed=${cold.parsed.pushed ?? "?"} assetsUp=${cold.parsed.assetsUp ?? 0} err=${cold.parsed.errors ?? 0})`);
	results.push({ name: "cold", times: [cold.ms], median: cold.ms, min: cold.ms, max: cold.ms, last: cold });

	note("▶ noop (nothing changed)");
	results.push(repeat("noop", [], repeats));

	note("▶ dry-noop (classify only)");
	results.push(repeat("dry-noop", ["--dry-run"], repeats));

	note(`▶ small-change (touch ${touchK} notes)`);
	results.push(repeat("small-change", [], repeats));

	note("▶ twoway-noop (--two-way, nothing changed)");
	results.push(repeat("twoway-noop", ["--two-way"], repeats));

	// Build report section.
	const ts = new Date().toISOString().replace("T", " ").slice(0, 19);
	const rows = results.map((r) => {
		const p = r.last.parsed;
		const counts = `push=${p.toPush ?? p.pushed ?? 0}, pushed=${p.pushed ?? 0}, assetsUp=${p.assetsUp ?? 0}, err=${p.errors ?? 0}`;
		return `| ${r.name} | ${fmt(r.median)} | ${fmt(r.min)} | ${fmt(r.max)} | ${r.times.length} | ${counts} |`;
	}).join("\n");

	const noopMed = results.find((r) => r.name === "noop")?.median ?? 0;
	const dryMed = results.find((r) => r.name === "dry-noop")?.median ?? 0;
	const execOverhead = noopMed - dryMed;

	const section = [
		`## Run: ${label} (${ts})`,
		"",
		`- commit \`${gitHash}\`, notes ${noteCount}, repeats ${repeats}, touch ${touchK}, node ${process.version}`,
		`- stack: docker-compose.syncperf.yml (vector search OFF), CLI via tsx`,
		`- decomposition: noop − dry-noop ≈ execute/asset overhead = **${fmt(execOverhead)}**`,
		"",
		"| scenario | median | min | max | runs | counts |",
		"|---|---|---|---|---|---|",
		rows,
		"",
	].join("\n");

	let head = "";
	if (!fs.existsSync(REPORT)) {
		head = [
			"# Obsidian Sync — benchmark (2026-06-21)",
			"",
			"Замеры времени синка на ~2000 заметках через obsidian-sync CLI (тот же",
			"`classify`/`execute`, что и в плагине), стек `docker-compose.syncperf.yml`",
			"без векторного поиска. Каждый прогон `bench.mjs` дописывает секцию ниже —",
			"так baseline и «после фикса» сравниваются в одном файле.",
			"",
			"Сценарии: **cold** (первый пуш всех заметок), **noop** (повтор без изменений —",
			"idle, где мёртвый кэш пере-хэширует всё), **dry-noop** (только classify),",
			"**small-change** (правка K заметок), **twoway-noop** (`--two-way` без изменений).",
			"",
			"---",
			"",
		].join("\n");
	}
	fs.writeFileSync(REPORT, (fs.existsSync(REPORT) ? fs.readFileSync(REPORT, "utf8") : head) + section);

	console.log(`\n\x1b[32m✓ appended results to ${path.relative(REPO, REPORT)}\x1b[0m`);
	console.log(`  noop median ${fmt(noopMed)}, dry ${fmt(dryMed)}, exec overhead ${fmt(execOverhead)}`);
}

main();
