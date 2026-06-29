---
title: "Performance"
free: true
lang_redirect: "[[ru/user/perfomance]]"
chart_bench: "[[pushnotes_bench.datachart.csv]]"
chart_ramp: "[[pagecache_ramp.datachart.csv]]"
---

## In short

- **Sync** (the everyday write): one changed note pushes in tens of milliseconds even on a 10 000-note vault; the *first* full import of 10 000 notes takes ~14 s on one core.
- **Serving** (the read hot path): a single small server (Hetzner cpx32, 4 vCPU) serves **~4 000 real documentation pages per second** at 100 % success with a p99 of ~30 ms.
- **Memory**: ~80 MB per 100 notes, ~1 GB per 10 000.
- **Anonymous page cache**: a €6.49/month shared box (Hetzner cx23, 2 vCPU) sustains **~8 500 req/s** of gzipped pages for 10 continuous minutes: 0 errors, 0 % CPU steal, ≈ €0.0003 per million requests.

Getting an *honest* serving number turned out to be surprisingly hard — three measurement traps, each costing roughly 2×:

1. **Dev mode** recomputes every asset's hash on each request (~2× slower) → benchmark with `DEV=false`.
2. **Load generator on the same box** steals CPU from the server (~2× slower) → run it from a separate machine.
3. **Cold cache / a single URL** flatters the result → hit many real pages with a warmed cache.

Our naïve first attempt was ~4× too pessimistic. The numbers below are after controlling for all three. Read on for the detail and the charts.

---

How fast is a vault sync, and how big can your vault get before you feel it? We measured the `pushNotes` mutation — the call the Obsidian plugin and the CLI make on every sync — on vaults of 10 to 10 000 notes.

The charts below are live [[chartdata|datachart]] blocks reading the raw benchmark CSV from this vault — the same feature you can use for your own data.

## Method

- Each run starts a fresh server with an empty SQLite database, pinned to 1 or 4 CPU cores (`taskset` + `GOMAXPROCS`).
- Synthetic notes of ~500 bytes: frontmatter, headings, a wikilink, a list, a code block.
- **Initial push** — all N notes in one `pushNotes` call (one write transaction).
- **1 note** — a typical small sync: one changed note pushed into a vault of N notes (median of 3).
- **10% of vault** — a batch sync: N/10 changed notes in one push (median of 3).
- Memory is the server's resident set size (RSS) after the run; peak from `/proc`.
- Benchmark script: `scripts/bench-pushnotes.sh` in the repository.

Important context: while a push is being processed, the server holds the database write lock — other writers (background jobs, other pushes) wait in line. So "push time" here is also "how long the database is locked".

## Initial push: whole vault in one call

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart_bench" },
  "config": {
    "title": { "text": "Initial push, ms (less is better)" },
    "tooltip": { "trigger": "axis" },
    "legend": {},
    "xAxis": { "type": "category", "name": "notes" },
    "yAxis": { "type": "log", "name": "ms" },
    "series": [
      { "type": "bar", "name": "1 core", "encode": { "x": "notes", "y": "initial_1c" } },
      { "type": "bar", "name": "4 cores", "encode": { "x": "notes", "y": "initial_4c" } }
    ]
  }
}
```

Scaling is close to linear: ~1.2 s for 1 000 notes on one core, ~14 s for 10 000. Four cores cut that roughly in half: note rendering parallelizes but the database writes do not.

## Incremental sync: the everyday case

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart_bench" },
  "config": {
    "title": { "text": "Incremental push, ms (less is better)" },
    "tooltip": { "trigger": "axis" },
    "legend": {},
    "xAxis": { "type": "category", "name": "vault size, notes" },
    "yAxis": { "type": "log", "name": "ms" },
    "series": [
      { "type": "line", "name": "1 note, 1 core", "encode": { "x": "notes", "y": "incr1_1c" } },
      { "type": "line", "name": "1 note, 4 cores", "encode": { "x": "notes", "y": "incr1_4c" } },
      { "type": "line", "name": "10% of vault, 1 core", "encode": { "x": "notes", "y": "incr10p_1c" } },
      { "type": "line", "name": "10% of vault, 4 cores", "encode": { "x": "notes", "y": "incr10p_4c" } }
    ]
  }
}
```

