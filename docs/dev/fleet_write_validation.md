# Fleet: what an agent wrote, and whether it should have

**Status:** plan. Nothing here is built except where marked.

The fleet is a harness — skills-as-role-notes plus codellm, which pretends to be
an LLM and executes code instead. Its transport is now demonstrably working and
observable: a chain of roles runs end to end, and the admin shows each step, what
it wrote and what triggered it (`docs/dev/webhooks.md` → "Цепочки доставок").

What is NOT covered is the half where a real model is involved. Everything
exercised so far ran against `cmd/mockserver` with `configs/llm.jsonnet`, which
ignores the prompt and writes by a fixed rule. So the plumbing is tested and the
semantics are not: whether the model produced the shape the role demanded, and
whether it wrote where it was allowed to.

Two gaps follow from that, and they are the next work.

## Gap 1 — a denied write looks like a successful run

`ScopedKB` rejects a write outside `write_patterns`, records it in
`agentruntime.Result.Denials`, and feeds the error back to the model as a tool
result so it can self-correct (`cmd/fleet/internal/agentruntime/runtime.go`). For
real-LLM roles that is deliberate: the model learns and retries.

But the denial never leaves the fleet. `webhookutil.AgentResponse` has no field
for it, so trip2g stores nothing, and the chain screen shows a delivery that
succeeded with an empty **Wrote** column. The operator's reading is "the agent
did nothing"; the truth is "the agent tried to write to a path it does not own".
On a real model this is the single most likely failure — an instruction says
`segments/` + basename and the model writes `segments/<title>.md` instead.

Plan:

1. Carry denials in the agent response next to `costs` — a list of
   `{tool, path, reason}`, not a free-form string.
2. Store them on the delivery row (one more JSON column, same shape as `costs`)
   and show them on the chain step, in the place that today says "Wrote: —".
3. Test that self-correction actually happens: a stub LLM that writes out of
   scope on its first call and in scope on the second, asserting one denial
   recorded and the note written. Today nothing pins that loop.

Note the asymmetry to keep: `HardFailApply` makes an apply failure fatal for
code roles (all-or-nothing), while real-LLM roles keep the soft, self-correctable
path. A denial is not an apply failure and must stay soft in both.

## Gap 2 — `strict`: lint what the model wrote before applying it

Scope answers "may this path be written". Nothing answers "is this content
usable". A role that asks for `[[WikiLinks]]` and a strict frontmatter block gets
whatever the model felt like producing, and the damage lands in the vault where
the next role reads it.

Proposal: a role-level `strict: true` flag that runs the write through a linter
before it is applied. On failure the write is rejected and the reason goes back
to the model as a tool result — the same loop that already handles scope denials,
so the model gets a chance to fix its own output.

Checks worth having, cheapest first:

| Check | Why |
|---|---|
| Broken `[[wikilinks]]` | The wiki roles exist to build a link graph; a link to a note that does not exist is the failure mode, and it is silent today |
| Frontmatter present and parseable | Roles demand a strict block (`id`, `source_transcript`, `kind`); a missing one breaks the next role's input |
| Required frontmatter keys | Declared per role, e.g. `strict_frontmatter: [id, kind]` |
| Path shape | The instruction says "`segments/` + basename of the source"; a derived path can be checked against the trigger |
| Empty or truncated content | A run that returns two words is a failed run, not a write |

Open questions, to settle before building:

- **Where the linter runs.** In the fleet (it has the content and the model to
  talk back to) or in trip2g's write path (it has the note graph, so it is the
  only side that can resolve a wikilink). Wikilink checking probably forces the
  second, which means the scoped write API must be able to answer "rejected,
  because" rather than just refusing.
- **Reject vs warn.** A rejected write costs another model round trip; a warning
  costs nothing but nobody reads it. Likely: reject under `strict`, warn
  otherwise, and surface warnings on the chain step.
- **Budget.** Each rejection is another LLM call against the same
  `max_tokens`/`max_steps` ceiling. A strict role needs headroom, or it will run
  out of steps fixing its own frontmatter.

## Testing plan

Three levels, none of which exist beyond the first.

**1. Mocked, deterministic (built).** `e2e/krisp-ingest.spec.js` and the demo
stand in `docker-compose.yaml` cover the transport: role discovery, webhook
registration, delivery, write-back, and the chain. Keep these as the regression
net — they catch the class of bug found today (a lane that silently stopped
firing events).

**2. Mocked, adversarial (missing).** Same stand, but the stub LLM misbehaves on
purpose: writes out of scope, returns malformed frontmatter, writes a broken
wikilink, returns nothing, exceeds the step ceiling. Each case asserts what the
operator ends up seeing — a denial on the step, a warning, a failed delivery.
This is where Gap 1 and Gap 2 get their tests, and it needs no real model.

**3. Live model, smoke (missing).** One real transcript, one real model, run by
hand — not on a schedule. What to look at afterwards, in this order: what each
step wrote (the chain step now expands the stored content), what it cost, and
whether anything was denied. Three or four such runs will say more about
extraction quality than a week of the cron running unattended.

