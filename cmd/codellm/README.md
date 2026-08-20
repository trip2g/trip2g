Check krisp blocks:

```bash
KRISP_BASE_URL=http://localhost:19092 \
KRISP_TOKEN=any \
CODELLM_ALLOWED_PROGRAMS=python,bash \
CODELLM_SANDBOX_NETWORK=true \
CODELLM_EXPOSE_ENV_PREFIX=KRISP_ \
make aircodellm
```

## Tools

What a block can use inside the runtime image (`Dockerfile.codellm`). The list is
deliberately short: it is meant to be pasted into the system prompt so the model
writes against what is actually installed instead of guessing.

Interpreters are `python`, `node` and `bash` (`internal/coderun/interpreters.json`),
gated at runtime by `CODELLM_ALLOWED_PROGRAMS`.

That file also declares the env each interpreter needs to find its packages, so
the paths baked into the image live next to the interpreter they belong to
rather than in Go. An entry is applied only when its `if_exists` path is
present, which is what keeps a dev run outside the image (`make aircodellm`)
from being pointed at a bundle that is not there:

```json
{"name": "node", "cmd": ["node"], "ext": ".js",
 "env": [{"name": "NODE_PATH", "value": "/usr/lib/node_modules",
          "if_exists": "/usr/lib/node_modules"}]}
```

The top-level `env` array applies to every interpreter — the CA variables live
there, since `curl`, `git`, node and python all need them. Change a path here
and you must change `Dockerfile.codellm` to match, or the guard silently drops
the variable; `TestInterpretersJSON_ShippedEnv` pins the pair together.

### fleetkit

A block's contract with the fleet is one line of JSON on stdout —
`{"changes": [...], "answer": "..."}`. `fleetkit` builds it, and the notes that
go into it, so a role does not re-derive the shape or glue YAML by hand:

```python
import fleetkit

meetings = ...
changes = [
    fleetkit.note('transcripts/%s.md' % m['id'], {'title': m['name']}, m['text'])
    for m in meetings
]
fleetkit.emit(changes, 'ingested %d' % len(changes))
```

| Function | What it returns |
|----------|-----------------|
| `render(meta, body='')` | markdown: YAML frontmatter + body, quoting decided by `yaml.safe_dump` |
| `note(path, meta, body='')` | a write change carrying a rendered note |
| `write(path, content)` | a change that replaces the whole note |
| `patch(path, find, replace)` | a change that swaps the first occurrence of `find` |
| `emit(changes, answer)` | prints the stdout contract — call once, and last |
| `bag()` | the delivery bag from `FLEET_INPUT`; `{}` outside a delivery |
| `note_frontmatter(path)` | the frontmatter of a note in the bag; raises if the bag has no such path |
| `parse_frontmatter(content)` | the frontmatter block of a markdown string; empty when there is none |

`note_frontmatter(path)` reads a note the delivery bag carries, so a lint role
is plain code with no schema DSL to learn:

```python
data = fleetkit.note_frontmatter('calls/a.md')
if not data.title:
    issues.append('calls/a.md: no title')
```

A missing key reads as `None` (python) / `undefined` (node) rather than
raising, so the `if` says what it looks like. A path the bag does not carry
raises instead — that is the role's `attach_notes` being too narrow, not a
finding about the note. Bring `jsonschema` or `zod` if you want a schema; the
helper deliberately has no opinion.

**Both languages, same names.** The twins are function-for-function identical
and the node one keeps snake_case for that reason: a role moved between them
should not have to relearn the API. `node` renders with YAML **version 1.1** to
match PyYAML — not cosmetic, since under the 1.2 default a timestamp-shaped
string is emitted bare and reads back as a timestamp, and `yes` reads back as
`true`. `TestFleetkitRuns_EmitsTheContract` runs both and compares the parsed
result.

Sources are `cmd/codellm/fleetkit/fleetkit.py` and `cmd/codellm/fleetkit/node/`,
installed by `Dockerfile.codellm` to `/usr/lib/python3/dist-packages/fleetkit.py`
and `/usr/lib/node_modules/fleetkit` — on the default `sys.path` and where both
`require` and `import` resolve, inside the subtree the sandbox grants.

