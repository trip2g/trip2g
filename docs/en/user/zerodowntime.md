---
title: "Zero-downtime deploys"
free: true
lang_redirect: "[[ru/user/zerodowntime]]"
---

Shipping a new version of trip2g should not drop a single read. trip2g gives your orchestrator — or even a bare server with no load balancer — exactly the health signals and warmup behavior needed to swap versions while readers keep getting their pages.

## The two health endpoints (you need BOTH)

trip2g serves two probes on its internal port. Wiring both correctly is the whole game — they answer different questions.

### `/livez` — liveness (mandatory)

Returns **200 whenever the process can answer**, even while it is warming up or draining. It answers "is the process alive?". The orchestrator uses it to **restart a hung instance**. It must **never** return 503 during warmup — otherwise the orchestrator decides a perfectly healthy, still-warming instance is broken and kills it → crash loop, it never finishes warming.

### `/readyz` — readiness (mandatory)

Returns **200 only when the instance is fully ready to serve** (warmed up, writer slot acquired). Returns **503 while warming up and while shutting down**. The load balancer and the deploy gate use it to decide whether to **route traffic**. This is what keeps the old version serving until the new one is genuinely ready.

(`/healthz` is also served, kept for backward compatibility.)

## Why warmup matters

On boot trip2g builds its in-memory state — rendered note views, the search index — before it can serve a single page. That is seconds on a small vault, longer on a big one. If a deploy sends traffic to the new instance before that finishes, readers get errors. The fix: the new instance warms up **while the old one keeps serving**, and `/readyz` gates the switch so traffic only moves once the new instance is ready.

## The handoff knobs

- `--shutdown-grace-period` — on shutdown trip2g flips `/readyz` to 503 and keeps serving for this long before it actually stops. Set it **≥ your load balancer's unhealthy-detection window** (e.g. 5 s for a 1–2 s health-check interval) so the balancer drains the old instance before it dies. Too short → a few `502 Bad Gateway` at the cutover.
- `--simple-backup-on-shutdown=false` — skip the final backup during a rolling deploy; a peer is already taking over (see [[backup]]).
- `--vacuum-cron` — off by default; leave it off (see [[litestream]]).

## Recipes

### Managed: Fly.io

The simplest path. `fly launch` scaffolds a `fly.toml`; map its health checks to `/readyz` and `/livez`, and Fly's deploy gates on them automatically. For true zero-downtime SQLite you pair it with **LiteFS** (read replicas) — Fly's bluegreen strategy cannot share a volume, and LiteFS sidesteps that by giving each machine its own replica. Read Fly's own write-ups: [Litestream VFS](https://fly.io/blog/litestream-vfs/), [Introducing LiteFS](https://fly.io/blog/introducing-litefs/), [LiteFS docs](https://fly.io/docs/litefs/), [Seamless deployments](https://fly.io/docs/blueprints/seamless-deployments/).

### Self-hosted: Nomad + Traefik + Consul

The recipe we measured:

- Nomad `update { canary = 1, auto_promote = true, health_check = "checks" }` with a Consul `check` on `/readyz`.
- **Consul service discovery** (not plain Nomad SD): the catalog is health-gated, so the warming canary never receives traffic until `/readyz = 200`. This alone removes "route to a warming instance" errors.
- App graceful drain (`--shutdown-grace-period` ≥ the check interval) removes "route to a dying instance" errors.

Result: ~99 % of reads stay 200 through a deploy; the last ~1 % are retryable 502s at the cutover edge, which an idempotent-GET retry middleware masks.

### Self-hosted: systemd, no orchestrator

Blue-green with two units (`trip2g@blue` / `trip2g@green`) on different ports and a reverse proxy. Caddy is a clean fit:

```
:80 {
	reverse_proxy 127.0.0.1:8081 127.0.0.1:8082 {
		health_uri /readyz
		health_interval 1s
	}
}
```

Deploy: start the idle colour, wait for its `/readyz = 200`, add it to the proxy, remove the old one, stop it. `caddy reload` is graceful (no dropped connections).

### Bare server, one port, NO load balancer

trip2g can hand the listening port from the old process to the new one with no proxy at all, via **SO_REUSEPORT** (the new process binds the shared port only *after* it is warm, so the kernel routes to it only once it can serve) or socket fd-passing. The single-port path measured ~99.8 %; the tail is in-flight connections cut at the old process's exit, which a graceful drain removes.

## Measured numbers

All on a Hetzner **cpx32** (4 vCPU / 8 GB, x86, AMD), single SQLite writer, vegeta at 80–100 rps through a rolling deploy. Full method + per-run data in the developer research log.

| Deploy path | reads kept (200) |
|---|---|
| Naive (proxy not health-gated) | ~77 % |
| Traefik active `/readyz` health-check | ~98 % |
| Consul SD + app graceful drain | ~99 % |
| SO_REUSEPORT, single port, no LB | ~99.8 % |

The pattern: a proxy that is not health-gated routes to the warming instance (503s) and to the dying instance (502s). Gate it on `/readyz` (Consul, or the proxy's own health check) and give the app a real drain window, and dropped reads go to near zero.