This is what you feel day to day: you edit a note, the plugin pushes it. Even at 10 000 notes a single-note push takes ~0.4 s on one core and ~0.2 s on four. The renderer caches unchanged notes, so the cost grows with vault size (re-indexing), not with how much you changed — pushing 1 000 changed notes (10% of a 10 000-note vault) takes ~2.3 s on one core, far from 1 000 × the single-note cost.

## Memory

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart_bench" },
  "config": {
    "title": { "text": "Server RSS after sync, MB" },
    "tooltip": { "trigger": "axis" },
    "legend": {},
    "xAxis": { "type": "category", "name": "notes" },
    "yAxis": { "type": "value", "name": "MB" },
    "series": [
      { "type": "bar", "name": "RSS, 1 core", "encode": { "x": "notes", "y": "rss_1c" } },
      { "type": "bar", "name": "RSS, 4 cores", "encode": { "x": "notes", "y": "rss_4c" } },
      { "type": "line", "name": "peak, 1 core", "encode": { "x": "notes", "y": "peak_1c" } },
      { "type": "line", "name": "peak, 4 cores", "encode": { "x": "notes", "y": "peak_4c" } }
    ]
  }
}
```

The server keeps rendered notes in memory: ~80 MB for a hundred notes, ~180 MB for a thousand, ~1 GB for ten thousand. Plan your hosting accordingly — a 1 GB VPS comfortably serves vaults up to a few thousand notes.

## Vector search memory

Enabling vector search (semantic similarity, powered by an embedding model like bge-m3) builds an in-memory index alongside the note cache. The index stores one embedding vector per note — bge-m3 produces 1024-dimensional float32 vectors, so each note costs 4 KB.

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart_bench" },
  "config": {
    "title": { "text": "RSS with vs without vector index, MB" },
    "tooltip": { "trigger": "axis" },
    "legend": {},
    "xAxis": { "type": "category", "name": "notes" },
    "yAxis": { "type": "value", "name": "MB" },
    "series": [
      { "type": "bar", "name": "no vector, 1 core", "encode": { "x": "notes", "y": "rss_1c" } },
      { "type": "bar", "name": "vector, 1 core", "encode": { "x": "notes", "y": "rss_vec_1c" } },
      { "type": "bar", "name": "no vector, 4 cores", "encode": { "x": "notes", "y": "rss_4c" } },
      { "type": "bar", "name": "vector, 4 cores", "encode": { "x": "notes", "y": "rss_vec_4c" } }
    ]
  }
}
```

| Vault | RSS no vector (1c / 4c) | RSS with vector (1c / 4c) | Overhead |
|------:|------------------------:|--------------------------:|---------:|
| 10 | 67 / 68 MB | 70 / 69 MB | ~2 MB |
| 100 | 82 / 79 MB | 84 / 84 MB | ~5 MB |
| 1 000 | 176 / 136 MB | 205 / 179 MB | **+29–43 MB** |
| 10 000 | 1050 / 837 MB | 964 / 875 MB | **+38 MB** |

At small vault sizes the vector index is invisible (40 KB at 10 notes). It becomes significant at 1 000+ notes: ~30–45 MB extra. At 10 000 notes it adds ~40 MB on top of the base note cache — consistent with the theoretical 1 024 dims × 4 bytes × N notes.

## Raw numbers

| Vault | Initial push 1c / 4c | 1 note 1c / 4c | 10% batch 1c / 4c | Peak RSS 1c / 4c |
|------:|---------------------:|---------------:|------------------:|-----------------:|
| 10 | 40 / 8 ms | 4 / 2 ms | 4 / 2 ms | 68 / 69 MB |
| 100 | 113 / 81 ms | 6 / 8 ms | 13 / 10 ms | 83 / 80 MB |
| 1 000 | 1.2 / 0.5 s | 39 / 22 ms | 96 / 62 ms | 182 / 149 MB |
| 10 000 | 13.8 / 6.8 s | 382 / 224 ms | 2.3 / 0.7 s | 1186 / 902 MB |

