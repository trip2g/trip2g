#!/usr/bin/env node
// @ts-check
//
// Generate a deterministic Obsidian vault for the obsidian-sync perf benchmark.
//
//   node scripts/syncperf/gen-vault.mjs [--count 2000] [--out tmp/syncperf-vault]
//   node scripts/syncperf/gen-vault.mjs --touch 5 --rev 1   # mutate first K notes
//
// What it does:
//   - writes <count> deterministic .md notes (frontmatter publish:true + ~300
//     words + a few wikilinks), spread 50-per-folder (area-00/note-0000.md ...)
//   - lays down a real .obsidian copied from docs/.obsidian, but replaces the
//     trip2g symlink with a REAL copy of the built plugin + an isolated data.json
//     (so it never touches your dev obsidian-sync/data.json)
//   - writes a .gitignore so the vault is never committed
//
// Notes are pure markdown (no binary assets) — the benchmark isolates note sync.
// The apiKey in data.json is left blank here; scripts/syncperf/setup.mjs mints it.

import fs from "node:fs";
import path from "node:path";
import zlib from "node:zlib";
import { fileURLToPath } from "node:url";

const REPO = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..");

function parseArgs() {
	const a = process.argv.slice(2);
	const out = {
		count: 2000,
		out: path.join(REPO, "tmp", "syncperf-vault"),
		touch: 0,
		rev: 1,
		apiUrl: "http://localhost:20071",
		// asset mode: "unique" = 1 PNG per note (2000 real uploads),
		// "shared" = all notes reference one shared.png (N upload attempts of 1 file),
		// "none" = no assets.
		assetMode: "unique",
	};
	for (let i = 0; i < a.length; i++) {
		const [k, inlineV] = a[i].includes("=") ? a[i].split(/=(.*)/s) : [a[i], undefined];
		const val = () => inlineV ?? a[++i];
		switch (k) {
			case "--count": out.count = parseInt(val(), 10); break;
			case "--out": out.out = path.resolve(val()); break;
			case "--touch": out.touch = parseInt(val(), 10); break;
			case "--rev": out.rev = parseInt(val(), 10); break;
			case "--api-url": out.apiUrl = val(); break;
			case "--asset-mode": {
				const m = val();
				if (!["unique", "shared", "none"].includes(m)) { console.error(`--asset-mode must be unique|shared|none`); process.exit(1); }
				out.assetMode = m; break;
			}
			case "--no-assets": out.assetMode = "none"; break;
			default: console.error(`unknown arg: ${k}`); process.exit(1);
		}
	}
	return out;
}

