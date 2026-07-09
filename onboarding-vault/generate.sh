#!/usr/bin/env bash
# Regenerate vault.zip with the current built plugin from obsidian-sync/.
# Run via: go generate ./onboarding-vault/...
# or: make vault-zip
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PLUGIN_SRC="$REPO_ROOT/obsidian-sync"
PLUGIN_DST="$REPO_ROOT/onboarding-vault/.obsidian/plugins/trip2g"

# Copy built plugin artifacts from obsidian-sync/ into the vault.
# Skipped when obsidian-sync/ is absent (e.g. Docker build context) — the
# committed files in onboarding-vault/.obsidian/plugins/trip2g/ are used as-is.
if [ -f "$PLUGIN_SRC/main.js" ]; then
	for f in main.js trip2g-sync.mjs styles.css manifest.json; do
		cp "$PLUGIN_SRC/$f" "$PLUGIN_DST/$f"
	done
	VER=$(jq -r .version "$PLUGIN_SRC/manifest.json")
	echo "Copied plugin v$VER from obsidian-sync into onboarding-vault"
else
	echo "obsidian-sync not present — packing committed plugin files"
fi

# Repack vault.zip with deterministic timestamps so the zip is reproducible.
python3 - "$REPO_ROOT" <<'PYEOF'
import zipfile, pathlib, sys

repo_root = pathlib.Path(sys.argv[1]).resolve()
vault_dir = repo_root / "onboarding-vault"
out_zip   = vault_dir / "vault.zip"

EXCLUDE = {
    vault_dir / "embed.go",
    vault_dir / "generate.sh",
    vault_dir / "vault.zip",
    vault_dir / ".obsidian" / "workspace.json",
    # data.json contains real credentials — never bundle
    vault_dir / ".obsidian" / "plugins" / "trip2g" / "data.json",
    vault_dir / "AGENTS.md",
}

FIXED_DATE = (2024, 1, 1, 0, 0, 0)

entries = sorted(vault_dir.rglob("*"))

with zipfile.ZipFile(out_zip, "w", compression=zipfile.ZIP_DEFLATED) as zf:
    for path in entries:
        if path in EXCLUDE:
            continue
        rel = path.relative_to(repo_root)
        hidden = 0x02 if path.name.startswith(".") else 0   # FILE_ATTRIBUTE_HIDDEN for dotfiles (Windows)
        if path.is_dir():
            zi = zipfile.ZipInfo(str(rel) + "/", date_time=FIXED_DATE)
            zi.compress_type = zipfile.ZIP_STORED
            zi.external_attr = (0o40755 << 16) | 0x10 | hidden
            zf.writestr(zi, "")
        else:
            zi = zipfile.ZipInfo(str(rel), date_time=FIXED_DATE)
            zi.compress_type = zipfile.ZIP_DEFLATED
            zi.external_attr |= hidden
            zf.writestr(zi, path.read_bytes())

print(f"Wrote {out_zip} ({out_zip.stat().st_size} bytes)")
PYEOF
