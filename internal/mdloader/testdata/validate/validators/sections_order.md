An article has exactly these second-level sections, in this order.

```jsonnet
local want = ['Intro', 'Steps', 'FAQ'];
local got = [n.text for n in nodes if n.type == 'heading' && n.level == 2];

if got == want then []
else [{
  level: 'error',
  msg: 'sections must be %s, got %s' % [
    std.join(' → ', want),
    std.join(' → ', got),
  ],
}]
```
