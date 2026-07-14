# Forms → Notes: generation, storage, and the EAV duplication problem

**Status:** Open design (not decided, 2026-07-14). Sibling to `forms_comments.md`.
**Builds on:** `forms.md` (spec + submit pipeline), `change_webhooks.md` (the delivery mechanism).

## TL;DR

A form submission should be able to become a **note**, generated from a Jet template in the form spec, so the existing change-webhook pipeline delivers it to agents/automations with no new notification channel. The current all-admin email on submit is a demo stub and should go. Open question: whether the note **replaces** the EAV store (which now looks duplicative) or coexists with it, and whether a note is even created (a "virtual" change-webhook without a note is a considered but awkward alternative). Not decided. Two security bugs in the current pipeline are being fixed independently of this design (see "Critical fixes" below).

## Why this came up

- The submit path ends with `EnqueueSendFormSubmitEmail` (`internal/case/submitform/resolve.go:117`) — an email to every admin. This was only wired to test the form feature end to end. It does not scale (amplification on the small prod VM; flagged by both security reviews) and it hardcodes notification policy into the core.
- Writing a note already triggers the whole change-webhook fan-out (`change_webhooks.md`): glob match → batch → goqite delivery → optional agent response applied via `InsertNote`. So "turn a submission into a note" reuses that entire mechanism instead of adding a second one. Notification policy moves to userland (a webhook on the submission path), which is the project's extension model (Jet + webhooks, not a plugin API).

## Direction (leaning, not locked)

Generate the note **from a Jet template declared in the form spec**, not from a fixed field-to-frontmatter mapping. The author controls the shape:

```yaml
form:
  fields: [ ... ]
  note:
    path: "_submissions/{{ .Form }}/{{ .Timestamp }}-{{ .SubmitID }}.md"   # server-composed
    template: |
      ---
      form: {{ .Form }}
      email: {{ .Fields.email }}
      hidden: true
      ---
      {{ .Fields.message }}
```

Wiring mirrors the opaque-port pattern (see the Env-pattern refactor): `submitform.Env` gains a `WriteSubmissionNote(...)` port; `app` wraps `insertnote.Resolve`. After the validated field values are in hand, render the template and write the note.

### Hard constraints (any implementation must hold these)

1. **The note path is server-composed from the form spec, never from submitter input.** A guest supplies field *values* only. A user-controlled path turns anonymous submit into an arbitrary-write primitive. The path template is admin-authored frontmatter; the timestamp/id come from the server.
2. **PII containment.** The submission carries email/IP; a note propagates to the git mirror and to every subscribed webhook. Only fields the form explicitly marks public go into the note, and the note is `hidden: true` by default (not rendered, not indexed) — same treatment as the `_`-prefixed landing section notes. This is the same whitelist-not-blacklist rule the security review demanded for public comment reads.
3. **Rate limiting becomes mandatory, not optional.** Note-per-submit *amplifies* the DoS surface versus the EAV-only path: each flood submission now also writes a note, hits the git mirror, and fans out webhooks, all through the single SQLite writer. The note option cannot ship without a per-IP submit limit under it.

## The EAV duplication problem (the actual open question)

The current store is EAV: `form_submits` + `form_string_values` / `form_int_values` / `form_bool_values` (`db/schema.sql`). If every submission also becomes a note, we have two representations of the same data, and the EAV one starts to look redundant.

Three options, none clearly right yet:

- **A. Note is a projection, EAV stays canonical.** Keep EAV as the source of truth (structured admin filter UI is already built on it via `form_submits_filter`); the note is an opt-in side effect for the webhook/agent path. Cost: genuine duplication, two things to keep in sync, PII lives in two places.
- **B. Note is canonical, drop EAV.** Attractive for comments (comments are content). Cost: structured queries/admin filtering now mean parsing frontmatter across many notes instead of SQL; the existing filter UI would need rework. Loses the cheap structured store for surveys/lead-gen.
- **C. Virtual change-webhook, no note written.** Fire the change-webhook delivery for a synthetic path without materializing a note. Avoids duplication and PII-in-vault. Cost: awkward — the webhook contract assumes a real note the agent can read/patch; a phantom path breaks the "agent returns changes applied via InsertNote" loop and the debug/replay tooling. Considered, currently disfavored.

No decision. B is cleanest *if* comments become the primary use and the admin filter UI is rebuilt on note queries; A is the low-risk incremental step; C is a dead-end unless the webhook contract is generalized to "events" rather than "note changes."

### What weakens the case for EAV (arguments for B)

- **Admin review comes for free from the magazine layout.** If submissions are notes under `_submissions/<form>/`, the admin browses them with the existing magazine layout over that folder — reverse-chronological cards, no bespoke admin UI. That was EAV's main advantage (the `form_submits` filter UI); a note folder plus magazine covers the common case without new frontend.
- **"Inbox forms" as a custom template.** An inbox (contact/lead/support queue) is then just a custom layout over the submission folder — the same pattern as kanban_template/theme_editor: a self-contained HTML layout reading notes via GraphQL. No new subsystem; the form writes notes, the inbox template reads them.

These push toward B (note canonical) for content-shaped forms. EAV still earns its place only where structured aggregate queries matter (numeric survey stats, exportable tabular lead data) — so the real split may be per-form: `store: note` (content/comments/inbox) vs `store: eav` (structured), not one global choice.

## Critical fixes (decoupled from the above, doing now)

These are current bugs in the shipped pipeline, independent of the note-generation direction:

1. **No note-readability check on submit** (`GetFormSpec` at `cmd/server/notes.go:292` + `submitform.Resolve`). Only `can_submit` is checked, never `canreadnote.Resolve`. With enumerable `note_version_id`, a guest reads the form spec of, and submits to, a paywalled/subgraph/signin-gated note's guest form. Fix: gate the submit path on note readability; treat `note_version_id` as untrusted.
2. **Turnstile fails open on empty secret** (`internal/turnstile/verify.go:37-40`) — same code path in prod, no boot guard. Fix: fail closed in production when a SiteKey is configured but the SecretKey is empty (refuse the `turnstile: true` submit / surface a startup error), keep the dev no-op behind an explicit signal.
3. **No default length cap** when a field omits `max_length` (`resolve.go:163-167`) — accepts up to the 10 MB body cap, stored verbatim; counts bytes not runes. Fix: a global hard per-field cap regardless of spec, and `utf8.RuneCountInString` for the bounds.

Rate limiting (absent everywhere on `submitForm`) and the spoofable `X-Forwarded-For` IP are real gaps but are coupled to the note-generation/comments direction and are larger changes; they are tracked here, not in the immediate fix.

## References

- Security audit (Codex + Fable, 2026-07-14): findings above, plus `success_url` open-redirect (author-scoped, LOW), duplicate-field-name → partial write + 500, `can_submit` unknown-value default-allow.
- `forms_comments.md` — public reads/threading/moderation; shares constraints 2 and 3.
- `change_webhooks.md`, `shared_webhooks.md` — the delivery mechanism a submission note would ride.
