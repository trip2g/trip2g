# Subprocess Agent (subot)

## Цель

Универсальная обёртка, которая превращает любой CLI-инструмент в webhook-агента trip2g. Получает webhook, запускает подпроцесс (Claude CLI, Gemini CLI, скрипт), передаёт контекст, возвращает результат.

```bash
go run ./cmd/subot -cmd 'claude -p' -listen :3334
```

---

## Архитектура

```
┌─────────────┐     POST /webhook     ┌──────────────┐     stdin      ┌───────────┐
│   trip2g    │ ───────────────────▶  │    subot     │ ────────────▶ │ claude -p │
│  (webhook)  │                       │              │               │ gemini    │
└─────────────┘                       │  - saves     │ ◀──────────── │ any CLI   │
      ▲                               │    api_token │     stdout    └───────────┘
      │                               │  - launches  │                     │
      │    changes / API calls        │    subprocess│                     │
      └───────────────────────────────│  - proxies   │◀────────────────────┘
                                      │    MCP       │     MCP tools
                                      └──────────────┘
```

---

## Flow

1. **Webhook приходит** — subot получает POST с instruction, changes (или cron context), api_token
2. **Сохраняет токен** — в `/tmp/subot_{webhook_id}`
3. **Формирует промпт** — из instruction + контекста изменений
4. **Запускает subprocess** — команда из `-cmd` флага
5. **Передаёт промпт** — через stdin
6. **Subprocess работает** — через MCP bridge или просто выводит результат в stdout
7. **subot парсит вывод** — формирует changes для response
8. **Возвращает результат** — trip2g применяет изменения

---

## Промпт из instruction

Webhook instruction — это текст, который subot передаёт подпроцессу. Два режима:

### Прямой текст

```
Instruction: "Review this note for grammar and clarity"
```

subot передаёт как есть + добавляет контекст изменений.

### Ссылка на заметку

```
Instruction: "Follow instructions from prompts/reviewer.md"
```

subot:
1. Видит ссылку на заметку
2. Читает `prompts/reviewer.md` через API (api_token из webhook)
3. Передаёт содержимое заметки как промпт

Промпты хранятся и версионируются как обычные заметки. Можно редактировать в Obsidian.

---

## MCP bridge

subot может работать как MCP-мост между CLI-агентом и trip2g API.

При запуске с `-mcp-bridge`:
1. subot стартует локальный MCP-сервер
2. Генерирует временный MCP config: `/tmp/subot_mcp_{id}.json`
3. Передаёт конфиг подпроцессу (через флаг или ENV)

Для Claude CLI:
```bash
claude -p --mcp-config /tmp/subot_mcp_{id}.json
```

### MCP tools

| Tool | Описание |
|------|----------|
| `read_note(path)` | Прочитать заметку |
| `update_note(path, find, replace)` | Атомарный find/replace |
| `push_notes(updates)` | Создать/заменить заметки |
| `list_notes(pattern)` | Список заметок по glob-паттерну |
| `commit_notes()` | Закоммитить изменения |

Все вызовы проксируются в trip2g GraphQL API с api_token. Scoped permissions (read_patterns, write_patterns) наследуются от webhook.

---

## Конфигурация

### Флаги

```bash
subot \
  -cmd 'claude -p'          # команда для subprocess
  -listen :3334              # адрес HTTP сервера
  -webhook-secret SECRET     # HMAC верификация
  -mcp-bridge                # включить MCP bridge
  -timeout 300s              # таймаут subprocess
```

### ENV

| Переменная | Описание |
|------------|----------|
| `SUBOT_CMD` | Команда для subprocess |
| `SUBOT_LISTEN` | Адрес HTTP сервера (default `:3334`) |
| `SUBOT_WEBHOOK_SECRET` | HMAC secret |
| `SUBOT_MCP_BRIDGE` | Включить MCP bridge (`true`/`false`) |
| `SUBOT_TIMEOUT` | Таймаут subprocess (default `300s`) |

---

## Структура кода

```
agents/subprocess/
├── main.go          — entry point, flags, ENV config
├── webhook.go       — HTTP handler для webhook
├── subprocess.go    — запуск CLI, stdin/stdout
├── prompt.go        — формирование промпта из webhook payload
├── mcp.go           — MCP bridge server (проксирует в trip2g API)
└── store.go         — token store (/tmp/subot_{id})
```

---

## Примеры использования

См. [claude_agent.md](claude_agent.md) для конкретной конфигурации с Claude CLI.

### Любой CLI-скрипт

```bash
go run ./cmd/subot -cmd './scripts/linter.sh' -listen :3335
```

`linter.sh` получает на stdin контекст изменений, выводит на stdout исправления.

### Python-скрипт

```bash
go run ./cmd/subot -cmd 'python3 scripts/summarize.py' -listen :3336
```

### Цепочка агентов

Через depth control можно строить цепочки:

```
Человек правит blog/post.md (depth=0)
  → change webhook (max_depth=1) → subot + claude → грамматика (depth=1)
    → change webhook (max_depth=2) → subot + gemini → SEO-оптимизация (depth=2)
      → sync → Obsidian → человек модерирует
```