**This is an API that role notes in user vaults import.** Those notes are not in
this repo, have no build step and are not rolled back with the image, so a
rename or a signature change breaks them at runtime with nothing to catch it
first. `TestFleetkit_SourceExports` pins the exported names for that reason;
add functions freely, change existing ones only deliberately.


### Python

Installed into `/usr/lib/python3/dist-packages`.

| Package | What it is for |
|---------|----------------|
| `requests` | HTTP client for REST calls: sessions, headers, timeouts, uploads |
| `httpx` | HTTP client with async and HTTP/2 when `requests` is not enough |
| `beautifulsoup4` | Forgiving HTML parsing when an API is only a web page |
| `lxml` | Fast HTML/XML backend for bs4, plus XPath for XML APIs |
| `python-dateutil` | Parses the arbitrary date formats third-party APIs return |
| `pyjwt` | Encode, decode and verify JWTs (`import jwt`) |
| `cryptography` | HMAC/RSA/ECDSA request signing; backs RS256/ES256 JWTs |
| `tenacity` | Retry with exponential backoff against flaky APIs |
| `pydantic` | Validate and reshape response JSON into typed models |
| `pyyaml` | YAML configs, OpenAPI specs |
| `jsonschema` | Validate frontmatter or an API response against a JSON Schema |
| `tzdata` | Timezone database for `zoneinfo` — `/usr/share/zoneinfo` is denied by the sandbox |

Not installed on purpose: `pandas`/`numpy` (~120 MB for work the `csv` module and
`mlr` already cover), `orjson` (stdlib `json` is enough at this size).

### Node.js

Installed globally into `/usr/lib/node_modules`. **Both `require` and `import`
work** — write whichever.

The details behind that, since they are easy to break:

- A block is written to `run.js` in a temp dir and executed as `node run.js`;
  the fence tag picks the interpreter, the `ext` field names the file. For node
  the extension is load-bearing, because it selects the module system: `.mjs` is
  always ESM, `.cjs` always CommonJS, `.js` depends on the nearest
  `package.json` — and there is none in a temp dir.
- That would make `.js` CommonJS, except Node 20.19+ **detects ESM syntax** in
  `.js` and switches by itself. So a block using `import` or top-level `await`
  runs as ESM with no flag and no separate fence tag. No node flag can do this
  for a file: `--input-type=module` applies only to `-e` and stdin.
- Resolution is the part that does not follow automatically. `NODE_PATH`
  (set for the node interpreter in `interpreters.json`) is consulted **only** by
  `require`; ESM ignores it entirely and instead walks up from the script
  looking for `node_modules`. `Dockerfile.codellm` symlinks
  `/tmp/node_modules` → `/usr/lib/node_modules` so that walk succeeds, since a
  block always runs from a temp dir under `/tmp`. Landlock permits it because
  the resolved path is in the read-only allowlist.
- Consequence when changing any of this: dropping the symlink silently breaks
  every `import` with `ERR_MODULE_NOT_FOUND` while `require` keeps working, and
  moving `TMPDIR` off `/tmp` does the same. `scripts/test-codellm-sandbox.sh`
  covers both syntaxes so the break surfaces.

ESM-only packages are still avoided in the list below — not for resolution
reasons any more, but because a CommonJS-capable package works under both
syntaxes and an ESM-only one does not.

| Package | What it is for |
|---------|----------------|
| `axios` | HTTP client: JSON, headers, timeouts, interceptors |
| `cheerio` | jQuery-style HTML parsing |
| `zod` | Validate and reshape response payloads |
| `dayjs` | Date parsing, formatting and arithmetic |
| `lodash` | Reshaping data: `groupBy`, `chunk`, `get` |
| `jsonwebtoken` | Sign and verify JWTs |
| `csv-parse` | CSV parsing, streaming or synchronous |
| `axios-retry` | Retry policies layered onto axios |
| `pg` | PostgreSQL client |
| `yaml` | YAML parse/stringify; what `fleetkit` renders frontmatter with |
| `ajv` / `ajv-formats` | JSON Schema validation, the node counterpart to `jsonschema` |
| `qs` | Nested query-string parsing and serialization |

Node 20 already provides `fetch`, `crypto.randomUUID()` and full ICU, so
`node-fetch`, `uuid` and `crypto-js` are not installed.

### CLI

On `PATH` in `/usr/bin`, for `bash` blocks and for shelling out.

