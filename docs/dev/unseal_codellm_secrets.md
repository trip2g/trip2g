# Sealed secrets for codellm role notes

**Status:** Built, except the seal HTTP endpoint (2026-08-21). The `seal` CLI, the unseal step and the bag contract are in; `/_system/codellm/seal` is the remaining piece.
**Builds on:** `codellm_extraction.md` (env passthrough, the `requires_secrets` hardening it sketches at the end of "The wire protocol"), `fleet_run.md`, `internal/dataencryption`.

## TL;DR

A code role that needs a secret (the live case: `KRISP_TOKEN` in
`docs/fleet/krisp/roles/transcript-ingest.md`) can only get it today by the operator putting
the value in codellm's own environment and naming it in `CODELLM_EXPOSE_ENV`. Adding a secret
means editing docker-compose and redeploying, and the allowlist is flat — every code role on
that codellm sees every exposed value.

The chosen design puts the **ciphertext in the role note's frontmatter** and keeps the master
key in codellm alone. The note declares which frontmatter fields are sealed; codellm decrypts
exactly those, substitutes the plaintext into the delivery bag for that one run, and the role's
code reads them like any other frontmatter field. fleet never holds the key and never sees
plaintext; trip2g stores the ciphertext as ordinary note content and knows nothing about it.

```yaml
---
description: Krisp meetings → transcript notes (cron ingest, deterministic)
fleet_id: codellm
mode: cron
cron_schedule: "* * * * *"
write_patterns:
  - transcripts/**
unseal: [krisp_token]
unseal_env_key: SEAL_KEY          # optional; SEAL_KEY is the default
krisp_token: sealed:v1:9nR2v0QkX8tWc1pE...
krisp_base_url: https://api.krisp.ai
---
```

```python
cfg = fleetkit.frontmatter()      # the role's own frontmatter, ciphertext included
sec = fleetkit.secrets()          # what codellm unsealed for this run

base_url = cfg.krisp_base_url.rstrip('/')
token = sec.krisp_token
```

There is no `unseal()` call in the role, no env var mapping, and no master key inside the
sandbox.

**Decided: unsealed values go in their own `secrets` section, not inline in the frontmatter.**
Inline would read more uniformly — `cfg.krisp_token` beside `cfg.krisp_base_url` — and that
uniformity is why this design beat the alternatives in the first place. It loses to one thing:
the bag is what a role dumps while debugging, and `print(cfg)` or `emit(answer=json.dumps(bag))`
would publish the token into vault content, which is the store this whole feature exists to keep
clean. A separate section costs one asymmetry in the role's source and takes the secret out of
the object most likely to be dumped whole. The asymmetry is also honest: those two values are not
the same kind of thing.

Note what it does not buy: dumping the *whole bag* still carries the secrets. This narrows the
accident, it does not remove it.

## The pivotal fact

**The instance operator is also the owner of the secrets.** There is no multi-tenant hosted
fleet where several vault owners each hold their own credentials.

This single fact decides the design, so it is recorded here rather than left implicit:

- It makes a **symmetric** master key coherent. The standard objection to a Rails-style
  `credentials.yml.enc` in a product is that the party doing the encrypting is a user who must
  not hold a key that decrypts everyone's secrets. With operator == secret owner that objection
  is void — this is exactly Rails' trust structure, where the encrypters are the team who
  legitimately hold `master.key`.
- It removes the need for a per-vault-owner scoping mechanism, which is the one thing an
  embedded ciphertext gives for free and a shared `secrets` table does not.
- It makes "the operator can read the secret at run time" a tautology rather than a leak.

If that fact ever changes, most of the reasoning below has to be redone.

State it precisely, because the useful form is narrower than "single operator": what the design
relies on is **single role authorship** — that whoever can create a note under the watched
`--agents-folder` glob is trusted with the secrets sealed anywhere in the vault. Operator and vault
writer are not automatically the same set; the system already has delegated-admin sessions and
note-writing agents such as the telegram inbox. Everything below scales with who can author a role
note, not with who owns the deployment.

## What this does and does not protect

codellm hands the plaintext to the code it executes, so whoever operates the codellm host can
always read it — `/proc/<pid>/environ`, the bag file, a patched binary, or simply a `print()`
added to the role. No storage scheme changes that; it is a deploy-time property.

