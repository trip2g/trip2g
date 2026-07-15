---
title: Codellm unlabelled demo
fleet_id: codellm
---

# Codellm unlabelled demo

This note exercises filtering by fleet without a language predicate.

This Bash block creates a small JSONL stream.

```bash
printf '%s\n' '{"task":"parse","done":true}' '{"task":"render","done":false}'
```

The Python block keeps only completed tasks and passes JSONL to the next block.

```python
import json
import sys

for line in sys.stdin:
    task = json.loads(line)
    if task["done"]:
        print(json.dumps(task))
```

The final Bash block displays the task names from the filtered stream.

```bash
jq -r '.task'
```
