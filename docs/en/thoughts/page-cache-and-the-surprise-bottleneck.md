---
title: "The page cache and the surprise bottleneck"
free: true
lang_redirect: "[[ru/thoughts/page-cache-and-the-surprise-bottleneck]]"
---

The profiler said the plain documentation page was 30× more expensive than the custom-designed landing. We expected the opposite.

Two pages under load: the Jet home (a custom layout with a mesh grid, brand design, hand-tuned markup) and a default-template docs page, the kind you get without any configuration. Common sense says the fancy one costs more to render. The profiler said the docs page ran at ~28 ms/request; the Jet home ran at ~0.9 ms/request. Thirty times slower. All of it in one place: a synchronous, per-request cosine similarity scan over every note embedding, labeled "similar notes."

---

## The expensive thing was invisible

Here is what was happening per request: trip2g loads the page, assembles the layout, and, before returning, iterates over all note vectors, computes cosine similarity for each one, and picks the top matches to show as "related reading." This runs on every request, for every reader, with no caching. At modest vault sizes (hundreds of notes) it is ~28 ms of pure compute on the read path, consuming ~94% of the CPU budget for that request.

The Jet home had none of this. Its template parsing was already cached; the page assembled fast. Visually more complex, computationally cheaper by more than an order of magnitude.

Lesson: profile, do not assume. The bottleneck is rarely where it looks.

---

## What the anonymous page cache does

Making the scan faster would help. Not running it on every request helps more. The anonymous page cache stores the final, already-gzipped HTML response keyed by (path, host, note version, config, language). On a cache hit, the server copies pre-built bytes out of memory. The cosine scan does not run, the template engine does not run, the gzip compressor does not run.

The key is the version tag: when a note is updated, its cached entry is invalidated. The scan runs once on the first request after a content change, pays the 28 ms cost, and stores the result. All subsequent readers (potentially thousands per second) pay nothing.

Measured on a 12-core dev box with a single page under continuous load: the docs page served **421 req/s before the cache** and **8 504 req/s after it warmed**, a 20× improvement. CPU on cache hits: ~0%.

The Jet home was already fast. After the cache: also ~9 000 req/s. The cache equalizes expensive and cheap pages at the same throughput ceiling. That ceiling turns out to be network and syscall overhead, not application logic.

**Why anonymous-only.** The cache skips logged-in users. An authenticated viewer could be an admin (who sees edit controls), a subscriber (who is paywall-exempt), or have other per-user state that differs from the anonymous view; serving them a cached anonymous page risks showing the wrong thing. There is one nuance: on the default template, the admin/anonymous difference (login button, edit badge) is rendered by a client-side script, so the server sends the same HTML to both. The cache could safely extend to logged-in users on default-template pages, but not on custom layouts or paywalled notes. The current design takes the simpler path: any authenticated request bypasses the cache.

---

## How we measured, and the three traps

The benchmark ran on 2026-06-29 on two real Hetzner shared-vCPU VMs:

- **cx23** (2 vCPU / 3.8 GB, €6.49/month): the server under test
- **cx33** (4 vCPU / 7.7 GB, €8.99/month): the load generator

Cross-network, separate machines. [vegeta](https://github.com/tsenart/vegeta) at unlimited rate for 12 seconds (burst), then for 10 minutes (sustained).

Result: cx23 served ~9 000 req/s in the burst and 8 557 req/s averaged over 10 minutes. Zero errors. CPU steal 0% throughout.

Three setups we also tried, each with a different trap:

**Trap 1: loopback (same-box) attack.** We ran vegeta on the same cx23 it was attacking. It measured ~5 400 req/s. The load generator competed with the server for the same two cores. The server CPU was not at 100%; the attacker was. What we measured was the attacker's ceiling, not the server's. The cross-network number is the real server ceiling.

**Trap 2: weak attacker, strong server.** We ran trip2g on the stronger cx33 and attacked from the weak cx23. The cx33 server showed only 43–45% CPU. The cx23 could not generate enough load to find the cx33's real limit. That number is attacker-limited, not a server measurement.

**Trap 3: CPU steal as the throttle probe.** Shared vCPUs are subject to fair-use throttling: the provider can reclaim CPU time when the physical host is contended. The signal is the `steal` metric in CPU stats. During the entire 10-minute run, steal stayed at 0%. No throttling. The measured 8 557 req/s is what the server actually delivered.

The 10-minute run matters here. A 12-second burst with 0% steal is insufficient: some providers allow short bursts through and throttle sustained load. The sustained run was the check, and it passed. The caveat remains: 24/7 maximum load on a shared vCPU could eventually trigger fair-use limits. For that, a dedicated-vCPU instance holds without risk.

---

## What comes next

The anonymous page cache hides the cosine scan from cache hits. It does not eliminate it. Two cases still pay the full cost:

- **Cache misses:** the first request after a content change.
- **Authenticated users:** logged-in readers bypass the cache.

The next optimization is structural: precompute "similar notes" once per note reload, store the result alongside the rendered HTML, and serve it from memory on every read. The O(N) cosine scan moves off the read path entirely, replaced by a single in-memory lookup. When vaults grow large enough that even the one-time scan becomes slow, an ANN index is the next step after that.

This benchmark told us where to look. Before measuring, we assumed the expensive case was the Jet layout; the profiler corrected that assumption. The correction pointed straight at the optimization that will eventually make the cache unnecessary for most of what it currently hides.
