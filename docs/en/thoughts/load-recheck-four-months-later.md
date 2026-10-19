---
title: "Re-measured four months later: what holds and what drifted"
description: "In June trip2g served 4 000 pages a second. In October we measured it again on 10 000 notes with a profiler on every path. The cache holds the same numbers; sync, search and MCP had grown new bottlenecks."
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

## What drifted

**Sync on a large vault.** One changed note syncs into a 10 000-note vault in about 3.3 seconds. In June it was tens of milliseconds. The initial import takes about 104 seconds against 14. Memory after load is 5–6 GB against 1 GB. Importing 10 000 notes in one call runs into the 60-second handler timeout.

**Reads during a sync.** At 200 requests per second, while 100 notes sync, median read latency grows to 2–6 seconds. In June the same scenario stayed at 2 ms. Every note reload clears the page cache, and all readers go to a full render at once, competing for CPU with the reload itself.

**A full render costs twice as much.** The main cause was visible in the profile at first glance: on every request the template walks all notes to find RSS feeds for `<link rel="alternate">`. That is 42 % of all allocations on the render path. The feed list only changes when notes load, so computing it there once is enough. The fix moves that walk to load time.

## Where time went to waste

The most useful output of these runs is not the numbers but the profiles. Almost every bottleneck turned out to be work done on each request although its result does not change between note loads.

**MCP `tools/list`.** A static 15 KB answer ran at about 250 rps on four cores. Every request built an MCP SDK server with the whole tool catalog; the SDK parsed every tool's JSON schema twice and then marshalled it again the slow way, and afterwards we parsed the finished answer twice more to make sure it carried no error. The fix: the catalog a caller sees is built into a server once and reused. 1.15 ms → 0.33 ms per request, twelve times fewer allocations, the wire answer unchanged to the byte.

**Sitemap.** 88 % of this endpoint's CPU went to gzipping the same bytes. The sitemap is already generated when notes load. The fix compresses it there too and serves the ready bytes per request.

**The log.** The feed note pointed at an `rss` layout the documentation vault did not have. Every request to the feed wrote a warning with the list of all layouts — 1.1 million lines and 560 MB in one run. The feed itself answered an empty 200. The fix: one warning at load, on the note itself, and no empty 200.

**Search.** 100–700 rps on four cores, with about 70 % of CPU in match highlighting. For each of 20 results bleve reloads the whole document from the index, and the note body is stored in the index twice. In the end more than half of the CPU goes to the garbage collector. bleve already returns match positions, so highlighting can be built from them — and returned not as HTML with `<mark>` but as structure: text segments flagged "match". Then the frontend no longer has to dig `<mark>` out with a regular expression.

**Note queries.** Every call serialised all notes the reader may see to JSON again — 20 % of the request's CPU, with another 66 % in Jsonnet itself. The fix keeps the serialised notes until the next load, separately per access class.

## What was actually dangerous

One note query could take down the whole process. A Jsonnet program with tail recursion whose depth came from a visitor parameter slipped past the interpreter's stack limit, and Go died of a stack overflow — not a panic that can be recovered, a fatal error. The query has to be in the vault already, but in our setup agents write notes too. Ordinary recursion was caught by the limit and returned an error; this was not.

A heavy guest GraphQL request — five searches with the full HTML of every result — returned 1.5 MB and held 43 rps. The complexity limit did not see it: a search cost as much as any other field. The fix: a search costs 10, and a document inside a search result another 5.

## Status on October 19

- MCP `tools/list` — in a pull request.
- Sitemap, the feed and the GraphQL search cost — in a pull request.
- The note query crash, the feed walk, the notes JSON for note queries, our own search highlighting, the sync regression — in progress.

Once the fixes ship we will measure again and update the [[en/user/perfomance|performance page]].

## What follows from this

**Numbers age faster than code.** The June write-up did not lie; it described a path that has barely been touched since. The bottlenecks grew next to it: in sync, in the template, in new endpoints. Re-measuring belongs on a schedule, not after something breaks.

**"Once, at load" is this engine's main trick.** trip2g is fast because it renders on write, not on read. Nearly every finding of this run is a place where that was forgotten: feeds, sitemap, the MCP catalog, the notes JSON for queries. Each one looked harmless on its own.

**A profile beats a benchmark.** A benchmark says what is slow. A profile says what is done for nothing — and most often that is one line that can move to load time.