| Tool | What it is for |
|------|----------------|
| `jq` | Query and reshape JSON |
| `curl`, `wget` | HTTP from a shell block |
| `mlr` (miller) | Convert and reshape CSV/TSV/JSON — the CLI answer to "no pandas" |
| `sqlite3` | Scratch database; joins and aggregates over CSV without writing code |
| `openssl` | HMAC signing, base64, certificate inspection |
| `rg` (ripgrep) | Search dumps and payloads |
| `git` | Fetch a repo or config the script needs |
| `gh` | GitHub API and repos: issues, PRs, releases, `gh api` for raw calls |
| `unzip`, `xz`, `gzip`, `tar` | Unpack API exports |
| `dig`, `nc` | Diagnose why an endpoint is unreachable |
| `yq` | YAML/XML into a jq pipeline |
| `file`, `gawk`, `sed`, `grep` | Ordinary shell plumbing |

### Sandbox constraints

Under the default `CODELLM_SANDBOX=native`, blocks run under landlock with
read-only access to `/bin`, `/sbin`, `/usr/bin`, `/usr/sbin` and `/usr/lib*`
only. Consequences worth knowing when adding tools:

- Anything installed to `/usr/local` or `/opt` is **unreachable**. A downloaded
  binary has to go to `/usr/bin`; Python packages need
  `pip --target=/usr/lib/python3/dist-packages`, because Debian's pip otherwise
  defaults to `/usr/local/lib/python3.11/dist-packages` and ignores `--prefix`.
- `/etc` is denied apart from a named handful of files, so the system CA store
  is invisible. The image copies the bundle to `/usr/lib/ssl-certs/ca-bundle.crt`
  and the global `env` in `interpreters.json` exports `SSL_CERT_FILE`,
  `CURL_CA_BUNDLE` and `GIT_SSL_CAINFO` when that file exists. Python (certifi) and Node (CAs compiled in) do not
  depend on it.
- `/usr/share/zoneinfo` is denied, so the python interpreter declares an empty
  `PYTHONTZPATH` to send `zoneinfo` to the `tzdata` package instead. Bash
  `TZ=...` does not work.
- The network is in a private namespace unless `CODELLM_SANDBOX_NETWORK=true`.
  That flag also adds `/etc/resolv.conf`, `/etc/hosts`, `/etc/nsswitch.conf`,
  `/etc/services` and `/etc/gai.conf` to the allowlist — without them name
  resolution fails and only raw IPs are reachable. `/etc/ssl/openssl.cnf` is
  always granted: node aborts at startup without it, even for an offline block.
- The child environment is scrubbed to the declared env plus `PATH` and
  `FLEET_INPUT`; secrets reach a block only through `CODELLM_EXPOSE_ENV` /
  `CODELLM_EXPOSE_ENV_PREFIX`. Tools that authenticate from the environment need
  their var named there — `gh`, for one, is unauthenticated unless codellm runs
  with `CODELLM_EXPOSE_ENV=GH_TOKEN` and holds the value.

### Running the enforcing sandbox under an orchestrator

Docker's defaults do **not** allow `CODELLM_SANDBOX=native` — the run fails
closed ("refusing to run unsandboxed") rather than degrading silently. The
minimum grant, established by bisection:

```hcl
# Nomad job — docker driver
config {
  cap_add      = ["sys_admin"]              # PID + mount namespaces
  security_opt = ["apparmor=unconfined"]    # docker-default denies remounting / private
}
```

`privileged = true` also works but grants far more than needed. Docker's default
seccomp profile needs no change. The Nomad **client** must permit the capability
as well — `sys_admin` is not in the docker driver's default `allow_caps`:

```hcl
plugin "docker" {
  config {
    allow_caps = ["chown", "dac_override", "fowner", "fsetid", "kill", "mknod",
                  "net_bind_service", "setfcap", "setgid", "setpcap", "setuid",
                  "sys_chroot", "audit_write", "sys_admin"]
  }
}
```

Landlock is applied best-effort: on a kernel without it the path restrictions
are silently weaker while namespaces still apply. Verify the host supports it
(`grep landlock /sys/kernel/security/lsm`) rather than assuming.

Verify a deployment with `scripts/test-codellm-sandbox.sh`, which checks both
that the toolbox works and that the denials still hold.
