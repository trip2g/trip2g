import esbuild from "esbuild";
import { sassPlugin } from "esbuild-sass-plugin";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Output goes directly into the Go package for //go:embed
const outfile = path.resolve(__dirname, "../../internal/defaulttemplate/defaulttemplate.css");

await esbuild.build({
  entryPoints: [path.resolve(__dirname, "src/index.scss")],
  bundle: true,
  outfile,
  minify: true,
  plugins: [
    sassPlugin({
      type: "css",
    }),
  ],
  loader: { ".css": "css" },
  logLevel: "info",
});

console.log(`Built: ${outfile}`);
