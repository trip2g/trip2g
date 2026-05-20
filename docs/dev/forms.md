# Forms in Notes

A note can declare a form right in its frontmatter. trip2g embeds the spec into the page as `<script id="form-spec">`, accepts submissions via GraphQL, and stores them next to the note. This is not a separate module — it is a property of the note. Everything lives in the shared database and flows through the same machinery (roles, frontmatter patches, webhooks).

## Data Flow

```
Frontmatter         RawMeta              FormSpec               <script id=form-spec>
  form: ─────► RawMeta["form"] ─► formspec.ParseFromRawMeta ─► templateviews.Note.FormSpecJSON
                                                                       │
                                                                       ▼
                                                                  JSON in HTML
                                                                       │
                                                                       ▼
                                                          JS client → submitForm mutation
                                                                       │
                                                                       ▼
                                                          submitform.Resolve → form_submits + EAV
                                                                       │
                                                                       ▼
                                                          EnqueueSendFormSubmitEmail
```

## Key Files

| File | Role |
|------|------|
| `internal/formspec/spec.go` | `FormSpec` / `FormField` types, `ParseFromRawMeta`, `ParseFormRef` |
| `internal/templateviews/note.go` | `Note.FormSpecJSON()` — JSON for `<script>` in custom layouts |
| `internal/defaulttemplate/template.go` | `Ctx.FormSpecJSON()` — same for the default template, plus `form_ref` resolution via NVS |
| `internal/defaulttemplate/views.html` | Emits `<script id="form-spec">` |
| `internal/case/submitform/resolve.go` | `Resolve`, `validateFields`, `checkCanSubmit` |
| `internal/graph/schema.graphqls` | `submitForm`, `SubmitFormPayload`, `FormSubmitDeniedPayload`, `ErrorPayload`, `admin.formSubmits`, `admin.formNotes`, `markFormSubmitProcessed` |
| `cmd/server/main.go` | `submitform.Env` implementation (`GetFormSpec`, `IsAdmin`, `UserID`, `RequestIP`, EAV-insert methods, email enqueue) |
| db migrations | `form_submits`, `form_string_values`, `form_int_values`, `form_bool_values`, `form_file_values` |

## Frontmatter

Three shapes:

```yaml
# 1. Single form
form:
  can_submit: admin              # guest | admin | paid_user
  success_url: /thanks           # redirect target after success
  fields:
    - { name: email, type: email, required: true }
```

```yaml
# 2. Multiple named forms
forms:
  contact: { fields: [{ name: email, type: email }] }
  survey:  { can_submit: admin, fields: [{ name: rating, type: int, min: 1, max: 5 }] }
```

```yaml
# 3. Reference to a shared spec
form_ref: "[[templates/comment_form]]"    # or: form_ref: templates/comment_form.md
```

Parsing: `formspec.ParseFromRawMeta` handles a single `form:` block and each entry of `forms:`. `form_ref` is resolved at render time through NVS (`Ctx.FormSpecJSON` in the default template). Submissions are always attached to the **referencing** note, not the spec source.

## Field Types

| `type` | Stored in | Validators (in `validateFields`) |
|---|---|---|
| `text` | `form_string_values` | `required`, `min_length`, `max_length`, `enum: ["a","b"]` |
| `email` | `form_string_values` | `required` (format via `net/mail.ParseAddress`) |
| `int` | `form_int_values` | `required`, `min`, `max`, `enum: [1,2,3]` |
| `bool` | `form_bool_values` | `required`, `enum: [true]` (consent pattern) |
| `file` | `form_file_values` | **not implemented** — returns `ErrorResult{Message: "file_upload_not_supported"}` |

EAV schema: one row per submission in `form_submits`, plus one row per value in a typed table. This avoids ballooning the schema for arbitrary field sets.

## Authorisation — `can_submit`

Enforced in `submitform.Resolve.checkCanSubmit` (`resolve.go`):

