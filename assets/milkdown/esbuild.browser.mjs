import esbuild from "esbuild";
import path from "path";
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const isWatch = process.argv.includes("--watch");

// IIFE bundle for mol/require() usage
// Exports to global $trip2g_milkdown_bundle
const iifeConfig = {
	entryPoints: [path.resolve(__dirname, "src/index.ts")],
	bundle: true,
	platform: "browser",
	target: "es2020",
	format: "iife",
	globalName: "$trip2g_milkdown_bundle",
	logLevel: "info",
	sourcemap: false,
	treeShaking: true,
	minify: false,
	outfile: path.resolve(__dirname, "milkdown.js"),
	external: [],
};

if (isWatch) {
	const ctx = await esbuild.context(iifeConfig);
	await ctx.watch();
	console.log("👀 Watching for changes...");
} else {
	await esbuild.build(iifeConfig);
	console.log("✅ IIFE bundle built: milkdown.js");
	console.log("📝 Types: milkdown.bundle.d.ts (manual)");
}