Only after 3 is worth automating: a fixed transcript, a pinned model, and
assertions on the shape of the output rather than its wording.

## Built: agents may not author role notes

**Status: implemented.** A role that can write note content can write a *role
note*, and a role note declares its own `write_patterns`, `tools` and `model`.
So a role confined to `transcripts/**` could mint a successor with
`write_patterns: ["**"]`, and the reconciler would pick it up out of band, on
its own poll, outside any delivery-chain depth limit — `max_depth` bounds
recursion, not authorship. The realistic trigger is not a malicious operator but
prompt injection through note content, which reaches an LLM role as
`changed_files` content and can arrive from outside the vault (telegram inbox,
forms).

`ScopedKB` now denies it (`cmd/fleet/internal/agentruntime/roleguard.go`):

- The marker is `fleet_id` in a note's YAML frontmatter — necessary and
  sufficient, since `DiscoverRoles` skips a note with an empty `fleet_id`
  ("untagged roles are never claimed") and one tagged for another fleet. The
  check keys on the marker and never on the path, so role notes stay free to
  live anywhere in the vault.
- `write_note` is checked against the content being written. `patch_note` costs
  one read: the edit is applied server-side by trip2g's `updateNotes`, so fleet
  does not otherwise read the note. Matching on the `replace` fragment alone
  would be cheaper and is not enough — retagging an existing role note changes
  only the fleet_id VALUE, so the marker never appears in the fragment.
  `applyPatchPreview` mirrors trip2g's semantics exactly (unique match, replaced
  once) and both the current and the resulting content are checked, so editing an
  existing role note is refused as well as creating one.
- The verification read goes through the underlying KB, not the ScopedKB: a role
  may hold write scope over a path without read scope, and the guard must not be
  defeated by the role's own `read_patterns`.
- An unreadable note fails closed, under a distinct error
  (`ErrRoleGuardUnverifiable`), so an infrastructure failure is never reported as
  an accusation of role authoring. "Not found" fails closed too: `remoteKB` reads
  through the delivery-scoped client, so an absent note and one outside this
  token's read scope look identical, and letting "not found" through would hand a
  role that can write a path it cannot read the very bypass the unscoped
  verification read exists to prevent. It is classified as an apply failure
  rather than a denial, and wraps the underlying cause, so the denial log does not
  fill with false role-authoring hits.
- The marker parser mirrors `goldmark-meta`, which is what actually produces a
  note's meta in trip2g — not a hand-written idea of what frontmatter looks like.
  The first version did the latter and shipped five bypasses (`----` fences,
  mismatched fence lengths, longer dash runs, unterminated blocks that goldmark
  still closes at EOF, and duplicate YAML keys that yaml.v2 accepts and v3
  rejects). `TestDeclaresRoleMatchesGoldmark` now diffs the two parsers over a
  corpus and asserts the only property that matters: if trip2g would run it as a
  role, the guard must see a role.
- `--allow-role-authoring` (env `TRIP2G_FLEET_ALLOW_ROLE_AUTHORING`) turns it off
  fleet-wide for operators who do want agents managing roles. It logs a WARN at
  startup when off, because a guard silently disabled is worse than none.

The patch is conditional on the bytes that were verified. `ScopedKB` hashes the
content it read and `remoteKB.PatchIfUnchanged` sends it as `expectedHash`, so
trip2g compares against the live note inside the same atomic mutation
(`internal/case/updatenotes`). Without it the guard verified one version and
trip2g patched whatever was current — a window that is wider than a race, because
`remoteKB.Read` is served from an overlay seeded once per delivery and never
refreshed. Stale bytes now produce a loud hash mismatch instead of an unverified
write. A concurrent edit therefore fails the run, which is the intended trade.

fleet computes the hash itself rather than asking for it, so no schema or query
changed; the two implementations are tied together by a golden value asserted from
both sides (`TestContentHashGolden` / `TestHashContentGolden`), because a silent
drift would look like a concurrency fault rather than a bug. A KB without the
optional `conditionalPatcher` interface (`FileKB`, test doubles) is patched
unconditionally, as before.

Future: the overlay could carry the hash of what fleet last wrote, making a
write-then-patch of the same note in one run conditional too. Not needed yet — a
role doing that is better rewritten.

This closes authorship, not the underlying property that a role declares its own
scope. A human-authored role still does — which is intended: the point is that
only humans mint roles.

Gap 1 above is what makes the denial usable: the run log carries the reason
(`logToolCall` stores it in `data.reason` at WARN level), so the delivery trace
shows *why* rather than leaving the operator to read an empty **Wrote** column.

## Related

- [fleet_run.md](fleet_run.md) — running the fleet, and the krisp demo stand.
- [webhooks.md](webhooks.md) — delivery chains, and why an unchanged write raises
  no event.
- [agent_runtime_design.md](agent_runtime_design.md) — the scoped tool loop.
- [unseal_codellm_secrets.md](unseal_codellm_secrets.md) — sealed secrets in role
  frontmatter; role authorship is the escalation path that design depends on
  being closed.
