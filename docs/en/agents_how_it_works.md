---
title: "How LLM agents work — and how trip2g runs them"
free: true
lang_redirect: "[[ru/agents_how_it_works]]"
---

An LLM agent is a language model in a loop: it calls tools, reads the results, and repeats until it either calls `finish` or hits a budget limit. trip2g's fleet lets you define such agents as ordinary notes — the frontmatter is the configuration, the body is the instruction — and runs them automatically whenever a matching note in your vault changes. Reads and writes go through a scoped per-delivery token that enforces exactly which paths the agent may touch.

## How an LLM agent works

A language model on its own answers a single prompt and stops. An agent extends that with a loop and tools.

The loop:

1. The model reads the current context (system prompt + conversation history).
2. If it has enough information, it calls `finish` and the run ends.
3. Otherwise it picks a tool — search the knowledge base, read a note, write a note — and the runtime executes that call.
4. The tool result is appended to the conversation, and the model runs again from step 1.

"Tools" in modern LLM APIs are not plugins or external services. They are JSON function schemas sent to the model on every request. The model returns a structured call (function name + arguments); the runtime executes it and feeds the output back. This mechanism is called function-calling (or tool-use). trip2g agents use it — not MCP.

**Why scope and permissions matter.** The model can request any tool call its instructions suggest. Without a scope layer, a poorly-written instruction or a prompt-injected note could cause the agent to read or overwrite things it should not. Enforcing read and write path patterns at the runtime level — not just in the prompt — is what makes access boundaries stick.

**When an agent is overkill.** If the task is deterministic — reformat a note, apply a template, compute a value — a plain prompt or a script is simpler and cheaper. Use the iterative agent loop when the right sequence of steps is not known in advance and the model needs to navigate based on what it finds. Anthropic's Building Effective Agents guide says it plainly: start with the simplest tool that works.

## How trip2g does it: the fleet

### Roles as notes

An agent in trip2g is called a **role**. A role is a note in your vault like any other, but read by the fleet daemon instead of a human:

- **Frontmatter** = configuration: which model to use, which paths the agent may read (`read_patterns`), which it may write (`write_patterns`), what triggers it (`trigger_include`, `trigger_on`), budget limits (`max_tokens`, `max_steps`), timeout (`timeout_seconds`), concurrency policy (`concurrency`), and fan-out mode (`for_each`).
- **Body** = the instruction, rendered as a Jet template against four variables: `changed_files`, `change_file`, `attached_notes`, `depth`.

A minimal role note:

```yaml
---
model: gpt-4o-mini
tools: [read_note, write_note]
read_patterns: ["drafts/**"]
write_patterns: ["published/**"]
mode: change
trigger_on: [update]
trigger_include: ["drafts/**"]
max_tokens: 4000
max_steps: 6
concurrency: skip
---
You are an editor. Read the draft at {{ change_file.Path }}, rewrite it for clarity,
and write the result to published/{{ change_file.Title }}.md.
```

### Triggered by note changes

When you save a note that matches a role's `trigger_include` patterns, trip2g fires a change-webhook delivery to the fleet. The fleet verifies the HMAC signature, renders the role body as a Jet template against the trigger context, and runs the agent loop.

The fleet registers and maintains these webhooks automatically through a reconcile loop — you do not set up webhooks by hand. Drop a role note into the agents folder and the fleet registers the corresponding webhook on the next poll. Remove the note and the webhook is deregistered.

### The scoped tool loop

The agent has five tools:

| Tool | What it does |
|------|-------------|
| `search(query)` | Full-text search within the read scope |
| `read_note(path)` | Read a note's full content (read scope only) |
| `write_note(path, content)` | Create or replace a note (write scope only) |
| `patch_note(path, find, replace)` | Surgical find→replace in an existing note (write scope only); fails if `find` is absent or matches more than once |
| `finish(answer)` | End the run with a summary answer |

Every read and write is checked against `read_patterns` and `write_patterns` using doublestar glob matching. An out-of-scope request is rejected and the error is fed back to the model so it can adjust — denials are never silently swallowed. The model is told in its system prompt which patterns apply, and any attempt to go outside them is surfaced explicitly.

A role can restrict which tools are available at all via the `tools` frontmatter field. `finish` is always available regardless.

Writes happen **during the run**, not after: each `write_note` or `patch_note` call issues an immediate note update through the per-delivery scoped token. The fleet reports `changes: []` in the HTTP response because the writes are already applied by the time the response is sent.

### Safety controls

