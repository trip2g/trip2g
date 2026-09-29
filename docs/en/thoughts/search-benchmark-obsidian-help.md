---
title: "trip2g search against qmd and obsidian-hybrid-search"
free: true
lang_redirect: "[[ru/thoughts/search-benchmark-obsidian-help]]"
---

*What we did: built a bilingual set of 236 queries against Obsidian's help docs, ran trip2g, qmd and obsidian-hybrid-search on it and scored everything with one script, without trusting the numbers in other people's READMEs. We found two reasons trip2g was losing in Russian. We added alias search: in its own query category it lifted nDCG@5 from 0.81 to 0.98. The biggest gain came from fields an agent passes over MCP: plus 0.10 on the Russian base set and about 0.05 on the others. Read it if you run search over notes and want to see how it looks next to its neighbours.*

After [[en/thoughts/rag-under-the-hood|How others search]] one question was left that reading code can't answer: which of these searches is actually better? The obsidian-hybrid-search README compares itself with qmd: 0.733 against 0.659 nDCG@5. But those are the author's numbers, on the author's machine, and the qmd run results aren't in the repository. We decided to measure it ourselves.

## The set

The obsidian-hybrid-search author built a good test set: 58 hand-written queries against Obsidian's help docs, with correct and partly correct answers. The help docs are exactly our case: ordinary notes with frontmatter, wikilinks and aliases.

The help docs repository also holds translations into thirty-odd languages, Russian among them: the same 176 pages. An English page and its Russian counterpart share a `permalink`, so the correct answers carry over to Russian without manual labelling. We translated the queries the way a person would type them, not word for word.

Another 120 queries were written by a separate agent that went through the help docs and collected the cases where search usually trips:

- the query is exactly a note's alias;
- typos: "backlnks", «обратные ссылкы»;
- Russian morphology: «как синхронизировать» ("how to sync") against a page called «Синхронизация» ("Sync");
- paraphrases with no words in common with the page;
- a question in one language with the answer in the other;
- an answer buried deep inside a long page;
- syntax: `==highlight==`, `$$`, `[[note#^block]]`.

It checked every answer against the page text. That gives four sets: English and Russian base, 58 queries each, and English and Russian tricky, 57 and 63.

## How we measured

- **trip2g**: our hybrid search. bleve full-text search with Russian and English stemming, vector search on bge-m3, fusion with RRF.
- **obsidian-hybrid-search 0.15.2**: SQLite, FTS5, multilingual-e5-small vectors, alias and trigram matching.
- **qmd 2.8.3**: with its own query expansion and reranker. On our server without a GPU a single query took three to seven minutes, so we ran qmd on a MacBook with an M3 Max and Metal. Two variants: defaults, and Qwen3-Embedding instead of the English embeddinggemma. That is the only setting qmd needs for Russian.

Every run was scored by one script using obsidian-hybrid-search's own formulas: nDCG with 1 for a correct answer and 0.5 for a partly correct one.

## Results

nDCG@5, higher is better:

| Set | obsidian-hybrid-search | qmd | qmd + Qwen3-Embedding | trip2g | trip2g + aliases | trip2g + agent fields |
|---|---|---|---|---|---|---|
| English base | 0.726 | 0.717 | 0.731 | 0.742 | 0.742 | **0.798** |
| English tricky | 0.722 | 0.733 | 0.764 | 0.767 | 0.785 | **0.815** |
| Russian base | 0.671 | 0.608 | 0.668 | 0.603 | 0.603 | **0.702** |
| Russian tricky | 0.684 | 0.670 | 0.771 | 0.815 | 0.815 | **0.866** |

## What we learned

**The README numbers didn't match ours.** obsidian-hybrid-search reproduced itself almost exactly: 0.726 against the claimed 0.733. But qmd came out noticeably better than in that comparison: 0.717 instead of 0.659. A gap of 0.07 shrank to 0.01. We don't know why: a different qmd version, a different machine or a different mode. Which is exactly why comparing by other people's tables is a bad idea.

