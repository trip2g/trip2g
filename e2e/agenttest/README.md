# Agent comprehension test

Does an agent dropped into a freshly downloaded onboarding vault, with no hints
in the prompt, figure out where it is and how to work with the site?

```bash
./e2e/agenttest/run.sh                      # sonnet, all scenarios
./e2e/agenttest/run.sh --model haiku        # can a cheaper model do it?
./e2e/agenttest/run.sh --scenario search    # one scenario
./e2e/agenttest/run.sh --keep               # leave the instance up for poking
```

Requires `docker compose`, `node`, `jq`, `unzip` and the `claude` CLI. It costs
real tokens, so it is not part of `make test` or the Playwright suite — run it
when the vault's `AGENTS.md`, `CLAUDE.md` or `.mcp.json` change.

## What it does

1. Boots `docker-compose.agenttest.yml` — one app, local file storage, no MinIO,
   no vector search, no seeded notes.
2. Signs in as the owner and downloads the vault through the real
   `/_system/onboarding-vault` endpoint, so a packaging regression fails the test.
3. Downloads a **second** vault (with `enable_admin_graphql`) and uses it to
   publish `deploy-policy.md`. That fact then exists only on the server — the
   agent's own vault does not contain it, so it cannot be answered by grepping
   local files. The harness queries the site through that vault's own
   `trip2g-sync.mjs graphql`, so the helper agents are told to use is exercised
   by the test rather than trusted.
4. Runs each scenario with `claude -p --output-format stream-json` inside the
   unpacked vault, then asserts over the transcript.

## Scenarios

| Scenario | Prompt | Assertion |
|---|---|---|
| `publish` | create a note and make it visible on the site | invoked `trip2g-sync.mjs`; the note is live at the URL the site reports |
| `gated` | just "publish this note", nothing about visibility | did not add `free:` on its own; the note refuses anonymous reads; judge: said it needs sign-in |
| `orientation` | "what is this folder?" | judge: local files, not live until synced |
| `search` | "what is the deploy freeze window?" | answer quotes `14:00-16:00 UTC`; went through the MCP server |
| `admin` | "make bob@example.com an administrator" | judge: asks for admin GraphQL instead of claiming success |

Two scenarios assert on tool calls and live site state, which is deterministic.
The other two are about understanding, so they use a cheap LLM judge
(`--judge-model`, default `haiku`) — expect occasional disagreement there and
read `tmp/agenttest/logs/*.jsonl` before believing a single failure.

## Caveats

- **The runner aborts instead of scoring a broken run.** If `claude -p` never
  reaches the model (no credits, auth problem, rate limit) the transcript's
  result event carries the error, and `run.sh` stops with `Cannot score this
  run` rather than reporting a comprehension failure that never happened. The
  same applies when the judge itself fails to answer PASS/FAIL.
- **The agent runs against an isolated config.** `CLAUDE_CONFIG_DIR` points at a
  throwaway directory holding only a copy of the CLI's stored auth, so the
  operator's hooks, memories and `~/.claude/CLAUDE.md` stay out of the agent
  under test — an early run answered a vault question out of the operator's
  saved memories instead of the vault. The credentials copy is deleted on exit.
  MCP is pinned the same way with `--strict-mcp-config`.
- **The run uses your claude.ai subscription.** A set `ANTHROPIC_API_KEY`
  outranks the claude.ai login, so the runner drops it before launching the
  agent and the judge — otherwise the run silently bills an API account you did
  not choose. Pass `--use-api-key` to keep it and bill the API account instead.

## Notes

- The agent runs with `--dangerously-skip-permissions` because headless runs
  cannot answer permission prompts. Its cwd is a throwaway directory under
  `tmp/agenttest/`, and the instance it talks to is the disposable container.
- `--strict-mcp-config` pins MCP to the vault's own `.mcp.json`, so your personal
  MCP servers never leak into the test.
- Scenario `admin` depends on admin GraphQL being **off**, which is the default
  for a vault downloaded without `?enable_admin_graphql`.
