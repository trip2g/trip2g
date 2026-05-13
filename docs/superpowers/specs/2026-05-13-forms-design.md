# Forms in Notes

**Date:** 2026-05-13  
**Status:** Draft

## Overview

Notes can declare a form in their YAML frontmatter. Once declared, the note page embeds the form spec as JSON, a GraphQL mutation accepts submissions, and a query returns public submits. Use cases: lead collection on landing pages, homework submission in courses, comments on any note.

Frontmatter patches can bulk-apply a form spec to many notes at once (e.g., enable comments site-wide without touching individual notes).

A `form_ref` field can point to another note's `form:` block to avoid duplicating config across many notes.

---

## Frontmatter Spec

Three options — inline single form, multiple named forms, or reference to another note:

```yaml
# Option A: single inline form (form_id = "")
form:
  can_submit: guest        # guest | paid_user | admin
```

```yaml
# Option B: multiple named forms on one note
forms:
  contact:
    can_submit: guest
    fields:
      - name: email
        type: email
  survey:
    can_submit: paid_user
    fields:
      - name: rating
        type: int
```

```yaml
# Option B: reference another note's form: block (avoids duplicating config)
form_ref: "[[Comment Form]]"   # resolve by permalink (same as defaulttemplate ContentRefWikiLink)
# or by path:
form_ref: templates/comment-form.md   # resolve by file path (ContentRefFile)
```

Uses the same `parseContentRef` logic as the default template (`internal/defaulttemplate/widgets.go`). When `form_ref` is present, the system resolves the reference, reads the `form:` block from that note, and uses it as the spec. Submissions are still stored against the current note's `note_path_id`.

---

```yaml
form:
  can_submit: guest        # guest | paid_user | admin
  turnstile: false         # true = require Cloudflare Turnstile — v2 (not available in Russia)
  # can_read + moderated — v2 (public comments use case)

  fields:
    - name: email
      type: email
      required: true

    - name: body
      type: text
      required: true

    - name: rating
      type: int
      min: 1
      max: 5

    - name: agree
      type: bool
      required: true

    - name: attachment
      type: file
      accept: [pdf, jpg, png]
      max_size: 5mb

    - name: parent_id
      type: int             # optional — include to enable threaded display on frontend
```

### Field Types

| Type | Stored in | Validation options |
|------|-----------|--------------------|
| `text` | `form_string_values` | `required`, `min_length`, `max_length`, `enum: ["a","b"]` |
| `email` | `form_string_values` | `required` (format validated server-side) |
| `int` | `form_int_values` | `required`, `min`, `max`, `enum: [1,2,3]` |
| `bool` | `form_bool_values` | `required`, `enum: [true]` (consent checkbox pattern) |
| `file` | `form_file_values` + MinIO | `accept` (list of extensions), `max_size` | **v2 — not implemented in MVP; returns `file_upload_not_supported` error** |

### Access Levels

`can_submit` accepts: `guest`, `paid_user`, `admin`.

- `guest` — unauthenticated visitors allowed
- `paid_user` — requires active purchase with access to this specific note
- `admin` — only vault admins

### Moderation

- `moderated: false` (default) — submits are immediately `visible`; admin can hide post-factum
- `moderated: true` — submits start as `pending`; admin must approve before they appear publicly

---

## Database Schema

```sql
-- One row per form submission
CREATE TABLE form_submits (
    id               INTEGER PRIMARY KEY,
    note_version_id  INTEGER NOT NULL REFERENCES note_versions(id),
    form_id          TEXT NOT NULL DEFAULT '',   -- "" for single form, named for forms: map
    user_id          INTEGER REFERENCES users(id),
    ip               TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'visible',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- EAV value tables
CREATE TABLE form_string_values (
    submit_id  INTEGER NOT NULL REFERENCES form_submits(id),
    field_name TEXT NOT NULL,
    value      TEXT NOT NULL,
    PRIMARY KEY (submit_id, field_name)
);

CREATE TABLE form_int_values (
    submit_id  INTEGER NOT NULL REFERENCES form_submits(id),
    field_name TEXT NOT NULL,
    value      INTEGER NOT NULL,
    PRIMARY KEY (submit_id, field_name)
);

CREATE TABLE form_bool_values (
    submit_id  INTEGER NOT NULL REFERENCES form_submits(id),
    field_name TEXT NOT NULL,
    value      INTEGER NOT NULL,  -- 0 | 1
    PRIMARY KEY (submit_id, field_name)
);

-- File metadata (actual file lives in MinIO)
CREATE TABLE form_file_values (
    submit_id    INTEGER NOT NULL REFERENCES form_submits(id),
    field_name   TEXT NOT NULL,
    storage_key  TEXT NOT NULL,
    filename     TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes   INTEGER NOT NULL,
    PRIMARY KEY (submit_id, field_name)
);
```