Measured 2026-06-11 on a 12-core ARM64 dev machine restricted with `taskset`, SQLite in WAL mode, single server process. Your absolute numbers will differ; the shape of the curves will not.

## Why page rendering is this fast

The numbers above are for write operations. Page reads are a different story: p50 response time is **3–6 ms** at any vault size, even under 10 concurrent users.

This is not an accident; it is an architectural choice. trip2g sits somewhere between a static site generator and a traditional CMS:

- **Static generator** (Gatsby, Hugo, Eleventy): builds HTML files to disk on deploy. Reads are instant because they're just file serving. But every content change requires a full rebuild — minutes for large sites, no live editing.
- **Traditional CMS** (WordPress, Ghost): renders every page on request — queries the database, fetches relations, runs templates. Flexible and live, but every reader hits the database.
- **trip2g**: renders HTML *on write* (during `pushNotes`) and stores the result in memory. Reads serve that pre-rendered HTML directly from a hash map — no database, no template engine on the hot path. The cost of rendering is paid once per content change, shared across all readers.

Keeping thousands of rendered pages in RAM turns out to be surprisingly cheap. A synthetic 500-byte note renders to ~800 bytes of HTML; 10 000 notes is roughly 8 MB of content. The rest of the ~1 GB RSS at that scale is Go runtime, SQLite WAL buffers, and search indexes — not the page cache. A $5/month VPS with 1 GB of RAM comfortably serves vaults of several thousand notes without touching the disk on reads.

There is still room to go further — the current architecture renders notes individually and assembles the final page (layout, injections) on each request. Pre-rendering complete pages would push latency even lower. But even without that, trip2g is already orders of magnitude faster than traditional database-backed CMS platforms on equivalent hardware.

### Measured on real content

