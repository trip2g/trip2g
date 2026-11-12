# EnterpriseRAG-Bench kit

Runs trip2g search against [EnterpriseRAG-Bench](https://github.com/onyx-dot-app/EnterpriseRAG-Bench)
(Onyx, MIT): a synthetic company's 511 962 documents and 500 questions with gold documents and
answers. Results and what they changed: `docs/dev/retrieval_eval.md`,
`docs/{en,ru}/thoughts/search-benchmark-enterprise-rag.md`. Memory and timing: `docs/dev/sizing.md`.

Run by hand, not in CI: one pass needs a 16 GB machine, about two hours of embedding and paid
API calls. The corpus and the questions are not committed.

## What a run needs

- A work directory (`$ERB_WORK`) with `all_documents.zip` (the benchmark's GitHub release),
  `questions.jsonl` (its repo root), a dev build `./trip2g` and `.key` holding an OpenRouter key.
- Docker for the S3 container, and for `numpy` if the host has none.
- An embedder. `up.sh` uses `baai/bge-m3` through OpenRouter: $0.01 per million tokens, about
  $1.50 for the 50k sample. Its vectors are unit length, which search relies on.

## Run

```bash
cd "$ERB_WORK"
go build -tags dev -o ./trip2g <repo>/cmd/server
K=<repo>/scripts/enterprise-rag-bench

# every gold document plus 50 000 random ones
python3 $K/prepare.py all_documents.zip questions.jsonl 50000 corpus50k.jsonl 42

$K/up.sh 21201                     # S3, the server, a session token, an API key
$K/guard.sh 2.0 &                  # kills the server past $2 of key usage
$K/sampler.sh > samples.csv &      # RSS, CPU, database size every 15 s
python3 $K/push.py corpus50k.jsonl 21201 1000
# wait until `select count(*) from goqite` in bench.sqlite3 reaches 0

python3 $K/run_search.py base      # logs/run-base.jsonl
python3 $K/score.py logs/run-base.jsonl questions.jsonl
```

Compare two builds by keeping `bench.sqlite3`, swapping `./trip2g` and running
`run_search.py` under another label: the vectors stay, only search changes.

| Script | What it does |
|---|---|
| `lanes.py` | The same questions against the stored vectors and SQLite FTS5, alone and fused: tells which lane loses a document. Needs `numpy`. |
| `diag.py` | Chunk sizes per source, which lane finds each gold document, a re-chunking experiment. |
| `answers.py` | Answers a sample with a small model and scores them with the benchmark's judge prompts. `chunks` mode hands the model the closest chunks instead of document heads. |
| `http.sh` | Stepwise load on the note page, `/livez` and search with `hey`. |
| `metrics_log.sh` | A full Prometheus snapshot every minute. |

## Gotchas

- **Ports.** Nomad hands out 20000–32000 on dh0; a clash kills the instance at start with no retry.
- **Five documents hold a NUL byte**, and one such note fails its whole push batch. Strip them
  after `prepare.py`; the published runs did.
- **Kill the server with `kill -9`**, and restart `guard.sh` after every restart: it exits with the server.
- **`-storage-db-limit=0`** is required; the 100 MB default stops the push near 14 000 notes.
- **The sample reads easier than the leaderboard.** Most near-duplicates fall out of a 10% sample.
- **The published answer scores used `head` mode and one judge model**, not the benchmark's three-judge vote.
