# Claude Agent

## Цель

Конкретная конфигурация [subprocess agent](subprocess_agent.md) для Claude CLI. Получает промпт из instruction вебхука, запускает `claude -p`, Claude работает с заметками через MCP bridge.

---

## Запуск

```bash
go run ./cmd/subot \
  -cmd 'claude -p' \
  -listen :3334 \
  -mcp-bridge \
  -webhook-secret $SECRET
```

Claude CLI в pipe-режиме (`-p`) читает промпт из stdin и выводит результат в stdout. С `-mcp-bridge` Claude получает MCP tools для чтения/записи заметок.

---

## Flow

```
trip2g webhook
  │
  ▼
subot получает POST
  │
  ├── instruction: "Review blog/post.md for clarity. See prompts/reviewer.md"
  ├── changes: [{path: "blog/post.md", event: "update", content: "..."}]
  ├── api_token: "eyJ..."
  │
  ▼
subot формирует промпт
  │
  ├── читает prompts/reviewer.md через API (развёрнутая instruction)
  ├── добавляет контекст: какие файлы изменились, их содержимое
  ├── генерирует MCP config с api_token
  │
  ▼
claude -p --mcp-config /tmp/subot_mcp_{id}.json
  │
  ├── Claude читает промпт из stdin
  ├── Claude вызывает read_note("blog/post.md") через MCP
  ├── Claude анализирует, формирует правки
  ├── Claude вызывает update_note("blog/post.md", find, replace) через MCP
  │
  ▼
subot завершает, возвращает результат trip2g
```

---

## Примеры вебхуков

### AI-ревьюер (change webhook)

При изменении заметок в `blog/` — Claude проверяет грамматику и стиль.

```
URL: http://localhost:3334/webhook
Include patterns: ["blog/**"]
On create: true, On update: true, On remove: false
Instruction: "Follow instructions from prompts/reviewer.md"
Pass API key: true
Read patterns: ["blog/**", "prompts/**"]
Write patterns: ["blog/**"]
Max depth: 1
```

`prompts/reviewer.md`:
```markdown
# Reviewer Prompt

Review the changed notes for:
- Grammar and spelling
- Clarity and readability
- Consistent formatting

Fix issues directly using update_note tool. Only fix clear errors, don't rewrite style.
```

### AI-суммаризатор (cron webhook)

Каждое утро генерирует summary за вчера.

```
URL: http://localhost:3334/webhook
Cron: 0 9 * * *
Instruction: "Follow instructions from prompts/daily-summary.md"
Pass API key: true
Read patterns: ["blog/**", "prompts/**"]
Write patterns: ["digests/**"]
```

`prompts/daily-summary.md`:
```markdown
# Daily Summary Prompt

1. List all notes in blog/** modified in the last 24 hours
2. Generate a brief summary of changes
3. Write to digests/YYYY-MM-DD.md using update_note with marker $DIGEST$
```

### AI-тегировщик (change webhook)

При создании новой заметки — Claude предлагает теги.

```
URL: http://localhost:3334/webhook
Include patterns: ["blog/**"]
On create: true, On update: false, On remove: false
Instruction: "Read the new note. Suggest 3-5 tags based on content. Add them to frontmatter using update_note: find 'tags: []' replace 'tags: [tag1, tag2, ...]'"
Pass API key: true
Read patterns: ["blog/**"]
Write patterns: ["blog/**"]
Max depth: 1
```

---

## Промпты как заметки

Ключевой паттерн: instruction в вебхуке ссылается на заметку, а не содержит промпт напрямую.

Преимущества:
- **Версионирование** — история изменений промпта
- **Obsidian** — редактирование промптов в привычном редакторе
- **Длина** — промпт может быть сколько угодно длинным
- **Переиспользование** — один промпт для нескольких вебхуков
- **AI-доступ** — Claude может прочитать промпт через `read_note`

Конвенция: промпты живут в `prompts/`.

---

## Альтернативные LLM

Тот же subot с другим `-cmd`:

```bash
# Gemini
go run ./cmd/subot -cmd 'gemini --pipe' -listen :3334

# Llama (через ollama)
go run ./cmd/subot -cmd 'ollama run llama3' -listen :3334

# Любой скрипт с LLM API
go run ./cmd/subot -cmd 'python3 scripts/openai_agent.py' -listen :3334
```

MCP bridge работает одинаково — tools не зависят от конкретного LLM.
