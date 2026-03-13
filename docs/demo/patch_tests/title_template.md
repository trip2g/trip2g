---
# Expected patches: "Site title suffix" (priority 100)
# Expected meta: { title: "Title Template Test — Test Site", free: true }
# Tests: meta + { title: meta.title + " — Test Site" }
free: true
title: Title Template Test
---

This page tests title template patch using jsonnet expression:
`meta + { title: meta.title + " — Test Site" }`

The patch should append " — Test Site" to the original title.

Final title should be: "Title Template Test — Test Site"
