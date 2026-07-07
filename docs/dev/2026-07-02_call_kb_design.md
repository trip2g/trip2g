# Call → Knowledge Base pipeline — design (2026-07-02)

Consolidated design for turning call transcripts (Krisp) into a linked, navigable trip2g knowledge base. Captures the decisions from the 2026-07-02 session so nothing is lost. Sample vault: `~/krisp-run/vault-out/`. Article: PR #102. Pipeline evidence: `~/krisp-run/2026-07-02_boundary_and_extraction.md`.

## Goal
Extraction INTO a knowledge base (linked concept graph + navigable notes), NOT summary + chat-with-a-bot. The KB is durable, auditable, and compounds over time.

## Note taxonomy
- **transcript** (`transcripts/YYYY-MM-DD_slug.md`) — raw Krisp transcript, verbatim, never edited. The source. `type: transcript`, `call_id` (ulid), `created_at` from the UUIDv7 id.
- **call / разбор** (`calls/YYYY-MM-DD_slug.md`) — the analysis. First paragraph = summary (magazine card). Links to its transcript (frontmatter `transcript:` + inline). `title_source: inferred` (Krisp's calendar title is unreliable — see below), `speakers`, `needs_review`.
- **daily** (`daily/YYYY-MM-DD.md`) — entry point of the pipeline. **Action-item checkboxes at the TOP** (today; "сделаю завтра" → next day's daily), **log below**, **no day summary**. Prev/next-day links.
- **log** (`log/topic.md`) — append-only journal of one recurring thought; each addition under a `### [[YYYY-MM-DD]]` heading; never rewrite, only append.
- **concept / term** (`concepts/slug.md`) — the glossary graph. `[[wikilinks]]`, `aliases`, `mentions` (unquoted int).

See [[project_secondbrain_note_taxonomy]] for the user's methodology this models.

## Provenance (invariant)
Every fact traces to the source. We have topic boundaries, so: **quote the raw segment verbatim** as a blockquote with a link + timerange to the transcript. The quote IS the evidence (zero hallucination in the evidence); the LLM разбор/summary sits above it. Applies to разбор topic sections and concept "Упоминания". Approximate segment-start time is fine; never fabricate exact times.

## Pipeline = event-driven fleet cascade
Standalone `~/krisp-run/build_vault.py` is the "run it yourself without fleet" path. The trip2g-native path is a chain of fleet role-notes wired by change-webhooks:
1. **Ingest** — python `executor: code` role pulls raw calls (Krisp API) → writes `transcripts/*.md`. The ONLY source-specific stage.
2. change-webhook on `transcripts/**` → **topic segmentation** (mini-V2 coarse boundaries) writes segments.
3. segments written → **extraction/distribution (растаскивание)** writes concept notes, appends logs, writes daily action-checkboxes, and the разбор note linking back to the transcript.

Drop a raw transcript → the KB assembles itself. Dogfoods change-webhooks + exec:code. See [[project_kb_construction_engine]], [[project_agent_runtime]].

## Authoritative metadata (Krisp is unreliable)
- **Time**: decode the UUIDv7/ULID id (`int(id[:8],16)<<16` ms), NOT the local clock (user's clock is wrong) and NOT Krisp's calendar match (it picks the wrong event by time — documented-fragile, purely time-based). Drives `created_at` (RFC3339, the key the magazine sorts on) + daily bucketing.
- **Title**: prefer LLM-inferred topic title (`title_source: inferred`); Krisp's calendar title is actively wrong for this user.
- **Speakers**: Krisp API participant list; fall back to inference; `needs_review: true`.

## Entity resolution (the hard part)
- **Specificity gate**: concept notes only for proper nouns or terms that recur with a stable meaning. Lowercase generic descriptions ("мероприятие про агентов") stay as prose or resolve to a real named entity — they do NOT become notes.
- **Referent-aware reconcile** (not name-only): on a surface-name match, compare the new mention's context to the existing note's definition. Same referent → append. Different referent → mint a NEW note with a disambiguating title (`... (HR, 2026-06-19)` vs `... (маркетинг, 2026-06-27)`). Never silently merge. This is the fix for the observed over-merge (Claude/Codex → ChatGPT).
- **Stable slug + alias table** so links survive re-titling/disambiguation.
- **needs_review** on low-confidence merges. Identity = note id/content, not the surface title (Zettelkasten).
- Drift within one note is fine (evergreen append) as long as it's the same referent; the bug is false merges, fixed by the above.

## Navigation
Calls magazine (home), terms magazine, by-day (daily notes), `_header`/`_footer`. Magazine mode = `content: [magazine]` + `magazine_include_files` glob; date key is `created_at` (RFC3339), not `date`. See `docs/dev/2026-07-02_call_magazine_layout.md`.

## Maintenance fleet roles (future)
- **link-healer** — reads note warnings (GraphQL `AdminNoteWarnings` / `trip2g lint`), fixes broken `[[wikilinks]]` (create missing stub or repoint) via `patch_note`; `needs_review` when ambiguous.
- **concept-gardener** — cron role that finds notes conflating two referents (contradictory definition / diverging mentions) and splits them.

## Cost
~$0.13–0.18 per 50-min call, all gpt-5.4-mini. Boundaries mini-V2 coarse. Re-runs reuse cached extractions (`work-vault/`) at ~$0.

## Status
Sample vault at `~/krisp-run/vault-out/` (8 calls); served locally via `memcli up`. Article PR #102. Reconcile hardening (specificity gate + referent-aware + disambiguation) and the maintenance roles are designed here but not yet implemented.
