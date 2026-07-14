#!/usr/bin/env bash
# End-to-end demo: run the Krisp python skill THROUGH the real codellm service.
#
# It builds+starts the codellm binary on loopback, POSTs an OpenAI
# /v1/chat/completions request whose message carries the skill markdown and whose
# fleet_input message carries the delivery bag (krisp_token + n), then captures
# the request+response with secrets redacted.
#
# codellm as-built scrubs the child env to PATH+FLEET_INPUT only (no secret
# passthrough is wired), so the token reaches the python via the $FLEET_INPUT bag.
# The native sandbox denies network egress, so we run CODELLM_SANDBOX=off for this
# demo (the python must reach api.krisp.ai). See README.md for the productized path.
#
# It runs the REAL internal/codellm service via the standalone driver in
# ./standalone (no-auth mode the package documents). The cmd/codellm binary
# hard-wires a delegated-admin browser gate whose fleet-lane token is not built
# yet, so it 401s any server-to-server call without a monolith cookie. Same
# handler/execution path either way. See README.md.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$HERE/../.." && pwd)"
ENV_FILE="${KRISP_ENV_FILE:-/home/alexes/projects2/trip2g_agent_queue/.env}"
N="${N:-3}"
ADDR="127.0.0.1:8092"
OUT="$HERE/transcript.json"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"; [ -n "${SRV_PID:-}" ] && kill "$SRV_PID" 2>/dev/null || true' EXIT

# --- read the secret (never printed, never committed) ---
KRISP_TOKEN="$(grep -E '^KRISP_TOKEN=' "$ENV_FILE" | tail -1 | cut -d= -f2- | tr -d '"'"'"'')"
[ -n "$KRISP_TOKEN" ] || { echo "no KRISP_TOKEN in $ENV_FILE" >&2; exit 1; }

# --- build + start codellm (real internal/codellm service, standalone no-auth) ---
echo "building codellm-standalone..." >&2
( cd "$REPO" && go build -o "$WORK/codellm" ./_private_drafts/krisp_codellm_demo/standalone )
CODELLM_ADDR="$ADDR" CODELLM_SANDBOX=off CODELLM_ALLOWED_PROGRAMS=python \
  "$WORK/codellm" >"$WORK/codellm.log" 2>&1 &
SRV_PID=$!

# wait for /healthz
for _ in $(seq 1 50); do
  curl -sf "http://$ADDR/healthz" >/dev/null 2>&1 && break
  sleep 0.2
done
curl -sf "http://$ADDR/healthz" >/dev/null || { echo "codellm did not start"; cat "$WORK/codellm.log"; exit 1; }

# --- build the OpenAI request: skill body + fleet_input bag + Begin ---
SKILL="$(cat "$HERE/skill.md")"
BAG="$(jq -nc --arg t "$KRISP_TOKEN" --argjson n "$N" '{krisp_token:$t, n:$n}')"
REQ="$(jq -nc --arg body "$SKILL" --arg bag "$BAG" '{
  model: "codellm",
  messages: [
    {role:"system", content:("You are a scoped trip2g micro-agent.\n" + $body)},
    {role:"system", name:"fleet_input", content:$bag},
    {role:"user", content:"Begin."}
  ],
  tools: [{type:"function", function:{name:"write_note"}}]
}')"

echo "POST http://$ADDR/v1/chat/completions ..." >&2
RESP="$(curl -sS -X POST "http://$ADDR/v1/chat/completions" \
  -H 'Content-Type: application/json' --data "$REQ")"

if ! printf '%s' "$RESP" | jq -e . >/dev/null 2>&1; then
  echo "non-JSON response from codellm:" >&2
  printf '%s\n' "$RESP" >&2
  RESP="$(jq -nc --arg raw "$RESP" '{error:{message:$raw, type:"non_json_response"}}')"
fi

# --- redact secret from the captured request, save transcript (via files: the
#     full transcripts blow past ARG_MAX if passed on the command line) ---
printf '%s' "$REQ" | jq -c --arg t "$KRISP_TOKEN" \
  '(.messages[] | select(.name=="fleet_input") | .content) |= (fromjson | .krisp_token = "***REDACTED***" | tojson)' \
  >"$WORK/req_redacted.json"
printf '%s' "$RESP" >"$WORK/resp.json"

jq -n --slurpfile request "$WORK/req_redacted.json" --slurpfile response "$WORK/resp.json" \
  --arg addr "$ADDR" --arg when "$(date -u +%FT%TZ)" '{
    note: "codellm Krisp python skill — end-to-end transcript (secrets redacted)",
    captured_at: $when, endpoint: ("http://" + $addr + "/v1/chat/completions"),
    sandbox: "off (network needed for api.krisp.ai)",
    request: $request[0], response: $response[0]
  }' >"$OUT"

echo "transcript written to $OUT" >&2

# --- public-safe sample: real Krisp note bodies + meeting titles elided, so the
#     contract/flow is reviewable in the (PUBLIC) repo without leaking private
#     call content. transcript.json (full, real) stays local under gitignored
#     _private_drafts and is NEVER committed. ---
SAMPLE="$HERE/transcript.sample.json"
jq '
  .note = "codellm Krisp python skill — SANITIZED sample (real call content elided; full transcript stays local)"
  | .request.messages = [ .request.messages[]
      | if .name == "fleet_input" then .
        elif .role == "system" then {role, content: ("<system message: skill.md body, \(.content|length) chars — see skill.md>")}
        else . end ]
  | .response.choices[0].message.tool_calls = [ .response.choices[0].message.tool_calls[]
      | if .function.name == "write_note" then
          .function.arguments = ((.function.arguments|fromjson)
            | {path: "calls/YYYY-MM-DD-xxxxxxxx.md",
               content: "<elided: \(.content|length) chars of a real Krisp note (frontmatter + transcript)>"} | tojson)
        elif .function.name == "finish" then
          .function.arguments = ((.function.arguments|fromjson)
            | {answer: "Pulled 3 latest Krisp call(s) [titles/details elided for public repo]"} | tojson)
        else . end ]
' "$OUT" >"$SAMPLE"
echo "sanitized sample written to $SAMPLE" >&2
# Print a short human summary of what codellm returned.
printf '%s' "$RESP" | jq -r '
  .choices[0].message.tool_calls as $tc |
  if $tc then
    ($tc | map(select(.function.name=="write_note") | (.function.arguments|fromjson.path)) ) as $paths |
    ($tc | map(select(.function.name=="finish") | (.function.arguments|fromjson.answer)) | .[0]) as $ans |
    "wrote \($paths|length) note(s): \($paths|join(", "))\nanswer: \($ans)"
  else "ERROR: " + (.error.message // "no tool_calls") end'