We loaded this very documentation — 35 real pages (the `/en/user` set: custom HTML inside the default template) — onto a Hetzner **cpx32** (4 vCPU AMD EPYC-Genoa, 8 GB RAM) and hit them with [vegeta](https://github.com/tsenart/vegeta) **from a separate machine** (so the load generator doesn't steal the server's CPU), round-robin across all pages (~52 KB each), in **production mode** with a warm cache:

| Load | Success | p50 | p95 | p99 |
|------:|:-------:|----:|----:|----:|
| 1 000 req/s | 100 % | 2.7 ms | 3.9 ms | 10 ms |
| 2 000 req/s | 100 % | 1.4 ms | 3.3 ms | 12 ms |
| 3 000 req/s | 100 % | 1.8 ms | 6.2 ms | 16 ms |
| 4 000 req/s | 100 % | 2.6 ms | 13 ms | 32 ms |
| 5 000 req/s | 100 % | 9.1 ms | 74 ms | 169 ms |
| 6 000 req/s | 100 % | 241 ms | 645 ms | 898 ms |

Every request succeeds (100 % `200`) right through 6 000 req/s; the limit is latency, not error count. A single 4-core node comfortably serves **~4 000 real pages per second** with a p99 of ~30 ms; the knee is around 4 000–5 000 req/s, past which latency climbs into the hundreds of ms as the node becomes CPU-bound. That's the per-node ceiling on this hardware, and where a second node (or a read replica — see [[zerodowntime]] / [[litestream]]) earns its keep. Measured 2026-06-22.

> **How to benchmark this honestly.** Run with `DEV=false` (dev mode recomputes asset hashes per request, ~2× slower), run the load generator on a *separate* machine (sharing the box steals ~2× the CPU), and hit many real pages with a warm cache. Skipping any of these understated our first run by ~4×.

Two things dominate the per-request cost at the ceiling: **gzip compression** of each response, and a handful of **per-request database lookups** in the render path (access check, embedded Telegram links, HTML injections, view tracking). Both are cacheable: pages are static between writes, so there is clear headroom for a future optimization pass to push the ceiling several times higher. *(That optimization is planned, not yet done.)*

### Static files, and a bandwidth reality check

We also served the same 35 pages as plain files from nginx on the same node, to size the engine's overhead against raw static serving. The surprise: at ~52 KB per page, **both** nginx and trip2g top out in the same low-thousands req/s range — the wall there is **network bandwidth** (52 KB × 5 000 ≈ 2 Gbit/s) plus per-request gzip, not trip2g's render path. So on this hardware and page size, trip2g already runs close to raw static file serving; the engine's per-request cost (layout assembly + the bookkeeping DB lookups) would only become the dominant limit on smaller pages or a faster link. Placing trip2g cleanly on the spectrum from static files to a database-backed CMS like WordPress needs that cleaner setup (smaller pages / simpler hardware) — noted as future work.

## How small a box can run it

We deployed trip2g on a range of cheap VMs and load-tested each **from a separate machine** (so the generator never steals the server's CPU), in production mode (`DEV=false`):

| Box | vCPU / RAM | ~Price | Served at 100 % | Restart under load |
|---|---|---|---|---|
| Hetzner cpx32 | 4 / 8 GB | ~€10/mo | ~4 000 rps (real 52 KB doc pages) | — |
| Hetzner cpx22 | 2 / 4 GB | ~€6/mo | ≥4 000 rps (its `/` landing, ~43 KB — not saturated) | **0 dropped** |
| DigitalOcean basic | 1 / 512 MB | **$4/mo** | ~500 rps* | **0 dropped** |

\*Its single core holds **~500 rps** of the `/` page (~43 KB) at 100 % with p99 < 50 ms; the knee is ~600–700 rps and past ~900 latency falls apart. Measured with the generator in the same US-East area (~23 ms away) — a first run with the generator in Germany showed a misleading "1 000 rps", but that 86 ms transatlantic link just throttled the generator so the droplet never got pushed. Prices are provider list rates.

*Measurement note: these box-comparison numbers predate the 2026-06-29 page cache, so they show throughput without it. Within the table, the Hetzner rows were measured after the earlier write-path optimizations and the DigitalOcean row before them. For page-cache throughput on comparable hardware, see the section below.*

Two findings matter more than the raw rates.

**A deploy drops nothing.** Restarting the service under live traffic — `systemctl restart` while requests were firing — dropped **zero** connections on both the 2-core Hetzner node and the $4 droplet (20 000 / 20 000 and 12 000 / 12 000 requests answered `200`, no refusals). The listening socket stays open across the restart via **systemd socket activation**, so in-flight requests wait in the kernel queue for a few hundred milliseconds instead of being refused. One server, no load balancer, no downtime — see [[zerodowntime]].

**No object storage needed.** With the local file-storage backend (`--storage-backend=local`), trip2g keeps assets and backups on the server's own disk — no MinIO, no S3. It booted on the $4 droplet using ~57 MB of RAM. The cheapest VM you can rent is enough to run a real site.

## What this means for you

- **Vaults up to ~1 000 notes**: everything is instant — syncs are tens of milliseconds.
- **~10 000 notes on a small server**: everyday syncs are still sub-second, but the *first* push of the whole vault takes ~14 s on one core. During that window the database is locked for other writers; on very large vaults prefer an initial import during quiet hours.
- **More cores help rendering, not locking**: a 4-core server halves big-push times. The lock is held for the whole push either way, so many *concurrent* writers gain less than the raw numbers suggest.

## Anonymous page cache: 8 500 req/s on a €6.49/month box

The note-cache architecture above pre-renders HTML and holds it in memory, but per-request gzip and a handful of database lookups remain on the hot path. Both costs are now also eliminated for anonymous readers by a second layer: the **anonymous page cache**, which stores the final pre-gzipped HTML keyed by (path, host, note version, config, language). A cache hit copies bytes out of memory, with no re-render and no gzip.

The effect: a default-template docs page previously cost ~28 ms/request (dominated by a synchronous per-request vector-search "similar notes" pass using cosine similarity over all note embeddings, ~94% of CPU) while a Jet landing page cost ~0.9 ms/request. After the cache warmed, both pages served at the **same ~9 000 req/s**. The cache equalizes cheap and expensive pages.

**Benchmark: 2026-06-29, Hetzner shared-vCPU VMs**

| Box | vCPU / RAM | Price |
|---|---|---|
| cx23 (server under test) | 2 / 3.8 GB | **€6.49/month** |
| cx33 (load generator) | 4 / 7.7 GB | €8.99/month |

Attacked from the cx33 over the network (separate machines, cross-network).

**12-second burst (max rate):** ~9 000 req/s, server CPU ~78%, steal 0%.

**10-minute sustained run (max rate):**

| Metric | Value |
|---|---|
| Total requests | 5 134 192 |
| Throughput | **8 557 req/s** |
| vs. burst | −5% |
| p50 | 25.8 ms |
| p99 | 86 ms |
| Max | 323 ms |
| Errors | 0 |
| CPU steal | **0% throughout** |

Five million requests, ten minutes, zero errors.

The chart below shows how latency changes as the offered request rate climbs toward and past saturation.

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart_ramp" },
  "config": {
    "title": { "text": "Latency vs request rate, cx23 (less is better)" },
    "tooltip": { "trigger": "axis" },
    "legend": {},
    "xAxis": { "type": "category", "name": "req/s" },
    "yAxis": { "type": "log", "name": "ms" },
    "series": [
      { "type": "line", "name": "p50", "encode": { "x": "rate", "y": "p50" } },
      { "type": "line", "name": "p95", "encode": { "x": "rate", "y": "p95" } },
      { "type": "line", "name": "p99", "encode": { "x": "rate", "y": "p99" } }
    ]
  }
}
```

p50 stays sub-millisecond up to ~6 000 req/s. The knee is at ~7 000–8 000 req/s, where p50 jumps from 1.2 ms to 146 ms. The box saturates near 9 000 req/s. Success rate was 100% throughout: the server degrades on latency under overload rather than refusing requests.

**Cost framing:** 8 557 req/s ≈ 739 million requests per day on a €6.49/month box — roughly **€0.0003 per million requests**, or about €0.29 per billion. For any realistic traffic volume, the serving cost per request is negligible.

> **Pre-cache comparison (12-core dev box, single page):** the default-template docs page served **421 → 8 504 req/s** after the cache warmed — a **20× improvement**. CPU on cache hits dropped from ~95% to ~0%.

### Three measurement traps

Two alternative setups produce misleadingly low numbers:

**Loopback (same-box) attack.** Running vegeta on the same cx23 measured only ~5 400 req/s — the load generator competed with the server for the same two cores. The server CPU was not at 100%; what we measured was the attacker's ceiling, not the server's. The cross-network number is the real ceiling.

**Weak attacker, strong server.** Running trip2g on the stronger cx33 and attacking from the cx23 showed only ~44–45% CPU on the cx33. The cx23 could not generate enough load to find the cx33's real limit. That number is attacker-limited, not a server measurement.

**CPU steal as the throttle probe.** Shared vCPUs can be fair-use-throttled by the provider. The signal is the `steal` metric in CPU stats. During the entire 10-minute run, steal stayed at 0%: no throttling. For workloads that sustain maximum load 24/7, a dedicated-vCPU instance avoids this risk. For typical bursty traffic, shared is enough.

### What the cache covers

The anonymous page cache covers unauthenticated readers only. Logged-in users bypass it. The reason: an authenticated viewer could be an admin (who sees edit controls), a subscriber (who is paywall-exempt), or have other per-user state that differs from the anonymous view; serving them a cached anonymous page risks showing the wrong thing. There is one nuance: on the default template, the admin/anonymous difference (login button, edit badge) is rendered by a client-side script, so the server sends the same HTML to both. The cache could safely extend to logged-in users on default-template pages, but not on custom layouts or paywalled notes. The current design takes the simpler path: any authenticated request bypasses the cache. The underlying synchronous vector-search "similar notes" computation (~28 ms) remains on the read path for cache misses and authenticated requests. The planned next step is precomputing "similar notes" once per note reload, moving the O(N) cosine scan off the read path entirely. The benchmark found this; [[en/thoughts/page-cache-and-the-surprise-bottleneck|The page cache and the surprise bottleneck]] tells the full story.