```go
switch spec.CanSubmit {
case "", formspec.CanSubmitGuest:
    return nil                                          // allow
case formspec.CanSubmitAdmin:
    if env.IsAdmin(ctx) { return nil }
    return &DeniedResult{Reason: DeniedAdminRequired}
case formspec.CanSubmitPaidUser:
    return &DeniedResult{Reason: DeniedNotSupported}    // not implemented yet
default:
    return nil                                          // forward-compatible
}
```

`Env.IsAdmin` lives on `*app` (`cmd/server/main.go`): pulls `appreq.UserToken()` and checks `token.IsAdmin()`. The token can arrive either as a cookie (`trip2g_e2e=…`) or as a Bearer (`t2g_…`); the admin role is set by `personaltoken.Resolver` when the token owner has a row in the `admin` table.

The GraphQL return is a union:

```graphql
union SubmitFormOrErrorPayload =
    SubmitFormPayload          # { submitId: Int! }
  | FormSubmitDeniedPayload    # { reason: String! }
  | ErrorPayload               # { message, byFields[] }
```

Type mapping is in `internal/graph/schema.resolvers.go` (`mutationResolver.SubmitForm`).

## Reading Submissions

| Query | Access | Purpose |
|---|---|---|
| `admin.formSubmits(notePathId)` | admin (cookie / Bearer t2g_) | List of submissions for one note with EAV values |
| `admin.formNotes` | admin | All notes that have at least one submission — for dashboards |
| `admin.unprocessedFormSubmitsCount` | admin | Counter for the admin badge (stub right now) |
| `markFormSubmitProcessed(submitId, comment)` | admin | Mark a submit as processed, with an optional comment |

API keys work: `personaltoken.Resolver.Resolve` assigns `Role: "admin"` when the token's owner has an `admin` row, so `checkAdmin` lets them through.

## Default Template and Custom Layouts

The default template (`views.html`) does not actually render the form — it only emits the JSON. Rendering is the client JS's job. That JS does not currently exist in `defaulttemplate.js`, which means the form is **not interactive out of the box** — you either render it via a custom layout, or wait for the default renderer to land.

Sample custom layout: `docs/_layouts/forms/example.html` — builds fields from the spec, calls `submitForm`, handles `SubmitFormPayload` / `FormSubmitDeniedPayload` / `ErrorPayload`, and performs the `success_url` redirect.

`templateviews.Note.FormSpecJSON()` returns the same JSON shape as `defaulttemplate.Ctx.FormSpecJSON()`, minus `form_ref` resolution — without NVS access, custom layouts can't resolve refs yet.

## Tests

- `internal/formspec/spec_test.go` — yaml → FormSpec parsing
- `internal/case/submitform/resolve_test.go` — validation, success, `file_upload_not_supported`, `can_submit: admin` allow/deny, `paid_user` → not_implemented
- `internal/templateviews/note_test.go` — `FormSpecJSON` for empty case, single form, named map
- `e2e/forms.spec.js` — embedded `<script id=form-spec>`, submission via mutation, anon → `admin_required` on an admin-only form, admin → success, `success_url` present in spec, admin read via `admin.formSubmits`

## Current Spam Mitigation

- `can_submit: admin` removes anonymous spam entirely
- The existing user-ban system covers authenticated abusers
- `turnstile: true` in the spec is **not validated** — accepted but ignored in `submitform.Resolve`

## Not Implemented (Roadmap)

### File field — `type: file`

Currently returns `file_upload_not_supported`. What's needed:

- `Upload` scalar in gqlgen with multipart support on top of fasthttp/adaptor
- Upload to MinIO/S3: `forms/{note_path_id}/{submit_id}/{field_name}/{filename}`
- Metadata written to `form_file_values` (schema already exists in the design doc)
- Presigned GET URLs for admin UI and API consumers
- Validators: `accept: [pdf, jpg]`, `max_size: 5mb`

Without this, forms for homework, portfolios, and job applications stay neutered — text submits work, file attachments don't.

### Payment field — `type: payment`

Hypothetical type: the field asks not for a value but for **proof of payment**. Flow:

- Before submitting, the client kicks off `createPaymentLink` (already in the schema) with `noteVersionId` plus the amount from the spec
- After a successful payment the provider webhook (Stripe / CloudPayments / Boosty / Patreon — we already integrate all four) tags the purchase with "form N paid"
- `submitForm` checks the payment and only then accepts the submission
- The payload returns `submitId` + `purchaseId`