**On the Russian base set, trip2g lost to everyone.** There are two reasons, both visible in individual queries.

1. **Full-text search needs every word of the query to appear in the text.** The query «свойства заметки во frontmatter» ("note properties in frontmatter") finds nothing in full-text search: the Russian page doesn't have those four words together in those forms. Only vector search is left. obsidian-hybrid-search joins words with OR and matches them by prefix, so it doesn't have this problem.
2. **Long pages win on their best chunk.** In vector search a note is scored by its best chunk. The Obsidian CLI page has 163 chunks, and at least one of them lands close to almost any query. That page and the Obsidian URI reference kept coming out on top for queries that had nothing to do with them.

**Aliases helped where they should.** trip2g didn't read the `aliases` frontmatter field at all. We added a separate lane for an exact match of the query against a note's title or alias, as obsidian-hybrid-search has, with weight 2 in RRF. In the "query equals an alias" category the score rose from 0.813 to 0.980; over the whole English tricky set, from 0.767 to 0.785. Behind that are two queries that were not found without aliases. Nothing changed on the other sets, neither up nor down. There is no gain in Russian because the Russian aliases in the help docs are the English page titles, and trip2g already found those queries.

> [!tip] The main finding: hand query expansion to the agent
> qmd rewrites the query into three variants with its own 1.7-billion-parameter model, on the server, for every query. But MCP search always has an agent on the other side: Claude, GPT or another model many times larger. It already reads the question and knows the context and the language. Ask it to send those three variants along with the query and two problems are solved at once: trip2g doesn't have to carry a generative model and run it on every query, and the variants come from a far stronger model. On our set this gave the biggest gain of anything we tried.

**The agent fields gave the biggest gain.** When an agent connected over MCP passes three optional fields with the query (a short query of two or three words, a rephrasing, and a sentence that could stand in the answer), trip2g comes first on all four sets. On the Russian base set the gain is ten points and clearly above noise; on the other sets it is about five points, and on English tricky it is within noise. The details, including how we chose the field names, are in [[en/thoughts/query-expansion-by-the-agent|Let the agent expand the query]].

**qmd sags in Russian without tuning and recovers with one setting.** By default qmd uses an English embedding model and scores 0.608 on the Russian base set. With Qwen3-Embedding it scores 0.668, and 0.771 instead of 0.670 on the Russian tricky set. It's one environment variable, but afterwards the whole index has to be re-embedded.

## What we found along the way

- **A bug in trip2g.** If the frontmatter has an empty `description:` or `description: null`, the server cannot build the notes, and the whole sync batch of a hundred notes fails. The Obsidian help docs have two such pages per language. For the benchmark we stripped those lines; the bug itself will be fixed separately.
- **obsidian-hybrid-search returns paths in Unicode NFD form**: `й` and `ё` are split into a letter and a mark. Comparing paths byte for byte, as we did in the first run, drops its Russian score from 0.671 to 0.447.
- **obsidian-hybrid-search misses Russian inline tags**: `#note` is found, `#заметка` is not. Frontmatter tags work.

## Caveats

- **The embedding models differ.** trip2g uses bge-m3 with 568 million parameters, obsidian-hybrid-search uses multilingual-e5-small, qmd uses the 300-million-parameter embeddinggemma. We compared the systems as they are installed, not the retrieval models as such.
- **The sets are small.** At 57–63 queries a set, three points of nDCG is a couple of queries. We would not call a 0.01–0.02 difference between systems significant. A bootstrap clearly confirms only the agent-field gain on the Russian base set.
- **Models wrote the tricky queries and the agent fields.** The tricky queries were checked against the help text, but an independent review of their labels is not finished. The agent fields were written by a separate agent given only the questions and the field descriptions; with a weak agent the gain will be smaller.

## How to repeat it

The scripts, the query sets and the raw run results are in the repository: `scripts/obsidian-help-bench/`, `testdata/eval/obsidian-help/`, `docs/superpowers/eval-runs/obsidian-help/`. The Obsidian help docs themselves aren't licensed for redistribution, so the script downloads them at the same commit the answers were labelled against.
