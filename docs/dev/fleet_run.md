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
3. A full-admin API key with MCP admin tools enabled (see below).
4. Role notes under the `roles/` folder in trip2g (or whatever
   `--agents-folder` points at).

---

## API key setup

The fleet needs a full-admin API key with the MCP admin-tools lane enabled.
Create one via GraphQL:

```graphql
# Step 1: create the key — note value (plaintext) and apiKey.id.
mutation {
  admin {
    createApiKey(input: { description: "fleet-local" }) {
      ... on CreateApiKeyPayload { value apiKey { id } }
      ... on ErrorPayload { message }
    }
  }
}

# Step 2: enable MCP admin tools on that key.
mutation {
  admin {
    setApiKeyMcpAdminTools(input: { id: "<id-from-step-1>", enabled: true }) {
      ... on SetApiKeyMcpAdminToolsPayload { apiKey { id } }
      ... on ErrorPayload { message }
    }
  }
}
```

The value from step 1 is passed as `--admin-api-key`. The fleet uses this key
for all read/write operations and for reconciling webhooks.

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
| `--admin-api-key` | `FLEET_ADMIN_API_KEY` | — | **Required.** Full-admin API key (X-Api-Key header) |
| `--fleet-secret` | `FLEET_SECRET` | — | **Required.** HMAC seed; fleet derives a per-role secret from it to verify delivery signatures |
| `--llm-base-url` | `FLEET_LLM_BASE_URL` | _(empty)_ | OpenAI-compatible base URL (e.g. `https://api.openai.com/v1`). Empty means the default OpenAI endpoint |
| `--llm-api-key` | `FLEET_LLM_API_KEY` | — | **Required.** LLM provider key |
| `--default-model` | `FLEET_DEFAULT_MODEL` | `gpt-4o-mini` | Model name used for roles without an explicit `model:` frontmatter field |
| `--token-ceiling` | — | `100000` | Hard per-run token cap; overrides any `max_tokens:` in a role note |
| `--step-ceiling` | — | `25` | Hard per-run step cap; overrides any `max_steps:` in a role note |
| `--agents-folder` | `FLEET_AGENTS_FOLDER` | `roles/` | Note-path LIKE prefix used when discovering role notes in trip2g |
| `--offered-tools` | — | `search,read_note,patch_note,write_note` | Comma-separated list of tools the fleet exposes to agents; roles may only use a subset |
| `--poll-seconds` | — | `30` | Discovery + reconcile interval in seconds |

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
  --admin-api-key   <key-from-createApiKey> \
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

### Pointing a role at the example board

Copy `docs/demo/fleet/roles/triage.md` and `docs/demo/fleet/boards/sprint.md`
into trip2g (e.g. via `updateNotes`). The triage role's `trigger_include`
targets `boards/sprint.md`. Any update to the board fires the triage agent.

---

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
