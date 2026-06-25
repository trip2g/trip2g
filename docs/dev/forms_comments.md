# Forms as a Universal Submissions API — Public Reads, Threading, Moderation

**Status:** Design (approved 2026-06-25). Precursor to an implementation plan.
**Builds on:** `docs/dev/forms.md` (the "Built-in JS for forms and comments" roadmap, ~lines 188–198).

## TL;DR

Finish the documented forms-v2 roadmap so **any** form can expose a public, threaded, moderated feed of its own submissions — and ship a built-in JS renderer that draws both the form *and* the feed with no custom layout. "Comments" is not a new subsystem; it is a form convention plus a frontend interpretation. The server never learns the word "comment" or "thread".

Two properties make this cheap:

- **Zero database migrations.** The `status` column already exists; `parent_id` is stored as an ordinary EAV `int` field.
- **The backend stays generic.** Threading, sort, and "comment-ness" are JS concerns driven by the form spec. If someone wants a different structure, they declare different fields and write their own JS.

`type: file` is a defined **Phase 2** fast-follow, kept migration-free by an upload-ticket model (presigned PUT / JWT route) that stores the file's storage key as a normal string field.

## Why now / motivation

The forms feature already stores submissions (typed EAV), gates them by `can_submit`, and protects them with Turnstile. What is missing is the *read* side and a default renderer: today a form is not even interactive out of the box (`forms.md:120`). The smallest useful thing that unlocks the most value is a public, moderatable, optionally-threaded read of submissions — i.e. comments — exposed through one generic mechanism that doubles as a universal "public submissions" API.

## Current state (verified)

What exists:

- `submitForm` mutation + EAV storage: `form_submits` + `form_string_values` / `form_int_values` / `form_bool_values` (`db/schema.sql:728–748`, `internal/case/submitform/resolve.go`).
- `form_submits.status` column: `text not null default 'visible'` (`db/schema.sql:733`). **Not set on insert today** — `InsertFormSubmit` takes only `(note_version_id, form_id, user_id, ip)` (`queries.write.sql:1123`), so every submit currently defaults to `visible`.
- Admin reads: `admin.formSubmits`, `admin.formNotes`, `markFormSubmitProcessed`; field assembly via `loadFormSubmitFields` (`internal/graph/form_helpers.go:35`).
- Admin UI catalog: `assets/ui/admin/formsubmits/`.
- Turnstile captcha on by default (`internal/case/submitform/resolve.go`, the `VerifyTurnstile` path).
- Object storage (for Phase 2): dual backend MinIO + local disk behind one `Storage` interface (`cmd/server/storage.go:19`), presigned GET (`internal/miniostorage/storage.go:182`), `PutPrivateObject` (`miniostorage.go:281`, `localstorage.go:157`); working multipart `Upload` scalar proven by `uploadNoteAsset` (`internal/graph/schema.graphqls:1663`, `handler.go:72`).

What is missing:

- No **public** read of submissions — only admin-gated.
- No moderation status transition (only `markFormSubmitProcessed`, which sets processed metadata, not visibility).
- No `can_read`, `moderation`, `sort`, or `private` keys in the form spec; no `parent_id` recognition.
- No built-in JS renderer in `defaulttemplate.js` (forms.md:120, 190).
- `unprocessedFormSubmitsCount` is a stub that panics (`forms.md:263`).
- For Phase 2: `form_file_values` does **not** exist despite `forms.md:35,72,147` claiming it does; the file resolver discards bytes (`schema.resolvers.go:2636`); there is no served route for private objects on the local backend.

## Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | Scope = full comments feature, end-to-end. | Backend API + threading + moderation + built-in JS renderer + admin UI. |
| 2 | "Comments" = generic form convention + JS, **plus** a `preset: comments` sugar. | Universal API; one mechanism. Power users write full specs; common case is a one-liner. |
| 3 | `parent_id` is a **form field**, not a DB column. Field present → tree, absent → flat list. | Backend stays generic; threading is a JS interpretation. No migration. |
| 4 | Threading renders with **unlimited nesting + CSS indent cap**. | Matches "tree-structured" intent; survives mobile. Depth stored at full fidelity. |
| 5 | Moderation is a per-form spec flag `moderation: pre|post`. **Default = `pre`.** | Author chooses; safe default for public guest forms. |
| 6 | Identity is **anonymous by default**; `name`/`email` captured only if the author adds those fields. | A comment is just a submission. Server still records `ip` + `user_id` for audit. |
| 7 | Sort is a spec setting `sort: newest|oldest` (default `newest` for top level), read by JS. | Configurable in frontmatter, not hardcoded. |
| 8 | A spec field can be `private: true` to exclude it from public reads. System/PII columns are never exposed publicly. | Prevents leaking `email` / `ip` in a public feed. |
| 9 | `type: file` is **Phase 2**, via an upload-ticket (presigned PUT / JWT route); store the storage key as a string field. | Keeps the whole feature migration-free; decouples bytes from `submitForm`. |
| 10 | **Zero database migrations** for Phase 1 and Phase 2. | `status` exists; `parent_id` and file-key are EAV values. |

