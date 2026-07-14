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

// Three outputs:
//  - mermaid.js — the glue + diagram-type router + pan/zoom/export
//    enhancements (@mostlylucid/mermaid-enhancements is small enough to
//    bundle eagerly here, ~15 KB gzipped).
//  - mermaid.min.js — mermaid.js itself, wrapped as a window.mermaid global,
//    lazy-loaded by the glue for diagram types beautiful-mermaid can't render.
//  - beautifulmermaid.min.js — beautiful-mermaid wrapped as a
//    window.beautifulMermaid global, lazy-loaded by the glue for the 6
//    diagram types it supports. Heavy (pulls in ELK.js, ~1.6 MB minified),
//    so it stays its own lazy chunk rather than joining mermaid.js.
const builds = [
  { ...base, entryPoints: [path.resolve(__dirname, "src/index.ts")], outfile: path.resolve(__dirname, "../mermaid.js") },
  { ...base, entryPoints: [path.resolve(__dirname, "src/lib.ts")], outfile: path.resolve(__dirname, "../mermaid.min.js") },
  { ...base, entryPoints: [path.resolve(__dirname, "src/lib-beautiful.ts")], outfile: path.resolve(__dirname, "../beautifulmermaid.min.js") },
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
  console.log("Built: mermaid.js, mermaid.min.js, beautifulmermaid.min.js");
}