// Deterministic PRNG (mulberry32) so re-generation is byte-stable.
function rng(seed) {
	let s = seed >>> 0;
	return () => {
		s |= 0; s = (s + 0x6d2b79f5) | 0;
		let t = Math.imul(s ^ (s >>> 15), 1 | s);
		t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

const WORDS = (
	"note graph link publish sync vault server client hash content version " +
	"obsidian markdown frontmatter folder asset commit delta change event " +
	"subscription badge conflict resolve classify push pull remote local cache " +
	"performance latency payload throughput mutex reload render pipeline index " +
	"query schema resolver migration column counter snapshot baseline scenario"
).split(" ");

const pad = (n, w) => String(n).padStart(w, "0");
const notePath = (idx) => `area-${pad(Math.floor(idx / 50), 2)}/note-${pad(idx, 4)}.md`;
const assetPath = (idx) => `area-${pad(Math.floor(idx / 50), 2)}/note-${pad(idx, 4)}.png`;
const assetName = (idx) => `note-${pad(idx, 4)}.png`;

// Minimal valid 1x1 RGB PNG encoder. Color varies per note so each file has a
// distinct hash → real asset uploads (not content-addressed skips).
const CRC_TABLE = (() => {
	const t = new Uint32Array(256);
	for (let n = 0; n < 256; n++) {
		let c = n;
		for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
		t[n] = c >>> 0;
	}
	return t;
})();
function crc32(buf) {
	let c = 0xffffffff;
	for (let i = 0; i < buf.length; i++) c = CRC_TABLE[(c ^ buf[i]) & 0xff] ^ (c >>> 8);
	return (c ^ 0xffffffff) >>> 0;
}
function pngChunk(type, data) {
	const len = Buffer.alloc(4); len.writeUInt32BE(data.length);
	const t = Buffer.from(type, "ascii");
	const crc = Buffer.alloc(4); crc.writeUInt32BE(crc32(Buffer.concat([t, data])));
	return Buffer.concat([len, t, data, crc]);
}
function png1x1(idx) {
	const sig = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
	const ihdr = Buffer.alloc(13);
	ihdr.writeUInt32BE(1, 0); ihdr.writeUInt32BE(1, 4); ihdr[8] = 8; ihdr[9] = 2; // 1x1, 8-bit RGB
	const raw = Buffer.from([0, idx & 0xff, (idx >> 8) & 0xff, (idx >> 4) & 0xff]); // filter + RGB
	return Buffer.concat([sig, pngChunk("IHDR", ihdr), pngChunk("IDAT", zlib.deflateSync(raw)), pngChunk("IEND", Buffer.alloc(0))]);
}

function genBody(idx, count, assetMode) {
	const embed = assetMode === "unique" ? assetName(idx) : assetMode === "shared" ? "shared.png" : null;
	const rand = rng(idx + 1);
	const pick = () => WORDS[Math.floor(rand() * WORDS.length)];
	const paras = [];
	for (let p = 0; p < 4; p++) {
		const words = [];
		for (let w = 0; w < 60; w++) words.push(pick());
		const s = words.join(" ");
		paras.push(s.charAt(0).toUpperCase() + s.slice(1) + ".");
	}
	// A few deterministic wikilinks to other notes (graph realism for the renderer).
	const links = [1, 7, 50, 333]
		.map((off) => (idx + off) % count)
		.map((j) => `[[${notePath(j).replace(/\.md$/, "")}]]`);
	const tags = `[bench, area-${pad(Math.floor(idx / 50), 2)}]`;
	return [
		"---",
		`title: Note ${pad(idx, 4)}`,
		"publish: true",
		`tags: ${tags}`,
		"---",
		"",
		`# Note ${pad(idx, 4)}`,
		"",
		...(embed ? [`![[${embed}]]`, ""] : []),
		paras[0],
		"",
		"## Related",
		"",
		links.map((l) => `- ${l}`).join("\n"),
		"",
		"## Details",
		"",
		paras[1],
		"",
		paras[2],
		"",
		paras[3],
		"",
	].join("\n");
}

function setupObsidian(outDir, apiUrl) {
	const srcDot = path.join(REPO, "docs", ".obsidian");
	const dstDot = path.join(outDir, ".obsidian");
	fs.rmSync(dstDot, { recursive: true, force: true });
	// Copy everything except the trip2g symlink (broken when relocated).
	fs.cpSync(srcDot, dstDot, {
		recursive: true,
		dereference: false,
		filter: (src) => !src.endsWith(path.join(".obsidian", "plugins", "trip2g")),
	});
	// Real copy of the built plugin so the live plugin works in this vault,
	// fully isolated from the dev obsidian-sync/data.json.
	const pluginDst = path.join(dstDot, "plugins", "trip2g");
	fs.mkdirSync(pluginDst, { recursive: true });
	for (const f of ["main.js", "manifest.json", "styles.css"]) {
		const src = path.join(REPO, "obsidian-sync", f);
		if (!fs.existsSync(src)) {
			console.error(`✗ missing ${src} — run: (cd obsidian-sync && npm run build)`);
			process.exit(1);
		}
		fs.copyFileSync(src, path.join(pluginDst, f));
	}
	// Preserve an already-minted key across re-gen if present.
	const dataPath = path.join(pluginDst, "data.json");
	let apiKey = "";
	const keyFile = path.join(REPO, "scripts", "syncperf", ".syncperf-api-key");
	if (fs.existsSync(keyFile)) apiKey = fs.readFileSync(keyFile, "utf8").trim();
	const data = {
		syncDirs: [{ path: "/", apiKey, apiUrl, twoWaySync: true }],
		skipPushConfirmation: true,
		hideSyncStatus: false,
	};
	fs.writeFileSync(dataPath, JSON.stringify(data, null, 2));
	// Marker so the hot-reload plugin reloads trip2g when we overwrite main.js.
	fs.writeFileSync(path.join(pluginDst, ".hotreload"), "");
}

function main() {
	const args = parseArgs();
	const outDir = args.out;

	if (args.touch > 0) {
		// Mutate the first K notes (small-change scenario) — append a rev marker.
		for (let i = 0; i < args.touch; i++) {
			const p = path.join(outDir, notePath(i));
			let body = fs.readFileSync(p, "utf8").replace(/\n<!-- rev \d+ -->\n?$/, "\n");
			body = body.replace(/\n+$/, "\n") + `<!-- rev ${args.rev} -->\n`;
			fs.writeFileSync(p, body);
		}
		console.log(`✏️  touched ${args.touch} notes (rev ${args.rev})`);
		return;
	}

	fs.mkdirSync(outDir, { recursive: true });
	// Clean previously generated notes (areas) but keep .obsidian / state.
	for (const e of fs.readdirSync(outDir)) {
		if (e.startsWith("area-")) fs.rmSync(path.join(outDir, e), { recursive: true, force: true });
	}

	fs.rmSync(path.join(outDir, "shared.png"), { force: true }); // stale from a previous --asset-mode shared
	for (let i = 0; i < args.count; i++) {
		const p = path.join(outDir, notePath(i));
		fs.mkdirSync(path.dirname(p), { recursive: true });
		fs.writeFileSync(p, genBody(i, args.count, args.assetMode));
		if (args.assetMode === "unique") fs.writeFileSync(path.join(outDir, assetPath(i)), png1x1(i));
	}
	if (args.assetMode === "shared") fs.writeFileSync(path.join(outDir, "shared.png"), png1x1(0));
	fs.writeFileSync(path.join(outDir, ".gitignore"), "*\n");
	setupObsidian(outDir, args.apiUrl);

	const assetMsg = args.assetMode === "unique" ? ` + ${args.count} unique 1x1 PNGs`
		: args.assetMode === "shared" ? ` + 1 shared 1x1 PNG (referenced by all)` : "";
	console.log(`✓ generated ${args.count} notes${assetMsg} in ${outDir}`);
	console.log(`  .obsidian ready (trip2g plugin copied, apiUrl=${args.apiUrl})`);
}

main();
