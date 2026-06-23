import esbuild from "esbuild";
import path from "path";
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const isWatch = process.argv.includes("--watch");

const base = {
  bundle: true,
  platform: "browser",
  target: "es2020",
  format: "iife",
  logLevel: "info",
  sourcemap: false,
  treeShaking: true,
  minify: true,
};

// Two outputs: the tiny glue (codeblock.js) and the highlight.js library
// wrapped as window.hljs (highlight.min.js), lazy-loaded by the glue.
const builds = [
  { ...base, entryPoints: [path.resolve(__dirname, "src/index.ts")], outfile: path.resolve(__dirname, "../codeblock.js") },
  { ...base, entryPoints: [path.resolve(__dirname, "src/lib.ts")], outfile: path.resolve(__dirname, "../highlight.min.js") },
];

if (isWatch) {
  for (const cfg of builds) {
    const ctx = await esbuild.context(cfg);
    await ctx.watch();
  }
  console.log("Watching...");
} else {
  for (const cfg of builds) {
    await esbuild.build(cfg);
  }
  console.log("Built: codeblock.js, highlight.min.js");
}
