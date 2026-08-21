---
title: "Fleet: the agent sidecar"
free: true
lang_redirect: "[[ru/user/fleet]]"
---

Fleet is a small daemon that runs next to your trip2g hub and turns notes into agents. You write a role note: frontmatter says when to run and what the agent may touch, the body is the instruction. Drop it into the agents folder and the agent exists. Edit a matching note in your vault and the agent runs. Delete the role note and the agent is gone. No pipeline config, no YAML repo, no redeploys: the vault is the control plane.

If you want the theory first (what an LLM tool loop is, why scope enforcement has to live in the runtime), read [[en/agents_how_it_works|How LLM agents work]]. This page is the practical side: what fleet does, and a full pipeline built on it.

In this article:

- [What fleet does](#what-fleet-does)
- [Roles are notes](#roles-are-notes)
- [Triggers: on change and on schedule](#triggers-on-change-and-on-schedule)
- [Tools and scope](#tools-and-scope)
- [Code roles](#code-roles)
- [Fan-out and templating](#fan-out-and-templating)
- [Budgets, spend, and shutdown](#budgets-spend-and-shutdown)
- [Worked example: a knowledge base that builds itself](#worked-example-a-knowledge-base-that-builds-itself)
- [Running fleet](#running-fleet)

## What fleet does

Fleet is a sidecar: a separate process (`cmd/fleet` in the trip2g repo) that connects to a hub over its normal API. The hub stays a dumb event source; fleet does the thinking. On startup and every 30 seconds after, fleet scans the agents folder (default `roles/`), parses each role note, and registers the matching webhooks on the hub for itself. When a note in your vault changes, the hub delivers the event to fleet, fleet runs the agent, and the agent's writes land back in the vault as ordinary note versions.

Two kinds of agents run under one roof:

- **LLM roles** (the default): a model in a tool loop that searches, reads, and writes notes until it finishes or hits a budget.
- **Code roles**: a Python, Bash, or Node program in a sandbox, for deterministic steps that need no model.

Because agents read and write plain notes, anything that renders notes doubles as an agent UI. A [[en/user/kanban|kanban board]] over task notes is a live control surface: an agent moves a card by editing frontmatter, and you see it move.

## Roles are notes

A role note is frontmatter plus body. The frontmatter is the full agent config:

```yaml
---
model: gpt-4o-mini
tools: [read_note, write_note]
read_patterns: ["drafts/**"]
write_patterns: ["published/**"]
mode: change
trigger_on: [update]
trigger_include: ["drafts/**"]
for_each: changed_files
max_tokens: 4000
max_steps: 6
max_depth: 2
concurrency: skip
---
You are an editor. Read the draft at {{ change_file.Path }}, rewrite it
for clarity, and write the result to published/{{ change_file.Title }}.md.
```

The keys, verified against the runtime:

| Key | Meaning |
|-----|---------|
| `fleet_id` | which fleet claims this role. It also picks the kind of run: a fleet pointed at codellm executes the body as code, one pointed at a model runs it as prose. A role with no `fleet_id` is claimed by nobody and never runs |
| `model`, `tools` | which model, which tools this role may call |
| `read_patterns`, `write_patterns` | glob scopes the runtime enforces on every read and write |
| `mode` | `change`, `cron`, or `both` |
| `trigger_include`, `trigger_exclude`, `trigger_on` | which note paths and events (`create`, `update`, `remove`) fire the role |
| `cron_schedule` | cron expression for scheduled roles |
| `attach_notes` | glob of notes pre-loaded into the delivery as context |
| `for_each` | fan out: one run per `changed_files` item or per `attached_notes` item |
| `max_tokens`, `max_steps`, `timeout_seconds` | per-run budget (timeout defaults to 300s) |
| `max_depth` | cascade depth limit; loop protection |
| `concurrency` | `skip`, `queue_one`, or `allow_overlap` when deliveries pile up |
| `env_passthrough`, `env_prefix` | code roles only: which env vars the code receives, narrowing what the codellm operator allowlisted |
| `unseal`, `unseal_env_key` | code roles only: which frontmatter fields hold sealed secrets, and which key opens them |

Fleet validates every role at discovery time and refuses bad ones out loud: a role that declares a tool the fleet does not offer, a change role with empty triggers, a body that references `change_file` without fanning out over changed files. Misconfiguration fails at poll time, not silently at 3 a.m.

## Triggers: on change and on schedule

**Change triggers** are the core loop: notes trigger agents. Save a note matching `trigger_include`, and the hub fires a [[en/user/webhooks|change webhook]] delivery to fleet. Fleet verifies the HMAC signature, renders the role body against the trigger context, and runs the agent. Because one role's writes can match another role's triggers, roles chain into cascades. The `depth` counter and `max_depth` cap keep cascades from becoming loops.

**Cron triggers** run roles on a schedule via [[en/user/webhooks|cron webhooks]]: a nightly digest, a weekly link-checker, a poller that ingests an external source. `mode: both` combines the two in one role.

You never register a webhook by hand. Fleet's reconcile loop creates, updates, and removes them on the hub to match the role notes it finds.

## Tools and scope

An LLM role gets five tools:

| Tool | What it does |
|------|-------------|
| `search(query)` | full-text search within the read scope |
| `read_note(path)` | read a note (read scope only) |
| `write_note(path, content)` | create or replace a note (write scope only) |
| `patch_note(path, find, replace)` | surgical find and replace; fails unless `find` matches exactly once |
| `finish(answer)` | end the run with a summary |

Every call is checked against `read_patterns` and `write_patterns` at the runtime level, not in the prompt. An out-of-scope request is denied and the denial is fed back to the model. Writes go through a per-delivery scoped token the hub mints for that one delivery, so the agent physically cannot reach paths outside its scope, and every write is a normal note version you can inspect and revert.

## Code roles

Not every step needs a model. Pagination, format conversion, pulling an external API: these are deterministic, and running them through an LLM adds cost and noise. Point the role at a fleet backed by codellm and the body runs as a program instead:

````markdown
---
fleet_id: codellm
mode: cron
cron_schedule: "*/30 * * * *"
write_patterns: ["transcripts/**", "logs/**"]
---
```python
import json, fleetkit

cfg = fleetkit.frontmatter()
# fetch new items, then emit the writes:
fleetkit.emit([fleetkit.note("transcripts/2026-07-02_standup.md", {"title": "Standup"}, "...")])
```
````

There is no `executor` key. The role names a fleet, the fleet decides what kind of run it is, and the same note can move between them by changing one line.

The body's first fenced code block is the program (`python`, `bash`, or `node`). The delivery context arrives as JSON in the file named by `$FLEET_INPUT` — `fleetkit.bag()` reads it, and `fleetkit.frontmatter()` gives you the role note's own frontmatter, so a role's configuration lives in the note that declares it. The program prints a `{"changes": [...]}` object to stdout, and every change is applied through the same `write_patterns` enforcement as `write_note`. Code execution is not a scope bypass.

Isolation is strict by default, and it is codellm's, not the fleet's: the child runs in an OS-level sandbox (Linux namespaces plus Landlock filesystem confinement, no network unless the operator allows it) that fails closed on unsupported systems. Code execution as a whole is off until the codellm operator enables specific interpreters.

A code role that needs a token reads it from the run environment, which the codellm operator fills, or carries an encrypted value in its own frontmatter. Both paths are covered in [[en/user/codellm-secrets|Secrets for code roles]].

## Fan-out and templating

The role body is a Jet template rendered per delivery. Four variables are available: `changed_files` (the notes that triggered the delivery), `change_file` (the single current note when fanning out), `attached_notes` (context pre-loaded via `attach_notes`), and `depth`. Secrets never enter the template, and referencing an undefined variable stops the delivery before any model call.

`for_each: changed_files` runs the agent once per changed note; `for_each: attached_notes` once per attached note. A batch of ten edits becomes ten focused runs instead of one confused run, and one item failing does not abort the rest.

## Budgets, spend, and shutdown

Three hard limits bound every run: `max_tokens`, `max_steps`, and `timeout_seconds`. The model cannot raise any of them, and the fleet operator's ceilings (`--token-ceiling`, default 100,000; `--step-ceiling`, default 25) cap whatever a role asks for: the effective budget is the minimum of the two.

Spend is attributed, not guessed. Every delivery response reports `tokens_used` and `steps`, and the hub records them in its webhook delivery logs, so you can see which role spent what. The writes themselves are note versions with history.

Operationally, fleet shuts down politely: on SIGTERM it stops accepting deliveries, drains in-flight runs for up to `--shutdown-grace-seconds` (default 30), and deregisters its webhooks. For rolling deploys, `--keep-webhooks-on-shutdown` leaves the webhooks in place and lets the hub retry deliveries until the new fleet instance is up.

## Worked example: a knowledge base that builds itself

The clearest fleet pipeline in production use turns raw call transcripts into a linked knowledge base. The full result is described in [[en/user/calls-knowledge-base|Calls into a knowledge base]]; here is the same pipeline seen as fleet roles.

```
Krisp API
   │  cron: code role (ingest)
   ▼
transcripts/<date>_<slug>.md    raw transcript, verbatim, never edited
   │  change webhook on transcripts/** → segmentation role
   ▼
segments/<id>.md                topic map with time ranges
   │  change webhook on segments/** → extraction role
   ▼
calls/, concepts/, log/, daily/  the knowledge base assembles itself
```

Each stage is one role note:

1. **Ingest** is a code role on a cron schedule. It pulls new calls from the Krisp API and writes raw transcript notes. This is the only source-specific stage: swap the script and the same downstream pipeline processes YouTube captions or meeting bot output instead.
2. **Segmentation** is an LLM role triggered by `transcripts/**`. It reads the transcript from the delivery payload and writes a coarse topic map with time ranges.
3. **Extraction** is an LLM role triggered by `segments/**`. It writes the call note, mints or appends concept notes, adds action checkboxes to the daily note, and appends to topic logs. Every claim quotes the transcript segment verbatim, so facts trace to the source.

Nobody invokes anything. A transcript lands and the change-webhook cascade does the rest; `max_depth` bounds the chain and nothing triggers on the output folders, so it terminates. Cost sits around $0.15 in model calls for a 50-minute call. The pipeline lives at [github.com/trip2g/krisp_knowledge](https://github.com/trip2g/krisp_knowledge).

The pattern generalizes: an ingest code role per source, plus source-agnostic LLM roles chained by change webhooks. That is fleet's shape for any "raw input in, structured knowledge out" problem.

## Running fleet

Fleet ships in the trip2g repo as a standalone binary:

```bash
go build -o fleet ./cmd/fleet

./fleet \
  --trip2g-url https://your-hub.example \
  --callback-url http://fleet-host:9090 \
  --trip2g-admin-personal-token "<hub personal token>" \
  --fleet-secret "<any random seed>" \
  --llm-base-url https://openrouter.ai/api/v1 \
  --llm-api-key  "<key>" \
  --agents-folder roles/
```

Every flag has a `TRIP2G_FLEET_<FLAG>` environment variable equivalent. `--llm-base-url` takes any OpenAI-compatible endpoint, including a local model — or a codellm instance, which is what makes this fleet run code bodies instead of prose.

Sandboxing and the interpreter allowlist are codellm's settings, not fleet's. Fleet runs no code itself.

The `--trip2g-admin-personal-token` is fleet's only credential on the hub: a [[en/user/mcp|personal token]] belonging to an admin. You do not mint it by hand — set `OWNER_PERSONAL_TOKEN_VALUE` on the hub to a `t2g_` value and it seeds the token for the owner at boot, then pass the same value to fleet. Fleet never holds the hub's signing secret, so the credential is one revocable row: revoke it in the admin UI and fleet stops within half a minute, without restarting the hub. A restart will not bring a revoked row back — restoring access means a new value.

Two switches make roles cheap to develop:

- `--dry-run` discovers and validates every role, prints the resolved config, and exits without registering anything.
- `--once role.md --vault ./my-vault` runs a single role against a local folder with no hub and no webhooks: edit, run, inspect the output files, repeat.

Start with one role, a cheap model, and tight `write_patterns`. Widen from there once the runs look right.
