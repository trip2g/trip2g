#!/bin/bash
# Native-sandbox smoke test for the codellm runtime image (Dockerfile.codellm).
#
# The landlock sandbox needs CLONE_NEWPID/CLONE_NEWNS, which Docker's default
# caps and seccomp profile deny — that is why docker-compose.test.yml runs
# codellm with CODELLM_SANDBOX=off. This script starts the image with the
# privileges the enforcing posture actually needs and checks two things:
#
#   1. the toolbox is reachable from a block under CODELLM_SANDBOX=native
#      (python/node/bash packages, CA bundle, NODE_PATH, CLI tools)
#   2. the sandbox still denies what it is supposed to deny
#      (/etc, /usr/local, parent env secrets)
#
# Usage: ./scripts/test-codellm-sandbox.sh          # builds, then tests
#        SKIP_BUILD=1 ./scripts/test-codellm-sandbox.sh
#        CAPS=limited ./scripts/test-codellm-sandbox.sh   # minimal grant, not --privileged
#
# Run under sudo if your user is not in the docker group.
# Exits non-zero on the first failed check.
#
# CAPS=limited is the minimum an orchestrator has to grant, established by
# bisection: CAP_SYS_ADMIN for the PID/mount namespaces, and an unconfined
# AppArmor profile because the stock docker-default profile denies the private
# remount of /. Docker's default seccomp profile needs no change. With neither
# granted the run fails closed — coderun refuses to execute unsandboxed.

set -e

IMAGE=${IMAGE:-codellm-sandbox-test}
NAME=${NAME:-codellm-sandbox-test}
PORT=${PORT:-28099}
KEY=${KEY:-sandbox-smoke-key-0123456789abcdef}
SENTINEL_VALUE=PARENT_SECRET_MUST_NOT_LEAK
ROOT=$(cd "$(dirname "$0")/.." && pwd)

if [ "${CAPS:-full}" = "limited" ]; then
  PRIV=(--cap-add SYS_ADMIN --security-opt apparmor=unconfined)
else
  PRIV=(--privileged)
fi

cleanup() { docker rm -f "$NAME" >/dev/null 2>&1 || true; }
trap cleanup EXIT

if [ -z "$SKIP_BUILD" ]; then
  echo "==> building $IMAGE"
  docker build -f "$ROOT/Dockerfile.codellm" -t "$IMAGE" "$ROOT"
fi

cleanup
echo "==> starting codellm (CODELLM_SANDBOX=native, ${PRIV[*]})"
docker run -d --name "$NAME" -p "$PORT:8082" "${PRIV[@]}" \
  -e CODELLM_ADDR=0.0.0.0:8082 \
  -e CODELLM_TRIP2G_URL=http://example.invalid \
  -e CODELLM_API_KEY="$KEY" \
  -e CODELLM_ALLOWED_PROGRAMS=python,bash,node \
  -e CODELLM_SANDBOX=native \
  -e CODELLM_SANDBOX_NETWORK=true \
  -e FLEET_SANDBOX_SENTINEL="$SENTINEL_VALUE" \
  "$IMAGE" >/dev/null

for _ in $(seq 1 40); do
  curl -sf "http://localhost:$PORT/healthz" >/dev/null 2>&1 && break
  sleep 0.25
done
if ! curl -sf "http://localhost:$PORT/healthz" >/dev/null 2>&1; then
  echo "FAIL: server did not come up"
  docker logs "$NAME" 2>&1 | tail -30
  exit 1
fi

PORT="$PORT" KEY="$KEY" SENTINEL_VALUE="$SENTINEL_VALUE" python3 - <<'PYEOF'
import json, os, sys, urllib.error, urllib.request

URL = "http://localhost:%s/v1/chat/completions" % os.environ["PORT"]
HDR = {"Content-Type": "application/json",
       "Authorization": "Bearer " + os.environ["KEY"]}

def run(lang, code):
    """Execute one block; return (answer, error) — exactly one is non-empty."""
    body = json.dumps({"model": "codellm",
                       "messages": [{"role": "user",
                                     "content": "```%s\n%s\n```" % (lang, code)}]}).encode()
    req = urllib.request.Request(URL, data=body, headers=HDR)
    try:
        resp = json.load(urllib.request.urlopen(req, timeout=120))
    except urllib.error.HTTPError as e:
        return "", e.read().decode("utf-8", "replace")[:300]
    except Exception as e:
        return "", str(e)[:300]
    msg = resp["choices"][0]["message"]
    for call in msg.get("tool_calls") or []:
        if call["function"]["name"] == "finish":
            return json.loads(call["function"]["arguments"]).get("answer", ""), ""
    return msg.get("content") or "", ""

EMIT_PY = 'import json; print(json.dumps({"changes": [], "answer": ANSWER}))'

