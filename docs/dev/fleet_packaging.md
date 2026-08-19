# Fleet packaging

**TL;DR:** the trip2g image ships two static binaries — `/trip2g` (the primary, `CMD`) and `/fleet` (the agent host, secondary). The base image is **LLM-only**: no language interpreters. If a role uses `executor: code`, you add the interpreter yourself — extend the image or copy the static `/fleet` binary into your own runtime. This keeps the default image small and the code-execution surface an explicit operator choice (it's off by default anyway — `--allowed-programs` is empty).

## One image, two binaries

The main `Dockerfile` builds both binaries (`CGO_ENABLED=0`, fully static) into one Alpine image:

```
/trip2g   # CMD — the server
/fleet    # secondary — run as a sidecar / second container
```

Run trip2g normally; run the fleet from the same image by overriding the command:

```yaml
# docker-compose
services:
  trip2g:
    image: ghcr.io/trip2g/trip2g:latest        # CMD ["/trip2g"]
    environment:
      - OWNER_PERSONAL_TOKEN_VALUE=t2g_...     # seeded for the owner at boot
  fleet:
    image: ghcr.io/trip2g/trip2g:latest
    command: ["/fleet"]
    environment:
      - TRIP2G_FLEET_TRIP2G_URL=http://trip2g:20081
      - TRIP2G_FLEET_CALLBACK_URL=http://fleet:9090
      - TRIP2G_FLEET_TRIP2G_ADMIN_PERSONAL_TOKEN=t2g_...  # = OWNER_PERSONAL_TOKEN_VALUE
      - TRIP2G_FLEET_LLM_BASE_URL=...
      - TRIP2G_FLEET_LLM_API_KEY=...
      # ... see cmd/fleet flags; every flag has a TRIP2G_FLEET_<NAME> env
```

trip2g stays a dumb event source; the fleet reconciles its own webhooks against it. One image is the whole unit.

The fleet holds no signing secret. Its one credential is a trip2g personal
token: set the same value on both services, and the app seeds the matching
`user_tokens` row for the owner at boot under the name
`system: seeded by OWNER_PERSONAL_TOKEN_VALUE`. Rotating means a new value and
a restart — the same pass revokes the row the old value carried. Revoking the
row in the admin UI stops the fleet without a restart, and a restart does not
bring it back: the way back is a new value.

## Enabling `executor: code`

A role with `executor: code` runs code through an interpreter (python, node, bash…). The base image has none. Two ways to add them — both rely on the binary being static, so it drops into any runtime:

**A. Extend the image** (simplest):

```dockerfile
FROM ghcr.io/trip2g/trip2g:latest
RUN apk add --no-cache python3        # or nodejs, bash, ...
# run with --allowed-programs python (or TRIP2G_FLEET_ALLOWED_PROGRAMS=python)
CMD ["/fleet"]
```

**B. Copy the static binary into your own runtime** (when you want a different base):

```dockerfile
FROM python:3.12-alpine
COPY --from=ghcr.io/trip2g/trip2g:latest /fleet /usr/local/bin/fleet
ENTRYPOINT ["fleet"]
```

Then opt code execution in explicitly:

```
--allowed-programs python,node        # default empty = code execution disabled
```

Keep the interpreter set minimal, and isolate the fleet container (network policy, read-only FS, non-root) — see [[process_isolation]]. The base image carries no interpreters precisely so you make this choice deliberately, per deployment.
