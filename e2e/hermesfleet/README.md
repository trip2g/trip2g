# Hermes fleet demo

Proves `cmd/fleet` can run on a **real** Hermes agent instead of the mock LLM:
drop a task note in `tasks/`, sync, and a result note shows up in `results/` —
both on the site and, after a two-way sync, in the vault folder on disk.

```
vault → trip2g → change webhook → fleet → hermesllm → hermes → ChatGPT/Codex
```

```bash
./e2e/hermesfleet/run.sh            # boot, run both tasks, tear down
./e2e/hermesfleet/run.sh --keep     # leave the stack up for poking
```

## Why the `hermesllm` shim

Hermes exposes an OpenAI-compatible chat-completions API, but it ignores the
`tools` field and never returns `tool_calls` — it is an agent, not a raw model.
Fleet's whole loop is tool calls (`read_note`, `write_note`), so it cannot talk
to Hermes directly. `cmd/hermesllm` sits between them: it accepts fleet's
OpenAI request, drives Hermes, and hands the answer back as the `tool_calls`
fleet expects.

## Prerequisites

- `docker compose`, `node`, `jq`, `unzip`, `python3`.
- A logged-in Codex CLI: `~/.codex/auth.json` must exist (`codex login`).
  `run.sh` translates its ChatGPT tokens into Hermes' own `auth.json` format and
  seeds them into the `hermes-data` volume before the stack starts. The tokens
  are never printed and the seed directory is deleted on exit.

It costs real tokens, so it is **not** part of `make test` or the Playwright
suite — run it by hand.

## What it does

1. Seeds the Hermes home volume (`auth.json` + `config.yaml`, uid/gid `10000`),
   then brings up `docker-compose.hermes.yml` with per-run API keys.
2. Signs in as the owner (dev code `111111`) and downloads a vault through the
   real `/_system/onboarding-vault` endpoint, with `enable_admin_graphql` so the
   harness can query the site through the vault's own `trip2g-sync.mjs graphql`.
3. Publishes `roles/task-runner.md` and waits for fleet to register its change
   webhook — a task created before that would never be delivered.
4. Publishes `tasks/task1.md` ("compute 10 + 20") and `tasks/task2.md`
   ("multiply 7 by 6") in the same round, then polls `results/` until both
   answers land. Two independent tasks, so one lucky answer is not enough.
5. Runs a second sync with `--two-way` and asserts `results/*.md` now exist on
   disk in the vault folder.

Container logs for all four services are dumped to `tmp/hermesfleet/logs/`, on
success and on failure alike — the teardown grabs them before removing the stack.

## Ports

| Service | Host | Container | Notes |
|---|---|---|---|
| `app` | 20281, 20282 | 20281, 20282 | public + internal (`/healthz`) |
| `fleet` | 29095 | 9090 | `/health` |
| `hermesllm` | 29096 | 8088 | `/healthz` |
| `hermes` | 28642 | 8642 | `/health`; published for debugging only |

## Notes

- Hermes cold start is 20-40 s and a first inference round-trip is ~10 s, hence
  the 60 s healthcheck `start_period` and the 5-minute result poll.
- The role note is the whole configuration: `fleet_id: hermes` partitions it to
  this fleet, `trigger_include: ["tasks/**"]` plus `max_depth: 1` keep the
  write-back to `results/` from re-triggering the webhook, and
  `write_patterns: ["results/**"]` is what stops the agent writing anywhere else.
- The `trip2g-hermesfleet-hermes-data` volume holds **your real ChatGPT OAuth
  access and refresh tokens**, copied from `~/.codex/auth.json`. `docker compose
  down -v` on exit takes it with it, so they do not outlive the run; `--keep`
  leaves them in place. To clean up a kept stack later:

  ```bash
  docker compose -f docker-compose.hermes.yml down -v
  docker volume rm trip2g-hermesfleet-hermes-data   # fallback, if the above did not remove it
  ```

  Both work without any environment variables set.
