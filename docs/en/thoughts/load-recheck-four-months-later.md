---
title: "Re-measured four months later: what holds and what drifted"
description: "In June trip2g served 4 000 pages a second. In October we measured it again on 10 000 notes with a profiler on every path. The cache holds the same numbers, search, MCP and the template had grown new bottlenecks, and the published sync number turned out to be measured on synthetic notes."
free: true
lang: en
lang_redirect: "[[ru/thoughts/load-recheck-four-months-later]]"
---

*What this is: in June we published [[en/user/perfomance|performance]] numbers. Over four months trip2g gained note queries, federation, MCP on the official SDK and a dozen small things on the render path. We measured everything again and turned the profiler on for every endpoint. This is what held, what broke and what is being fixed.*

A published number stays true for exactly the path it measured. Everything around that path has moved since.

## How we measured

Three independent runs, each in its own 4 vCPU cloud sandbox, in production mode (`DEV=false`):

1. Stress per request type on the project's own documentation (908 notes): pages, static files, search, GraphQL, MCP, sitemap, reads during writes.
2. Note queries and every MCP tool the way an agent calls them, plus 2 000 synthetic notes with varied frontmatter.
3. Page serving and sync on 10 000 notes, A/B: the June commit `8552a649` against current main on the same machine.

A sandbox is one machine, and the load generator sits next to the server. That is trap two from the June write-up: absolute numbers come out lower than the real ones. So in the third run the server and the generator are pinned to different cores, and the comparison is not against the publication but June against October under identical conditions.

## What holds

**The anonymous page cache.** About 8 500 requests per second at 54 % server CPU — the generator hits its ceiling, not trip2g. The same figure as in June.

**Small pages and static files.** A small page does about 47 000 rps at p99 19 ms with 200 concurrent clients, and that too is the generator's ceiling. Without an `Accept-Encoding: gzip` header the same client decompressed every response itself and stalled at 7 500 rps — a fourth measurement trap.

**MCP tools** answer fast and correctly, and per-role auth works. On a small vault a sync does not block reads: throughput drops 25–30 %, no errors.

## What turned out to be something else

**Sync on a large vault.** One changed note synced into a vault of 10 000 realistic notes in about 3–5 seconds, and memory after load was 5–7 GB. The June write-up says tens of milliseconds and 1 GB. The first version of this text called that a regression. It is not one.

We measured the June commit and October's main the same way. On the June write-up's method — synthetic notes of 500 bytes — October's code reproduces the published numbers. On realistic notes (~9 KB on average, with links, embeds, tables) the June code is exactly as slow: 4.7 seconds per note against 4.8 in October, 7.4 GB against 7.6. The published number was honest for synthetic notes and said nothing about a real vault.

The slowness comes from the reload model: every sync renders every note again. The note cache saves parsing, not rendering, so the cost of one note grows with the size of the whole vault. This is not a defect a one-line fix removes but an architectural property, and the performance page now says so plainly.

**One real regression turned up.** Since August a vault with at least one note carrying a `route:` to a custom domain re-rendered every note once more per such host on each reload. trip2g's own documentation has such a note — the multi-domain demo. Rendering for hosts took 57 % of reload CPU, and its HTML 2.7 times the memory of the main HTML. The fix: only notes with their own route on a host are pre-rendered for it; the rest render on the first request to that host, once per snapshot.

**Reads during a sync.** In one run at 200 requests per second, while 100 notes synced, median read latency grew to 2–6 seconds against 2 ms on the June code. Part of the gap is the per-host rendering: the June run used documentation without the multi-domain demo. The rest is common to both versions: every reload clears the page cache, and readers all go to a full render at once, competing for CPU with the reload itself.

**A full render cost twice as much.** The main cause was visible in the profile at first glance: on every request the template walked all notes to find RSS feeds for `<link rel="alternate">`. That was 42 % of all allocations on the render path. The feed list only changes when notes load, and it is now computed there: 0.9 ms per render → 2 nanoseconds.

## Where time went to waste

The most useful output of these runs is not the numbers but the profiles. Almost every bottleneck turned out to be work done on each request although its result does not change between note loads.

**MCP `tools/list`.** A static 15 KB answer ran at about 250 rps on four cores. Every request built an MCP SDK server with the whole tool catalog; the SDK parsed every tool's JSON schema twice and then marshalled it again the slow way, and afterwards we parsed the finished answer twice more to make sure it carried no error. The fix: the catalog a caller sees is built into a server once and reused. 1.15 ms → 0.33 ms per request, twelve times fewer allocations, the wire answer unchanged to the byte.

**Sitemap.** 88 % of this endpoint's CPU went to gzipping the same bytes. The sitemap is already generated when notes load. The fix compresses it there too and serves the ready bytes per request.

**The log.** The feed note pointed at an `rss` layout the documentation vault did not have. Every request to the feed wrote a warning with the list of all layouts — 1.1 million lines and 560 MB in one run. The feed itself answered an empty 200. The fix: one warning at load, on the note itself, and no empty 200.

**Search.** 100–700 rps on four cores, with about 70 % of CPU in match highlighting. For each of 20 results bleve reloads the whole document from the index, and the note body is stored in the index twice. In the end more than half of the CPU goes to the garbage collector. bleve already returns match positions, so we now build highlighting from them and return it not as HTML with `<mark>` but as structure: text segments flagged "match". The frontend no longer digs `<mark>` out with a regular expression, note bodies left the index, and search got about twice as fast.

**Note queries.** Every call serialised all notes the reader may see to JSON again — 20 % of the request's CPU, with another 66 % in Jsonnet itself. The fix keeps the serialised notes until the next load, separately per access class.

## What was actually dangerous

One note query could take down the whole process. A Jsonnet program with tail recursion whose depth came from a visitor parameter slipped past the interpreter's stack limit, and Go died of a stack overflow — not a panic that can be recovered, a fatal error. The query has to be in the vault already, but in our setup agents write notes too. Ordinary recursion was caught by the limit and returned an error; this was not.

A heavy guest GraphQL request — five searches with the full HTML of every result — returned 1.5 MB and held 43 rps. The complexity limit did not see it: a search cost as much as any other field. The fix: a search costs 10, and a document inside a search result another 5.

## Status on October 20

Merged: MCP `tools/list`, the sitemap, the feed and the GraphQL search cost, the feed walk at load, our own search highlighting as segments. In a pull request: the guard against note query crashes, the notes JSON cache for note queries. In progress: lazy rendering for custom domains.

Once it all ships we will measure again.

## What follows from this

**A benchmark on synthetic data measures the benchmark.** The June sync number was true for 500-byte notes and said nothing about a vault of ten-kilobyte notes. We caught it only because we ran the same scenario on realistic data and on the old code — and saw the "regression" had always been there. Compare version with version under the same conditions, not a new number with an old publication.

**"Once, at load" is this engine's main trick.** trip2g is fast because it renders on write, not on read. Nearly every finding of this run is a place where that was forgotten: feeds, sitemap, the MCP catalog, the notes JSON for queries. Each one looked harmless on its own.

**A profile beats a benchmark.** A benchmark says what is slow. A profile says what is done for nothing — and most often that is one line that can move to load time.
