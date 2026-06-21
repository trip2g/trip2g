// In-process warm benchmark — eliminates the ~340ms tsx/node startup floor that
// dominates the per-invocation CLI numbers. Times classify (the idle path:
// re-hash all local files + fetch server hashes + classify) across N iterations
// in ONE process, like the long-running plugin does.
//
//   TRIP2G_API_KEY=... TRIP2G_ENDPOINT=http://localhost:20071/_system/graphql \
//   BENCH_FOLDER=tmp/syncperf-vault BENCH_ITERS=15 \
//   npx tsx scripts/syncperf/warmbench.mts
//
// Imports are relative to THIS file; transitive bare imports (graphql-request)
// resolve from obsidian-sync/src, so run with obsidian-sync's tsx.

// Dynamic import: handles ESM/CJS interop for the tsx-transpiled .ts modules
// (a static `import {NodeEnv}` from a .mts can't see the named export).
const { NodeEnv } = await import("../../obsidian-sync/src/sync/cli/env");
const { classifySync } = await import("../../obsidian-sync/src/sync/classify");

const folder = process.env.BENCH_FOLDER ?? "tmp/syncperf-vault";
const iters = parseInt(process.env.BENCH_ITERS ?? "15", 10);
const apiUrl = process.env.TRIP2G_ENDPOINT ?? "http://localhost:20071/_system/graphql";
const apiKey = process.env.TRIP2G_API_KEY ?? "";

function mkEnv() {
	return new NodeEnv({
		folder, prefix: "", apiUrl, apiKey,
		twoWaySync: false, verbose: false, conflictResolution: "local", meta: {},
	});
}

const ms = (ns: bigint) => Number(ns) / 1e6;

async function timeIt(label: string, fn: () => Promise<void>) {
	await fn(); // warm
	const ts: number[] = [];
	for (let i = 0; i < iters; i++) {
		const t = process.hrtime.bigint();
		await fn();
		ts.push(ms(process.hrtime.bigint() - t));
	}
	ts.sort((a, b) => a - b);
	const med = ts[ts.length >> 1];
	console.log(`${label.padEnd(28)} median ${med.toFixed(1)}ms  (min ${ts[0].toFixed(1)}, max ${ts[ts.length - 1].toFixed(1)}) over ${iters}`);
	return med;
}

// Full classify: getLocalFiles + (re)hash all + getServerHashes + classify.
await timeIt("classify (idle, full)", async () => { await classifySync(mkEnv()); });

// Local-only portion: hash all local files (no network) — isolates the dead-cache cost.
const env = mkEnv();
await timeIt("local hash all (2000)", async () => {
	const files = await env.getLocalFiles();
	for (const f of files) {
		const content = await env.readFileContent(f.path);
		await env.computeHash(content);
	}
});

// Network-only portion: fetch all server hashes — isolates the payload/round-trip cost.
await timeIt("fetch server hashes", async () => { await env.getServerHashes(); });