---

## GraphQL API

### Scalar

```graphql
scalar Upload
```

### Types

```graphql
type FormSubmit {
  id: Int!
  noteVersionId: Int64!
  formId: String!
  user: User
  ip: String!           # visible to admin only
  status: FormSubmitStatus!
  createdAt: Time!
  fields: [FormSubmitField!]!
}

type FormSubmitField {
  name: String!
  stringValue: String
  intValue: Int
  boolValue: Boolean
  fileUrl: String       # presigned GET URL for file fields
}

enum FormSubmitStatus {
  pending
  visible
  hidden
}

type FormSubmitsConnection {
  nodes: [FormSubmit!]!
  pageInfo: PageInfo!
}
```

### Mutation: submitForm

```graphql
mutation {
  submitForm(input: SubmitFormInput!): SubmitFormOrErrorPayload!
}

input SubmitFormInput {
  noteVersionId: Int64!
  formId: String            # "" (default) for single form, named key for forms: map
  turnstileToken: String    # v2, ignored in MVP
  fields: [FormFieldValueInput!]!
}

input FormFieldValueInput {
  name: String!
  stringValue: String
  intValue: Int
  boolValue: Boolean
  fileValue: Upload     # reserved for v2 — returns file_upload_not_supported in MVP
}

union SubmitFormOrErrorPayload = FormSubmit | ErrorPayload | RequestCaptchaPayload
```

### Mutation: setFormSubmitStatus — v2

Status management for moderated forms (public comments). Not implemented in MVP.

### Query: formSubmits — v2

Public query for comments/public submits. Not implemented in MVP.

### Query: adminFormSubmits (admin)

```graphql
query {
  admin {
    formSubmits(notePathId: Int64!, formId: String): AdminFormSubmitsConnection!
  }
}
```

`formId` is optional — omit to see all forms for a note, pass `""` or a named key to filter.

---

## Form Spec Embedding

When rendering a note that has a `form:` frontmatter key, the server embeds the form spec in the HTML output as:

```html
<script id="form-spec" type="application/json">
{
  "note_version_id": 123,
  "forms": {
    "": { "can_submit": "guest", "fields": [...] },
    "survey": { "can_submit": "paid_user", "fields": [...] }
  }
}
</script>
```

`forms` always uses string keys. Single `form:` frontmatter maps to key `""`. The `note_version_id` is included so the frontend can pass it directly to `submitForm`.

The default JS renders the form for key `""` at the end of the page. Users can suppress default rendering and implement their own by reading `#form-spec`.

---

## Captcha

**v2 — not implemented in MVP.** Cloudflare Turnstile is not reliably available in Russia. Will be added if spam becomes a problem. `turnstileToken` in `SubmitFormInput` is accepted but ignored in MVP.

---

## Email Notifications

A background job is enqueued after each successful submit (always, no config needed). New job type analogous to `sendsignincode`. Email sent to all vault admins via Resend API containing:
- Note title and path
- Submission date, user/guest
- All field values (files as filenames only)

---

## Spam Protection

- **Cloudflare Turnstile** on every submit (configurable, skip if no site key)
- **User bans** — existing admin ban system covers authenticated spammers
- `can_submit: paid_user` or `can_submit: admin` eliminates anonymous spam entirely

No server-side rate limiting in v1. Captcha + access level is sufficient for most cases.

---

## Admin UI

New section in admin panel per note:
- List of `form_submits` with date, user/IP, status, field values
- For file fields: link to presigned GET URL
- `setFormSubmitStatus` controls: approve (pending → visible), hide (visible → hidden), restore (hidden → visible)
- Filter by status

---

## Use Cases

| Use case | `can_submit` | `can_read` | `moderated` |
|----------|-------------|------------|-------------|
| Lead collection (landing) | guest | admin | false |
| Homework submission | paid_user | admin | false |
| Public comments | guest | guest | false |
| Moderated comments | guest | guest | true |

**Bulk comments via frontmatter patches:** Create a patch that injects the `form:` block into all notes matching a path pattern. No per-note changes needed.

---

## Open Questions

- gqlgen multipart `Upload` scalar compatibility with the fasthttp/adaptor stack — verify at implementation time.
- File storage path in MinIO: `forms/{note_path_id}/{submit_id}/{field_name}/{filename}` — confirm convention matches existing MinIO usage.
