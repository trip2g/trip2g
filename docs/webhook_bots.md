# Webhook Bots: паттерны и примеры

## Концепция

Webhook bots — внешние программы, которые подключаются к trip2g через систему вебхуков. Бот получает уведомления (change или cron), обрабатывает их и возвращает изменения обратно. Боты работают как автономные процессы — могут быть на любом языке, на любом сервере.

Два типа триггеров:
- **Change webhook** — бот реагирует на изменения заметок (линтер, индексатор, AI-ревьюер)
- **Cron webhook** — бот запускается по расписанию (дайджест, инбокс, генератор)

Подробности: [change_webhooks.md](change_webhooks.md), [cron_webhooks.md](cron_webhooks.md), [shared_webhooks.md](shared_webhooks.md).

---

## Паттерны

### 1. Sync bot (return changes)

Бот получает webhook, обрабатывает, возвращает `changes` в response body. trip2g сам применяет изменения. Бот не вызывает API.

```
trip2g POST → bot → {changes: [...]} → trip2g applies
```

Плюсы: простота, атомарность, не нужен HTTP-клиент в боте.
Минусы: ограничен таймаутом (default 60s).

Поддерживает [find/replace](update_note_mutation.md) для атомарных вставок.

### 2. Async bot (use API)

Бот получает webhook с `api_token`, возвращает `202 Accepted`, работает через API (pushNotes, updateNote, commitNotes).

```
trip2g POST → bot → 202 → bot calls API → trip2g
```

Плюсы: нет ограничения по времени, может делать несколько операций.
Минусы: сложнее, нужен HTTP-клиент, race conditions.

### 3. Subprocess bot (subot)

Бот-обёртка: получает webhook, запускает CLI-подпроцесс (Claude, Gemini, любой LLM), передаёт контекст и инструкции. Подпроцесс работает с заметками через MCP или stdin/stdout.

```
trip2g POST → subot → launches `claude -p` → AI reads/writes notes → trip2g
```