This unlocks paid leads, paid comments, and paid newsletter signups without a separate backend.

### Agent handling — submission webhooks

Largely composable from existing pieces: `webhooks` + `submitForm`. What still needs explicit wiring:

- A new event for `change_webhooks`: "new submission" (alongside the current on_create/on_update/on_remove for notes)
- Webhook payload: spec + field values + link to the note
- A HAT token (`docs/dev/hat.md`) scoped to the note so the agent can edit it, reply to the submitter, or create a new page in the response

### `can_submit: paid_user`

Currently returns `not_implemented`. What's needed: `Env.HasPaidAccess(ctx, notePathID) bool`, backed by `ListActiveSubgraphAccessesByUserID` or a direct purchase check. Full integration with the paywall (`docs/dev/monetization.md`).

### Captcha (any provider)

`turnstileToken` is already part of `SubmitFormInput`, but it's ignored in `submitform.Resolve`. The plan is broader than "turn Turnstile on":

- Pluggable providers: Cloudflare Turnstile, hCaptcha, Yandex SmartCaptcha (the last one is required for RU users where Turnstile is unreliable)
- Provider selection in admin config (site key + secret + provider id)
- A `captcha: true` flag in the form spec — the provider isn't chosen in frontmatter
- The spec JSON includes `captcha: { provider, site_key }` so the client knows which widget to render
- The server validates the token with the matching provider based on `captcha.provider` from admin config

Without captcha, `can_submit: guest` is unsafe in production. With captcha, it is the only sane defence against anonymous spam.

### Built-in JS for forms and comments

`defaulttemplate.js` does not render the form today — it only emits the JSON. A default client renderer would make forms work out of the box, no custom layout required. What it should do:

- **Render a form from the spec** — fields, validation, `submitForm` call, union handling (`SubmitFormPayload` / `FormSubmitDeniedPayload` / `ErrorPayload`), `success_url` redirect.
- **List comments** — for forms marked public (`can_read: guest`, v2). A feed of existing submissions appears under the form: name, date, body, status (`visible` / `pending` / `hidden`). It would read from a not-yet-implemented public `formSubmits` query (today only `admin.formSubmits` exists).
- **Threading** — if the spec has a `parent_id: int` field, the JS nests replies. Submissions with a `parent_id` show indented under the parent.
- **Admin moderation** — when the viewer is an admin, each submission gets `approve / hide / pin / mark processed` controls.
- **Captcha widget** — rendered above the submit button based on `spec.captcha`.

The same JS is useful as a utility for custom layouts: you can pull just the feed component (for read-only pages) or just the form renderer.

## Why This Matters — Agent-Driven Micro SaaS

When file + payment + webhook land, a closed loop emerges:

1. **Frontend** — a regular note with a form. SEO, sharing, any custom layout.
2. **Auth + paywall** — already there.
3. **Submission storage** — already there, typed EAV.
4. **Payment** — once the payment field exists, payment attaches to a submission.
5. **Trigger** — the webhook fires to an agent with a HAT token.
6. **Agent logic** — the agent replies to the user, writes a personal note via `pushNotes`, or grants access to a private folder via a subgraph.
7. **Delivery** — the user gets an email with a link to their personal note, or open access to the right subgraph.

The site owner can ship a landing page with payment intake, lead processing, and access delivery — **with no separate backend**. Note is the landing. Form is intake. Webhook is orchestration. Agent is business logic. trip2g is the runtime.

## Known Gaps

- `form_ref` is not resolved in custom layouts (`templateviews.Note.FormSpecJSON`) — only in the default template. Custom layouts that need a ref must inline the spec, or wait for a `NVS.FormSpecJSON(note)` helper.
- `unprocessedFormSubmitsCount` is a stub — it panics.
- No rate-limiting on guest forms. Not an issue with `can_submit: admin`, but mandatory once `paid_user` is enabled.
- The `comment` field on `markFormSubmitProcessed` is in the schema but not exposed in the admin UI.
