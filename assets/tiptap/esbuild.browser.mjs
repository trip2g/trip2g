import esbuild from "esbuild";
import path from "path";
import { fileURLToPath } from 'url';
import { createRequire } from 'module';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);
const isWatch = process.argv.includes("--watch");

// Inline CSS as <style> injected at runtime (same approach as milkdown).
const inlineCssPlugin = {
	name: 'inline-css',
	setup(build) {
		build.onResolve({ filter: /\.css$/ }, (args) => {
			const resolved = require.resolve(args.path, { paths: [args.resolveDir] });
			return { path: resolved, namespace: 'inline-css' };
		});
		build.onLoad({ filter: /.*/, namespace: 'inline-css' }, async (args) => {
			const result = await esbuild.build({
				entryPoints: [args.path],
				bundle: true,
				write: false,
				minify: false,
				loader: { '.css': 'css' },
				logLevel: 'warning',
			});
			const css = result.outputFiles[0].text;
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

const iifeConfig = {
	entryPoints: [path.resolve(__dirname, "src/index.ts")],
	bundle: true,
	platform: "browser",
	target: "es2020",
	format: "iife",
	globalName: "$trip2g_tiptap_bundle",
	logLevel: "info",
	sourcemap: false,
	treeShaking: true,
	minify: false,
	outfile: path.resolve(__dirname, "tiptap.js"),
	external: [],
	plugins: [inlineCssPlugin],
};

if (isWatch) {
	const ctx = await esbuild.context(iifeConfig);
	await ctx.watch();
	console.log("Watching for changes...");
} else {
	await esbuild.build(iifeConfig);
	console.log("IIFE bundle built: tiptap.js");
}