## Phase 1 — Comments core (zero migrations)

### 1. Form-spec additions — `internal/formspec/spec.go`

Add to the spec parser:

- `can_read: guest` — opt submissions into public reads. Default: admin-only (unchanged).
- `moderation: pre | post` — per-form. Default `pre`. Controls insert status.
- `sort: newest | oldest` — feed order for top-level entries. Default `newest`.
- `private: true` on a field — exclude that field's values from public reads.
- Recognize an `int` field named `parent_id` (no special storage — see §3). Its presence is what the JS uses to decide tree vs list.
- `preset: comments` inside a `form:` block expands server-side to:
  ```yaml
  can_submit: guest
  can_read: guest
  moderation: pre
  fields:
    - { name: body, type: text, required: true }
    - { name: parent_id, type: int }   # threading
  ```
  The author may extend/override: add `name`/`email` fields, set `moderation: post`, change `can_submit`, mark fields `private`, etc.

The current `FormField` struct and `rawField` parser (`spec.go:28–39, 59–68`) gain the new keys. (Phase 2 adds `accept`/`max_size` here.)

### 2. Insert status — `queries.write.sql` + `make sqlc`

`InsertFormSubmit` gains a `status` argument. `submitform.Resolve` computes it from the spec: `pre` → `pending`, `post` (or no moderation) → `visible`. Forms without `moderation` keep today's `visible` default — **no behavior change for existing forms.** This is a query change, not a migration (the column exists).

### 3. `parent_id` storage — no change

