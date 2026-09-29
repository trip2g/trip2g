---
title: "How others search: five knowledge-search systems"
free: true
lang_redirect: "[[ru/thoughts/rag-under-the-hood]]"
---

*What we did: took apart, from code and from a write-up, how search works in five other systems, and compared it with ours. For Hindsight we also checked on the live models what happens to Russian text. AnythingLLM is a popular self-hosted chat-with-your-documents app. qmd is Tobi Lütke's local search engine for markdown. Cerebras Knowledge is the company's internal knowledge base, answering 15,000 questions a day. Hindsight is memory for AI agents, built on a graph of facts. obsidian-hybrid-search is MCP search over an Obsidian vault, the closest to us in purpose. Each system gets its own section, with a summary table at the end. Read it if you are building search over notes and want to know which decisions everyone makes the same way and where there is a real choice.*

RAG means very different things. For some it is a single call to a vector database. For others it is a six-stage pipeline with two language models and heuristics tuned per data source. From the outside they look the same: ask a question, get an answer with links. The difference only shows up in the code, and in which queries the system handles badly.

We read the code of AnythingLLM, qmd, Hindsight and obsidian-hybrid-search as of September 2026, and the Cerebras engineering write-up from 16 July 2026. Numbers and model names below come from the code, not the READMEs: in places qmd's README lags behind its code.

## Baseline: trip2g

A short summary of our search, to compare against. The details are in [[en/thoughts/search-benchmark|Benchmarking search]].

- **Two searches in parallel.** Full-text (BM25 on bleve) and vector. The full-text index holds every note twice: once with Russian stemming, once with English.
- **Structure-aware chunks.** About 450 tokens, and a heading always starts a new chunk. Every chunk begins with its heading breadcrumb: `Note > Section > Subsection`. On long documents this lifted nDCG@10 from 0.93 to 0.95.
- **Embeddings** come from any OpenAI-compatible model; ours is mainly bge-m3. Vectors are kept in memory and the query is compared against every chunk, with no approximate-search index.
- **Fusion** is Reciprocal Rank Fusion (RRF) with k = 60. It adds up positions in the lists, not the raw scores.
- **Reranker:** bge-reranker-v2-m3. Its score is blended with RRF half and half rather than replacing it, and it runs only when a request asks for it. Why it is built this cautiously is covered in [[en/thoughts/truth-about-reranking|The truth about reranking]].
- **No language model.** Nobody rewrites or expands the query. Search returns a list; the agent on the other side of MCP assembles the answer.

## AnythingLLM: a vector database and nothing else

AnythingLLM is a chat-with-your-documents app you install on a server or a laptop. It offers ten vector databases, around fifteen embedding providers and a polished UI. Its search is the simplest of the three.

**Chunking.** It uses LangChain's `RecursiveCharacterTextSplitter`, sized in characters: 1,000 characters by default, with a 20-character overlap. It ignores markdown structure. Every chunk starts with a `<document_metadata>` block holding the document's title, date and source link. That is how they give a chunk context: the whole document, not the section.

**Search is vector-only.** There is no full-text search at all. The raw user message goes to the database, with no chat history and no rewriting. Cosine similarity, a 0.25 threshold, and **four** chunks come back. An exact error name or flag in the query is found only if its embedding happens to land close by.

**There is a reranker, but only for LanceDB** (the default vector database). The model is `ms-marco-MiniLM-L-6-v2`, running inside Node on the CPU. Vector search first fetches between 10 and 50 candidates (10% of the database size). The reranker then **replaces** their order outright and keeps four. MiniLM was trained on English, so on Russian text it adds almost nothing. We tried replacing the order on our own benchmark: nDCG fell by 0.03, and by 0.53 on whole notes.

**The interesting part is how context is assembled, not the search itself.**

- *Pinned documents.* A user can pin a document, and the whole document goes into every answer, bypassing search.
- *Backfill from history.* If search finds fewer than four chunks, the rest are taken from the sources of earlier answers in the same conversation. This is how they handle follow-ups like "tell me more": instead of rewriting the question, they mix in what was already found.
- *Documents-only mode.* If the context is empty, the model is not called at all and the user gets a preset refusal. A simple, honest guard against made-up answers.

