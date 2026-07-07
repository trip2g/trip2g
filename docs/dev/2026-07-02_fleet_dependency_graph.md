# Fleet Dependency Graph (design)

**TL;DR.** Derive a static agent-dependency graph from role notes: edge A→B when A's
`write_patterns` can intersect B's trigger/read scope (glob-overlap decided by a small exact
pattern-intersection algorithm over the doublestar subset we use). Build it as a `fleet graph`
subcommand in `cmd/fleet` — trip2g is the single source of truth (all role notes across every
fleet + the webhook registry with `fleet:<id>:` markers), so one command sees every fleet.
Output: a `_fleet-graph.md` note with a Mermaid diagram (trip2g renders it natively — "see the
graph" = open a note) + machine JSON (satisfies task #29 "Fleet graph + internal API for UI,
JSON, localhost-only"). The same graph doubles as the multi-fleet guardrail: nodes colored by
owning fleet; roles claimed by 0 or >1 fleets, cycles, self-triggers, orphan writes, and
desired-vs-registered drift are flagged. v1 = static note-graph over disjoint `AgentsFolder`s;
v2 = `fleet:` frontmatter selector + liveness overlay + localhost `/graph.json`.

Status: design, not built. Grounded in `internal/fleet/{role,discovery,reconcile}.go` on
`feat/agent-runtime` and `internal/webhookutil/patterns.go`.

---

## 1. What the graph is

Nodes = roles (one per role note). Edges = "A's output can activate or feed B":

| Edge kind | Condition | Meaning |
|---|---|---|
| **trigger** (solid) | `overlap(A.write_patterns, B.trigger_include)` and `B.trigger_on ∩ {create, update} ≠ ∅` and B is `mode: change\|both` | A's writes fire B's change-webhook |
| **feed** (dashed) | `overlap(A.write_patterns, B.read_patterns)` and no trigger edge A→B | A writes notes B reads as context (incl. cron roles, which have no trigger edges — they are time-driven but still consume state) |

Everything comes from parsed `Role` fields (`role.go`): `WritePatterns`, `TriggerInclude`,
`TriggerExclude`, `TriggerOn`, `ReadPatterns`, `Mode`, `ForEach`, `Concurrency`, `MaxDepth`,
`CronSchedule`. No runtime data is required for v1 — the graph is note-derived, which is the
point: you can validate a fleet topology before any agent runs.

### Facts the derivation relies on (from code)

- Delivery matching is `doublestar.Match(pattern, path)` per changed path
  (`webhookutil.MatchesAny`, used by the change-webhook pipeline). So "A can fire B" is exactly
  "∃ path matching both A's write glob and B's include glob" — a glob-intersection question.
- Writes an agent performs are creates/updates (`updateNotes` via the scoped lane). A role with
  `trigger_on: [remove]` only gets a trigger edge from A if A can delete notes; v1 assumes
  agents don't delete → no remove-edges. Revisit if a delete tool ships.
- `trigger_exclude` narrows B's scope. Deciding "the exclude fully subsumes the include∩write
  intersection" is glob containment — more machinery than it is worth. v1 rule: if
  `overlap(A.write, B.trigger_exclude)` also holds, keep the edge but annotate it
  `maybe-excluded` (dotted label). Conservative: never hides a real edge.

## 2. Glob-overlap semantics (`Overlap(p, q) bool`)

Question: can two doublestar patterns match a common path? Requirements: **sound toward true**
(a spurious edge is a review nuisance; a missing edge hides a live trigger chain), exact for the
subset we actually use, cheap (patterns are tiny).

Supported subset (everything `parseList` can carry — note it splits on spaces/commas, so
patterns never contain spaces): literals, `*`, `?`, `[...]` classes, `**` segments, `{a,b}`
braces. Algorithm:

1. **Brace expansion.** `expandBraces(p)` → list of brace-free patterns.
   `Overlap(P, Q)` = any expanded pair overlaps.
2. **Segment recursion.** Split both on `/`. `rec(i, j)`, memoized on `(i, j)`:
   - both exhausted → `true`;
   - `p[i] == "**"` → `rec(i+1, j) || (j < len(q) && rec(i, j+1))` (`**` matches zero or more
     segments); symmetric for `q[j]`;
   - one exhausted → `true` iff the other side has only `**` segments left;
   - otherwise → `segOverlap(p[i], q[j]) && rec(i+1, j+1)`.
3. **Within-segment intersection.** `segOverlap(a, b)`, memoized on positions:
   - both empty → `true`;
   - `a` head `*` → `segOverlap(a[1:], b) || (b ≠ "" && segOverlap(a, b[1:]))`; symmetric;
   - otherwise compare heads as character sets (`?` = universal, `[...]` = the class, literal =
     singleton): sets intersect → recurse on both tails.
4. **Unknown construct** (future doublestar syntax we don't model) → return `true` with a
   diagnostic. Conservative by design.

This is a product-NFA emptiness check specialized to globs — exact for the subset, worst-case
exponential only in pathological patterns that never occur in role frontmatter (memoization
makes realistic inputs linear-ish).

Examples: `Overlap("roles/*", "transcripts/**")` = false (first segments `roles` vs
`transcripts` don't intersect). `Overlap("wiki/**", "wiki/topics/*.md")` = true.
`Overlap("logs/*.md", "logs/run-[0-9]*.md")` = true.

### `for_each` fan-out

`for_each` (`changed_files` / `attached_notes`) does not change edge existence — the trigger
scope is the same webhook; fan-out multiplies *runs per delivery*, not reachability. In the
graph it is an **amplification annotation**: the node gets a fan-out badge, and every outgoing
edge from a fan-out node is labeled `×N` (unbounded multiplier). This matters for cycle
severity (below): a cycle through a `for_each: changed_files` + `allow_overlap` node can
multiply writes each lap even though depth is bounded.

### Cycle detection

Directed graph over **trigger edges only** (feed edges don't cause runs). Tarjan SCC:

- **Self-trigger**: `Overlap(A.write_patterns, A.trigger_include)` — the most common footgun,
  flagged even when depth would stop it.
- **Cycle**: any SCC with >1 node, or a node with a self-loop.
- **Severity vs the depth guard.** `max_depth` already bounds every chain: a webhook is skipped
  when incoming `depth >= max_depth`, and `pass_api_key` mints the next token with `depth+1`
  (see `change_webhooks.md`). So no cycle is infinite — but a cycle still burns budget up to the
  depth ceiling, and fan-out amplifies within it. Report per cycle:
  `lap_budget = min(max_depth over members)` (default 1, per reconcile `MaxDepth`), plus flags
  for `for_each` members and `allow_overlap` members.
  - `lap_budget == 1` and no fan-out → **WARN** (structurally a loop, practically fires once).
  - `lap_budget > 1`, or any member has `for_each` + `allow_overlap` → **ERROR** (real
    token-burn risk; this is the "catch structure ahead of time" complement to the depth cap
    called out in agent.md Phase 4).

## 3. Where it's built and how it's shown

**Build from trip2g, ship as a `fleet graph` subcommand.** trip2g is the only place that has
the complete picture: every role note of every fleet (they are all just notes) **and** the
webhook registry, where ownership is already encoded in reconcile markers
(`fleet:<FleetID>:<notePath>#<specVer>` and `fleetcron:<FleetID>:...`, `reconcile.go`). A fleet
daemon only knows its own `AgentsFolder`; the graph must not.

Not a third binary: `cmd/fleet` grows subcommand dispatch (`fleet graph [flags]`; bare
invocation stays the daemon for compatibility, `--once`/`--dry-run` keep working). The
subcommand:

1. **Reads the registry** over the existing admin lane (`trip2ggql.ListChangeWebhooks` /
   `ListCronWebhooks`), parses markers → `{fleetID, notePath, specVer, webhookID, url}`. This
   yields (a) the set of role-note paths that are actually live, (b) which fleet owns each, and
   (c) each fleet's callback base (webhook `url` minus `/deliver/<key>`), which is how the
   liveness probe finds `/health` without extra config.
2. **Discovers role notes** via `notePaths(filter: {like})` + meta, reusing
   `Discovery.DiscoverParsed` generalized to take a list of folders. v1 folder list: the union
   of `--agents-folder` flags plus folders inferred from registry marker paths (so the graph
   sees fleets you didn't list). Parse with `ParseRole`; run `Validate`-level checks but keep
   invalid roles as flagged nodes (like `--dry-run` does) — a broken role is exactly what the
   graph should show.
3. **Derives** nodes, edges, cycles, and conflicts (§2, §4–5).
4. **Emits**:
   - `_fleet-graph.md` — a note written via the admin lane (`updateNotes`), containing a
     Mermaid `flowchart LR` + a findings table. trip2g renders Mermaid natively
     (`docs/dev/mermaid.md`), so viewing the graph = opening a note in any browser, no UI work.
   - `--json <path|->` — the machine graph (schema below).
   - `--probe` — GET each fleet's derived `<callback-base>/health` (exists today,
     `cmd/fleet/main.go:93`) and overlay liveness: fleet swimlane header gets `up`/`down`,
     answering "who sits where, and is it running".

Mermaid conventions:

- node label: role name + trigger badge (`⚡` change / `⏰` cron / `⚡⏰` both) + `for_each` badge;
- `classDef` per fleet (color = owning fleet), a distinct class for **conflict** (0 or >1
  owners) and **invalid** (parse/validate error);
- solid arrows = trigger edges, dashed = feed edges, `×N` label on fan-out edges,
  `maybe-excluded` label where `trigger_exclude` overlaps;
- pure entry points (trigger scope nobody writes into — human-edited folders) get an implicit
  `human` source node so the flow reads left-to-right.

JSON schema (stable, for the future admin UI):

```json
{
  "generated_at": "...",
  "fleets": [{"id": "krisp", "callback": "http://...", "health": "up|down|unknown"}],
  "nodes": [{
    "note_path": "roles/segmenter.md", "fleet_ids": ["krisp"],
    "mode": "change", "for_each": "changed_files", "concurrency": "allow_overlap",
    "max_depth": 1, "registered": true, "valid": true, "errors": []
  }],
  "edges": [{"from": "...", "to": "...", "kind": "trigger|feed",
             "via": {"write": "wiki/**", "match": "wiki/topics/*"},
             "fanout": true, "maybe_excluded": false}],
  "findings": [{"severity": "error|warn|info", "kind": "cycle|conflict|orphan-write|drift|...",
                "nodes": ["..."], "detail": "..."}]
}
```

**Task #29 fit** ("Fleet graph + internal API for UI, JSON, localhost-only"): the JSON emitter
is the internal API payload; v2 additionally serves it from the fleet daemon's existing mux as
`GET /graph.json` (same `ListenAddr`, which deployments already bind to localhost — no auth
added, no new port). The Mermaid note covers the "UI" half with zero frontend work; a real
admin view later consumes the same JSON.

## 4. Multi-fleet partition

Multiple fleets against one trip2g already work — ownership is the marker prefix, and
`Reconcile` never touches foreign markers. What is *not* guarded is role assignment:

- **v1 rule — disjoint `AgentsFolder`s.** Each fleet discovers `LIKE folder%`
  (`discovery.go:likePattern`). Folders MUST be pairwise non-overlapping (no prefix of
  another), or two fleets parse the same role note and register two webhooks for it =
  duplicate execution. Nothing enforces this today; the graph does (§5). One fleet per role;
  a role note outside every configured folder is simply not a role.
- **v2 rule — frontmatter selector.** `fleet: krisp` in role frontmatter; discovery becomes
  "notePaths in scope + meta filter `fleet == cfg.FleetID`" instead of pure path prefix. A role
  then lives wherever it makes sense semantically (next to the data it tends), assigned by
  meaning, not by folder. This rides the planned discovery-selector / lang-aware `notePaths`
  direction: the natural implementation is a meta-key filter on the `notePaths` query
  (`NotePathsFilter` gains `metaKey/metaValue` or similar) so selection happens server-side,
  not by fetching everything. Rules: exactly **one fleet per role** (`fleet:` is a scalar, not
  a list); **untagged = ignored** (not picked up by any selector-mode fleet; a fleet may opt in
  as `--default-fleet` to claim untagged roles under its folder for migration). v1 and v2
  coexist: a fleet runs in prefix mode or selector mode; the graph reads both.

## 5. Conflict + validation via the graph

The graph is the "are two fleets running the same thing" guardrail. Findings, computed from the
same derivation pass:

| Finding | Rule | Severity |
|---|---|---|
| **Ownership conflict** | role note matched by >1 fleet (overlapping folders in v1; duplicate `fleet:` claims can't happen in v2, but folder-mode + selector-mode overlap can) — detected both statically (folder prefix overlap) and from the registry (same `notePath` in markers of two `FleetID`s) | ERROR |
| **Unclaimed role** | parseable role note owned by 0 fleets (folder not covered / untagged in selector mode) | WARN |
| **Cycle / self-trigger** | §2, with `lap_budget` + fan-out severity rule | WARN/ERROR |
| **Orphan write** | some `write_patterns` entry overlaps no one's `trigger_include` or `read_patterns` | INFO (often fine — logs, human-facing output; the note says so) |
| **Dangling trigger** | `trigger_include` no role writes into | INFO, rendered as a `human` entry point, not an error |
| **Drift** | registry marker with no matching current role (stale webhook — fleet down or `KeepWebhooksOnShutdown` leftovers), or valid role with no marker (fleet hasn't reconciled it) | WARN |
| **Invalid role** | `ParseRole`/`Validate` error (same checks as `--dry-run`) | ERROR node, kept in the graph |

Coloring: fleet color per node; conflict nodes red regardless of fleet; drift-only markers
shown as ghost nodes.

## 6. Change surface + phasing

New code, all read-only against trip2g except the one `_fleet-graph.md` note write:

| Piece | Where | Notes |
|---|---|---|
| Glob intersection | `internal/fleet/graph/overlap.go` | `Overlap(p, q) bool` per §2; table-driven tests are the bulk of the work |
| Derivation | `internal/fleet/graph/derive.go` | roles + markers → nodes/edges/SCC/findings; pure functions, no IO — trivially testable |
| Emitters | `internal/fleet/graph/mermaid.go`, `json.go` | golden-file tests |
| Registry + note IO | reuse `trip2ggql` (`ListChangeWebhooks`, `ListCronWebhooks`, `DiscoverRoles`) + one `updateNotes` call over the admin lane | one new genqlient op at most |
| CLI | `cmd/fleet`: subcommand dispatch, `fleet graph --agents-folder ... [--json -] [--note _fleet-graph.md] [--probe]` | bare invocation unchanged (daemon) |

**v1 — static note-graph.** Subcommand + overlap + derivation + Mermaid/JSON emitters; folder
list from flags ∪ registry markers; conflicts/cycles/drift findings; disjoint-folder rule
enforced as a finding. No server changes, no schema changes.

**v2 — multi-fleet overlay + selector.** `fleet:` frontmatter selector (needs the `notePaths`
meta filter — small server change, coordinate with the lang-aware `notePaths` work);
`--default-fleet` migration mode; `--probe` liveness overlay; `GET /graph.json` on the daemon
mux (localhost-only by deployment convention, closing task #29's "internal API for UI" half);
optionally a dynamic overlay from `webhook_delivery_logs` later (agent.md Phase 4's
static+dynamic split) — out of scope here.

Open questions for review: (a) does `_fleet-graph.md` live at vault root or under a fleet
folder (root keeps it fleet-agnostic; it must NOT match any role's `trigger_include`, or the
graph note itself triggers agents — the emitter should check its own path against the graph and
refuse/warn); (b) whether `fleet graph` runs on a cron webhook itself eventually (a role with
`executor: code`? cute, but v1 is a manually-run CLI); (c) v2 selector key name (`fleet` vs
`fleet_id`).
