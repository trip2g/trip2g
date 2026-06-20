# RAG is a band-aid: from passive retrieval to a federated context mesh

*Audience: technical users who've hit the ceiling of classic RAG, AI builders, researchers who want their agents to actually know things. ~1,200 words. Voice: see [`../00_brief.md`](../00_brief.md).*

---

Retrieval-Augmented Generation solved a real problem. Language models hallucinate. They go stale. Their training data ends at a cutoff date and doesn't include your notes, your team's decisions, or anything that happened last Tuesday. RAG was the fix: embed your documents, store the vectors, retrieve the relevant chunks at query time, paste them into the context window, and suddenly the model knows about your stuff.

It works. The problem is what you give up to make it work.

## The original sin of partitioning

To build a RAG pipeline, you slice your knowledge into chunks (paragraphs, pages, fixed-token windows), embed each chunk as a vector, and store them in a database. The database answers nearest-neighbor queries: given a question, return the N most similar chunks.

That design commits what I'd call the original sin of partitioning. Your knowledge starts as a connected graph of ideas. A research note links to a paper, the paper links to a methodology, the methodology constrains a decision you made six months ago. Those connections are what give the knowledge its meaning. The vector database throws them out. It stores fragments without context, without provenance, without the web of relationships that made the original worth writing.

Microsoft's GraphRAG team put numbers on this in 2024. Classic RAG fails on questions requiring synthesis across a whole dataset: the "what is the overall theme of this body of work?" queries that matter most. Graph-structured retrieval, which preserves entity relationships, answered those queries substantially better ([From Local to Global: A GraphRAG Approach](https://arxiv.org/abs/2404.16130)). The chunk was the problem. The chunk lost too much.

## Lost in the middle

There's a second problem that arrives when you try to fix the first one.

The instinct is: give the model more context. Bigger context window, more chunks retrieved, more of the document included. But Liu et al. showed in 2023 that this doesn't work the way you'd expect. When relevant information appears in the middle of a long context, performance drops sharply. Models are best at using information at the beginning and the end of their context window. Everything buried in the middle degrades ([Lost in the Middle: How Language Models Use Long Contexts](https://arxiv.org/abs/2307.03172)).

So retrieving more chunks doesn't straightforwardly help. You're more likely to push the useful signal into the degraded middle. The retrieval has to be precise, or you're paying the cost of a large context without getting the benefit.

The underlying problem is that RAG doesn't actually solve the retrieval question: it offloads it to an embedding similarity function. That function has no understanding of the structure of your knowledge, the trust relationships between sources, or which parts of a document are relevant to which kinds of questions. It finds things that are textually similar to the query. That's useful. It's also limited in ways that matter as soon as your agents start doing real work.

## The ownership break

Classic RAG also breaks ownership, which gets less attention than the technical limitations.

To run a RAG pipeline on your notes, you extract the text, embed it, and store the vectors in a database. That database is a copy (a transformed copy, but a copy). It lives somewhere, usually in a cloud service, controlled by whoever runs that service. The original meaning of your note is now partially residing outside your control.

For personal notes, this is a privacy problem. For organizational knowledge, it's a compliance problem. For a peer circle where three people want to share research without merging their personal files, it's structurally wrong: to query across everyone's knowledge, you'd have to centralize all of it in one vector store, and then the "separate owners" part is fiction.

MemGPT (now Letta) identified a related problem at the agent level. Their 2023 paper framed it as the RAM-vs-disk question: agents need to manage what's in the context window versus what's in external storage, and they need to do it explicitly, like an OS manages memory ([MemGPT: Towards LLMs as Operating Systems](https://arxiv.org/abs/2310.08560)). The insight was that passive retrieval ("here are some chunks") is not enough. Agents need to navigate memory architectures actively, deciding what to surface and when.

The same logic applies to federated knowledge. Passive retrieval from a central vector store assumes one owner, one trust boundary, one corpus. Real knowledge doesn't look like that.

## What structured, federated context looks like

The alternative starts from different premises.

First: the original files stay where they are. Each person's notes live on their own machine. The knowledge base is not a vector store. It's a live index built on top of the actual documents, which remain the source of truth.

Second: access control travels with the data, not with the query. When you make a folder shareable with a trusted peer, you're defining a boundary. An agent querying across your circle's nodes can reach what you've made available, and nothing else. The boundary doesn't live in a permissions table in someone else's database. It's enforced at the node level, on hardware you control.

Third: the query is structured. Instead of embedding similarity over decontextualized chunks, agents can issue semantic queries against nodes that understand the structure of the underlying documents: backlinks, tags, metadata, headings, the connections that give the knowledge its shape. Citations come back with the answer: here's the node, here's the document, here's the passage.

This is what federated search means in practice. One agent question, answered across a mesh of trusted bases, each enforcing its own access rules, each returning structured results with provenance. Not a paste of chunks from a central database. A query trace across a network of owned knowledge.

## The honest risk

There's a real threat to this whole argument. If frontier models ship context windows of 10 million tokens with sub-second latency and zero "lost in the middle" degradation, the case for complex retrieval architectures weakens considerably. Why maintain a federated mesh when you can dump everything into one massive prompt?

Ownership and cost don't go away even if context windows grow. Sending your entire knowledge base (and your friends' knowledge bases) to a cloud model on every query is expensive, slow for large vaults, and requires uploading files you may not want to upload. Access control evaporates. The "separate owners" property is gone.

Those constraints matter enough to most users that structured retrieval doesn't disappear even in the large-context future. But the honest position is that very large context windows do reduce the retrieval precision requirement. Trip2g's value is clearer now than it will be in three years. We'd rather say that plainly than pretend otherwise.

## The point

Classic RAG is a workaround for a model limitation that was real in 2022. It solved the hallucination and staleness problems by giving models access to external text. But it introduced new problems: it partitioned knowledge into decontextualized fragments, broke ownership, and gave agents no way to query across trust boundaries.

The transition from passive retrieval to a federated context mesh isn't about better embedding models or larger vector stores. It's about treating knowledge as a graph of owned, connected, access-controlled documents, and giving agents the ability to walk that graph across the people they're allowed to query. RAG retrieved documents. A context mesh routes context, and that difference is structural.

---

*Trip2g turns local Markdown vaults into federated knowledge nodes with per-folder access control and structured agent queries. Self-hosted, MIT licensed. More in [the research notes](../../dev/market_research_notebooklm.md). Primary sources: [GraphRAG](https://arxiv.org/abs/2404.16130), [Lost in the Middle](https://arxiv.org/abs/2307.03172), [MemGPT](https://arxiv.org/abs/2310.08560), [Local-First Software](https://www.inkandswitch.com/local-first/).*
