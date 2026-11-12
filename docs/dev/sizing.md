# Sizing: what a large vault costs

Measured on one run of 50 722 notes (EnterpriseRAG-Bench sample, 2026-11-04/05): 2.6 chunks per
note on average, bge-m3 vectors (1024 dimensions), a 12-thread Xeon E-2276G, commit `9f805d36`
plus the search change of #500. Kit: `scripts/enterprise-rag-bench/`. One corpus and one machine:
treat the numbers as an order of magnitude, and the rows marked "estimate" as untested.

## Memory

| Notes | Full-text index | Peak RSS | Steady RSS | Status |
|---|---|---|---|---|
| 1 017 | on disk | about 290 MB during a reload | 145 MB of live heap | measured on the pool, see `search_index_on_disk.md` |
| 13 000 | in memory | 8.1 GB | — | measured, few vectors loaded yet |
| 50 722 | in memory | 23.6 GB | about 21 GB | measured, half the vectors loaded, under `GOMEMLIMIT=24GiB` |
| 50 722 | on disk | 6.5–9.3 GB | 4.2–5.4 GB | measured, all vectors loaded |
| 512 000 | in memory | about 240 GB | — | estimate, linear from 50k |
| 512 000 | on disk | about 60–90 GB | — | estimate, linear from 50k |

- **Set `SEARCH_INDEX_PATH` on anything past a few thousand notes.** In memory the index was
  10.5 GB of a 12.3 GB live heap, about 465 KB per note against 5 KB of text.
- **With the index on disk, the rest is notes and vectors.** 118 449 chunk vectors are about
  0.5 GB as float32; the note graph and rendered HTML take most of the remainder. The split
  was not profiled.
- **Peaks come from reloads**, which hold the old and the new note graph at once. The highest
  on-disk figure, 9.3 GB, was a first index build.

## Disk

| What | 50 722 notes |
|---|---|
| SQLite database | 1.3 GB, of it chunks with vectors 0.8 GB |
| Full-text index on disk | 1.5 GB, twice that briefly while segments merge |

The default `-storage-db-limit` is 100 MB: the push stopped near 14 000 notes until it was lifted.

## Time

| What | 50 722 notes |
|---|---|
| Start, index in memory | rebuilt every start: 204 s already at 13 000 notes |
| Start, index on disk | 463–497 s the first time, 33–45 s after |
| Push of 1 000 notes | 19 s at 15 000 notes in the vault, 34 s at 50 000 |
| Embedding | 3 notes/s at the default queue settings, 9 with `GLOBAL_QUEUE_LIMIT=20` and `GLOBAL_QUEUE_POLL_INTERVAL=500ms`; the remote embedder then answered `overloaded` 140 times |
| Search, one client | p50 1.2 s, most of it the remote embedding of the query |
| Search, 20 clients | 15 requests/s, p50 1.2 s |
| Note page | 1.9 s, 2.6 requests/s at most: similar notes, issue #501 |

## Not measured

- The full 512 000 documents, or any vault between 50 000 and that.
- Text-only search without vectors at this size.
- A local embedder: a CPU TEI gave about 1 text/s, which would take weeks for this sample.
- Anonymous page views served from the page cache.
