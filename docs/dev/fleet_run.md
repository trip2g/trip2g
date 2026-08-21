# Fleet: internal run guide

**Status:** unreleased / internal. The fleet feature is not yet publicly
announced. This document is for developers running or testing the fleet locally.

The fleet (`cmd/fleet`) is the trip2g agent host. It polls trip2g for role
notes, reconciles change-webhooks so deliveries are routed back to itself,
and runs a scoped agent loop per delivery that writes results back via the
per-delivery scoped token. trip2g itself requires no change — it stays a dumb
event source.

---

## Prerequisites

1. A running trip2g instance with `DEV=true` (DevMode disables the SSRF guard
   so the fleet's loopback callback URL is accepted when delivering webhooks).
2. An LLM endpoint compatible with the OpenAI chat-completions API
   (e.g. OpenAI, Ollama, a local proxy).
3. A personal token belonging to an admin, seeded by the instance (see below).
4. Role notes under the `roles/` folder in trip2g (or whatever
   `--agents-folder` points at).

---

## Admin credential setup

The fleet's admin lane authenticates with a trip2g personal token (`t2g_*`)
whose user is an admin. You do not mint it by hand: set
`OWNER_PERSONAL_TOKEN_VALUE` on the instance and boot seeds the matching
`user_tokens` row for the owner, under the reserved name
`system: seeded by OWNER_PERSONAL_TOKEN_VALUE`.

```sh
# on the trip2g instance
OWNER_PERSONAL_TOKEN_VALUE=t2g_$(openssl rand -hex 32)
```

Pass that same value to the fleet as `--trip2g-admin-personal-token`. Changing
the value and restarting revokes the row the old one carried. Revoking the row
in the admin UI stops the fleet within the resolver's 30-second cache, and a
restart does not bring it back — the way back is a new value.

---

## Flags

All flags fall back to the corresponding environment variable when set.
Required flags have no default and cause a startup error if absent.

| Flag | Env var | Default | Notes |
|------|---------|---------|-------|
| `--fleet-id` | `FLEET_ID` | `fleet1` | Marker ID embedded in webhook descriptions; distinguishes concurrent fleet instances |
| `--listen` | `FLEET_LISTEN` | `:9090` | HTTP listen address for the delivery endpoint |
| `--callback-url` | `FLEET_CALLBACK_URL` | — | **Required.** trip2g-reachable base URL of this fleet (no trailing slash). trip2g posts webhook deliveries here |
| `--trip2g-url` | `TRIP2G_BASE_URL` | `http://localhost:8081` | Base URL the fleet uses to call trip2g's GraphQL / MCP endpoints |
| `--trip2g-admin-personal-token` | `TRIP2G_FLEET_TRIP2G_ADMIN_PERSONAL_TOKEN` | — | **Required.** Personal token (`t2g_*`) of an admin user; seeded by the instance from `OWNER_PERSONAL_TOKEN_VALUE` |
| `--fleet-secret` | `FLEET_SECRET` | — | **Required.** HMAC seed; fleet derives a per-role secret from it to verify delivery signatures |
| `--llm-base-url` | `FLEET_LLM_BASE_URL` | _(empty)_ | OpenAI-compatible base URL (e.g. `https://api.openai.com/v1`). Empty means the default OpenAI endpoint |
| `--llm-api-key` | `FLEET_LLM_API_KEY` | — | **Required.** LLM provider key |
| `--default-model` | `FLEET_DEFAULT_MODEL` | `gpt-4o-mini` | Model name used for roles without an explicit `model:` frontmatter field |
| `--token-ceiling` | — | `100000` | Hard per-run token cap; overrides any `max_tokens:` in a role note |
| `--step-ceiling` | — | `25` | Hard per-run step cap; overrides any `max_steps:` in a role note |
| `--agents-folder` | `FLEET_AGENTS_FOLDER` | `roles/` | Note-path LIKE prefix used when discovering role notes in trip2g |
| `--offered-tools` | — | `search,read_note,patch_note,write_note` | Comma-separated list of tools the fleet exposes to agents; roles may only use a subset |
| `--poll-seconds` | — | `30` | Discovery + reconcile interval in seconds |
| `--allow-role-authoring` | `TRIP2G_FLEET_ALLOW_ROLE_AUTHORING` | `false` | Let agents create and edit **role notes** (`fleet_id` in frontmatter). Off by default: a role declares its own `write_patterns`, so authoring one escalates scope. Logs a WARN at startup when on. See [fleet_write_validation.md](fleet_write_validation.md) |

---

## Networking

The fleet listens on `--listen` and registers `--callback-url` as the webhook
target in trip2g. trip2g posts delivery payloads to `<callback-url>/deliver/<key>`.

**Pure-host run** (both trip2g and fleet on the same machine):

```
--listen        127.0.0.1:9099
--callback-url  http://127.0.0.1:9099
```

DevMode (`DEV=true`) disables trip2g's SSRF guard so deliveries to loopback
addresses are accepted.

**Docker compose setup** (trip2g in a container, fleet on the host):

The app container has `extra_hosts: host.docker.internal:host-gateway`, so it
can reach host processes at `host.docker.internal`. The fleet still listens on
the host loopback; the callback URL must use the Docker bridge name:

```
--listen        127.0.0.1:9099
--callback-url  http://host.docker.internal:9099
```

The `FLEET_CALLBACK_HOST` env var in `e2e/fleet-kanban.spec.js` controls this
choice at test time (default: `127.0.0.1`; set to `host.docker.internal` for
the compose setup).

---

## Minimal launch example

```sh
go run ./cmd/fleet \
  --fleet-id        myfleet \
  --listen          127.0.0.1:9099 \
  --callback-url    http://127.0.0.1:9099 \
  --trip2g-url      http://localhost:8081 \
  --trip2g-admin-personal-token $OWNER_PERSONAL_TOKEN_VALUE \
  --fleet-secret    $(openssl rand -hex 32) \
  --llm-base-url    https://api.openai.com/v1 \
  --llm-api-key     $OPENAI_API_KEY \
  --default-model   gpt-4o-mini \
  --agents-folder   roles/ \
  --offered-tools   search,read_note,patch_note \
  --poll-seconds    10
```

On startup the fleet:
1. Calls trip2g to list notes under `roles/` and parse each as a role note.
2. For each role, upserts a change-webhook (description: `fleet:<id>:<path>#<ver>`)
   with callback URL `<callback-url>/deliver/<hmac-key>`.
3. Begins serving `POST /deliver/<key>` and polls for role changes every
   `--poll-seconds` seconds.

---

## Trigger → delivery → agent loop → write-back

1. A user (or another agent at `depth=0`) edits a note that matches a role's
   `trigger_include` glob.
2. trip2g fires the change-webhook: `POST <callback-url>/deliver/<key>` with
   an HMAC-signed JSON payload containing `changes[]`, `attached_notes[]`,
   `depth`, and a short-lived `api_token`.
3. The fleet verifies the HMAC, renders the role-note body as a Jet template
   against the trigger context (`change_file`, `changed_files`,
   `attached_notes`, `depth`), and calls `agentruntime.Run`.
4. The agent loop calls the LLM, executes tool calls (`search`, `patch_note`,
   etc.) using the scoped `api_token`, and returns a result.
5. The fleet aggregates the result and responds `200 OK` to the webhook call.
   trip2g records the delivery as successful.
6. Because the role sets `max_depth: 1`, any note written by the agent in step 4
   carries `depth=1`. trip2g refuses to re-fire the webhook for depth ≥ max_depth,
   so the loop terminates.

### Template variables per `for_each` mode

The role body is a Jet template rendered against four variables. Which of them
are populated depends on the role's `for_each` frontmatter:

| `for_each` | `change_file` | `changed_files` | `attached_notes` |
|------------|---------------|-----------------|------------------|
| `""` (default — one run for the whole delivery) | `nil` | full list of all changes | full list of all attached notes |
| `changed_files` (one run per change) | the current change | full list (unchanged) | full list |
| `attached_notes` (one run per attached note) | `nil` | full list | one-element list — the current note |

`depth` is always set. In `attached_notes` mode the current note is exposed as
the single element of `attached_notes` (there is no singular note slot), so
iterate it with `{{ range attached_notes }}`.

**Footgun:** a body that references `change_file` (e.g. `{{ change_file.Path }}`)
without `for_each: changed_files` renders against a `nil` `change_file` and
**fails the delivery**. In the default and `attached_notes` modes, walk
`{{ range changed_files }}` instead.

**Prompt injection:** note content is interpolated verbatim into the agent's
prompt, so a note author can attempt prompt injection. This is mitigated — not
eliminated — by the role's read/write scope and tool allowlist: even a hijacked
run can only read/write within the role's declared globs and call its
allowlisted tools.

### Pointing a role at the example board

Copy `docs/demo/fleet/roles/triage.md` and `docs/demo/fleet/boards/sprint.md`
into trip2g (e.g. via `updateNotes`). The triage role's `trigger_include`
targets `boards/sprint.md`. Any update to the board fires the triage agent.

---

## Krisp demo stand (mocks, no LLM, no credentials)

Runs the whole three-step agent pipeline against your `make air` hub with every
external dependency replaced by a mock. Services live in `docker-compose.yaml`
alongside minio, so this is your dev stand, not a separate one.

```
krisp-mock  synthetic meetings (3 of them), stands in for the Krisp API
codellm     runs the ingest role's python; no LLM involved
llm-mock    deterministic chat-completions stub for the two LLM roles
fleet-code  serves `fleet_id: codellm` roles — the cron ingest
fleet-llm   serves `fleet_id: llm` roles — segmentation and wiki extraction
```

One-time setup — both fleets authenticate with the hub's seeded owner token, so
`.env` needs the value `.air.fleet.toml` already uses, and `DEV=true` so the hub
accepts loopback callback URLs:

```sh
echo 'OWNER_PERSONAL_TOKEN_VALUE=t2g_devfleetdevfleetdevfleetdevfleetdevfleetdevfleetdevfleetdevfl' >> .env
# restart `make air` so boot seeds the token row
```

Then:

```sh
make krisp-demo            # build + start the five services
cd docs && node ../obsidian-sync/dist/trip2g-sync.mjs --folder .   # sync the roles into the vault
make krisp-demo-logs       # follow codellm + both fleets
```

The roles live in the vault at `demo/krisp/roles/` (committed under
`docs/demo/krisp/roles/`) and are scoped to `demo/krisp/**`, so the pipeline
never touches the rest of the vault. They are demo copies of the shipped roles in
`docs/fleet/krisp/roles/`, which use top-level paths instead.

What happens next, without you doing anything:

1. Both fleets discover their roles within ~5 s and register webhooks.
2. The cron ingest fires (every minute), pulls 3 synthetic meetings from
   krisp-mock and writes `demo/krisp/transcripts/<id>.md`.
3. Each transcript triggers segmentation → `demo/krisp/segments/<id>.md`.
4. Each segments note triggers wiki extraction → `demo/krisp/wiki/<id>.md`.

Three meetings, so three chains of three deliveries each, spread across two
fleets. Watch them in the admin under **System → Delivery Chains**: each chain
shows its root cron delivery at depth 0 and the two change deliveries hanging
off it, with the delivery that caused each hop. Sync again to pull the generated
notes down into `docs/demo/krisp/` (the output folders are gitignored).

The cron schedule is `* * * * *`, so the ingest re-runs every minute. It writes
the same three transcripts, so nothing new is produced — but the write still
fires the downstream webhooks, so chains keep appearing. Stop the stand with
`make krisp-demo-down` when you have seen enough.

## Demo e2e (standalone)

The fleet end-to-end spec exercises the full loop with a deterministic stub LLM.
It requires the compose test stack (`docker compose -f docker-compose.test.yml up`)
and does NOT run as part of `scripts/test-e2e.sh`.

Run it standalone:

```sh
# Host-only (fleet and app both on the host — use default callback host):
npx playwright test e2e/fleet-kanban.spec.js

# Docker compose (app in container, fleet on host):
FLEET_CALLBACK_HOST=host.docker.internal \
  APP_URL=http://localhost:20081 \
  npx playwright test e2e/fleet-kanban.spec.js
```

The spec seeds `boards/sprint.md` and `roles/triage.md` into trip2g, starts the
stub LLM and `cmd/fleet`, triggers a user edit, and asserts the agent appended
`@triaged` to the doing-card within 30 seconds.
