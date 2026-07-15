---
title: Codellm English demo
fleet_id: codellm
lang: en
---

# Codellm English demo

This note belongs to the Codellm demo fleet.

The first block emits JSONL records for the next step.

```bash
printf '%s\n' '{"name":"Ada","score":7}' '{"name":"Linus","score":10}'
```

The Python block reads stdin, adds a `passed` flag, and emits JSONL again.

```python
import json
import sys

for line in sys.stdin:
    item = json.loads(line)
    item["passed"] = item["score"] >= 8
    print(json.dumps(item))
```

The final Bash block prints the names of passing records.

```bash
jq -r 'select(.passed == true) | .name'
```
