# Obsidian Help retrieval benchmark

Compares trip2g search with [obsidian-hybrid-search](https://github.com/flowing-abyss/obsidian-hybrid-search)
(ohs) and [qmd](https://github.com/tobi/qmd) on Obsidian's own help docs, in English and in Russian,
with one scorer for all systems. Background and results: `docs/{en,ru}/thoughts/search-benchmark-obsidian-help.md`.

## Data (`testdata/eval/obsidian-help/`)

| File | What |
|---|---|
| `golden-en.json` | 58 queries against the English docs, hand-written by the ohs author (MIT, © flowing-abyss) |
| `golden-ru.json` | the same 58 in Russian; judgements mapped to the Russian pages through their shared `permalink` |
| `golden-tricky-{en,ru}.json` | 57 + 63 harder queries: aliases, typos, Russian morphology, cross-lingual, deep sections, syntax |
| `agent-fields-{en,ru}.json` | `short_query` / `rephrased_query` / `expected_answer` for every query, written by a model that saw only the query and the field descriptions |

Relevance is graded: `relevant_paths` 1.0, `partial_paths` 0.5. The help texts themselves have no
licence and are not committed; `prepare.py` fetches them at the commit the judgements were made against.

## Run

Everything generated goes to `$OHB_WORK` (default `tmp/obsidian-help-bench`).

```bash
cd scripts/obsidian-help-bench
python3 prepare.py                                   # vault-en, vault-ru

# trip2g: a dev build, an S3 endpoint and a bge-m3 embedding server (see trip2g_up.sh)
go build -tags dev -o "$OHB_WORK/trip2g" ../../cmd/server
./trip2g_up.sh en 21101 "$OHB_WORK/trip2g"
./trip2g_up.sh ru 21103 "$OHB_WORK/trip2g"
python3 run_trip2g.py en 21101 trip2g
python3 run_trip2g.py ru 21103 trip2g

# obsidian-hybrid-search
(cd "$OHB_WORK" && npm i obsidian-hybrid-search@0.15.2)
python3 run_ohs.py en && python3 run_ohs.py ru

# qmd (needs a GPU in practice: 3-7 min per query on a shared CPU, ~10 s on an M3 Max)
(cd "$OHB_WORK" && npm i @tobilu/qmd@2.8.3)
./qmd.sh

# agent-side query expansion on top of trip2g: a text-only instance plus the vector
# instance's database; the t2g-sim run must equal the plain trip2g run
./trip2g_up.sh en 21105 "$OHB_WORK/trip2g" '{}'
python3 agent_fields.py en 21105 "$OHB_WORK/t2g-en-21101/bench.sqlite3"

# score (same definitions as ohs's eval/metrics.ts)
python3 score.py ../../testdata/eval/obsidian-help/golden-en.json "$OHB_WORK"/runs/*-en.json
```

`probe_names.py` checks which names of the optional fields small models fill best (OpenRouter;
`OPENROUTER_KEY`). The committed runs behind the published numbers are in
`docs/superpowers/eval-runs/obsidian-help/`.

## Gotchas

- obsidian-hybrid-search returns NFD paths (`й` decomposed); `score.py` compares in NFC.
- A trip2g benchmark instance may not exit on SIGTERM; stop it with `kill -9` and check with
  `ps` which binary holds the port before trusting a "restarted" run.
- `t2g-sim` in the committed runs was built before the exact-name lane: it matches `trip2g-*`
  metric for metric (5 of 121 Russian lists differ only in how exact RRF ties are broken — Go
  sorts ties by permalink, the script by path). A text-only instance built from a binary with
  the name lane folds that lane into its "text" list, so rebuild it from the commit before
  `feat(search): exact title and alias lane` to reproduce `t2g-sim` exactly.
- qmd's HTTP `/query` endpoint takes pre-typed searches only; `run_qmd.py` calls the MCP `query`
  tool with plain text so qmd runs its own expansion, as `qmd query` does.