What sealing buys is that the secret is **not in the vault in cleartext** while the vault travels
through Obsidian sync, git, backups, and note sharing, and is not in cleartext in trip2g's DB or
its dumps. Note that today's env approach has the same property by a different route (the secret
is in no vault at all), so sealing is not competing against a plaintext baseline. What it
actually adds over today is:

- **Provisioning without redeploy.** A secret is added by editing a note, not by editing
  docker-compose and restarting codellm.
- **Scoping — of accidents, not of access.** `CODELLM_EXPOSE_ENV` is flat: every code role on that
  codellm gets every exposed value. A sealed blob is decrypted only for the role that carries it,
  which contains the blast radius of a careless role. It is not access control: the ciphertext is
  ordinary vault content, so anyone who can author a role note can copy another role's blob into
  their own frontmatter, declare it in `unseal`, and print it. AAD binding would close the copy
  (see Open questions); git history would not.
- **Per-role key separation.** `unseal_env_key` lets different roles seal against different
  master keys, so compromising one key does not open the others, and a key can be rotated by
  migrating roles one at a time instead of re-sealing everything at once.

## Options considered

### A. Status quo — codellm env + `CODELLM_EXPOSE_ENV`

The operator holds values in codellm's environment and allowlists the names
(`cmd/codellm/appconfig/config.go:175`); `buildChildEnv` supplies them from codellm's own
`os.Environ()` (`cmd/codellm/internal/coderun/coderun.go:476`). Empty allowlist = expose nothing.

Not broken, and for a single operator not even wrong — "the operator can read their own secret"
is tautological. Its real costs are ergonomic: adding a secret needs a redeploy, the allowlist
lives in docker-compose detached from the role note that depends on it (an undocumented surprise
dependency), and the allowlist is flat.

**Kept, and not as a fallback.** For an operator who already runs Docker secrets, Kubernetes
secrets or a Vault agent, the credential is already in a process environment and sealing is
duplicate bookkeeping. This is the normal path; sealing is what a role reaches for when a secret
belongs to it alone, or when adding one must not require a redeploy.

The one thing that changed is that `env_passthrough` / `env_prefix` came back as role fields, this
time as an **intersection**: the operator allowlist is still the boundary and a role can only
narrow its own share of it. Declaring neither keeps the previous behaviour (the whole allowlist),
so no existing role note is affected. Note this is a tightening, not a relaxation — before it, one
code role saw every secret every other code role on that codellm needed.

### B. `requires_secrets` manifest + the existing `secrets` table

trip2g already has an encrypted secret store: `secrets (key unique, value_crypt blob)`
(`db/schema.sql:829`), AES-256-GCM via `internal/dataencryption`, use cases under
`internal/case/admin/{setsecret,getsecret,listsecretkeys,deletesecret}`, an admin UI at
`assets/ui/admin/secrets/`, and no read-back mutation — values never leave the server. It is
already consumed by key-prefix convention (`change_webhooks:<id>:<name>` →
`GetSecretValues(ctx, prefix+"%")`, `cmd/server/config.go:168`).

The role note would carry only the *name* (`requires_secrets: [KRISP_TOKEN]`) — the hardening
`codellm_extraction.md` already sketches — and the value would live in that table.

Best rotation and revocation story of any option: one row to update, `deleteSecret` as an instant
kill switch, nothing in git. Most consistent with the existing webhook pattern.

**Rejected on cost and on architecture.** It needs GraphQL surface, admin UI work, a scoping
convention, fleet-side resolution, and delivery wiring — and delivery reverses a documented
decision: codellm's design explicitly refused to carry secrets in the request from fleet. Over
HTTP there is no way to reach codellm's child env without crossing the wire, so "inject into the
env, not the JSON body" is cosmetic; the reversal would have to be owned honestly. Against that,
the chosen design needs no trip2g change at all.

### C. Symmetric Rails-style ciphertext, encrypted through a web UI

Ciphertext in the note, master key in server config, and a UI where the user pastes a plaintext
secret and gets a blob back.

**Rejected.** The UI is the problem, not the ciphertext. If the browser holds the master key, a
decrypt-everything key is distributed to clients — strictly worse than today. If the server
encrypts on request, the plaintext crosses the wire to trip2g, which is byte-for-byte what
`setSecret` already does, so the blob buys nothing over the existing store while adding a second
crypto path.

The insight that killed it also killed the need for it: Rails' `credentials:edit` is a **CLI**,
not a web UI. With operator == secret owner, a `seal` subcommand is the whole "UI".

