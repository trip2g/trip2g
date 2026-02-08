import esbuild from "esbuild";
import path from "path";
import fs from "fs";
import { fileURLToPath } from 'url';
import { createRequire } from 'module';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);
const isWatch = process.argv.includes("--watch");

// Plugin to inline CSS imports as <style> tags injected at runtime.
const inlineCssPlugin = {
	name: 'inline-css',
	setup(build) {
		build.onResolve({ filter: /\.css$/ }, (args) => {
			const resolved = require.resolve(args.path, { paths: [args.resolveDir] });
			return { path: resolved, namespace: 'inline-css' };
		});
		build.onLoad({ filter: /.*/, namespace: 'inline-css' }, async (args) => {
			const css = await fs.promises.readFile(args.path, 'utf8');
			return {
				contents: `(function(){
					if (typeof document !== 'undefined') {
						var s = document.createElement('style');
						s.textContent = ${JSON.stringify(css)};
						document.head.appendChild(s);
					}
				})();`,
				loader: 'js',
			};
		});
	},
};

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
	plugins: [inlineCssPlugin],
};

if (isWatch) {
	const ctx = await esbuild.context(iifeConfig);
	await ctx.watch();
	console.log("Watching for changes...");
} else {
	await esbuild.build(iifeConfig);
	console.log("IIFE bundle built: milkdown.js");
}
