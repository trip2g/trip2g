---
title: "Let the agent expand the query"
free: true
lang_redirect: "[[ru/thoughts/query-expansion-by-the-agent]]"
---

*What we did: took apart how qmd expands a query, which I first got wrong. Proposed handing that work to the agent that is already connected over MCP: it passes three optional fields with the query. Chose the field names on three models and measured the gain on 236 queries: nDCG@5 rose on all four sets, and clearly so on the Russian base set, from 0.603 to 0.702. Read it if you run MCP search and are thinking about adding query rewriting.*

For [[en/thoughts/rag-under-the-hood|How others search]] I was taking apart [qmd](https://github.com/tobi/qmd), a local search engine for markdown notes. Its main feature is query expansion: its own Qwen3-1.7B model, fine-tuned to rewrite a query into several variants. I read that and assumed the model runs at indexing time, working out in advance how each note might be searched for. That is how other systems in the same article work. Cerebras turns Slack threads into cards when it writes them. Hindsight breaks text into facts when it writes it.

qmd does the opposite. At indexing time no model generates anything: notes are cut into chunks and the chunks are embedded. The model runs at search time, on every new query.

## How it works

You type `qmd query "how to create internal links between notes"`. Then:

1. **Probe.** A plain full-text search on the original query. If the top result is clearly ahead, expansion is skipped: the answer is already obvious.
2. **Expansion.** Otherwise the model gets the prompt `Expand this search query: …` and generates lines of three kinds:
   - `lex:` keywords for full-text search;
   - `vec:` a paraphrase for vector search;
   - `hyde:` a made-up paragraph showing what an answer might look like in a document.
3. **Routing.** Each variant goes only to its own search; the original query goes to both. That makes five to eight lists.
4. **Fusion** with RRF. The original query's lists count double.
5. **Reranking.** Qwen3-Reranker-0.6B scores the top 40 candidates.
6. **Cache.** The expansion and the reranker scores are stored, so the same query again answers in a fraction of a second.

Here is what the model produced in our run. The English query "how to create internal links between notes":

```
vec:  tips for linking notes within a document
vec:  methods for integrating internal links in notes
hyde: The process of create internal links between notes involves several steps.
      First, guide to adding links within note content. ...
```

Decent. And here is the Russian «как упорядочить заметки с помощью тегов» ("how to organize notes with tags"):

```
lex:  метод упорядочивания        (ordering method)
lex:  советы по                   (tips on)
hyde: The topic of how to sort notes with tag ordering covers steps ... is the most
      effective way to provide.  is the most effective way to provide.  is the most
      effective way to provide. ...
```

The keyword "tips on" matches anything. The made-up answer is in English for a Russian question, and it loops. That is no accident: the model was fine-tuned on three and a half thousand examples, and not one line of them is in Cyrillic.

## What it costs

Step 2 is generation, one token at a time, up to 600 tokens. Step 5 is forty full passes of a model over "query plus a chunk of up to 900 tokens". All of it repeats for every new query. On our server with no GPU, which we were also sharing with other jobs, one qmd query took three to seven minutes. On a MacBook with an M3 Max and Metal, about ten seconds. The obsidian-hybrid-search README quotes under a second for qmd on a Mac, but we could not reproduce that. And trip2g is a server, usually without a GPU.

## Where the agent comes in

When someone connects to a knowledge base over MCP, there is an agent on the other side: Claude, GPT or another large model. It already reads the user's question, knows the conversation and knows the language. Rewriting the query into two or three variants costs it a couple of lines of output and costs our server nothing.

qmd itself allows for this. Its MCP `query` tool accepts either a string or a ready-made list:

```json
{
  "searches": [
    {"type": "lex",  "query": "tags organize notes"},
    {"type": "vec",  "query": "how to sort and group notes by tags"},
    {"type": "hyde", "query": "Tags help you find and group notes: add a #tag to the text or to the tags property."}
  ],
  "intent": "the user wants to organize notes with tags"
}
```

If the agent sends `searches`, qmd's own model does not run at all. The MCP server's instructions teach the agent to fill these fields: what `lex` is, what `vec` is, why `hyde`, and to always pass `intent`. obsidian-hybrid-search does the same more simply with a `queries[]` parameter: several phrasings are searched separately and fused with RRF.

So the small model in qmd is a fallback for people searching from the command line without an agent. When there is an agent, expansion is better left to it.

> [!tip] Two problems, one move
> Handing query expansion to the calling agent solves two problems at once. First, trip2g doesn't have to carry a generative model, keep it in memory and run it on every query; on a server without a GPU that means minutes. Second, the variants come from a model many times larger than Qwen3-1.7B that understands Russian and knows what the conversation is about. Cheaper and better at the same time.

## What to call the fields

In qmd the fields are named `lex`, `vec` and `hyde`, after the search technique rather than the meaning. A model always sees a field's name but doesn't always read its description, so we tested three sets of names on three models: gpt-5.4-mini, gpt-5.4-nano and claude-haiku-4.5. Each got twenty questions from our set, and we looked at the first search call: were the fields filled, and with what.

| Names | mini | nano | haiku |
|---|---|---|---|
| `short_query`, `rephrased_query`, `expected_answer` | all filled, 1 wrong language | all filled, 2 wrong language | searched 18 times in 20, third field once |
| `short_query`, `rephrased_query`, `answer_excerpt` | all filled, 1 wrong language | third field 18 of 20, 7 wrong language | third field never |
| `lex_query`, `vec_query`, `hyde_passage` | all filled, 4 wrong language | third field 18 of 20, 2 wrong language | third field 9 times in 18 calls |

The plain names won: `short_query`, `rephrased_query`, `expected_answer`. qmd's jargon only helps Haiku, which recognizes HyDE by name.

The first version of the descriptions failed in an unexpected place. For the third field I wrote "describe what the answer might be", and mini dutifully returned "The documentation should explain what Bases are…". For vector search that is useless: help pages aren't written that way. Rewriting the description the other way round fixed it: "one or two sentences that could stand in the note itself, as a statement, the way documentation is written". One more thing: I first wanted to call `short_query` `keywords`, but with that name a model sends a comma-separated list, while full-text search needs two or three words, no more.

## What it gained

We ran all 236 queries of the Obsidian Help set: base and tricky, English and Russian. The variants for each query were written by a separate Claude Sonnet agent: it was given only the question texts and the field descriptions and told not to open the help docs or the answers. The files cannot prove it, but it shows indirectly: on the non-obvious queries its guesses often disagree with the right answers, which is how someone who doesn't know the answer writes. `short_query` went to full-text search only, the other two fields to vector search only. nDCG@5:

| Set | trip2g | + `short_query` only | + all three fields |
|---|---|---|---|
| English base | 0.742 | 0.794 | **0.798** |
| English tricky | 0.767 | **0.820** | 0.815 |
| Russian base | 0.603 | 0.653 | **0.702** |
| Russian tricky | 0.815 | 0.824 | **0.866** |

The biggest gain came where we were weakest: ten points on the Russian base set. That is the only gain clearly above noise: a bootstrap puts its 95% interval at +0.04 to +0.16. On English base and Russian tricky the interval barely clears zero; on English tricky it includes zero. In English almost all of the gain comes from `short_query` alone: full-text search needs every word to match, and two precise words instead of a long phrase remove that problem. Recall@10 reached 0.98–1.0 almost everywhere: the right page is nearly always in the top ten.

For comparison, qmd with its own expansion scored 0.717, 0.733, 0.608 and 0.670 on the same queries, and 0.731, 0.764, 0.668 and 0.771 with multilingual Qwen3 embeddings. On the Russian base set that qmd variant is within noise of trip2g with the fields. But its queries took about ten seconds each on an M3 Max with Metal. Our three fields cost the server two extra embeddings of short phrases and one extra full-text query: milliseconds.

## What it doesn't solve

- **Site search.** There a person types the query and there is no agent. This only improves MCP.
- **Weak agents.** Haiku rarely writes the third field; nano sometimes writes in the wrong language. The fields are optional: without them search works as before, with them it works better. The server writes the knowledge base's language straight into the field description.
- **The fields hurt some queries.** On the English tricky set, queries equal to an alias fell from 0.81 to 0.72 and questions about deep sections of long pages from 0.97 to 0.89. Syntax queries on English base fell from 0.83 to 0.73, long questions on Russian tricky from 0.87 to 0.70. The gain sits in short queries, paraphrases and how-to questions.
- **Cost on our side.** Each variant is one more embedding. Cheap, not free.
- **A strong model wrote the variants.** The gain was measured on its variants. With a weak agent it will be smaller; how much smaller is the next measurement.

We have already seen how much the behaviour of cheap models depends on the instructions: [[en/thoughts/mcp-instructions-make-cheap-models-faster|MCP instructions make cheap models faster]]. Same here: a field's name and one sentence in its description decide whether search gets a useful variant or a retelling of "what the documentation should explain".
