# Krisp × codellm demo — python skill pulling the latest calls

**What this shows (video-1 flagship):** codellm is "code as a deterministic LLM."
Hand it markdown with a fenced `python` block; it *executes* the block and returns
the writes as OpenAI `tool_calls`. Here the block pulls the **latest calls from the
live Krisp API** and codellm returns them as `write_note` tool_calls + one `finish` —
a real, runnable pipeline, no LLM in the loop.

End-to-end run succeeded: codellm executed the python, which hit the live Krisp API
and pulled the **3 latest real calls**, and codellm returned 3 `write_note` calls
(one note per call) plus a trailing `finish`. Evidence in `transcript.sample.json`
(sanitized; full real transcript stays local — see "Privacy" below).

## Files

| File | What it is |
|---|---|
| `skill.md` | The codellm python skill — a role body (`executor: code`) with one `python` block. Pulls the latest Krisp calls, emits the `{"changes":[...],"answer":"..."}` stdout contract. |
| `run_demo.sh` | Builds + starts the real `internal/codellm` service (standalone), POSTs the OpenAI request carrying `skill.md`, captures request+response (secrets redacted). |
| `standalone/main.go` | Thin driver that runs the **real `internal/codellm` service** in its documented no-auth standalone mode (see "The auth-gate finding"). Does not modify codellm. |
| `transcript.sample.json` | **Committed.** Sanitized transcript: real call bodies + meeting titles elided; shows the wire contract and that it worked. |
| `transcript.json` | **Local only, never committed** (gitignored `_private_drafts`). The full run with real call content. |

Run it:

```bash
bash _private_drafts/krisp_codellm_demo/run_demo.sh    # needs KRISP_TOKEN in the env file below
```

## The real Krisp fetch mechanism

Krisp has a **live private API** (there is no local export — the demo pulls from the
real source). Reverse-engineered from `~/krisp-run/krisp_analyze.py` (itself ported
from `trip2g_agent_queue/krisp/extract_krisp_transcripts.py`):

- Base: `https://api.krisp.ai/v2`, bearer auth via `KRISP_TOKEN`, plus Krisp's web
  headers (`krisp_header_app: web`, `krisp_header_web_project: note`, `Origin: https://app.krisp.ai`).
- `POST /meetings/list` `{sort:"desc", sortKey:"created_at", page, limit, isOwner:true}`
  → latest meetings (metadata: id, name, started_at, duration, speakers).
- `GET /block/{id}/tree` → the transcript tree; walk nodes with `speakerIndex` +
  `speech.text`, sort by `speech.start` → ordered turns.
- "Latest calls" = list sorted desc by `started_at`, drop demos / <60s, take N.

The token lives in `/home/alexes/projects2/trip2g_agent_queue/.env` as `KRISP_TOKEN`
(override with `KRISP_ENV_FILE=`). Verified live: `python3 ~/krisp-run/krisp_analyze.py list`.

## Secrets — how the key reaches the code, and the real path

The productized path (per `docs/dev/codellm_extraction.md`) is a **`requires_secrets`
manifest**: a skill declares `requires_secrets: [KRISP_TOKEN]` (capability, not value),
codellm holds the value in its own secret config and injects **only** the declared keys
into that skill's sandbox. **That manifest is designed, NOT built.**

**Reality of codellm as-merged:** it goes further than "not built" — codellm scrubs the
executed child's env to `PATH` + `FLEET_INPUT` only (`buildChildEnv`,
`internal/agentruntime/coderun.go:388`) and **never sets `EnvPassthrough`**
(`internal/codellm/server.go:171`). codellm's own `Config` has **no secret field** at
all. So even the "minimal: put the key in codellm's env" fallback is **not wired** —
codellm's process env cannot reach the python.

The **only** runtime channel into the executed block is the delivery bag →
`$FLEET_INPUT`. So for this demo the `KRISP_TOKEN` rides the bag: the `run_demo.sh`
request includes a `fleet_input` system message `{"krisp_token":"…","n":3}`, and the
python reads it from `$FLEET_INPUT`. The `codellm_extraction.md` design explicitly
calls "put the secret in the bag" the *wrong* long-term answer (the bag rides the HTTP
request, crossing the boundary we're hardening) — hence this is a **demo-only shortcut**,
and the manifest is the real path. The key is **never** in `skill.md` and **never**
committed; the runner redacts it from the captured transcript.

## Sandbox: network must be on

The python must reach `api.krisp.ai`, but codellm's default `native` sandbox denies
egress (`SandboxPolicy.Network` defaults false). The demo runs `CODELLM_SANDBOX=off`
so the block has network. A production Krisp skill needs the codellm container to allow
egress to Krisp (per-skill network policy), which the current on/native/besteffort knob
does not express at skill granularity.

## The auth-gate finding (why the standalone driver)

The shipped `cmd/codellm` binary hard-wires a **delegated-admin browser gate**: every
`/v1/chat/completions` request's session cookie is forwarded to the trip2g monolith's
`viewer{role}`; admin → serve, else **401**. Its server-to-server **fleet lane**
(`TokenCheck`, the mTLS/shared-token seam) is explicitly **left nil / not built**
(`cmd/codellm/main.go:49`). Confirmed: hitting the `cmd/codellm` binary directly returns
`401 unauthorized` with no monolith session.

Consequence: as-merged, **nothing can call codellm's `/v1` server-to-server** — not this
demo, and not fleet — until either the monolith is up with an admin cookie forwarded, or
the fleet-channel token is built. This demo therefore runs the **identical
`internal/codellm` service package** via `standalone/main.go` in its documented no-auth
mode (`Config.Auth` nil → no-op passthrough; the mode codellm's own tests use). Same
`handleChatCompletions` → `ExecCode` → `buildResponse` path; only the auth wrapper is
skipped. No codellm Go code was modified.

## What is needed for a fully-live, production demo

Everything below the app layer already works live (Krisp API + real calls + codellm
execution + the tool_calls contract). To make it production-live end-to-end:

1. **Fleet-channel auth** on `cmd/codellm` (`TokenCheck`), so a service — fleet, or a
   demo client — can call `/v1` without a browser admin cookie. Not built.
2. **`requires_secrets` manifest + per-skill secret injection** in codellm, so
   `KRISP_TOKEN` comes from codellm's secret store instead of the request bag. Not built.
3. **Per-skill egress policy** so a Krisp skill gets network to `api.krisp.ai` under a
   real sandbox (not the blunt `CODELLM_SANDBOX=off`).
4. Optionally, route this as a real fleet `executor: code` role (the `fleet_graphql.md`
   per-provider-fleet model) instead of a hand-built request.

None of these block the *story*: the demo proves codellm deterministically executes
python that pulls live Krisp data and returns structured writes. The gaps are the
secret-delivery and channel-auth productization, both already designed in
`docs/dev/codellm_extraction.md`.
