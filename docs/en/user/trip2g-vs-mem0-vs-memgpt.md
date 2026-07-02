---
title: "Agent memory compared: Mem0 vs MemGPT (Letta) vs Zep vs trip2g"
description: "An honest comparison of AI agent memory systems: Mem0's extraction layer, Letta's self-managed memory, Zep's temporal graph, and trip2g's memory-as-markdown. Pick-when verdicts for each."
free: true
lang_redirect: "[[ru/user/trip2g-vs-mem0-vs-memgpt]]"
---

Choosing memory infrastructure for an agent is confusing because the tools solve different problems under the same word. Mem0 is a memory layer you bolt onto an existing agent. Letta (formerly MemGPT) is a runtime where the agent manages its own memory. Zep builds a temporal knowledge graph from conversations. trip2g takes a fourth path: memory is a markdown knowledge base that humans and agents share.

There is no best overall; there is a best fit per constraint. This page gives each tool its own section with the tradeoff named, then a pick-when list you can act on.

*Updated: July 2026. Competitor claims are taken from their own docs and repos, as of this date.*

**Before any framework, check the baseline.** If your agent is Claude Code, it already has native memory: `CLAUDE.md` plus auto memory (Claude's own notes under `~/.claude/`), on by default. It is free and needs no setup, but it is machine-local, best-effort, and not shared with a team, so it is not in the matrix below as a framework. If you work solo on one machine, native memory (or a local plugin like MemPalace, MIT, on-device) may be all you need; the tools here earn their place when memory must be shared, durable, or queryable at scale. See [[en/user/claude-code-persistent-memory|persistent memory for Claude Code]] for the native-first walkthrough.

## Decision matrix

| | Mem0 | Letta (MemGPT) | Zep | trip2g |
|---|---|---|---|---|
| Architecture | memory layer / API over your stack | agent runtime with self-edited memory | temporal knowledge graph (Graphiti) | markdown knowledge base + MCP |
| Auto-extraction from chat | **yes** | agent decides what to save | **yes** | no, deliberate notes |
| Memory readable by humans | dashboard | via agent tools | graph explorer | **plain files: Obsidian, web, git diff** |
| Editable by humans | limited | limited | limited | **edit the file** |
| Recall benchmark published | LOCOMO results (self-reported) | research lineage (MemGPT paper) | LongMemEval 71.2% with GPT-4o (self-reported) | none |
| Retrieval cost benchmark | no | no | no | **15× median section-vs-note, reproducible** |
| Self-hosted | yes (OSS) + managed cloud | yes (OSS) + cloud | yes (community edition) + cloud | yes, MIT; single Go binary + SQLite |
| Infra to run | vector store (+ optional graph) | Postgres + runtime | graph DB stack | SQLite only |
| Memory doubles as a website | no | no | no | **yes** |
| Community size | largest (~60k GitHub stars, July 2026) | large, research pedigree | large | small |

Read the losing rows honestly. trip2g does not extract memories automatically, publishes no recall-accuracy score, and has the smallest community of the four. What it uniquely offers is the bottom-left corner: memory you can open, read, edit, version, and publish as ordinary documents.

## Mem0: the bolt-on memory layer

Mem0 sits between your agent and an LLM, watches the conversation, extracts facts, and serves them back later. It is the most popular option by community size, it has a managed cloud for zero-ops adoption, and its self-reported LOCOMO numbers beat naive long-context baselines while cutting tokens.

The tradeoff: extraction is itself an LLM pipeline. You pay per-message extraction calls, and what lands in memory is the extractor's judgment, not yours; auditing why the agent "remembers" something means digging through a dashboard rather than reading a document. On cost: the managed free tier is capped (on the order of 10k stored memories and 1k retrievals per month as of July 2026), graph memory is a paid feature, and self-hosting brings up its own stack (Docker plus a vector store such as Qdrant, and a local model runner for extraction).

**Pick Mem0 when** you want memory that accumulates with zero authoring discipline, you are fine with an extraction pipeline in the loop, and a managed cloud is a plus rather than a concern.

## Letta (MemGPT): the agent that manages its own memory

Letta grew out of the MemGPT paper, which framed the context window as an OS problem: memory pages in and out, and the agent itself decides what to keep in core memory versus archival storage. It is the most architecturally interesting option, and it is a full runtime: you build agents on Letta, you do not bolt it onto an existing one.

The tradeoff is exactly that: adopting Letta means adopting its runtime and its way of building agents. Memory quality depends on the agent's own editing decisions, which are impressive and occasionally puzzling in equal measure.

**Pick Letta when** you are building agents from scratch, want self-managed memory as a first-class design element, and value research lineage over drop-in simplicity.

## Zep: the temporal knowledge graph

Zep ingests conversations and business data into Graphiti, its open-source temporal graph engine: every fact carries validity intervals, so the system can answer "what was true then" and not just "what is true now". Zep's self-reported LongMemEval results (71.2% with GPT-4o against a 60.2% full-context baseline, per its own blog) made temporal reasoning its calling card.

The tradeoff: a knowledge graph stack is the heaviest of the four to operate, and the graph is a machine artifact; you query it, you do not read it like documentation.

**Pick Zep when** point-in-time correctness matters (support, CRM-adjacent agents, anything where facts expire) and you have the appetite for graph infrastructure.

## trip2g: memory as a markdown knowledge base

trip2g is not a memory pipeline; it is a server for a folder of markdown notes. The agent writes memory as `.md` files; a watcher syncs them within ~500 ms; recall happens over MCP with section-precise retrieval (`search → expand → note_html`). The same base is a website for humans, has per-note access control, and every edit is versioned in `note_versions` plus a git mirror. Setup is one command: [[en/user/memcli|memcli]].

Three things follow from "memory is documents":

1. **You can read your agent's memory** the way you read documentation, and fix it with an editor. No other tool in this table offers that.
2. **No extraction pipeline and no vector-DB stack to operate.** A single Go binary with SQLite; the vector index is built in.
3. **The memory has an audience.** The same notes serve teammates under access control, other agents over MCP, and (if you choose) readers as published pages. Federation fans one query across several bases ([[en/user/federation|how]]).

The honest flip side, stated plainly:

- **No auto-capture.** If the agent is not instructed to write notes ([[en/user/agent-memory|the rules are two lines]]), nothing is remembered. Mem0 and Zep capture without being asked.
- **No recall-accuracy score.** We have not run LongMemEval or LOCOMO. The number we do publish is retrieval cost: reading the answering section instead of the whole note is 15× cheaper at the median, 37× on long notes, with a stdlib-only script you can run against any instance: [[en/user/token-economy-bench|the benchmark]].
- **Smallest community of the four.** You will find fewer integrations and fewer Stack Overflow answers.

**Pick trip2g when** you want memory you can read, edit in Obsidian, and publish; when auditable, versioned memory matters more than automatic capture; when you want agent memory and a team knowledge base to be the same thing.

## When to pick what

- **Pick Mem0 when:** zero-discipline capture, managed option, biggest ecosystem.
- **Pick Letta when:** building agents on a runtime with self-managed memory.
- **Pick Zep when:** temporal correctness is the requirement.
- **Pick trip2g when:** the memory should be a human-readable, versioned knowledge base.
- **Pick none of the above when:** a `CLAUDE.md` file already covers your needs. Not every project needs memory infrastructure.

Whatever you pick, run your own evaluation on your own workload; every benchmark cited above, including ours, was published by the tool's own vendor.

## FAQ

**Is trip2g a Mem0 replacement?**
Only if your problem is "memory I can read and control". If your problem is "capture facts from thousands of user conversations automatically", it is not, and Mem0 fits better.

**Can trip2g and these tools coexist?**
Yes. Some teams keep an extraction layer for conversational trivia and a markdown base for durable knowledge. The bases do not compete for the same facts.

**Why doesn't trip2g run LongMemEval?**
The benchmark measures extraction-and-recall pipelines over conversation history. trip2g deliberately has no extraction step, so the honest number for it is retrieval cost, which we publish with a reproducible script instead.

**What does migration out look like?**
For trip2g: copy the folder of `.md` files, or `git clone` the mirror; there is no export step because the storage format is already the export format. Check each vendor's docs for their export paths.

**Which is cheapest to run?**
For self-hosting, trip2g is a single binary with SQLite. Mem0 OSS wants a vector store; Zep CE wants a graph stack; Letta wants Postgres plus its runtime. Managed clouds move the cost to a subscription instead.

## Related

- [[en/user/mcp-memory-server|MCP memory server overview]]: the hub page for this cluster
- [[en/user/claude-code-persistent-memory|Persistent memory for Claude Code]]: hands-on tutorial
- [[en/user/agent-memory|Long-term memory for AI agents]]: full setup reference
- [[en/user/llm-wiki|LLM Wiki]]: the pattern behind memory-as-knowledge-base
- [[en/user/token-economy-bench|Token economy, measured]]: our one published number, reproducible