См. [subot](#subot-subprocess-bot).

---

## Примеры ботов

### inboxbot — Telegram → inbox

Телеграм-бот, который слушает сообщения и записывает их в заметку-inbox.

- **Тип**: sync bot + cron webhook
- **Триггер**: cron (каждые 5 минут)
- **Механизм**: буферизует TG-сообщения, на cron tick возвращает changes с find/replace
- **Код**: `cmd/inboxbot/`
- **Дизайн**: [telegram_inbox_agent.md](telegram_inbox_agent.md)
- **Код**: `agents/tginbox/`

### subot — subprocess bot

Универсальная обёртка для запуска CLI-агентов (Claude CLI, Gemini CLI, любой LLM) как webhook-ботов.

- **Тип**: async bot + change/cron webhook
- **Триггер**: change (файлы изменились) или cron (по расписанию)
- **Механизм**: запускает subprocess, передаёт контекст, агент работает через MCP
- **Код**: `agents/subprocess/`

#### Идея

Одна команда превращает любой CLI-инструмент в webhook-бота:

```bash
go run ./cmd/subot \
  -cmd 'claude -p' \
  -listen :3334 \
  -webhook-secret $SECRET
```

Или с Gemini:

```bash
go run ./cmd/subot \
  -cmd 'gemini --pipe' \
  -listen :3334
```

#### Архитектура

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

#### Flow

1. **Webhook приходит** — subot получает POST с instruction, changes, api_token
2. **Сохраняет токен** — в `/tmp/subot_{webhook_id}`
3. **Формирует промпт** — из instruction + контекста изменений + описания доступных инструментов
4. **Запускает subprocess** — `claude -p` или другой CLI
5. **Передаёт промпт** — через stdin
6. **Агент работает** — читает/пишет заметки через MCP tools (subot проксирует в trip2g API)
7. **Агент завершается** — subot парсит stdout, формирует ответ
8. **Возвращает результат** — changes в response body или уже применил через API

#### MCP bridge

subot может работать как MCP-мост между CLI-агентом и trip2g:

```
claude -p --mcp-config /tmp/subot_mcp_{id}.json
```

subot генерирует временный MCP config, который указывает на локальный MCP-сервер subot'а. Этот сервер проксирует вызовы в trip2g GraphQL API с api_token.

Доступные MCP tools для агента:
- `read_note(path)` — прочитать заметку
- `update_note(path, find, replace)` — атомарный find/replace
- `push_notes(updates)` — создать/заменить заметки
- `list_notes(pattern)` — список заметок по glob-паттерну
- `commit_notes()` — закоммитить изменения

#### Инструкции из заметок

Ключевая фишка: instruction из webhook может быть ссылкой на заметку. subot читает заметку через API и передаёт как полный промпт:

```
Webhook instruction: "Read prompts/reviewer.md and follow instructions for changed files"
```

subot:
1. Получает instruction
2. Видит ссылку на `prompts/reviewer.md`
3. Читает заметку через `read_note` (api_token из webhook)
4. Передаёт содержимое как промпт для Claude

Это позволяет хранить и версионировать промпты как обычные заметки.

Преимущества хранения промптов в заметках:
- **Версионирование** — история изменений промпта в git
- **Obsidian** — редактирование промптов в привычном редакторе
- **Длина** — промпт может быть сколько угодно длинным
- **Переиспользование** — один промпт для нескольких вебхуков
- **AI-доступ** — агент может прочитать промпт через `read_note`

Конвенция: промпты живут в `prompts/`.

#### Конфигурация

```bash
subot \
  -cmd 'claude -p'          # команда для subprocess
  -listen :3334              # адрес HTTP сервера
  -webhook-secret SECRET     # HMAC верификация
  -mcp-bridge                # включить MCP bridge (опционально)
  -timeout 300s              # таймаут subprocess
```

#### ENV

| Переменная | Описание |
|------------|----------|
| `SUBOT_CMD` | Команда для subprocess (альтернатива флагу `-cmd`) |
| `SUBOT_LISTEN` | Адрес HTTP сервера |
| `SUBOT_WEBHOOK_SECRET` | HMAC secret |
| `SUBOT_MCP_BRIDGE` | Включить MCP bridge (`true`/`false`) |
| `SUBOT_TIMEOUT` | Таймаут subprocess |

#### Структура кода

```
cmd/subot/
├── main.go          — entry point, flags, ENV config
├── webhook.go       — HTTP handler для webhook
├── subprocess.go    — запуск CLI, передача stdin/stdout
├── prompt.go        — формирование промпта из webhook payload
├── mcp.go           — MCP bridge server (проксирует в trip2g API)
└── store.go         — token store (/tmp/subot_{id})
```

---

## Безопасность

### Depth protection

Все боты работают через shortapitoken с `depth+1`. Это предотвращает бесконечные циклы:

```
Человек правит blog/post.md (depth=0)
  → change webhook → линтер-бот (depth=1)
    → линтер пушит исправления (depth=1)
      → change webhook max_depth=1 → НЕ триггерится (1 >= 1)
```

### Chain tracing

`X-Webhook-Chain-ID` header передаётся во всех webhook вызовах одной цепочки:
- Генерируется при первом триггере (depth=0)
- Передаётся дальше с каждым вызовом вебхука в цепочке
- Записывается в delivery log
- Позволяет отследить полную цепочку обработки и найти все связанные deliveries

Пример:
```
Человек правит blog/post.md → chain_id: abc123
  → change webhook → линтер-бот (chain_id: abc123)
    → линтер пушит исправления → change webhook → SEO-бот (chain_id: abc123)
```

Все три delivery в логе будут иметь одинаковый `chain_id`, что позволяет увидеть всю цепочку автоматической обработки.

### Write scope

`write_patterns` ограничивают что бот может менять:
- Линтер: `write_patterns: ["blog/**"]` — только blog
- Inboxbot: `write_patterns: ["inbox/**"]` — только inbox
- Subot: настраивается per-webhook

### HMAC verification

Каждый webhook подписан HMAC-SHA256. Бот верифицирует подпись перед обработкой.

---

## Примеры настройки вебхуков

### Линтер (change webhook)

```
URL: http://localhost:3335/webhook
Include patterns: ["blog/**", "docs/**"]
On create: true, On update: true, On remove: false
Pass API key: false (sync mode — возвращает changes)
Write patterns: ["blog/**", "docs/**"]
Max depth: 1
```

### AI-ревьюер (change webhook + subot)

```
URL: http://localhost:3334/webhook
Include patterns: ["blog/**"]
On create: true, On update: true, On remove: false
Instruction: "Review the note for clarity and grammar. Read prompts/reviewer.md for style guide."
Pass API key: true (subot needs API access for MCP)
Write patterns: ["blog/**"]
Read patterns: ["prompts/**", "blog/**"]
Max depth: 1
```

### AI-тегировщик (change webhook + subot)

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

### Дайджест-генератор (cron webhook + subot)

```
URL: http://localhost:3334/webhook
Cron: 0 9 * * 1 (каждый понедельник в 9:00)
Instruction: "Generate a weekly digest. Read prompts/digest.md for format."
Pass API key: true
Write patterns: ["digests/**"]
Read patterns: ["blog/**", "prompts/**"]
```

### Telegram inbox (cron webhook + inboxbot)

```
URL: http://localhost:3333/webhook
Cron: */5 * * * * (каждые 5 минут)
Pass API key: true
Write patterns: ["inbox/**"]
```

---

## Obsidian loop

Важный паттерн: AI-правки попадают обратно в Obsidian vault через sync.

```
Человек пишет в Obsidian
  → sync → trip2g (pushNotes + commitNotes)
    → change webhook → AI-бот
      → AI правит заметку
        → sync ← trip2g
          → Obsidian показывает diff
            → Человек модерирует
```

Человек всегда видит что AI изменил и может принять или откатить. Obsidian — это UI для модерации AI-контента.
