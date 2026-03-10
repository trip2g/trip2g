import esbuild from "esbuild";
import path from "path";
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const isWatch = process.argv.includes("--watch");

const config = {
  entryPoints: [path.resolve(__dirname, "src/index.ts")],
  bundle: true,
  platform: "browser",
  target: "es2020",
  format: "iife",
  logLevel: "info",
  sourcemap: false,
  treeShaking: true,
  minify: true,
  outfile: path.resolve(__dirname, "toc.js"),
};

if (isWatch) {
  const ctx = await esbuild.context(config);
  await ctx.watch();
  console.log("Watching...");
} else {
  await esbuild.build(config);
  console.log("Built: toc.js");
}
