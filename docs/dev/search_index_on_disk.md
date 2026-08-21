# The full-text index on disk (SEARCH_INDEX_PATH)

**TL;DR:** `bleve.NewMemOnly` costs about 35x the size of the text it indexes.
Setting `SEARCH_INDEX_PATH` puts the index on disk instead, which turns 350 MB of
heap into 4 MB and turns an 8 second rebuild at every boot into a 4 millisecond
reopen. It is **off by default**: an empty path keeps the historical in-memory
index, byte for byte.

## The measurement that prompted this

A pool instance with 1017 notes was being OOM-killed on restart. A heap profile
showed 98.7% of live objects under `noteloader.(*Loader).Load`, and two thirds of
that under bleve. Measured on the production box, 2026-08-21, same corpus, same
mapping, one process:

| | `NewMemOnly` (upsidedown + gtreap) | on-disk scorch |
|---|---|---|
| heap | **+350 MB** | **+4 MB** |
| first build | 12.6 s | 7.9 s |
| one query | 82 µs | 239 µs |
| files on disk | none | ~45 MB after merge |
| reopening an existing index | nothing to reopen | **4 ms** |

The corpus is 10.3 MB of markdown. The in-memory index is the old upsidedown
format over a gtreap: every term posting is a separate key in an in-memory tree,
each field is indexed twice (ru and en analyzers), and `IncludeLocations` pulls
term vectors along. That is where the 35x comes from.

## Incremental updates were the thing to check, and they are fine

Editing notes is the normal case, so the second measurement was 200 single-note
updates against a warm index:

| | on-disk scorch | in-memory |
|---|---|---|
| 200 edits | 1.85 s | 1.70 s |
| p50 per edit | 6.0 ms | 4.0 ms |
| p95 | 33 ms | 31 ms |
| heap growth | +9 MB | none |

An edit costs what analysis costs, and analysis is the same in both. What differs
is disk churn: right after the 200 edits the directory held 105 MB in 220 files,
and 30 seconds later the background merger had collapsed it to 44 MB in 3 files.
Budget twice the index size as a transient peak, not as steady state.

## Durability

`Index()` returns once the document is in an in-memory segment; scorch persists
asynchronously. So a hard kill can in principle lose the tail. Measured: a
process that indexed all 1017 notes and then sent itself `SIGKILL` without
closing left an index that reopened cleanly in 11 ms with all 1017 documents
(2 ms after a clean `Close`).

Anything genuinely lost self-heals on the next load: the note's stored hash will
not match, so it is re-indexed.

## How it stays honest after a restart

Incremental indexing is driven by `contentHashes`, a map that lives only in
memory. A persisted index outlives it, which breaks two things unless handled:

- every note would look unknown and be re-indexed, wasting the point of the exercise;
- **deleted notes would haunt the index forever**, because the deletion pass walks
  `contentHashes`, and on a fresh process it starts empty.

So each document stores its content hash as a stored, never-indexed field, and
`adoptPersistedIndex` reads them back on the first load after opening a
persisted index: notes whose hash matches are adopted as already-indexed,
documents belonging to no current note are deleted. A full hash comparison over
this corpus takes 22 ms.

## Two things it deliberately does not do

**It never deletes an index it failed to open.** The tempting reading of "cannot
open" is "corrupt, rebuild it", and it is wrong here: a zero-downtime handoff runs
the old and the new instance at once, and the old one holds the lock. The loader
logs the failure and falls back to the in-memory index for that process. A truly
corrupt index is a human's call: delete the directory and restart.

**It does not share one directory between loaders.** The index lives at
`<path>/<loader>/<schema>`, e.g. `/data/search/live/v1`, so the `live` and
`latest` loaders never collide. Bump `searchIndexSchemaVersion` when the mapping
changes; directories from other versions are deleted on startup, which is what
makes a mapping change safe.

## Turning it on

```
SEARCH_INDEX_PATH=/data/search
```

In the pool this one value is enough for every instance, because `/data` is
already a per-slot bind mount. Empty or unset keeps the in-memory index.

Sizing: budget roughly 4-5x the size of the note text on disk, doubled for
transient merge headroom.
