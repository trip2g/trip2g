Images need alt text (a warning) and code blocks need a language (an error,
the highlighter can't guess).

```jsonnet
[
  {level: 'warning', line: n.line, msg: 'image %s has no alt text' % n.src}
  for n in nodes
  if n.type == 'image' && n.alt == ''
] + [
  {level: 'error', line: n.line, msg: 'code block without a language'}
  for n in nodes
  if n.type == 'code_block' && n.lang == ''
]
```
