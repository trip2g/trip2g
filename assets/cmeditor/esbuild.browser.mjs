import esbuild from "esbuild";
import path from "path";
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

await esbuild.build({
  bundle: true,
  platform: "browser",
  target: "es2020",
  format: "iife",
  logLevel: "info",
  sourcemap: false,
  treeShaking: true,
  minify: true,
  entryPoints: [path.resolve(__dirname, "src/index.ts")],
  outfile: path.resolve(__dirname, "../cmeditor.js"),
});
