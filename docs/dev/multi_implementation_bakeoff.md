# Multi-implementation bake-off: finding optimal solutions from chaos

## TL;DR / Вывод

Building the *same* feature several independent ways — one carefully **planned + tested**, others **single-shot, code-only** by different models — and then **diffing the results** is a powerful way to find both the best design *and* the bugs any single approach missed. Diversity of attempts + a comparison/selection step is a **search over the solution space**: the optimum emerges from the "chaos" of varied implementations. In this project the 3-way diff found **3 security holes + 1 functional gap in our own already-reviewed-and-hardened implementation** that its own review pass had missed.

## The experiment (this case: the agent runtime)

Same baseline (`d929991c` + a shared Phase-1 loop) and the **same spec** (`agent_runtime_design.md`), three independent implementations:

1. **Planned / orchestrated** — multi-agent TDD: brainstorm → design → 33-task plan → sectioned execution (A/B/C) → a holistic 3-dimension adversarial review → a 9-fix hardening pass. **Has tests.** (`feat/agent-runtime` — the base we ship.)
2. **Codex CLI** — a single autonomous run, **code-only, no tests** (`feat/agent-runtime-codex`).
3. **A single Opus agent** — one run, **code-only, no tests** (`feat/agent-runtime-opus`).

Then a **seam-by-seam 3-way comparison** (core loop, jsonnet/transform, concurrency, scope-enforcement, fleet, attach/attribution/migrations) picked a winner per seam and produced a prioritized "what to graft" list.

## Why it works (the insight)

- **Different approaches make different mistakes.** Diffing N implementations of one spec surfaces defects *by disagreement*: where they differ, at least one is wrong — and frequently the eventual winner is wrong too, in a way only visible against an alternative.
- **A second independent implementation is a different reviewer.** Same-lineage review (the author re-reading, or an adversarial pass on *one* tree) shares the author's blind spots. An independently-built tree catches what same-lineage review structurally cannot. Here it caught a `similarNotes` read-scope leak, a read-scope fail-open on empty patterns, a `resolveWikilinks` leak, and a dropped cron `attach_notes` — none of which our own design + review + hardening had found.
- **Plan vs no-plan both contribute.** The planned/TDD/hardened build won the *expensive-to-reverse* seams (concurrency race-safety, secret redaction) and is the only **tested** tree → it is the right **base**. The single-shot builds were faster and produced cleaner **local abstractions** in places (a shared materialize helper, a `canreadnote` enforcement chokepoint, an explicit `WebhookScoped` flag) worth **grafting in**. Neither alone is optimal; the optimum is **base + grafts**.
- **Optimal from chaos = generate-and-select.** Producing diverse candidates (the "chaos") plus a comparison/selection step is an optimization process — an ensemble / tournament. It costs more (≈3× the build) but buys correctness and design quality a single pass cannot.

## When it's worth it

- High-stakes, **security-sensitive**, or hard-to-reverse features where a missed bug is expensive (an isolated agent runtime holding scoped credentials qualifies).
- When parallel capacity is cheap (multiple models/agents) and the **spec is stable**.
- **Not** for small or throwaway work — the ≈3× cost dominates.

## The recipe (repeatable)

1. Freeze a **baseline commit + a spec**.
2. Build it **≥2 independent ways**: at least one *planned + TDD + reviewed* (the likely base), and **≥1 single-shot by a *different* model** (the independent "reviewer"). Isolate each in its own **git worktree**.
3. **Diff seam-by-seam**; per seam pick the winner + note what to graft.
4. Use the diff as a **bug-finder on the winner** — disagreement between trees is a "suspect" flag; verify each claimed bug against the real code before acting.
5. **Graft the clear wins** into the base with TDD; ship the base.

## Caveats & a real gotcha

- The single-shot builds had **no tests** by design — judge their *code/design*, not completeness.
- The comparison is itself an LLM judgment — **verify its claimed bugs against the source** before acting on them (we did; they were real).
- **Tooling can break isolation:** the Codex *companion* plugin re-rooted Codex to the *session* worktree and ignored the per-task `--cd`, nearly writing into our active worktree. Run `codex exec --cd <worktree>` **directly** (not via the companion), and verify the agent's process `cwd` before trusting it.
- **Cost:** this case ran the feature ≈3× + a comparison + a graft pass. Justified for a security-sensitive runtime; overkill for routine work.