Three hard limits enforce the runtime budget. The model cannot override any of them:

- **Token cap** (`max_tokens`): if cumulative token usage reaches the cap before `finish` is called, the run stops with status `capped`.
- **Step cap** (`max_steps`): if the tool loop reaches the step limit without `finish`, the run stops with status `max_steps`.
- **Timeout** (`timeout_seconds`): the run context is cancelled and the loop stops. Default is 300 seconds.

The fleet applies a ceiling on top of the role's own settings: `effective = min(role.max_tokens, fleet.token_ceiling)`. A role author cannot request more than the fleet operator's configured maximum.

**Loop prevention** (`max_depth`): trip2g passes a `depth` counter with every delivery. When a role's write triggers a subsequent webhook delivery, the depth increments. If `depth >= max_depth`, trip2g drops the delivery without running it. Setting `max_depth: 1` on a role that writes to its own trigger path is enough to prevent self-retriggering entirely.

**Concurrency** (`concurrency: skip`): if a delivery arrives while a previous run for the same role is still in flight, the new delivery is dropped rather than queued. This collapses rapid note-edit bursts into a single agent run per role.

### Templating and fan-out

The role body is a Jet template. Four variables are available at render time:

| Variable | Contents |
|----------|----------|
| `changed_files` | All notes that triggered this delivery |
| `change_file` | The single triggering note (set only when `for_each: changed_files`) |
| `attached_notes` | Context notes pre-loaded via `attach_notes` glob patterns |
| `depth` | Current delivery depth (0 = top-level trigger) |

`for_each: changed_files` runs the agent once per changed file in the delivery, with `change_file` set to that file. `for_each: attached_notes` runs once per attached note. Without `for_each`, the agent runs once with the full lists.

Secrets are never exposed to the template. Referencing an undefined variable in a Jet template is a render error that stops the delivery before any LLM call is made.

### Worked example: the Krisp pipeline

The Krisp case shows how two chained roles turn raw call transcripts into a structured knowledge graph with wikilinks.

```
transcripts/<id>.md   raw transcript (written by ingest, never by the LLM)
   │  save fires change_webhook → transcript-segment role
   ▼
segments/<id>.md      topic map (Minto headlines + [MM:SS–MM:SS] time ranges)
   │  write fires change_webhook → transcript-wiki role
   ▼
wiki/<id>.md          knowledge note with [[WikiLinks]] into the vault graph
```

`transcript-segment` triggers on `transcripts/**`, reads the transcript from the delivery payload (`change_file.Content`), and writes a segmentation map to `segments/` using only the `write_note` tool.

`transcript-wiki` triggers on `segments/**`, reads the segment map (delivery payload) and the original transcript via `read_note` (covered by `read_patterns: ["segments/**", "transcripts/**"]`), and writes a wiki note with `[[WikiLinks]]` to `wiki/`.

Both roles use `for_each: changed_files` and `max_depth: 3`. The chain terminates because nothing triggers on `wiki/**`. In a validated run using `qwen/qwen3-14b` via OpenRouter: step 1 completed in ~7.6K tokens (2 steps), step 2 in ~11.2K tokens (2 steps).

The raw transcript is written by a deterministic ingest step — not the LLM. This keeps the source auditable and re-processable: when the prompt or model improves, re-run the roles against the same raw note.

### The code executor role kind

A second role kind runs Python, Bash, or Node code for steps that do not need a language model — pagination, field mapping, format conversion. A role picks it by naming a fleet backed by codellm, the code-execution service; the note format and the trigger mechanism are the same, without LLM cost or latency. The program runs in an OS-level sandbox, and its writes pass through the same `write_patterns` enforcement as `write_note`. See [[en/user/fleet|Fleet: the agent sidecar]] for the full reference and how to run the daemon.

## Honest limits

**Model reliability on multi-step loops.** Smaller or cheaper models sometimes fail to call `finish` correctly, ignore scope denials, or hallucinate paths. The step and token caps contain the damage, but they do not prevent it. Test roles on a cheap model first; move to a stronger one if results are unreliable.

**Cost.** Every step in the tool loop uses tokens. A role with `max_steps: 10` on a large model with a large context can cost meaningfully per delivery. Set conservative budgets until you have measured actual usage in your setup.

**It writes to your vault.** `write_note` replaces the entire note content. `patch_note` is surgical but requires a unique match string — if the string appears more than once, the call fails and the note is untouched. If a role has broad `write_patterns`, a confused model can overwrite notes you did not intend it to touch. Keep `write_patterns` as narrow as the task allows.
