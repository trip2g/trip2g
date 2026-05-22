---
free: true
title: Form example
layout: forms/example
form:
  can_submit: admin
  turnstile: false
  fields:
    - name: email
      type: email
      required: true
    - name: rating
      type: int
      min: 1
      max: 5
    - name: message
      type: text
      required: true
      max_length: 2000
    - name: agree
      type: bool
      required: true
      enum: [true]
---

This is a working example of a public form embedded in a note. The form spec lives in the note's frontmatter, the layout reads it, and submissions go to the `submitForm` GraphQL mutation.

## What's on this page

- `form:` in the frontmatter — server-side source of truth for fields and validation
- `layout: forms/example` — a custom Jet layout (`docs/_layouts/forms/example.html`) that reads the spec and renders the fields
- A `<script id="form-spec">` block injected by `note.FormSpecJSON()` so the JS doesn't have to know about the schema
- A vanilla `fetch` to `/_system/graphql` calling `submitForm`

## Form spec used here

```yaml
form:
  can_submit: admin
  fields:
    - name: email
      type: email
      required: true
    - name: rating
      type: int
      min: 1
      max: 5
    - name: message
      type: text
      required: true
      max_length: 2000
    - name: agree
      type: bool
      required: true
      enum: [true]
```

The note is publicly readable (`free: true`), but `can_submit: admin` means only authenticated admins can post to it. Other visitors see the form, get a denial after pressing submit, and the JS shows a sign-in hint.

### Redirect on success

Add `success_url` to redirect the visitor to a thank-you page after a successful submit:

```yaml
form:
  can_submit: admin
  success_url: /demo/form_example?submitted=1
  fields:
    - name: email
      type: email
```

The URL is embedded into the `#form-spec` JSON and the default layout calls `window.location.href = success_url` once it gets `SubmitFormPayload` back from the server. Relative URLs stay on the same domain; absolute URLs let you point at any page (e.g. an external receipt).

## GraphQL mutation

```graphql
mutation Submit($input: SubmitFormInput!) {
  submitForm(input: $input) {
    __typename
    ... on SubmitFormPayload { submitId }
    ... on FormSubmitDeniedPayload { reason }
    ... on ErrorPayload { message byFields { name value } }
  }
}
```

`FormSubmitDeniedPayload.reason` returns one of:

| `reason` | Meaning |
|---|---|
| `admin_required` | the form's `can_submit: admin` rejects non-admins |
| `not_implemented` | `can_submit: paid_user` is recognised but not enforced yet |

Variables:

```json
{
  "input": {
    "noteVersionId": 123,
    "formId": "",
    "fields": [
      { "name": "email", "stringValue": "hi@example.com" },
      { "name": "rating", "intValue": 5 },
      { "name": "message", "stringValue": "Hello!" },
      { "name": "agree", "boolValue": true }
    ]
  }
}
```

`noteVersionId` is taken from the `note_version_id` field in the embedded `#form-spec` JSON — the same value the server validates against. `formId` is `""` for a single inline `form:` block; for `forms:` maps, use the named key.

Try the form below — it submits live to the server.
