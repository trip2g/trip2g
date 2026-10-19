---
title: "A flat-file, contentless CMS"
label: "flat-file CMS"
order: 9
motion_visualization: files
link: "[[en/thoughts/flat-file-cms]]"
related:
  - "[[en/thoughts/contentless-cms]]"
  - "[[en/thoughts/markdown-operating-system]]"
  - "[[en/user/self-hosted-wiki-sqlite]]"
  - "[[en/user/docs-image]]"
layout: mesh/circle
lang: en
free: true
noindex: true
lang_redirect: "[[ru/circle/files]]"
---

Content lives in markdown files in your own folder, not in the CMS's database. The server renders them at runtime: one Go binary on one SQLite file, no separate database server, no build step. Edit a file and the page changes; leave and you take every file with you. A whole base can even ship as one container image.
