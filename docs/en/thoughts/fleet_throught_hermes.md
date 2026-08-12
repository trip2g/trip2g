---
title: "Both sides wanted to hold the tools"
free: true
lang_redirect: "[[ru/thoughts/fleet_throught_hermes]]"
---

trip2g has an agent daemon, `cmd/fleet`. Drop a note into `tasks/`, sync, and a note shows up in `results/`. Its loop is ordinary function calling: the model asks for `read_note`, `write_note`, `finish`; fleet executes those and hands the results back.

I wanted that daemon to think on Hermes — NousResearch's open agent, `nousresearch/hermes-agent:v2026.7.7`, in a container on my own machine, running on the ChatGPT subscription I already pay for. No third-party API key, same argument as [[en/thoughts/hermes-wiki|the site-agent]].

Hermes exposes `POST /v1/chat/completions`. Fleet talks to `POST /v1/chat/completions`. I expected to change a URL.

## An agent is not a model

Hermes accepts the OpenAI `tools` array and ignores it. Silently: no error, no warning, just a normal answer with no `tool_calls` in it. Its own `/v1/capabilities` states this outright — `"tool_execution": "server"`. Hermes runs its own skills internally and returns final text.

For fleet that reads as something specific. Zero tool calls means the run is over and this is the answer. So every task finished instantly, with a polite paragraph and an empty `results/` folder. Nothing gets written to a note unless something emits a `write_note` call, and nothing ever would.

Two agent architectures, both of which want to be the one holding the tools. Hermes assumes the tools live on its side. Fleet assumes the tools live on its side, and it is right to: they are scoped GraphQL calls into a specific vault, with a specific token and a `write_patterns` allowlist. That is not a capability you hand to a container and hope.

## Reconstructing tool calls out of prose

The fix is a shim, `cmd/hermesllm`: OpenAI on one side, plain text on the other.

It takes fleet's tool schemas and renders them into a system preamble — the tool list, then the protocol:

```
Reply with a SINGLE JSON object and nothing else:
{"tool_calls":[{"name":"<tool>","arguments":{...}}]}

Rules:
- No prose, no explanations, no markdown code fences — the JSON object only.
- Use only the tool names listed above, with exactly their arguments.
```

Then it parses whatever comes back into real OpenAI `tool_calls` and hands them to fleet, which never learns anything unusual happened.

The part I did not anticipate was the transcript. Hermes accepts only `system`, `user` and `assistant`. It has no concept of a `tool` message, and fleet re-sends the whole conversation every turn. So the shim flattens: a tool result becomes a user turn, `Result of write_note: ...`, and an assistant turn's tool calls become `Called write_note with {...}`. Without that, Hermes would never see the outcome of its own calls, would keep asking for the same thing, and would never reach `finish`.

Tool use here is not an API feature. It is a text protocol, reconstructed at both ends.

The repo already had this shape and I did not notice until the shim was written. `cmd/codellm` is a fake LLM used in tests: it runs Go code and returns the results dressed as `tool_calls`. Same trick, different source of truth. Fleet cannot tell the difference, which is what the seam is for.

## It worked on the first try

I expected a week of prompt fighting: markdown fences, apologies, "here is the JSON you asked for", a wrapper object with a different key. I had already written the tolerant parser that walks every `{` in the text and takes the first balanced span that parses.

The first request came back in about ten seconds, clean, exactly as specified:

```json
{"tool_calls":[{"name":"write_note","arguments":{"path":"results/task1.md","content":"answer: 30\n"}}]}
```

No fence. No prose. That is the finding, and it changed how I think about this class of adapter. A frontier model asked to emulate function calling through prose discipline just does it. The mechanism a provider sells as a structured API feature is, at this point, mostly instruction-following with a schema validator bolted on.

## The demo

`e2e/hermesfleet/run.sh` runs the whole chain with nothing mocked:

```
vault → trip2g → change webhook → fleet → hermesllm → hermes → ChatGPT/Codex
  → write_note → scoped GraphQL → results/ → two-way sync → disk
```

Two task notes go in during the same round: `tasks/task1.md` says compute 10 + 20, `tasks/task2.md` says multiply 7 by 6. Four assertions — both result notes exist on the site with the right answers, 30 and 42, and both exist on disk in the vault folder after a two-way sync. Two independent tasks, because one lucky answer proves nothing.

The role note is the entire configuration. `trigger_include: ["tasks/**"]` with `max_depth: 1` keeps the write-back from re-triggering the webhook that produced it, and `write_patterns: ["results/**"]` is the only thing standing between the agent and the rest of the vault.

## The bill

Hermes charges about 14 500 prompt tokens per turn before any of my prompt is counted. That is its own system prompt and its skill catalogue, shipped on every call. Measured on a one-paragraph answer:

```json
{"prompt_tokens": 14470, "completion_tokens": 41, "total_tokens": 14511}
```

The first version of the shim reported zero usage. I had copied that from `codellm`, where it is honest — `codellm` runs Go locally and genuinely spends nothing. Through `hermesllm` the same zeros were a lie, and the lie was load-bearing: fleet computes both of its spend ceilings from reported usage. `max_tokens` and `TRIP2G_FLEET_TOKEN_CEILING` were both silently disabled, leaving `max_steps` as the only thing between a loop and a real invoice. A code review caught it.

That is the part worth writing down. An adapter that misreports cost disarms every guardrail downstream of it, and it does so quietly: nothing fails, no ceiling fires, the counters just read zero forever. The safe default for a translation layer is to pass through what the upstream actually said, especially when the number is inconvenient.

The other half of the bill is architectural. A server-side agent used as a raw model is an expensive substrate — you pay for its whole toolbox on every turn, whether or not you touch any of it. Fleet uses none of Hermes' skills. It only wants the reasoning, and it pays for the catalogue each time.

## The credentials

Hermes keeps its own auth store at `$HERMES_HOME/auth.json` and deliberately does not auto-import `~/.codex/auth.json`. The reason is good: the refresh token is single-use, so two holders sharing one OAuth grant will race, and the loser gets logged out.

The harness copies the tokens into the container's own store anyway. It works because the access token was still valid for days, so no refresh happens during a run and the host CLI's session survives untouched. That is a demo-grade shortcut, and I would rather say so than dress it up. `fleetagent3`, a sibling project, solves it properly: a broker holds the real credential and hands each agent a stand-in token it can revoke.

## What I take from it

Two agents cannot both hold the tools. One of them has to be demoted to a model, and the demotion is a translation layer.

The translation turned out cheap — a preamble, a parser, a transcript flattener — because the model on the other side follows a text protocol better than I assumed. The expensive parts had nothing to do with intelligence: what a turn costs, and who is allowed to hold the key.
