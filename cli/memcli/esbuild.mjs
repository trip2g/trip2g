import esbuild from "esbuild";
import fs from "fs";
import path from "path";

await esbuild.build({
  entryPoints: ["src/cli.ts"],
  bundle: true,
  platform: "node",
  target: "node20",
  format: "esm",
  minify: false,
  keepNames: true,
  logLevel: "info",
  banner: { js: "#!/usr/bin/env node" },
  outfile: "dist/memcli.js",
});
fs.chmodSync("dist/memcli.js", 0o755);
console.log("✅ memcli built: dist/memcli.js");

// Propagate the runnable bundle into the onboarding vault plugin dir so an agent
// that pulls the onboarding vault gets the memory bootstrap CLI alongside
// trip2g-sync.mjs. cwd here is cli/memcli; the vault is two levels up.
const root = fs.realpathSync(process.cwd());
const targets = [
  path.resolve(root, "../../onboarding-vault/.obsidian/plugins/trip2g/memcli.js"),
];
for (const target of targets) {
  if (!fs.existsSync(path.dirname(target))) continue; // standalone build: skip missing vault
  fs.copyFileSync("dist/memcli.js", target);
  fs.chmodSync(target, 0o755);
  console.log(`✅ Propagated memcli → ${path.relative(root, target)}`);
}