# (name, lang, code, predicate over the answer string)
CHECKS = [
    ("python packages import", "python", '''
import requests, httpx, bs4, lxml, dateutil, jwt, cryptography, tenacity, pydantic, yaml
from zoneinfo import ZoneInfo
ANSWER = "ok tz=%s pydantic=%s" % (ZoneInfo("Europe/Moscow").key, pydantic.VERSION)
''' + EMIT_PY, lambda a: a.startswith("ok ")),

    ("python https via certifi", "python", '''
import requests
ANSWER = "status=%d" % requests.get("https://api.github.com/zen", timeout=30).status_code
''' + EMIT_PY, lambda a: a == "status=200"),

    ("node packages require via NODE_PATH", "node", '''
const names = ["axios","cheerio","zod","dayjs","lodash","jsonwebtoken","csv-parse","axios-retry","pg","qs"];
const bad = names.filter(n => { try { require(n); return false } catch (e) { return true } });
console.log(JSON.stringify({changes: [], answer: bad.length ? "missing=" + bad : "ok path=" + process.env.NODE_PATH}));
''', lambda a: a == "ok path=/usr/lib/node_modules"),

    ("node https", "node", '''
require("axios").get("https://api.github.com/zen", {timeout: 30000})
  .then(r => console.log(JSON.stringify({changes: [], answer: "status=" + r.status})))
  .catch(e => console.log(JSON.stringify({changes: [], answer: "error=" + e.message})));
''', lambda a: a == "status=200"),

    # ESM ignores NODE_PATH; this resolves only via the /tmp/node_modules symlink.
    ("node esm import resolves", "node", '''
import _ from "lodash";
import { z } from "zod";
console.log(JSON.stringify({changes: [], answer: "esm lodash=" + _.VERSION + " zod=" + (typeof z.string === "function")}));
''', lambda a: a.startswith("esm lodash=") and a.endswith("zod=true")),

    ("cli tools present", "bash", '''
missing=""
for t in jq curl mlr sqlite3 openssl rg git gh unzip xz dig nc yq file gawk wget; do
  command -v "$t" >/dev/null || missing="$missing $t"
done
jq -nc --arg m "$missing" '{changes: [], answer: (if $m == "" then "ok" else "missing:\\($m)" end)}'
''', lambda a: a == "ok"),

    ("curl https uses the copied CA bundle", "bash", '''
code=$(curl -s -o /dev/null -w '%{http_code}' https://api.github.com/zen)
jq -nc --arg c "$code" --arg ca "$SSL_CERT_FILE" '{changes: [], answer: "code=\\($c) ca=\\($ca)"}'
''', lambda a: a == "code=200 ca=/usr/lib/ssl-certs/ca-bundle.crt"),

    # --- enforcement: these must be DENIED ---
    ("sandbox denies /etc", "bash", '''
if cat /etc/passwd >/dev/null 2>&1; then r=readable; else r=denied; fi
jq -nc --arg r "$r" '{changes: [], answer: $r}'
''', lambda a: a == "denied"),

    ("sandbox denies /usr/local", "bash", '''
if ls /usr/local >/dev/null 2>&1; then r=readable; else r=denied; fi
jq -nc --arg r "$r" '{changes: [], answer: $r}'
''', lambda a: a == "denied"),

    ("parent env secret is scrubbed", "bash", '''
if [ -z "$FLEET_SANDBOX_SENTINEL" ]; then r=absent; else r=LEAKED; fi
jq -nc --arg r "$r" '{changes: [], answer: $r}'
''', lambda a: a == "absent"),

    # fleetkit is shipped by the same Dockerfile and must be importable under the
    # enforcing sandbox, not just with CODELLM_SANDBOX=off.
    ("fleetkit imports and renders", "python", """
import fleetkit
n = fleetkit.note("a.md", {"title": "T: q"}, "# T")
lines = n["content"].split(chr(10))
q = chr(39)
ok = lines[0] == "---" and lines[1] == "title: " + q + "T: q" + q and lines[2] == "---"
ANSWER = "ok" if ok else "bad=" + repr(lines[:3])
""" + EMIT_PY, lambda a: a == "ok"),

    ("fleetkit bag is empty outside a delivery", "python", '''
import fleetkit
ANSWER = "empty" if fleetkit.bag() == {} else "unexpected"
''' + EMIT_PY, lambda a: a == "empty"),

    # The node twin ships in the same image and must resolve the same way the
    # other global packages do.
    ("fleetkit node twin renders identically", "node", """
const fleetkit = require('fleetkit');
const c = fleetkit.note('a.md', {title: 'T: q', when: '2026-01-15T10:00:00+00:00'}, '# T').content;
const want = ['---', 'title: "T: q"', 'when: "2026-01-15T10:00:00+00:00"', '---', '# T'].join(String.fromCharCode(10));
console.log(JSON.stringify({changes: [], answer: c === want ? 'ok' : 'bad=' + JSON.stringify(c)}));
""", lambda a: a == "ok"),

    ("python jsonschema and node ajv import", "python", '''
import jsonschema
jsonschema.validate({"a": 1}, {"type": "object", "required": ["a"]})
try:
    jsonschema.validate({}, {"type": "object", "required": ["a"]})
    ANSWER = "missing-required-not-caught"
except jsonschema.ValidationError:
    ANSWER = "ok"
''' + EMIT_PY, lambda a: a == "ok"),

    ("node ajv validates", "node", """
const Ajv = require('ajv');
const ok = new Ajv().compile({type: 'object', required: ['a']})({a: 1});
console.log(JSON.stringify({changes: [], answer: ok ? 'ok' : 'bad'}));
""", lambda a: a == "ok"),
]

failed = 0
for name, lang, code, ok in CHECKS:
    answer, err = run(lang, code.strip())
    if err:
        print("FAIL  %-38s error: %s" % (name, err.replace("\n", " ")))
        failed += 1
    elif ok(answer):
        print("ok    %-38s %s" % (name, answer))
    else:
        print("FAIL  %-38s got: %s" % (name, answer))
        failed += 1

print()
if failed:
    print("%d/%d checks FAILED" % (failed, len(CHECKS)))
    sys.exit(1)
print("all %d checks passed" % len(CHECKS))
PYEOF

status=$?
if [ $status -ne 0 ]; then
  echo "==> server logs"
  docker logs "$NAME" 2>&1 | tail -30
fi
exit $status
