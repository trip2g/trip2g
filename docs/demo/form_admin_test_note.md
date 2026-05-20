---
free: true
title: Form Admin Test Note
form:
  can_submit: admin
  success_url: /demo/form_admin_test_note?submitted=1
  fields:
    - name: email
      type: email
      required: true
    - name: message
      type: text
      required: true
---

E2E fixture: only admins can submit; on success the page redirects to itself with `?submitted=1`.