### D. Asymmetric sealed blob (age / libsodium sealed box)

codellm publishes a public key; the operator seals offline with `age`; only codellm can open it.
Structurally the only variant where plaintext never crosses any wire at encryption time, and the
only one that gives per-vault-owner scoping by construction.

**Rejected as unnecessary here.** Asymmetry earns its keep when the encrypting party must not be
able to decrypt — i.e. multi-tenancy, which the pivotal fact rules out. `filippo.io/age` is a
fine dependency, but it would buy a property nobody needs while `internal/dataencryption` already
provides AES-256-GCM in-module. Revisit if multi-tenancy ever becomes real.

### E. `unseal()` inside fleetkit

fleet forwards the frontmatter in the bag; the master key is exposed to the child through the
existing `CODELLM_EXPOSE_ENV` mechanism; the role calls
`token = fleetkit.unseal(fm.krisp_token)`.

Genuinely attractive: **zero Go changes in codellm**, decryption is an explicit, debuggable line
in the role's own code, and no new format knowledge anywhere in Go.

**Rejected** because it puts the master key in the environment of every code child. Any role can
then decrypt any blob it can reach (and it can reach other roles' frontmatter via `attach_notes`),
so the scoping win disappears — the flat-allowlist weakness of option A comes back in a new form.
It also needs `unseal` implemented twice, in `fleetkit.py` and its node twin
(`cmd/codellm/fleetkit/node/index.js`).

### F. Targeted unseal in codellm, injected as env vars

codellm decrypts the role's sealed fields and injects them into the child env under derived
names (`krisp_token` → `KRISP_TOKEN`), so existing role code reading `os.environ['KRISP_TOKEN']`
keeps working unchanged.

**Superseded by the chosen design.** It drags in a name-mapping rule (case conventions, collisions
with `CODELLM_EXPOSE_ENV` names, two roles wanting different values under one name), and it splits
the role's inputs across two sources: the secret from `os.environ`, the neighbouring non-secret
config from the bag. Substituting into the bag removes that whole class of questions.

## Chosen design

Option F's key handling with the bag as the delivery surface.

### The note

Three frontmatter fields matter:

| Field | Meaning |
|---|---|
| `unseal` | list of frontmatter field names whose values are sealed |
| `unseal_env_key` | name of the env var in codellm holding the master key; defaults to `SEAL_KEY` |
| the named fields | `sealed:v1:<base64(nonce‖ciphertext)>` |

`unseal` is an explicit declaration rather than sniffing values for a `sealed:` prefix, because:
codellm fails loudly on a declared field it cannot open, instead of silently passing ciphertext
through as an ordinary string (a typo in a blob becomes an unseal error, not a 401 from the
upstream API minutes later); a reader sees at the top of the note which fields are secret; and
codellm never attempts to decrypt arbitrary strings.

`sealed:v1:` carries the format version so the algorithm can change without invalidating existing
blobs.

### The flow

1. **fleet** puts the role's own frontmatter into the delivery bag. It does not decrypt anything
   and holds no key.
2. **codellm** reads `unseal` from the bag, resolves `unseal_env_key` against **its own**
   environment, decrypts those fields, and replaces the ciphertext with plaintext in the bag it
   writes to `$FLEET_INPUT`.
3. **The role's code** reads config through `fleetkit.frontmatter()` and unsealed values through
   `fleetkit.secrets()`. Both return the attribute-access mapping `fleetkit` already uses for
   frontmatter, so a missing key reads as `None` instead of raising.

The ciphertext stays in the frontmatter as it was written. codellm adds the plaintext alongside
rather than substituting it, so a role can still tell that a value was sealed, and the bag's
frontmatter section is a faithful view of the note.

trip2g needs no change: it stores and serves the note like any other, ciphertext included.

### Where it touches the code

**fleet** (`cmd/fleet/internal/fleet/`):

- `ParseRole` (`role.go:38`) currently consumes the flat frontmatter map `m` and drops it; it must
  retain it on `Role`, plus parse `unseal` with the existing `parseList` and `unseal_env_key`.
- `fleetinput.Input` (`internal/fleetinput`) gains a frontmatter field. It is the package already
  shared by fleet and codellm, so the contract lives in the right place.
- `buildInputBag` (`handler.go:237`) and `buildCronInputBag` (`handler.go:360`) populate it.

fleet parses `unseal` rather than passing the raw string because the frontmatter arrives
stringified: `noteViewResolver.Meta` (`internal/graph/schema.resolvers.go:3014`) renders each
value with `fmt.Sprintf("%v", value)`, so `unseal: [krisp_token]` arrives as the literal
`[krisp_token]` — the form `role.go:246` already warns about and `parseList` already handles.
fleet normalising it keeps that quirk out of codellm. The same stringification means bag
frontmatter is **strings only**: a numeric or boolean frontmatter value arrives as its text form,
so nothing downstream should expect typed values.

This weakens "fleet knows nothing about secrets" to **"fleet never sees plaintext and never holds
the key"** — it knows a list of field *names*. That is the property that matters and it is intact.

**codellm** (`cmd/codellm/`):

- `handleChatCompletions` (`internal/codellm/server.go:237`) already splits body and bag via
  `extractBodyAndBag` (`:297`); the unseal step goes between that and building `coderun.CodeInput`.
- The master key comes from config, alongside `ExposeEnv`/`ExposeEnvPrefix`
  (`appconfig/config.go:175`).
- Crypto is `internal/dataencryption` (AES-256-GCM), imported the same way `internal/fleetinput`
  already is. No new crypto, no new dependency.
- Sealing has two front doors, a `seal` subcommand and an HTTP endpoint — see below. Together they
  are the entire provisioning UI.

**fleetkit**: nothing. `bag()` already exposes the delivery bag; the frontmatter simply appears
inside it, and the node twin gets it for free.

### Key handling and guards

The master key must be **exactly 32 bytes** (`dataencryption.NewManager`). trip2g already has this
convention for `DataEncryption.Key` — a raw 32-character string in config, with boot refusing the
default value (`internal/appconfig/config.go:735`) — and it should not be reinvented.

**The key must never reach a child process.** That is the entire difference between this design
and option E, and it is undone silently by one line of docker-compose: listing `SEAL_KEY` in
`CODELLM_EXPOSE_ENV`, or letting `CODELLM_EXPOSE_ENV_PREFIX` sweep it in, forwards the master key
to every code child and turns this back into option E without anything visibly breaking.

So codellm refuses:

- **at startup**, if the default `SEAL_KEY` is in `ExposeEnv` or matches `ExposeEnvPrefix`;
- **at unseal time**, for whatever name a note supplied in `unseal_env_key`, since without a
  registry of key names codellm only learns custom names from notes.

Same predicate, two moments. It is a deploy mistake, so it should be loud.

A note may name any env var in `unseal_env_key`, and that is accepted: a wrong key cannot reveal
a value, because `NewManager` rejects anything that is not 32 bytes and AES-GCM rejects a wrong
key on the authentication tag. What remains is a presence-and-length oracle over codellm's whole
environment, and note that the residue is the success/failure **boolean**, not the error text —
generic messages narrow it but do not remove it. Bounded by the fact that both surfaces now
require auth. A declared `unseal` field that is *absent* from the frontmatter should fail as
loudly as one that cannot be opened. The thing to avoid is propagating the underlying error text
outward — `NewManager` reports `"data encryption key must be exactly 32 bytes, got %d"`, which
would give a note author a length-and-existence oracle over codellm's environment, surfacing
through the 422 into fleet's run logs and the delivery trace UI. codellm returns
`unseal failed for "<field>"` and keeps the detail in its own log.

### Sealing: CLI and endpoint

Producing a blob must not require a terminal on the codellm host, so sealing has two front doors
over one operation.

**CLI (built).** `codellm seal --env-key SEAL_KEY_V2` reads the plaintext from stdin and prints
`sealed:v1:...`. `--env-key` names the master key and defaults to `SEAL_KEY`, mirroring the form's
field. The value comes from stdin rather than a flag on purpose: an argv secret is visible in the
process table and lands in shell history — the same class of leak as a value in a query string.

**HTTP (not built yet).** One path from config, defaulting to `/_system/codellm/seal` (the `/_system/` prefix
matches trip2g's own admin endpoints). `GET` renders a small HTML form, `POST` performs the
sealing. Two fields — the env var name holding the master key, defaulting to `SEAL_KEY`, and the
value to seal — so the operator can paste a credential in a browser and copy the blob straight
into a note. It is always enabled; the base path is configurable for deployments where the default
collides or is inconvenient.

Four rules, none of them optional:

- **The value travels in the POST body.** The `GET` carries nothing but the empty form. Keep it
  that way: a secret that reaches a URL — a GET parameter, a redirect — lands in reverse-proxy
  access logs, browser history, and `Referer` headers, where a provisioning path leaks worse than
  the thing it protects.
- **Never log the value**, at any level. codellm's existing habit of logging exposed env *names*
  and never values (`logExposedEnv`, `internal/codellm/server.go:314`) is the standard to match.
- **Scrub the unsealed plaintexts from every error, preview, and returned log.** This is mandatory,
  not hygiene, because one existing path leaks them into trip2g's database deterministically.
  `ParseCodeOutput` embeds up to 200 bytes of the child's stdout verbatim when the JSON contract
  fails to parse (`internal/coderun/coderun.go:369-375`, `"...(got: %q)"`). The doc's own
  "realistic failure" — a `print(fm)` before the protocol line — is exactly what makes stdout
  unparseable, so the decrypted token rides that preview into the 422, into fleet's run error, into
  `AgentResponse.Message`/`Logs`, and into the delivery-log store and trace UI. That is the store
  sealing exists to keep clean. Captured stderr tracebacks take the same route. Unlike general
  output scanning — dismissed elsewhere in this document as a control — this is precise and cheap:
  codellm knows the exact strings it decrypted for this run and can redact those, with no
  heuristics and no false positives.
- **Generic errors.** Same rule as unseal: `NewManager`'s `"must be exactly 32 bytes, got %d"` must
  not reach the response, or the endpoint becomes a length-and-existence oracle over codellm's
  environment for anyone who can reach it.
- **Wrapped in `cfg.Auth`, like its neighbours.** This is a requirement, not an inherited property.
  `Handler` (`internal/codellm/server.go:214`) gates the two browser-facing endpoints —
  `/v1/chat/completions` and `/_system/codellm/graphql`, including the playground's `GET` — through
  `s.cfg.Auth`, which in production is `BrowserAuth(delegatedAdmin, TokenCheck)`: an api_key lane
  and a delegated-admin cookie lane. Only `/healthz` and `/v1/models` are open. A seal endpoint
  registered as a bare `mux.HandleFunc` would be the *weakest* thing on that mux, not an equal of
  the execution endpoint beside it. Both verbs go through the same middleware, the same way the
  GraphQL playground's `GET` already does.

  Note that `codellm_extraction.md`'s risk 2 ("an unauthenticated `/v1/chat/completions` is
  RCE-as-a-service") describes the state before that auth seam existed. It no longer holds, and
  reasoning that treats codellm as ungated — including any argument that a new endpoint cannot make
  things worse — is reasoning from a stale premise.

The endpoint carries no enable/disable switch. Authorization is already answered by `cfg.Auth`, so
a flag would be answering a *deployment* question inside the binary: whether this address is
published, to whom, and behind what — reverse proxy, VPN, mTLS — is an operations decision, and the
configurable base path exists to serve it. Always registered, always gated.

## Costs, honestly

- **Rotation of a leaked upstream token**: re-seal, edit the note, sync. Cheap at this scale, but
  it is note editing rather than a single row update.
- **Rotation of a master key**: re-seal every blob sealed against it. `unseal_env_key` softens
  this — roles migrate to a new key one at a time — but it never becomes a one-liner.
- **Vault history**: old ciphertext lives in the vault's git history forever, and a later leak of
  the master key decrypts it retroactively. This is the property Rails has and everyone accepts,
  but it should be named.
- **No instant revocation**: there is no `deleteSecret` equivalent. Killing a secret means editing
  the note.
- **The plaintext lands in the bag, and the bag is a file** (`FLEET_INPUT=`, `coderun.go:479`).
  Two consequences: the value is on disk inside the sandbox for the run, and — more likely to
  bite — the bag is the natural thing to dump while debugging. `print(fm)` or
  `emit(answer=json.dumps(bag))` publishes the token into vault content. Leaking through env
  requires deliberately reaching for `os.environ`; here the secret rides inside the object most
  likely to be dumped wholesale.
- **codellm learns the seal format.** Its "no vault, no KB, no auth, no secrets" invariant becomes
  "holds one master key". Weighed against N plaintext values in its env today, this is still a
  reduction.
- **Roles not on codellm silently do not work.** An `executor: llm` role's bag goes to a real LLM,
  where nothing unseals it. Safe — the model sees ciphertext — but quietly non-functional.
- **The chat endpoint is a decryption oracle for anyone holding a channel credential.** codellm
  trusts the bag: whoever can `POST /v1/chat/completions` — fleet with its api key, or any
  delegated-admin cookie — can submit a synthetic bag carrying *any* blob ever sealed under a key
  codellm holds, including ciphertext rotated away long ago and recovered from vault git history,
  together with a body that prints it. Putting the seal endpoint behind `cfg.Auth` does not touch
  this; the execution endpoint is the oracle. So "fleet never sees plaintext and never holds the
  key" describes the honest dataflow only — fleet's credential is sufficient to obtain any
  plaintext on demand. Acceptable under the pivotal fact, but this is the design's real trust
  boundary and should not be left implied intact.
- **The friction being removed was also the authorization gate.** Today, granting a role a secret
  requires editing docker-compose and redeploying — an out-of-band act that doubles as "the
  operator specifically approved this role getting this value". Under sealing, `unseal:
  [krisp_token]` committed into a note is sufficient on its own: the next run has the plaintext,
  with no separate infrastructure step. That is precisely the ergonomics being bought, and the
  authorization checkpoint is what it is bought with. It holds only while vault-edit rights and
  infra-deploy rights belong to the same person. A collaborator with Obsidian sync access but no
  docker-compose access would, under sealing, be able to grant secret access; today they could not.
  This is a narrower form of the pivotal fact, and it deserves its own line because it can stop
  being true earlier: the operator may still run role code they did not author line by line — a
  shared recipe, a generated script — and sealing removes the manual step that would have made them
  look at it.

## What this deliberately does not solve: exfiltration

Once a role holds a secret, nothing stops it from writing that secret into a note. Scope
enforcement checks the **path**, never the content (`ScopedKB` / `write_patterns`), so a role with
a legitimate secret and a legitimate write scope can publish the secret into the vault, and from
there onto the site. This is not a regression introduced by sealing — it is equally true of
today's `CODELLM_EXPOSE_ENV` path — but sealing does not improve it either.

One escalation path this design depends on is closed separately: a role that can
write note content could otherwise write a *new role note* declaring `unseal`
for any blob plus its own unrestricted `write_patterns`. `ScopedKB` now refuses
to let an agent author or edit a note carrying `fleet_id` — see
[fleet_write_validation.md](fleet_write_validation.md), "Built: agents may not
author role notes".

A stronger shape was considered and **deferred**: a separate request-building block type
(`saferequest`) taking a declarative request spec — jsonnet or JSON/YAML; `internal/jsonneteval`
already exists — where secrets are substituted only while constructing the outgoing request and
are never handed to general-purpose code at all. The role would describe *which requests it may
send*, not hold the credential.

It does not confine the secret on its own. If the spec can also choose the destination, a role
points its base URL at a service the author controls, that service echoes the credential back in
the response body, and ordinary code reads it out of the response — the secret is back in hand
having never been "held". Confinement requires pinning the **destination**, not just the shape of
the request. The templating is the easy half.

Scanning outgoing writes for the secret value is not an answer either. It catches an *accidental*
dump — `print(fm)`, `emit(answer=json.dumps(bag))`, which is the realistic failure here — and
codellm can do that precisely, since it knows exactly which plaintexts it unsealed for the run.
But base64, XOR, or splitting the value across notes defeats it trivially, and the network channel
bypasses it entirely. It is a safety net, in the same category as a linter. It is not a control.

## Where real confinement lives — and why it is not this mechanism

If the goal is for the executing code to be genuinely unable to leak a credential, the answer is
not a better way to store it. It is to **cut the code off from the network and route all egress
through a proxy that holds the credentials and injects them into outgoing requests**. Then the
code never holds the secret at all, and `write(secret)`, output scanning, and encoding tricks all
stop mattering — there is nothing in the sandbox to encode. The destination pinning that the
`saferequest` idea above was missing comes for free, because the proxy decides where traffic may
go.

Half of it already exists: codellm's sandbox has a network switch (`SandboxNetwork`), so "no
direct internet" is a knob, not a new mechanism.

Two shapes, with very different cost:

- **Intercepting proxy.** The role speaks HTTPS to the real host; the proxy terminates TLS with
  its own CA installed in the sandbox trust store, injects auth, and re-originates. `mitmproxy`
  with a script addon or Envoy with a Lua/WASM filter does this. TLS interception — not the
  allowlist — is where the complexity is.
- **Proxy as upstream.** The role speaks plain HTTP to a pinned route (`http://proxy/krisp/...`);
  the proxy adds authentication and originates TLS outward. No CA, no interception, and the
  destination is pinned by proxy config rather than by anything the note says. Substantially
  cheaper, and sufficient for a known set of upstreams.

Off-the-shelf tooling splits along the same seam and is usually assembled rather than bought
whole: egress allowlisting (`smokescreen`, Squid ACLs, Envoy as an egress gateway) on one side,
enrichment and audit (mitmproxy addons, Envoy Lua/WASM, API gateways with upstream-auth plugins)
on the other.

**This is deliberately out of scope.** Confinement is a property of the execution environment —
network policy and egress control — not of a secret-storage format, and building it is not this
mechanism's job. Sealing makes no attempt at it.

What `unseal` is, stated plainly: **a trade of confinement for provisioning ergonomics.** It buys
"add a secret by editing a note, scoped to that role, with no redeploy" and buys nothing at all
against a role that decides to publish the value it was given. That trade is acceptable here for
one reason — the pivotal fact above: the person authoring the roles is the person who owns the
secrets, so the role is not an adversary. Under any other trust structure the trade would be
wrong, and the proxy would be the requirement rather than the alternative.

The two are complementary rather than competing. If the proxy is ever built, secrets move into its
configuration and sealing becomes optional — at the price of a configured route per upstream,
which is exactly the provisioning ergonomics sealing exists to provide.

## Open questions


- Whether `unseal` should also be honoured on **attached** notes. **Answer: no, and the code
  already enforces it.** `buildAttachedNotes` (`internal/webhookutil/attached_notes.go`) constructs
  `Meta` explicitly from `tags` and `layout` only — arbitrary frontmatter never travels that path —
  and codellm unseals only the role's own frontmatter, so cross-role unsealing is impossible by
  construction rather than by omission.

  Recorded as a **standing invariant**, because the convenient answer is dangerous: `SEAL_KEY` is
  the default `unseal_env_key` for every role, so in the common case all roles share one key.
  Extending unsealing to attached-note frontmatter would let any role decrypt any other role's
  sealed fields by merely `attach_notes`-referencing it — reopening exactly the cross-role leak
  option E was rejected for, by a different route.

  Caveat: `AttachedNote.Content` carries the full note text, so another role's *ciphertext* is
  still readable. That is what makes the AAD question below matter, and it is why option E's
  rejection stands.

- **Standing invariant: the `exec` tool must never receive the bag.** `makeExecInvoker`
  deliberately sends no `fleet_input` message (`cmd/fleet/internal/agentruntime/runtime.go:683`,
  "No fleet_input message: exec runs inline"), so an LLM role's mid-run `exec` never sees unsealed
  values. If anyone later adds the bag to `exec` calls, plaintext flows into a real model
  conversation and out to the provider.

- ~~Whether to bind the ciphertext to its location with AAD.~~ **Decided: no AAD.** `dataencryption` passes `nil` additional
  data, so nothing ties a blob to the note and field it belongs to: anyone who can read a blob can
  paste it into their own role under the same key and have it decrypt — relocation, not a crypto
  break. The obvious binding is the note path, and it has a real cost: **renaming or moving a role
  note would break every blob in it**, silently, at the next tick — and moving notes around is a
  routine Obsidian operation. Binding only the field name does not prevent relocation, so the
  half-measure buys nothing. Note also that `EncryptData`/`DecryptData` take no AAD parameter and
  are shared with the `secrets` table and `cmd/tge2e`, so this means a new method beside them, not
  a change to them.

  Rejected because the cost lands on an everyday operation and the benefit does not. Moving a note
  between folders is routine in Obsidian, and binding to the path would break every blob in it
  silently, at the next tick. What binding buys is preventing one role from copying another's
  ciphertext — and under the pivotal fact the roles are written by the same person who owns the
  secrets, so that is not an adversary. Revisit only if role authorship ever spreads beyond one
  trusted party, which is the same condition that reopens the rest of this design.
- What the `--dry-run` reconcile report should say about a role whose `unseal` field cannot be
  opened — a startup-time validation would catch a bad blob before its first cron tick.
