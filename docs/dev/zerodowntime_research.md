# Zero-downtime deploy — research log

Live experiments on a disposable Hetzner node (cpx32, 4 vCPU/8 GB, x86, nbg1) with
`nomad -dev` + docker + Traefik v3.1 + vegeta. Goal: find a deploy path where reads
never drop while a slow-warmup, single-writer (SQLite) service is replaced.

This log records every scenario, the measured numbers, and the analysis. It is the
source of truth behind the user docs (`docs/{en,ru}/user/zerodowntime.md`).

## The app-side contract (Phase 1, PR #23)

Orchestrator-agnostic. The binary must:

1. **Read-only warmup** — build all in-memory state (NoteViews, bleve index,
   layouts) WITHOUT taking the SQLite writer slot or starting writer subsystems
   (queues, cron, patreon/boosty refresh). The OLD instance keeps writing while the
   NEW one warms.
2. **Writer-slot probe** — `BEGIN IMMEDIATE; COMMIT` once the write lock is
   grabbable, then start writer subsystems. (Honest-minimal; a hard Consul lease is
   a later phase.)
3. **Health endpoints** (k8s canon):
   - `/livez` — liveness: **always 200 while the process answers**, regardless of
     warmup/shutdown. A warming or draining instance is still ALIVE; the orchestrator
     uses this only for restart-on-hang. **Must never 503 during warmup** or the
     orchestrator kills the warming instance → crash loop.
   - `/readyz` — readiness: **200 only when fully ready** (writer slot acquired,
     writer subsystems started); **503 while warming and while draining**. The
     LB/orchestrator uses this for routing + deploy gating.
   - `/healthz` — legacy (kept; 503 on shutdown).

**Key lesson up front:** `/readyz` on the app is *necessary but not sufficient*. The
proxy/SD must actually gate traffic on it. The experiments below quantify this.

## Test rig

- A synthetic service mirrors trip2g's behavior: during a `WARMUP_SECONDS` window it
  returns **503 on BOTH `/` and `/readyz`** (like trip2g before NoteViews load), then
  200. So routing to a warming instance = a dropped read — exactly what we must avoid.
- Load: `vegeta attack -rate=80 -duration=40s`, deploy triggered ~5 s in.
- "Drop" = any non-200 to the read path during the deploy.

---

## E1 — Nomad-native SD + Traefik, NO LB health-check

Traefik `--providers.nomad`, service has a Nomad `check { path=/readyz }`, but the
Traefik **router/service has no own health-check**. Rolling canary deploy v1→v2.

**Result (300 req, curl loop): ~23 % dropped** — `63×503 + 7×502`.

- **503**: Nomad registers the canary's service endpoint as soon as the alloc runs;
  Traefik's nomad provider does **not** filter by check status, so it load-balances
  onto the **warming** canary for the whole ~12 s warmup → half the requests 503.
- **502**: when v1 is stopped, Traefik briefly still has its backend → bad gateway.

**Verdict:** Nomad check alone does NOT gate Traefik routing. Not zero-downtime.

## E2 — Nomad SD + Traefik LB active health-check on `/readyz`

Added service tags:
`traefik.http.services.web.loadbalancer.healthcheck.path=/readyz` (interval 1s,
timeout 1s). Rolling canary v6→v7.

**Result (vegeta, 3200 req @ 80 rps): Success 97.62 %** — `200:3124 503:40 502:36`.
Mean latency 1.9 ms.

- **503 (~1.2 %)**: Traefik treats a **newly-discovered backend as UP until its first
  active check runs** (~1 s window) → routes to the warming canary in that window.
- **502 (~1.1 %)**: deregister lag — old backend removed ~1 check-interval after stop.

**Verdict:** big improvement (23 % → 2.4 %), but **not** zero. The residual is
inherent to Traefik's "assume-up-on-add" + deregister lag.

## E3 — systemd blue/green + Traefik file-provider, explicit flip

No Nomad. Two systemd units `web-blue` (:8081) / `web-green` (:8082). Traefik
file-provider on :81 with `/readyz` health-check. Deploy script: start green → wait
green `/readyz=200` → add green to the file → remove blue from the file → stop blue.

**Result (vegeta, 3200 req @ 80 rps): Success 94.84 %** — `200:3035 503:165`.
**No 502** (blue removed from LB before being stopped — the explicit flip kills the
deregister-lag that caused E2's 502s).

- **503 (~5 %)**: comes from **Traefik file-provider reloads** — each rewrite of
  `dynamic.yml` briefly leaves the router with no UP server while Traefik rebuilds the
  service/health-checker. Two rewrites (add green, remove blue) → two transient gaps.

**Verdict:** explicit flip removes 502s but introduces **reload transients**. Mitigations
to try: a single atomic config swap instead of two; or a provider that updates
incrementally (Consul) rather than full-file reload.

---

## Analysis so far

| Path | Success | 503 cause | 502 cause |
|---|---|---|---|
| E1 Nomad SD, no LB hc | ~77 % | LB routes to warming canary (no health gate) | deregister lag |
| E2 Nomad SD + Traefik hc | 97.6 % | assume-up gap (~1 s before first check) | deregister lag |
| E3 systemd + file flip | 94.8 % | file-provider reload transients | none (explicit flip) |

The two residual failure modes are **(a) routing to a not-yet-healthy new backend**
and **(b) routing to an already-removed old backend**. Eliminating both needs a SD/LB
where a backend appears in the pool **only when its health check passes** and
disappears **before** it stops — i.e. the catalog is health-gated, not "assume-up".

**Hypothesis for true-zero: Consul service discovery.** Consul registers the service
but the instance only becomes a routable catalog entry once its check is passing, and
goes critical (removed) the moment it fails/deregisters — no assume-up window, smooth
incremental updates to Traefik's Consul provider. → E4 (next).

---

## E4 — Consul SD + Traefik Consul provider — PENDING

Plan: run Consul, register the service in Consul (via Nomad's consul provider or
direct), Traefik `--providers.consulcatalog`, repeat the vegeta rolling deploy.
Expect ~0 drops if the health-gated catalog hypothesis holds.

## Backup interaction — PENDING

- **simplebackup** shutdown-backup (`PerformBackup` in `waitForShutdown`) must be
  **skipped during a rolling handoff** (a peer is taking over; the departing instance
  backing up is redundant and can race the new writer). Gate behind a flag, off in the
  rolling path.
- **restore-on-boot** (`RestoreOnStartup`, pre-DB-init) must **not** run on a warming
  peer against a live shared DB — verify it skips when a local DB already exists.
- **Litestream** vs simplebackup: do they coexist? Litestream does continuous WAL
  replication; simplebackup does S3 snapshots. Need to confirm they don't fight over
  the WAL / checkpointing, and decide which owns backup in the zero-downtime model.
- Requires **MinIO** (S3) on the node. → separate backup doc.

## Endpoint decision (settled)

`/livez` (liveness, always 200) + `/readyz` (readiness, 503 until fully ready) +
`/healthz` (legacy). Documented as **mandatory** sections in the user doc.
