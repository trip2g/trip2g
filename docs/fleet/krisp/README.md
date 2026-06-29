# Krisp case — transcripts → segments → wiki, on the fleet rails

**Status:** internal / unreleased. A worked use case that runs the fleet agent runtime on real-shaped data: a raw call/voice transcript becomes a topic-segmented map and then a knowledge note with `[[WikiLinks]]` into the vault graph.

## Pipeline (2 chained roles)
```
transcripts/<id>.md   raw transcript (deterministic ingest — source of truth, never the LLM)
   │  edit fires change_webhook (role: transcript-segment, trigger transcripts/**)
   ▼
segments/<id>.md      step 1: topic-boundary map (Minto headlines, [MM:SS–MM:SS])
   │  write fires change_webhook (role: transcript-wiki, trigger segments/**)
   ▼
wiki/<id>.md          step 2: knowledge note + [[WikiLinks]] (resolves into the trip2g graph)
```
Chain works because both roles set `max_depth: 3` (segment's depth-1 write is allowed to fire the wiki role); the chain terminates because nothing triggers on `wiki/**`. Both roles use `for_each: changed_files` (the bodies reference `change_file`).

## Design decision
Raw-source-saving is **deterministic ingest**, not the LLM: the raw transcript is preserved verbatim (auditable, exact, re-processable). The LLM only *processes* (segment, extract). Re-run a role on the preserved raw note when the prompt/model improves — no re-fetch from Krisp.

## Files
- `roles/transcript-segment.md` — step 1 (ported from `trip2g_agent_queue/krisp/prompts/segments.md`).
- `roles/transcript-wiki.md` — step 2 (ported/adapted from `idea_note.md`; reads the source transcript via `read_note` for detail).
- `transcripts/sample-call.md` — synthetic 3-min planning call (3 topic shifts) for cheap testing.
- `segments/sample-call.md`, `wiki/sample-call.md` — captured outputs from a real run.

## Validated run (2026-06-29)
Both steps run end-to-end on the fleet's own engine (`agentruntime`) via the `cmd/agent` offline harness, model **`qwen/qwen3-14b`** over OpenRouter:
- step 1: `completed`, ~7.6K tokens, 2 steps — 3 correct topic segments, Minto headlines.
- step 2: `completed`, ~11.2K tokens, 2 steps — knowledge note + 7 `[[WikiLinks]]` ([[Блог]], [[Контекстная реклама]], [[Сеньор-разработчик]], …).
Cost ≈ fractions of a cent total. (`cmd/agent` is the same engine as the fleet, without the webhook plumbing — see `docs/dev/fleet_run.md` for the full fleet run.)

## To run on the real fleet
Push `roles/*` and `transcripts/sample-call.md` into a trip2g instance (`roles/` is the fleet's `--agents-folder`), start `cmd/fleet` (see `docs/dev/fleet_run.md`) with `--default-model qwen/qwen3-14b --llm-base-url https://openrouter.ai/api/v1`, then edit the transcript → the chain runs and writes `segments/*` + `wiki/*`.

## Next
- Port `trip2g_agent_queue/krisp/extract_krisp_transcripts` → write real Krisp transcripts as `transcripts/*` (deterministic ingest).
- Quality compare Qwen-14B vs gpt-5.4-mini on the same input.