## qmd: fully local, with two models per query

[qmd](https://github.com/tobi/qmd) is a search engine for local markdown files, exposed as a CLI and an MCP server. Everything runs on the user's machine through llama.cpp: three small GGUF models and one SQLite database. Technically it is the most elaborate of the three.

**Storage.** One SQLite file holds an FTS5 full-text index with Porter stemming and the vectors in sqlite-vec. Documents are stored by content hash. Embeddings carry a fingerprint, a hash of the model, the prompt templates and the chunking settings. Change any of them and everything is re-embedded automatically. We check that by hand.

**Chunking.** About 900 tokens with 15% overlap. The cut point is chosen by weight: an h1 is worth 100, an h2 90, a code fence 80, a blank line 20, a list item 5. The weight decays with distance from the target length, and it never cuts inside a code block. After cutting, each chunk is checked with the real tokenizer and split again if it is too long. For code you can switch on tree-sitter syntax parsing. Chunks carry no heading breadcrumb: only `title: … | text: …` is embedded.

**Context on paths.** You can attach a description to a folder: "meeting notes live here", "this is the 2023 archive". The description comes back with each result so the agent knows where the text came from. It is not indexed and not embedded.

**The main feature is query expansion.** qmd ships its own model: Qwen3-1.7B, fine-tuned to rewrite queries. It returns several lines of three kinds:

- `lex:` keywords for full-text search;
- `vec:` a paraphrase for vector search;
- `hyde:` a made-up answer, written the way a document might phrase it (the HyDE technique).

Each variant goes only to its own search. The original query goes to both, and its lists count double in RRF. Before calling the model, qmd tries plain full-text search. If the first hit is clearly ahead of the second (score of 0.85 or more and a lead of 0.15 or more), expansion is skipped because the answer is already obvious. Model outputs are cached in the same SQLite file.

**Fusion and reranking.** Everything is fused with RRF at the same k = 60, plus a bonus: +0.05 for a document that was first in any list, +0.02 for second or third place. The top 40 candidates go on. From each one, qmd picks the single chunk sharing the most words with the query, and Qwen3-Reranker-0.6B scores that chunk.

The scores are blended with a weight that depends on position: RRF counts for 75% in the top three, 60% for places 4 to 10 and 40% after that. The idea is that the top of a list where several searches agreed is nearly off-limits to the reranker, while the tail is fair game. It is the same idea as our 50/50 blend, done more carefully.

**What matters for us: Russian.** Each layer handles it differently.

- *Full-text search* splits Cyrillic into words correctly, but the Porter stemmer is English. To it, "кошка" and "кошки" are different words. Prefix matching on every query word helps a little, but every word has to match. One word in the wrong form and full-text search finds nothing.
- *Embeddings.* qmd's README calls the default model, embeddinggemma-300M, "English-optimized" and recommends Qwen3-Embedding for other languages. That is the one to use.
- *The reranker*, Qwen3, is multilingual.
- *Query expansion* is the weakest part. The 3,579 examples the model was fine-tuned on contain not one line of Cyrillic. The variant filter only understands ASCII: a Russian query leaves it no words to check against, so it lets every variant through, even off-topic ones. qmd does have a benchmark, but it runs on six documents, and the README numbers are approximate (BM25 about 0.5, hybrid about 1.0).

## Cerebras Knowledge: the data matters more than the ranking

The [Cerebras write-up](https://www.cerebras.ai/blog/how-we-built-our-knowledge-base) describes a company knowledge base built over Slack, code repositories, wikis, Jira and teams' own databases. Three months after launch it gets more than 15,000 questions a day, from people, automations and agents.

The most valuable part of the article is not the ranking formulas. It is what they do to the data **before** search.

**One table for everything.** Every source writes rows of the same shape into one Postgres table: embedding, summary, metadata. The article's illustrations show the vectors in pgvector: 3,072 dimensions with an HNSW index. The model is not named. A source defines three things: what the data is, how to connect to it and how often to refresh it. A team that wants to plug in its own database sends a pull request with a small Python module that writes rows into that table. Nothing else in the system has to change.

**Slack as the main source.** The engineers say plainly that embeddings over raw Slack text were not enough, and give three reasons:

- "ok, thanks Mike" and a detailed kernel explanation sit in the same feed;
- short messages beat long ones on cosine similarity;
- a message's meaning often depends on the messages around it.

Their answer is several kinds of search at once:

1. **Full-text** (a GIN index in Postgres) for exact strings: error text, flag names, host names. When an engineer pastes a whole error message, an exact match is almost always the best answer.
2. **Vector** for paraphrases. Someone asks "restore hangs after manifest load" and the answer said "checkpoint stalls on the NFS mount", with no words in common.
3. **IDF, word rarity**, separates signal from chatter. "Sounds good, thanks!" is close to anything in vector space but contains no rare words.
4. **Age decay**: Slack answers go stale. A six-month-old thread may describe infrastructure that no longer exists. When everything else is equal, the newer one wins.

**Threads are not embedded as-is.** A bot listens to Slack over a WebSocket. On every new message it re-fetches the whole thread and stores it as one row. The raw text goes straight into the full-text index. For vector search, a language model first turns the thread into a card:

- the question in one line, the way an engineer would search for it;
- a short summary;
- the resolution;
- the systems and code references mentioned.

The card is embedded, not the conversation. They report that accuracy rose noticeably once threads were normalized into one format.

**Bursting.** A card loses important remarks made off the main topic. So a run of consecutive messages from one author is embedded separately, with the thread topic in front. Not every run, only those that score high enough. The score is a weighted sum of three signals: a word with an IDF of 4.0 or more, a length of at least 200 characters, and a reaction on the message. The weights and the threshold are not published. A rejected burst does not vanish from search: it stays in the thread's full-text index and in the card. It just does not get its own vector, so "ok, thanks" never surfaces in vector search.

**Code.** Repositories, some larger than 40 GB, are indexed by CocoIndex. Cut points are tried from coarse to fine: class, then method, then block. One file yields several embeddings at different levels of detail. After each commit only the changed pieces are recomputed. Plain ripgrep over the source is also available.

**Ranking in the unified search.**

1. RRF with k = 60, each list weighted 1.0 by default. The illustration fuses six lists: vector over everything, full-text over everything, thread summaries, a **graph** (code relations: a class, a header file), vector over the wiki and full-text over Slack.
2. Duplicates collapse to one source, and each file may contribute only a limited number of results. The outcome is a varied top 20.
3. A reranker scores each candidate from 0 to 10, and the top ten are kept. The text calls it "a small reranker model"; the diagram labels it LLM rerank. So it is a language model assigning scores, not a cross-encoder.
4. **Context expansion.** If a wiki section matched, the two neighbouring sections are added so the heading, preconditions and caveats that chunking cut off are not lost.

**The planner.** In the web UI, a language model first picks which tools to call: `subsystem_index` (a summary per file), `search`, `search_slack`, `search_code` (ripgrep), `recent_prs`, `who_knows` (who knows the topic). The tools run in parallel, and another model call writes the answer.

**MCP has no planner.** Each tool is one kind of search with no language model inside: fast, cheap and predictable. The agent, Claude Code for example, decides the order of calls. We arrived at the same design when we built MCP for trip2g.

**Projects.** Once the base grew, searching everything at once stopped working: the compiler team does not want data-centre runbooks. A project is a named set of sources. The default project chosen at onboarding scopes every query, so a new hire gets answers from their own area on day one.

What the article leaves out: model names, chunk sizes for Slack and the wiki, how the graph search works, the decay formula and, above all, **any quality numbers**. The architecture is described in detail, but no measurements are shown.

## Hindsight: not document search, but an agent's memory

[Hindsight](https://github.com/vectorize-io/hindsight) by Vectorize solves a different problem. It does not search finished documents; it remembers what an agent has learned and hands it back on request. It has three verbs: `retain`, `recall` and `reflect`. You can feed it documents too, but it stores the facts extracted from them, not the documents. We read the code as of 28 September 2026.

**Writing: a model turns text into facts.** Input is cut into pieces of up to 3,000 characters, and a language model (gpt-4o-mini by default) breaks each piece into facts. A fact has fields for what, when, where, who and why, a time interval, a list of entities and causal links to earlier facts. The prompt asks the model to be selective ("will this be useful in six months?"), to replace "she" and "he" with names, and to turn "yesterday" and "last year" into exact dates. It is the same move as Cerebras's thread cards, taken down to single statements.

**The graph.** At write time facts are linked to each other:

- *by time*: facts within a day of each other, weighted by how close they are;
- *by meaning*: up to 50 nearest neighbours with a cosine of 0.7 or more;
- *causally*: caused by, causes, enables, prevents;
- *by entity*: shared people, places and things; entities are merged by spelling similarity (pg_trgm).

**Consolidation into observations.** After writing, a second model folds facts, 8 at a time, into observations. It decides what to create, update or delete, and every observation keeps a count of the facts that support it. The prompt's rules: prefer updating to creating, never delete history, never compute anything on your own.

**Search: four ways in parallel.**

1. *Vector*: over facts, with a similarity floor of 0.3.
2. *Full-text*: Postgres tsvector. Query words are joined with OR, keeping up to the 16 rarest. This was not done for quality: long queries used to hang the search for a minute.
3. *Graph*: from the 20 closest facts by meaning it spreads along entity, semantic and causal links.
4. *Time*: if the query says "last week" or names a date, an interval is extracted from the query; the search takes facts from that period and then follows temporal and causal links for up to five steps.

How many candidates each way takes is set by a "budget": 100, 300 or 1,000.

**Fusion and the reranker.** The lists are fused with RRF at k = 60. Then up to 300 candidates are scored by the `ms-marco-MiniLM-L-6-v2` cross-encoder, the same one AnythingLLM uses. The final score is the reranker's score multiplied by adjustments: for recency (±10%, linear over a year), for falling inside the time interval (±10%) and for the number of supporting facts (±5%). **The RRF position does not enter the final score**: the reranker replaces the first-stage order outright, as in AnythingLLM and as in our own failed experiment. The answer is cut to 4,096 tokens.

**Reflect.** An agent with tools works down a hierarchy: first ready "mental models" (stored answers to recurring questions, refreshed in the background), then observations, then raw facts. A memory bank has a "character": scepticism, literalism and empathy on a scale of 1 to 5. There used to be "opinions" with confidence as well, but they have been removed from the code.

**Measurements.** Hindsight is the only one of the five that publishes numbers on external sets: 94.6% on LongMemEval and 92% on LoCoMo. But those sets test conversational memory, not document search. The evaluation code lives in a separate repository, with Gemini as the judge. The authors' own blog says LoCoMo and LongMemEval barely separate systems any more: the whole history can simply be put into the model's context.

**Language: in Russian the memory breaks silently.** Everything defaults to English: the `bge-small-en-v1.5` embedding model, full-text search with the `english` dictionary, the `ms-marco-MiniLM-L-6-v2` reranker. Fact extraction still works: the language model understands Russian, and the prompt tells it to write facts in the input's language without translating. What breaks is searching what has been stored.

We tested the default models on one example. The question "Where does Anna work now?" and three facts: "Anna started a job at Yandex" (the right one), "Anna's cat is called Barsik", "The quarterly report is due on Friday". We ran the same example in English.

| | English | Russian |
|---|---|---|
| Embeddings: right / cat / report | 0.72 / 0.53 / 0.41 | 0.77 / 0.75 / 0.75 |
| Reranker: right / cat / report | +5.7 / −6.3 / −11.3 | +7.5 / +6.8 / +6.5 |

The English model has no Russian vocabulary and cuts Russian text into single letters. So any two Russian texts look alike to it: the gap between the right fact and an unrelated one is 0.02 instead of 0.31. The reranker calls everything relevant. Hindsight passes its score through a sigmoid, which gives 0.9994, 0.9989 and 0.9985. That difference is smaller than the recency adjustment (up to ±10%), so in Russian the order is decided by date, not meaning: yesterday's fact about the report beats March's fact about the job. Full-text search with the `english` dictionary indexes Russian words without morphology, so "работает" and "работу" are different words to it.

Nothing errors. Recall always returns something, and recent facts look plausible. It only shows on questions about the past. The fix is configuration: multilingual embeddings (`multilingual-e5-small`, bge-m3), a multilingual reranker (`mmarco-mMiniLMv2`, bge-reranker-v2-m3) and `russian` as the full-text language. All of these are listed in their own documentation. After switching the embedding model, the memory has to be re-embedded.

## obsidian-hybrid-search: the closest to us

[obsidian-hybrid-search](https://github.com/flowing-abyss/obsidian-hybrid-search) is an MCP server and CLI for searching an Obsidian vault. Its job is ours: ordinary notes with links, tags and aliases, queried by an agent. We read the code of version 0.15.2.

**Storage.** One SQLite file: an FTS5 full-text index, a separate trigram index over titles and aliases, and vectors in sqlite-vec. The index updates at startup and on file-system events, 5 seconds after a file changes.

**Chunking by heading, with a breadcrumb.** A short note becomes one chunk. A long one is split by heading, one chunk per section. A section that does not fit the model's window is cut with a 512-token window and a 64-token overlap. The overlap stays inside the section and never crosses a heading. Before embedding, each chunk gets `Title > H1 > H2` in front, the same trick we use.

**Embeddings:** `multilingual-e5-small`, quantized, on the CPU, about 30 MB. The `query:` and `passage:` prefixes are applied.

**Four kinds of search, RRF with k = 60 and weights:**

| Kind | Weight | How it works |
|---|---|---|
| vector | 1.5 | best chunk per note |
| full-text | 1.5 | title ×10, aliases ×5, body ×1; words joined with OR, prefix matching |
| exact alias match | 2.0 | the whole query equals a note's alias, case-insensitive |
| fuzzy title | 0.25 | share of trigrams matching the title and aliases; forgives typos |

The exact alias match weighs the most. If a person typed exactly what they call the note, that note is almost certainly the one they want.

**Reranker:** `gte-multilingual-reranker-base`, off by default. It is blended with RRF by position: RRF counts 75% for the first ten, 60% for the next ten and 40% after that. This is qmd's scheme.

**Links** are not part of ordinary search. They have a separate `related` mode: a graph walk from a note to a given depth, in both directions or one.

**Filters inside search.** Tags, frontmatter fields and folders are filtered in SQL, before ranking.

**Measurements.** The author keeps three sets and checks minimum metric values before every push:

- Obsidian Help: 171 notes, 58 queries, nDCG@5 0.733, Hit@1 0.72;
- Andy Matuschak's notes: 1,357 notes, 78 queries, nDCG@5 0.72;
- LongMemEval: nDCG@5 0.895, but with bge-m3 embeddings over an API.

Six Russian queries against the English help do noticeably worse: nDCG@5 0.46. The comparison with qmd (0.733 against 0.659 on the help set) exists only in the README; there are no qmd run results in the repository.

**Russian: search works, tags get lost.** The models are multilingual, so there is no silent breakage like Hindsight's. Full-text search has no morphology: "работает" finds "работает…" by prefix but not "работу", and the vectors fill the gap. One trap: the regular expression for inline tags understands only Latin script. `#note` is found and `#заметка` is not, so filtering by such a tag silently returns nothing. Frontmatter tags work.

## Summary table

| | trip2g | AnythingLLM | qmd | Cerebras Knowledge | Hindsight | obsidian-hybrid-search |
|---|---|---|---|---|---|---|
| Purpose | publishing notes, site search and MCP | chat with documents | local markdown search, CLI and MCP | company knowledge base | agent memory | MCP search over an Obsidian vault |
| Storage | bleve on disk, vectors in memory | 10 vector databases to choose from | one SQLite file (FTS5 + sqlite-vec) | one Postgres table | Postgres + pgvector | one SQLite: FTS5 + sqlite-vec |
| Full-text search | BM25, ru + en stemming | none | BM25, Porter stemming (English) | Postgres GIN | tsvector, OR over the 16 rarest words | FTS5, no stemming, OR and prefixes |
| Vector search | exhaustive scan, dot product | ANN in the chosen database | sqlite-vec; exact scan up to 20,000 chunks when filtered | pgvector, 3,072 dims, HNSW | pgvector, HNSW, over facts | sqlite-vec, exact scan |
| Other signals | — | — | — | IDF, age, code graph | graph: entities, meaning, causes; time | exact alias match, title trigrams |
| Chunking | ~450 tokens, by heading | 1,000 characters, no structure | ~900 tokens, weighted cut points, tree-sitter for code | thread card + bursts; code by class and method | pieces up to 3,000 characters → facts | by heading; large sections by a 512-token window |
| Context in the chunk | heading breadcrumb | document metadata | `title:` | thread topic | who / when / where inside the fact | heading breadcrumb |
| Model pass over the data | no | no | no | summary of every thread | facts at write time + consolidation into observations | no |
| Query rewriting | no | no | own model: lex / vec / hyde | planner picks the tools | no; a time interval is extracted from the query | no; several queries can be passed |
| Fusion | RRF, k = 60 | — | RRF, k = 60, original query ×2, bonus for top places | RRF, k = 60, per-list weights | RRF, k = 60, 4 lists | RRF, k = 60, weights 1.5 / 1.5 / 2 / 0.25 |
| Reranker | bge-reranker-v2-m3, on request | MiniLM, LanceDB only | Qwen3-Reranker-0.6B | a language model, 0–10 | MiniLM (English), up to 300 candidates | gte-multilingual-reranker, on request |
| How the reranker counts | 50/50 blend | replaces the order | position-based blend: RRF 75 / 60 / 40% | keeps the top 10 | replaces the order; recency and time adjustments | position-based blend: RRF 75 / 60 / 40% |
| Reranker sees | best chunk of the note | chunk | chunk with most query words | candidate after dedup | a fact | title + best chunk |
| After ranking | access checks | backfill from chat history | — | neighbouring wiki sections | cut to 4,096 tokens | —; links in a separate related mode |
| Scoping | note permissions | workspace | collections and metadata filters | projects and source permissions | memory banks, tags | tags, frontmatter, folders |
| Default result count | up to 20 | 4 | 5 in CLI, 10 in MCP | 10 | whatever fits in 4,096 tokens | 10 |
| Languages | ru + en explicitly | depends on the model | full-text and expansion: English | not stated | English by default; Russian search degrades silently | multilingual models; Russian inline tags get lost |
| Quality measurements | benchmark: 60 + 16 queries, nDCG@10 0.93 / 0.95 | none | 6 documents, thresholds in tests | not published | LongMemEval 94.6%, LoCoMo 92% (conversational memory) | 3 sets, 58–470 queries, nDCG@5 0.72–0.90 |

## What this shows

Everyone who takes search seriously has more in common than not. Full-text search alongside vector search appears in five of the six systems; only AnythingLLM lacks it, and its code shows the cost. RRF with k = 60 is used by all five that fuse anything. The reranker scores a piece of text that fits its window, never a whole document. What to do with the reranker's score is not settled: trip2g, qmd and obsidian-hybrid-search blend it with the first stage, while AnythingLLM and Hindsight put it in place of the first stage.

Where the systems differ is where they spend a language model. Spend it nowhere and you get trip2g, AnythingLLM and obsidian-hybrid-search: fast, predictable, and the agent on the other end fills the gaps. qmd spends it on the query, rewriting and expanding it, and skips that step for obvious queries. Cerebras spends it on the data, turning chatter into cards once, at write time. For Slack that looks like the only approach that works. Hindsight goes furthest: one model breaks every piece of input into facts at write time, a second folds the facts into observations, and a third reasons over them on request. For an agent's memory, where "yesterday" and "she" must become dates and names, that is justified. For markdown notes that a person has already written carefully, the gain is less obvious.

Worth running through our benchmark before arguing about any of it:

- **position-based blending as in qmd**, instead of a flat 50/50;
- **a bonus for first place in RRF**: cheap, a one-line change;
- **neighbouring sections after ranking, as at Cerebras**: we already have breadcrumbs, and MCP can already return a section by `toc_path`;
- **age decay**, but only for vaults where notes really go stale, such as feeds and logs;
- **query expansion**, the most expensive item: a model call per query and a separate model for Russian. It should be tested last.
