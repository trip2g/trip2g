# Test scenario: memcli OKF / LLM-wiki seed + lint

Branch: `feat/memcli-okf-llmwiki` (worktree `/home/alexes/projects2/trip2g-memcli-okf`).
What changed: `memcli up` now seeds an OKF / LLM-wiki starter, and there is a new
`memcli lint` command (plus `memory_lint` MCP tool). The `export --okf` idea is
deferred (see GitHub issue trip2g#26), so it is NOT part of this scenario.

The goal of this scenario is to test memory end to end with the new changes:
boot it, confirm a fresh vault is OKF-valid out of the box, exercise the
remember/recall loop, and confirm the lint commit-gate catches a broken note.

---

## 0. Prerequisites

- Docker available (the `up` path boots `ghcr.io/trip2g/trip2g:latest`, which must
  include the local-filestorage backend).
- Node 20+.
- Run everything from the worktree root: `cd /home/alexes/projects2/trip2g-memcli-okf`.
- Build first: `cd cli/memcli && npm install && npm run build`. Then `MEMCLI="node dist/memcli.js"`.
- Use a throwaway vault: `VAULT=$(mktemp -d)/memvault`.

If Docker is NOT available, run the **offline subset** only: steps 1, 2, 5, 8
(these need no server). Note in the report that the server-dependent steps were skipped.

---

## 1. Fresh vault is OKF-valid out of the box (offline, dry-run)

```bash
$MEMCLI up --no-hub --dry-run --folder "$VAULT"
```

**Expect:** output reports it would write `index.md`, `log.md`, `AGENTS.md`,
`SCHEMA.md`. No files written (dry-run).

## 2. Seed builders carry the right metadata (offline, unit tests)

```bash
cd cli/memcli && npm test
```

**Expect:** all tests pass, including:
- each seed builder output contains a `type:` line,
- `AGENTS.md` contains `mcp_method: instructions`,
- `SCHEMA.md` contains `mcp_method: schema`,
- `lintVault` detects missing `type`, an unresolved marker, missing index/log, and a stale note,
- `lintVault` passes clean on a well-formed vault.

## 3. Boot real memory (needs Docker)

```bash
$MEMCLI up --folder "$VAULT"
```

**Expect:** server live on http://localhost:24081, watcher started, and the four
seed files actually present on disk:

```bash
for f in index.md log.md AGENTS.md SCHEMA.md; do
  echo "== $f =="; head -6 "$VAULT/$f"
done
```

Every file must have a `type:` in frontmatter. `AGENTS.md` describes the
search → expand → note_html workflow and the commit-gate rule.

## 4. Lint a clean vault (PASS)

```bash
$MEMCLI lint --folder "$VAULT"; echo "exit=$?"
```

**Expect:** no error-level violations, `exit=0`. (A fresh seeded vault may still
warn on notes without `type`, but the seeds themselves should all have one.)

## 5. Lint catches a broken note (commit-gate)

```bash
printf -- '---\ntitle: Broken\n---\n\n## Conflict\nStatus: Unresolved\n' > "$VAULT/broken.md"
printf -- '---\ntitle: No type here\n---\n\nbody\n' > "$VAULT/notype.md"
$MEMCLI lint --folder "$VAULT"; echo "exit=$?"
```

**Expect:** an **error** for `broken.md` (commit-gate: `Status: Unresolved`),
a **warn** for `notype.md` (missing `type:`), and `exit` non-zero because of the
error-level violation. Then delete the two files and confirm lint is green again.

## 6. Remember / recall loop (needs Docker)

```bash
$MEMCLI daily "Costya runs an AI-native company; cares about ROI not token spend." --folder "$VAULT"
$MEMCLI log token-economy "section reads are ~15x cheaper than whole-note dumps" --folder "$VAULT"
```

**Expect:** `daily/<today>.md` created with `type: daily` frontmatter; `token-economy.md`
created with a `### [[<today>]]` section. Both OKF-valid. Re-run `lint` → green.

## 7. Recall via MCP (needs Docker)

Drive the MCP endpoint (`/_system/mcp`) — use the stdio adapter or raw JSON-RPC.
Mint/grab a token, then:

1. `search("token economy")` → returns a slim snippet + `toc_path`, NOT the whole note.
2. `note_html(path, toc_path)` → returns only the matching section.
3. Confirm the remembered fact ("~15x cheaper") comes back.
4. Confirm token efficiency: the section read is much smaller than the whole note.

Also call the new `memory_lint` MCP tool and confirm it returns the same verdict
as the CLI `lint`.

## 8. Idempotency + `--no-seed`

```bash
# Re-running up must not overwrite existing seeds:
$MEMCLI up --folder "$VAULT"           # seeds already exist → skipped, not clobbered
# A fresh vault with --no-seed writes NO starter files:
$MEMCLI up --no-hub --no-seed --dry-run --folder "$(mktemp -d)/v2"
```

**Expect:** step one reports the seeds already present (no overwrite). Step two does
NOT list index/log/AGENTS/SCHEMA among files it would write.

## 9. Teardown

```bash
$MEMCLI down --folder "$VAULT"
```

---

## Report format

For each step: PASS / FAIL, the command output (trimmed), and for any FAIL the exact
error. End with: does a fresh `memcli up` vault validate as OKF, does lint catch the
broken note, and does recall return the remembered fact through MCP. Note any step
skipped because Docker was unavailable.
