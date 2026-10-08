---
title: A form in a note
free: true
layout: form
form:
  can_submit: guest
  turnstile: true
  thanks: Your answers just reached the people who build trip2g.
  fields:
    - name: use_case
      type: text
      required: true
      enum: [Personal notes, Team knowledge base, Docs for a product, Paid content, Something else]
      label: What would you use trip2g for?
      hint: Pick the closest one.
    - name: feature_mcp
      type: bool
      group: Which features interest you?
      label: AI agents reading the site over MCP
    - name: feature_forms
      type: bool
      group: Which features interest you?
      label: Forms like this one
    - name: feature_paid
      type: bool
      group: Which features interest you?
      label: Paid access to notes
    - name: feature_layouts
      type: bool
      group: Which features interest you?
      label: Custom layouts
    - name: feature_obsidian
      type: bool
      group: Which features interest you?
      label: Publishing straight from Obsidian
    - name: try_it
      type: int
      required: true
      min: 1
      max: 5
      label: How likely are you to try trip2g?
      hint: 1 means not likely, 5 means you already use it.
    - name: comment
      type: text
      max_length: 200
      label: Anything else?
      hint: Optional. One line is plenty.
---

A four-question survey about trip2g, and a working example of a form in a note.

- **The note's frontmatter defines the form.** Fields, types, limits and who may answer all live under `form:`. See [[en/user/forms|Forms in notes]].
- **A layout renders it.** This page uses `form.html` from [form_template](https://github.com/trip2g/form_template), a single HTML file you drop into `_layouts/`.
- **Turnstile guards it.** Cloudflare's captcha appears when you press Submit.
- **Answers land in the admin panel,** under Forms. To send them from an app of your own, see [[en/user/spa#forms|Forms in an app]].

The source of this page, frontmatter included, is [docs/demo_form.md](https://github.com/trip2g/trip2g/blob/main/docs/demo_form.md) on GitHub.
