FAQ pages use only third-level headings: one `###` per question.

```jsonnet
[
  {
    level: 'error',
    line: n.line,
    msg: 'heading "%s" is h%d, only h3 is allowed' % [n.text, n.level],
  }
  for n in nodes
  if n.type == 'heading' && n.level != 3
]
```