A field named `parent_id` is stored in `form_int_values` keyed by `(submit_id, field_name)` like any int field (`db/schema.sql:743`, `queries.write.sql:1132`). Nothing special-cases it. On reply, the client submits `parent_id = <parent submitId>` as a normal field value. No backend parent validation: a dangling `parent_id` simply renders at top level (the JS's responsibility).

### 4. Public read query — `queries.read.sql` + sqlc + GraphQL

New public query:

```graphql
formSubmits(
  noteVersionId: Int64!
  formId: String
  parentId: Int        # optional: fetch one thread level
  sort: String         # newest | oldest
  limit: Int
  offset: Int
): FormSubmitsConnection!
```

Rules:
- Allowed only when the form's spec has `can_read: guest` (else empty / denied).
- Guests see `status = 'visible'` only; admins see all statuses (with status attached for moderation UI).
- Returns per submit: `id`, public field values (via `loadFormSubmitFields`), `createdAt`, `status`, and a display name (account name if `user_id`, else the `name` field if present, else "Anonymous").
- **Excludes** system/PII columns `user_id`, `ip`, `processed_by`, `comment`, and any field marked `private: true`. `parent_id` is returned (structural, used by JS to build the tree).

Reuses the existing EAV read plumbing; adds a status filter + field whitelist.

### 5. Moderation mutation — GraphQL

```graphql
setFormSubmitStatus(submitId: Int!, status: FormSubmitStatus!): FormSubmit!  # admin only
```

`status ∈ {visible, hidden}`. `markFormSubmitProcessed` stays for processed/notes metadata. Fix `unprocessedFormSubmitsCount` so the admin badge counts `status='pending'`.

### 6. Built-in JS renderer — `assets/defaulttemplate` → `defaulttemplate.js`

The missing default renderer from `forms.md:190`, triggered by the presence of `<script id="form-spec">` (`views.html.go:1456`). It:

- **Renders the form** from the spec — fields, validation, `submitForm`, union handling (`SubmitFormPayload` / `FormSubmitDeniedPayload` / `TurnstileRequiredPayload` / `ErrorPayload`), captcha widget, `success_url`. Works for *all* forms, not just comments.
- **Renders the feed** when `can_read: guest` — calls `formSubmits`, lists entries, honors `sort`.
- **Threads** when the spec has a `parent_id` field — unlimited nesting with a CSS indent cap (indent stops growing after ~5 levels). A reply box per node submits with `parent_id` set.
- **Admin controls** when the viewer is admin — inline approve/hide via `setFormSubmitStatus`.
- **Moderation feedback** — post-mod: new comment appears immediately; pre-mod: optimistic "awaiting moderation" placeholder.

The renderer is also usable piecemeal by custom layouts (feed-only, or form-only). Mirror the conditional-widget-load pattern used for mermaid/charts (`docs/dev/mermaid.md`).

### 7. Admin moderation UI — `assets/ui/admin/formsubmits/`

Extend the existing catalog: a status column, approve/hide buttons (→ `setFormSubmitStatus`), and a pending filter. Wire the fixed `unprocessedFormSubmitsCount` into the admin badge.

### 8. Anti-spam

- Turnstile stays on by default for guest forms.
- Default `moderation: pre` means guest spam is never shown publicly before review.
- Rate-limiting on guest submits is a known gap (`forms.md:264`) — noted as a fast follow, not blocking.

## Phase 2 — `type: file` via upload-ticket (still zero migrations)

Built after the Phase 1 renderer + public query land (it reuses both).

Flow:
1. Client requests an **upload ticket** (new mutation/endpoint). The server returns either a presigned PUT URL (MinIO) or a short-lived **JWT + `/_system/upload`** route (local backend, which writes via `PutPrivateObject`). The ticket encodes the allowed content-type and max size.
2. Client uploads the bytes directly to that URL — never through `submitForm`, so no GraphQL transaction holds the SQLite writer.
3. Client submits the form with the returned **storage key as a normal string field value** (`form_string_values`). No `form_file_values` table.
4. Read-back: presigned GET (MinIO) / JWT-signed `/_system/file` route (local), served with `Content-Disposition: attachment` so uploads can't execute as XSS on the app origin.

Spec additions: `type: file` with `accept` (allowlist) and `max_size`.

- **`max_size`: enforced server-side** in the ticket policy / upload route, plus a global hard cap. Mandatory — anonymous uploads on a small VM are a storage/DoS vector. Reuse `CheckStorageLimits` (`cmd/server/assets.go:53`).
- **`accept`: an allowlist baked into the ticket policy.** Treated as UX + junk-prevention, not a security boundary; security comes from private storage + attachment-disposition serving.

Frontend: a file input in the JS renderer that performs the ticket → upload → submit dance.

## Data model

No `CREATE TABLE` / `ALTER TABLE` in either phase. Confirmed: `status` exists with default `'visible'`; EAV value tables key by `(submit_id, field_name)` with no field-name restrictions; no `parent_id` column or threading concept exists anywhere today. (Per SQL-migration rule: if any future change *does* require a migration, confirm before creating it.)

## Testing (TDD — tests first)

Unit:
- `formspec`: `preset: comments` expansion; parsing of `can_read` / `moderation` / `sort` / `private`; `parent_id` int field recognized.
- `submitform`: status-on-insert by moderation mode (`pre`→pending, `post`→visible, none→visible).
- public `formSubmits`: status filter (guest sees only visible), PII/system column exclusion, `private` field exclusion, sort, `parentId` filter.

E2E (`e2e/forms.spec.js`):
- Guest comment under `moderation: pre` → not visible publicly; admin `setFormSubmitStatus(visible)` → appears.
- Reply with `parent_id` → nests under parent in the rendered feed.
- Admin `setFormSubmitStatus(hidden)` → vanishes from the guest read.
- A `private: true` `email` field is not present in the public read payload.

## Out of scope

- Jet `forms` / `db.Query` template variable (the bigger "submission data in templates" effort — `forms.md:216`).
- Pin, author edit/delete, commenter email notifications, pluggable captcha providers.

## Open points to confirm during review

- `private: true` field mechanism (vs. exposing all declared fields) — included as a safety default; confirm it's wanted.
- Default `sort = newest` for top-level entries.
- Exact spelling of the upload-ticket endpoint (`/_system/upload`) and whether it's a GraphQL mutation or a plain HTTP route.
