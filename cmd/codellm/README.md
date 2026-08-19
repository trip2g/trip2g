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
| `tzdata` | Timezone database for `zoneinfo` — `/usr/share/zoneinfo` is denied by the sandbox |

Not installed on purpose: `pandas`/`numpy` (~120 MB for work the `csv` module and
`mlr` already cover), `orjson` (stdlib `json` is enough at this size).

### Node.js

Installed globally into `/usr/lib/node_modules`, reachable via `NODE_PATH`.

**Write CommonJS (`require`).** `NODE_PATH` is not consulted for ESM `import`, so
`import` of a global package fails with `ERR_MODULE_NOT_FOUND`. This is also why
ESM-only packages (`p-retry` >= 5, `got`, `node-fetch` v3) are absent.

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
  and `coderun` exports `SSL_CERT_FILE`, `CURL_CA_BUNDLE` and `GIT_SSL_CAINFO`
  when that file exists. Python (certifi) and Node (CAs compiled in) do not
  depend on it.
- `/usr/share/zoneinfo` is denied, so `coderun` sets an empty `PYTHONTZPATH` to
  send `zoneinfo` to the `tzdata` package instead. Bash `TZ=...` does not work.
- The network is in a private namespace unless `CODELLM_SANDBOX_NETWORK=true`.
  That flag also adds `/etc/resolv.conf`, `/etc/hosts`, `/etc/nsswitch.conf`,
  `/etc/services` and `/etc/gai.conf` to the allowlist — without them name
  resolution fails and only raw IPs are reachable. `/etc/ssl/openssl.cnf` is
  always granted: node aborts at startup without it, even for an offline block.
- The child environment is scrubbed to the toolbox vars plus `PATH` and
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
