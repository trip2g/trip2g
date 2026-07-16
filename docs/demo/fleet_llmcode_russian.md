---
title: Codellm Russian demo
fleet_id: codellm
lang: ru
---

# Codellm Russian demo

Эта заметка принадлежит demo fleet Codellm и используется в тестах GraphQL и
отладчика последовательных markdown-блоков.

## JSONL pipeline

Первый блок выдаёт JSON Lines. Следующий блок читает их из stdin и выдаёт
следующий JSON Lines — это удобно выполнять по шагам в debugger.

```bash
printf '%s\n' '{"name":"Ada","score":7}' '{"name":"Linus","score":10}'
```

Следующий Python-блок читает JSONL из stdin, оставляет исходные поля и
добавляет boolean-поле `passed`: оно равно `true`, если `score` не меньше 8.

```python
import json
import sys

for line in sys.stdin:
    item = json.loads(line)
    item["passed"] = item["score"] >= 8
    print(json.dumps(item, ensure_ascii=False))
```

Последний Bash-блок читает преобразованный JSONL и печатает имена только тех
записей, у которых `passed` равен `true`.

```bash
jq -r 'select(.passed == true) | .name'
```

Ожидаемый вывод последнего блока — `Linus`.
