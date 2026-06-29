---
title: "The database was re-reading its own queries"
free: true
lang_redirect: "[[ru/thoughts/the-database-was-re-reading-its-own-queries]]"
---

After the page cache shipped, we profiled again. Rendering had disappeared from the hot path. A third of the CPU was now in SQLite's SQL parser.

The server: a cx23, 2 vCPU, €6.49/month on Hetzner. The profile, taken under load, showed `_yy_reduce` and the query planner sitting where the cosine similarity scan used to be. The database was parsing the same SQL strings on every single request and throwing the compiled result away.

---

## Why the parser was running at all

trip2g uses `modernc.org/sqlite`, a pure-Go SQLite driver. The version in use called `sqlite3_prepare_v2` on every statement execution: parse the SQL, build a query plan, execute, discard the compiled statement. The generated query layer issued one-shot `QueryContext(sqlString, args)` calls, so nothing kept a compiled statement around between requests.

Parsing `SELECT … WHERE id = ?` is identical every time. Only the `?` changes. Redoing it per request is pure waste.

---

## The fix, and why it required two moves

The anonymous page cache, described in the previous essay, already handled one piece of this: moving the cache lookup to before the database work means a cache hit now touches zero SQL. The parse cost vanished from the most common path entirely.

For cache misses and logged-in readers, the fix is a prepared-statement cache: prepare each distinct SQL string once, hold the compiled statement, reuse it on every call. Straightforward in principle. The non-obvious part: a prepared statement only helps if the driver keeps the compiled form. The old driver version did not cache at all. Even reusing a Go `*sql.Stmt` caused a re-parse. Two things had to change together: upgrade the driver (to a version that keeps a per-connection compiled-statement cache) and add a read-pool wrapper that holds one long-lived prepared statement per query string. Either change alone does nothing. The driver version jump spanned 16 releases.

One change without the other is a common trap in layered optimizations. The profiler is the only reliable way to tell whether an optimization actually landed.

---

## The numbers

Both measured on cx23 (2 vCPU, €6.49/month) with vegeta driven from a separate box.

Anonymous readers, page-cache hit: throughput rose from roughly 9,000 to roughly 16,400 requests/second, about 1.8x. The SQL parser and planner disappeared from the profile. The bottleneck is now network and HTTP serving overhead.

Logged-in readers bypass the page cache, so every request runs the full database work. After the prepared-statement cache, `_yy_reduce` and the query planner are gone from their profile too. What remains is irreducible: `_sqlite3VdbeExec`, walking the b-tree to execute the queries. Before the fix, they were paying parse plus plan on top of that.

---

## How we measured

The parse cost does not appear in microbenchmarks. It shows up under real concurrency, so we profiled the actual running binary under load.

Isolating the logged-in path required a small trick. Logged-in readers bypass the page cache, which is exactly what we wanted to measure: the full database work without cache hits masking it. To drive authenticated load, we minted a short-lived hot-auth-token, exchanged it for a session cookie, and ran vegeta against that authenticated session. The profile shows the query path directly.

Validating the driver upgrade required more care. A version jump of 16 releases touches enough internal behavior that unit tests are insufficient. We ran the full end-to-end suite against a real multi-container stack, not just in isolation.

---

## A static site generator without the wait

Step back from the numbers. ~16,400 cached pages per second on a €6.49/month box is static-site-generator territory: the throughput you would expect from flat files behind nginx. A static site generator pays for that speed with a build step. Hugo, Jekyll, or Astro on a CI pipeline rebuilds the whole site on every edit, and a change is not live until minutes of CI/CD have run.

trip2g has no build step. Editing a note re-renders that one note and invalidates its cache entry; everything else stays warm. The change is live in a fraction of a second, and the site still serves at static-site-generator throughput. That is the trade this work bought: the serving speed of a generated static site, without the generation.

---

## What's left

Logged-in readers still carry two costs the anonymous path does not. Every response is re-gzipped on the fly; there is no page cache for them yet. And each page view fires a handful of bookkeeping queries to record the visit.

The next steps: extend the page cache to logged-in readers on the default template (the server sends identical HTML to both anonymous and logged-in users there; the per-user differences are drawn in the browser by a client-side script), and trim the per-view bookkeeping queries.

The profiler keeps pointing at the next thing to fix. That is, more or less, how this works.

---

## What it adds up to

At 16,400 requests/second on a €6.49/month shared box, anonymous cached pages serve at throughput you would normally associate with static files: pre-built HTML, nothing to compute per request. Classic SSGs (Hugo, Jekyll, Astro deployed to GitHub Pages) achieve this by rebuilding the entire site on every publish. One note edit triggers a full rebuild, sometimes minutes of CI before the change is visible.

trip2g is still a dynamic server. The cache makes it behave like a static server for reads. When a note changes, only that note's cache entry is invalidated. Everything else stays warm. The updated note is live in a fraction of a second.

Throughput and update latency usually trade off. SSGs push throughput high by accepting rebuild time. The two-part optimization described here inverts that: the serving speed is comparable to a static site, the update path does not go through a build pipeline, and a single note change does not disturb the rest of the site.
