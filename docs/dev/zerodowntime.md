# Zero-downtime deploys (infra)

How the `infra/` Ansible deploy avoids dropping requests during a restart.

> **Status update (socket activation now implemented).** Parts of the prose below
> were written before the Phase-1 merge and assumed a single `/healthz`. Two things
> changed:
> 1. The app exposes **`/livez`** (liveness — 200 while the process is alive, incl.
>    warmup) and **`/readyz`** (readiness — 503 during warmup/drain) on the internal
>    port, plus the legacy `/healthz`.
> 2. **Socket activation is implemented** (`systemdListener` in `cmd/server/main.go`
>    inherits the `LISTEN_FDS` fd; `infra/socket.j2` defines the per-service
>    `.socket`). The listening socket now outlives a restart, so connections queue
>    in the backlog instead of being refused — **true zero-downtime on one server,
>    no LB, no SQLite contention.** The Traefik active check targets **`/livez`**
>    (not `/readyz`): with a single backend the socket queues the warmup, so the
>    check must keep the backend in rotation and only catch a genuinely dead process.
>
> Canonical references: `docs/dev/zerodowntime_research.md` (E1–E7 experiments) and
> `docs/en/user/zerodowntime.md` (user recipes). The sections below describe the
> earlier health-check + drain wiring, kept as the fallback path.

## Current deploy flow

Single server. Ansible (`infra/site.yml`) copies one shared binary to
`/opt/trip2g/bin/app`, then `systemctl restart`s each per-site systemd unit
(`trip2g`, `trip2g_landing`, `trip2g_demo2`, `trip2g_founder`,
`trip2g_internaldev`, ...). Each site is independent: its own domain, its own
SQLite DB, its own `/etc/{service}.env`. Traefik (v3, file/dynamic provider,
`infra/traefik_dynamic.yml.j2`) routes each domain to a **single** backend.

## App contract (verified, this branch)

The app already exposes the signals the infra needs — no `/livez` + `/readyz`
split, the single `/healthz` already encodes readiness:

| Signal | Where | Behaviour |
|--------|-------|-----------|
| `/healthz` | internal port (`--internal-listen-addr`) | `200` while serving; **`503` the instant SIGTERM arrives** (`a.stopped`), and during warmup |
| drain | on SIGTERM | flip `/healthz` → 503 → `sleep(--shutdown-grace-period)` → graceful HTTP `Shutdown` (bounded by `--shutdown-timeout`) |
| public traffic | `--listen-addr` | separate port from the internal/health one |

Flags (env equivalents use the `TRIP2G_` prefix, e.g. `TRIP2G_INTERNAL_LISTEN_ADDR`;
CLI args on `ExecStart` take precedence over env):

- `--listen-addr` — public port
- `--internal-listen-addr` — health/metrics port (default `:8082`)
- `--shutdown-grace-period` — drain window (app default 50ms; we set `5s`)
- `--shutdown-timeout` — graceful-shutdown bound (app default 1s; we set `10s`)

There is **no** `--simple-backup-on-shutdown` or `--vacuum-cron` flag. Shutdown
backup runs only if simple-backup is configured for that site; on the rolling
deploy path the restarted unit takes over, so no extra backup flag is needed.

## What is wired today (gets us to "no in-flight loss", ~99%)

1. **Per-site internal health port** (`infra/site.yml`, `service.j2`). Each
   service gets a unique `internal_addr` (`localhost:1908x`). Without this every
   unit would collide on the app default `:8082` and health checks would be
   meaningless.
2. **`/healthz` active health check** on the internal port for every trip2g
   `loadBalancer` (`traefik_dynamic.yml.j2`): `path: /healthz`, `interval: 1s`,
   `timeout: 1s`. Detection window ≈ 1–2s.
3. **Drain ≥ detection window.** `--shutdown-grace-period 5s` > the ~2s Traefik
   detection window, so on SIGTERM Traefik pulls the backend out of rotation
   **before** the socket closes. `TimeoutStopSec=20` (> grace + shutdown-timeout)
   plus `KillSignal=SIGTERM`/`KillMode=mixed` guarantee systemd never SIGKILLs
   mid-drain. This is the fix for the old "forever to restart" hang: the old
   binary no longer blocks the critical path, and systemd waits out the bounded
   drain instead of hanging.
4. **`retry-idempotent` middleware** (`retry: { attempts: 4 }`) on every trip2g
   router.

### Honest limit: retry is a near no-op with one backend

Traefik's `retry` retries against the **server pool**. With a single backend per
site, every retry during a restart resolves to the same just-killed backend, so
the unbound-port window between old-process-exit and new-process-listen still
refuses connections. The health check + drain shrink in-flight request loss to
near zero, but the restart gap itself is **not** closed by anything wired today.
`retry` becomes genuinely useful only once a second backend exists (see below);
it is kept now because it is harmless and covers transient upstream blips.

## The step to true 100%

Two ways to cover the unbound-port restart gap. For this topology
(**per-site embedded SQLite, one shared binary, file-provider Traefik**),
**socket activation is recommended** and blue-green is discouraged.

### Recommended: systemd socket activation (needs a small app change)

A `trip2g@.socket` owns the public port and **outlives** the service restart, so
the kernel queues incoming connections (extra ms of latency) instead of refusing
them; the new process inherits the already-bound fd via `LISTEN_FDS`.

- Exactly **one** process per site DB at all times → no SQLite contention.
- Unit count unchanged; Traefik config unchanged.
- **Follow-up app code change required:** the server currently calls
  `ListenAndServe(--listen-addr)`. It must instead use the passed listener fd
  when `LISTEN_FDS`/`LISTEN_PID` are set (Go: `net.FileListener` on fd 3),
  falling back to `--listen-addr` otherwise. Until that lands, socket activation
  cannot be enabled.

### Discouraged here: blue-green

Two units per site (`trip2g@blue`/`@green`) on two ports, both as Traefik
backends with the `/healthz` check; release starts the idle colour, waits for
`/healthz=200`, then stops the old. This is where the `retry` middleware finally
earns its place. But it is the **wrong tool for embedded per-site SQLite**: blue
and green run simultaneously during the cutover, and the DB is opened with
`busy_timeout=20000` (single writer). A write on the draining old colour during
handoff makes the other side block up to 20s / hit `SQLITE_BUSY`. It also doubles
the unit count and forces the release task to track the current colour. If ever
adopted, scope it to the flagship site only.

## Recommended release sequence

1. `ansible-playbook -i hosts site.yml -e ... ` builds/uploads the binary and
   re-renders `traefik_dynamic.yml` (health check + retry) and the units.
2. The `Restart service` handler `systemctl restart`s each unit. Per unit:
   SIGTERM → `/healthz` 503 → Traefik drops it within ~2s → app drains 5s →
   graceful shutdown → new process starts and warms → `/healthz` 200 → Traefik
   routes again.
3. Deploy Traefik config (`/etc/traefik/dynamic.yml`); the file provider
   hot-reloads it (no Traefik restart needed for dynamic changes).

To verify a single site live (do **not** run against prod casually):
`curl -s localhost:1908x/healthz` should be `ok`; during a restart it returns
`shutting down` (503) for the grace window.

## Risks / follow-ups

- **Restart gap is still open** until socket activation lands (the one app-code
  follow-up). Today's wiring is "minimal+drain", honestly ~99% with idempotent
  retry tails, not a hard 100%.
- `internal_addr` ports (`1908x`) are bound on localhost only and must stay
  unique per service and free of UFW exposure (they are not opened — only 80/443
  are).
