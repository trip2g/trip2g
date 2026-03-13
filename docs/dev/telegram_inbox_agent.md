# Inboxbot: Telegram → trip2g inbox

## Цель

Телеграм-бот, который слушает сообщения из разрешённых чатов и записывает их в заметку-inbox через cron webhook. Живёт в `agents/tginbox/` как пример внешнего агента для trip2g webhook API.

## Сценарий

```
Пользователь отправляет сообщение в Telegram
    ↓
Inboxbot буферизует сообщение в памяти
    ↓
Cron webhook срабатывает (каждые N минут)
    ↓
trip2g POST → inboxbot /webhook
    ↓
Inboxbot возвращает changes с find/replace
    ↓
trip2g вставляет сообщения в inbox.md через find/replace маркер
```

---

## Архитектура

```
┌─────────────┐     TG messages      ┌──────────────┐
│  Telegram   │ ───────────────────▶  │   Inboxbot   │
│  (users)    │                       │              │
└─────────────┘                       │  buffer []   │
                                      │              │
┌─────────────┐     POST /webhook     │  /webhook    │
│   trip2g    │ ───────────────────▶  │  handler     │
│  (cron wh)  │ ◀─────────────────── │              │
└─────────────┘     {changes: [...]}  └──────────────┘
```

### Компоненты

| Компонент | Описание |
|-----------|----------|
| TG listener | Слушает сообщения через Telegram Bot API (long polling) |
| Message buffer | In-memory буфер сообщений между cron тиками |
| Webhook handler | HTTP-эндпоинт для приёма cron webhook |
| Token store | Сохраняет `api_token` из webhook payload в `/tmp/` |

### Два режима работы

**1. Синхронный (рекомендуемый)**: бот возвращает `changes` в ответе на webhook — trip2g сам применяет изменения. Не требует API-вызовов от бота.

**2. Асинхронный**: бот использует `api_token` из webhook для вызова `updateNote` через GraphQL. Для случаев когда нужна немедленная запись (не ждать cron tick).

---

## Cron webhook интеграция

### Настройка webhook

```
URL: http://localhost:3333/webhook
Cron: */5 * * * *              (каждые 5 минут)
pass_api_key: true             (для async режима)
write_patterns: ["inbox/**"]   (ограничить запись)
read_patterns: ["inbox/**"]
instruction: "Flush buffered Telegram messages to inbox"
```

### Получение токена

При каждом cron trigger бот получает POST с `api_token`. Бот сохраняет токен:

```
/tmp/inboxbot_{webhook_id}    — файл с последним api_token
```

Токен нужен для:
- Async режима (вызов updateNote через API)
- Чтение заметок если нужно (query API)

Если файла нет → бот ещё не получал ни одного webhook → отвечает в TG: "Бот ещё не подключён к trip2g. Ожидаю первый webhook."

### Sync ответ (основной режим)

При получении cron webhook бот проверяет буфер и возвращает changes:

```json
{
  "status": "ok",
  "message": "Flushed 3 messages",
  "changes": [
    {
      "path": "inbox/telegram.md",
      "find": "$INBOX$",
      "replace": "## 2026-02-10 15:30 — Алексей\n\nТекст сообщения\n\n---\n\n## 2026-02-10 15:32 — Алексей\n\nЕщё одно сообщение\n\n---\n\n$INBOX$"
    }
  ]
}
```

Если буфер пуст:

```json
{
  "status": "ok",
  "message": "No new messages"
}
```

Формат find/replace — см. [docs/update_note_mutation.md](update_note_mutation.md).

---

## Формат сообщений в inbox

```markdown
# Telegram Inbox

$INBOX$
```

После нескольких сообщений:

```markdown
# Telegram Inbox

## 2026-02-10 15:30 — Алексей

Текст сообщения из Telegram.

Может быть многострочным.

---

## 2026-02-10 15:32 — Алексей

> Цитата (reply)

Ответ на цитату.

---

## 2026-02-10 15:45 — Алексей

📎 [photo.jpg](./assets/photo.jpg)

Подпись к фото.

---

$INBOX$
```

### Форматирование

| TG тип | Markdown |
|--------|----------|
| Текст | Как есть (markdown entities → markdown) |
| Reply | `> Цитируемый текст\n\nОтвет` |
| Фото | `📎 photo` + caption (без загрузки файлов в MVP) |
| Документ | `📎 document: filename.pdf` |
| Стикер | `[sticker: emoji]` |
| Форвард | `↩️ Forwarded from: Name\n\nТекст` |

### Группировка по дням (опционально)

Вместо одного `inbox/telegram.md` можно писать по дням:
- `inbox/telegram/2026-02-10.md`
- `inbox/telegram/2026-02-11.md`

Путь настраивается через ENV.

---

## Конфигурация (ENV)

| Переменная | Обязательная | Описание | Пример |
|------------|-------------|----------|--------|
| `BOT_TOKEN` | да | Telegram Bot API token | `123456:ABC-DEF...` |
| `ALLOWED_CHAT_IDS` | да | Разрешённые chat ID через запятую | `-100123,456789` |
| `LISTEN_ADDR` | нет | Адрес для webhook HTTP сервера | `:3333` (default) |
| `WEBHOOK_SECRET` | нет | HMAC secret для верификации webhook | из trip2g при создании |
| `INBOX_PATH` | нет | Путь заметки для inbox | `inbox/telegram.md` (default) |
| `INBOX_MARKER` | да | Маркер для find/replace вставки | `$INBOX$`, `<!-- INSERT -->`, любая строка |
| `DATE_FORMAT` | нет | Формат даты в заголовках | `2006-01-02 15:04` (default) |

### ALLOWED_CHAT_IDS

Критическая настройка безопасности. Бот игнорирует сообщения из чатов не в списке. Поддерживает:
- Личные чаты: `123456789`
- Группы: `-100123456789`

Без этой настройки бот не запустится.

---

## Структура кода

```
agents/tginbox/
├── main.go          — entry point, ENV config, запуск goroutines
├── bot.go           — Telegram bot: long polling, буферизация сообщений
├── webhook.go       — HTTP handler для cron webhook, формирование changes
├── format.go        — форматирование TG сообщений в markdown
└── store.go         — token store (/tmp/inboxbot_{id})
```

### main.go

```go
func main() {
    cfg := loadConfig()  // ENV

    buffer := NewMessageBuffer()
    tokenStore := NewTokenStore(cfg.WebhookID)

    // Telegram bot (goroutine)
    go runTelegramBot(cfg.BotToken, cfg.AllowedChatIDs, buffer)

    // HTTP server для webhook
    http.HandleFunc("/webhook", webhookHandler(cfg, buffer, tokenStore))
    log.Fatal(http.ListenAndServe(cfg.ListenAddr, nil))
}
```

### bot.go

```go
type Message struct {
    ChatID    int64
    From      string
    Text      string
    ReplyTo   *string
    Timestamp time.Time
    Type      string  // text, photo, document, sticker, forward
}

type MessageBuffer struct {
    mu       sync.Mutex
    messages []Message
}

func (b *MessageBuffer) Add(msg Message)
func (b *MessageBuffer) Flush() []Message  // возвращает и очищает буфер
func (b *MessageBuffer) Len() int
```

### webhook.go

```go
func webhookHandler(cfg Config, buffer *MessageBuffer, store *TokenStore) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Верифицировать HMAC подпись
        // 2. Парсить payload, сохранить api_token
        // 3. Flush буфер
        // 4. Форматировать сообщения в markdown
        // 5. Вернуть changes с find/replace
    }
}
```

### Зависимости

- `github.com/go-telegram-bot-api/telegram-bot-api/v5` — Telegram Bot API
- `crypto/hmac` — верификация webhook подписи
- Стандартная библиотека Go (net/http, encoding/json)

Не зависит от основного trip2g бинарника.

---

## HMAC верификация

Бот верифицирует подпись каждого webhook запроса:

```go
func verifySignature(body []byte, secret, signature string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

Если `WEBHOOK_SECRET` не задан — бот не проверяет подпись (dev режим).

---

## MCP Tool Definitions в webhook payload

### Идея

Когда `pass_api_key=true`, webhook payload может включать описание доступных API-инструментов. Два варианта:

**Вариант A: Inline tool definitions** — для простых агентов, описание инструментов прямо в payload (JSON).

**Вариант B: MCP endpoint (рекомендуемый)** — агент подключается к обратному MCP SSE endpoint как MCP-клиент, скачивает спеку инструментов, вызывает их через стандартный MCP протокол.

### Зачем

- AI-агент получает webhook → подключается к MCP endpoint → сразу знает какие операции доступны
- Не нужно хардкодить GraphQL запросы в агенте
- Самодокументирующийся API — агент читает описание инструментов через MCP
- Стандартный формат — совместимость с MCP-клиентами
- Через MCP можно не только читать/писать заметки, но и менять статус задачи (в процессе, завершена)

### Вариант A: Inline tool definitions (для простых агентов)

Формат в payload:

```json
{
  "version": 1,
  "id": 42,
  "instruction": "...",
  "api_token": "eyJhbGc...",
  "api_base_url": "https://example.com/graphql",
  "mcp_tools": [
    {
      "name": "read_note",
      "description": "Read a note's content by path. Returns markdown content.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "path": {
            "type": "string",
            "description": "Note path, e.g. 'inbox/telegram.md'"
          }
        },
        "required": ["path"]
      }
    },
    {
      "name": "update_note",
      "description": "Atomically find and replace text in a note. Use markers like <!-- INSERT --> for insertion points.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "path": {
            "type": "string",
            "description": "Note path"
          },
          "find": {
            "type": "string",
            "description": "String to find in current content"
          },
          "replace": {
            "type": "string",
            "description": "String to replace it with"
          }
        },
        "required": ["path", "find", "replace"]
      }
    },
    {
      "name": "push_notes",
      "description": "Create or fully replace notes. Requires commit_notes after.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "updates": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "path": {"type": "string"},
                "content": {"type": "string"}
              },
              "required": ["path", "content"]
            }
          }
        },
        "required": ["updates"]
      }
    },
    {
      "name": "list_notes",
      "description": "List notes matching a glob pattern.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "pattern": {
            "type": "string",
            "description": "Glob pattern, e.g. 'inbox/**' or '*'"
          }
        }
      }
    }
  ]
}
```

### Вариант B: MCP endpoint (рекомендуемый)

Вместо отправки полных tool definitions в payload, trip2g передаёт `agent_mcp_url` — URL обратного MCP SSE endpoint:

```json
{
  "version": 1,
  "id": 42,
  "instruction": "...",
  "api_token": "eyJhbGc...",
  "agent_mcp_url": "https://example.com/mcp/agents/webhook-42"
}
```

Агент:
1. Подключается к `agent_mcp_url` как MCP-клиент (SSE transport)
2. Скачивает список доступных инструментов через `tools/list`
3. Вызывает инструменты через `tools/call` (стандартный MCP протокол)
4. trip2g транслирует вызовы в GraphQL-мутации с `api_token`

Доступные инструменты через MCP:
- `read_note(path)` — прочитать заметку
- `update_note(path, find, replace)` — атомарный find/replace
- `push_notes(updates)` — создать/заменить заметки
- `list_notes(pattern)` — список заметок по glob-паттерну
- `commit_notes()` — закоммитить изменения
- `update_task_status(task_id, status)` — **новое**: изменить статус задачи (pending/in_progress/completed)

Endpoint `agent_mcp_url` живёт только в рамках обработки одного webhook вызова (scoped session). Автоматически закрывается после завершения агента.

### Как агент использует MCP tools

Три варианта:

**Вариант A: Синхронный ответ (самый простой)**

Агент использует описание tools для понимания формата ответа. Вместо вызова API — возвращает `changes` в response body. Сервер применяет изменения сам.

```json
{
  "changes": [
    {"path": "inbox/telegram.md", "find": "$INBOX$", "replace": "...$INBOX$"}
  ]
}
```

MCP tools здесь — документация формата для агента.

**Вариант B: Асинхронный (через API)**

Агент возвращает `202 Accepted`, потом вызывает API используя `api_token`:

```
POST https://example.com/graphql
Authorization: Bearer eyJhbGc...
Content-Type: application/json

{"query": "mutation { updateNote(input: {path: \"inbox.md\", find: \"$INBOX$\", replace: \"...\"}) { ... on UpdateNotePayload { notePathId } } }"}
```

MCP tools описывают доступные операции, агент транслирует в GraphQL.

**Вариант C: MCP endpoint (рекомендуемый)**

Агент подключается к `agent_mcp_url` как MCP-клиент (вариант B выше). Это даёт:
- Стандартный протокол взаимодействия (MCP)
- Динамическое обнаружение инструментов (tools/list)
- Доступ к дополнительным операциям (update_task_status)
- Scoped session — endpoint живёт только в рамках обработки webhook

Агент может комбинировать подходы: использовать MCP для чтения/записи заметок и синхронный ответ (changes) для финальных изменений.

### Настройка

Колонка в `cron_webhooks` и `change_webhooks`:

```sql
include_mcp_tools boolean not null default false
```

Если `true` и `pass_api_key=true` → payload включает `mcp_tools` и `api_base_url`.

MCP tools определяются на сервере как константа (не хранятся в БД). Фильтруются по `read_patterns`/`write_patterns` вебхука — если write запрещён, `update_note` и `push_notes` не включаются.

---

## Жизненный цикл бота

### Запуск

```bash
BOT_TOKEN=123:ABC \
ALLOWED_CHAT_IDS=-100123,456789 \
LISTEN_ADDR=:3333 \
INBOX_PATH=inbox/telegram.md \
./inboxbot
```

### Первый запуск

1. Бот стартует, начинает слушать TG
2. Cron webhook ещё не настроен → бот работает, но не может записывать
3. Админ создаёт cron webhook в trip2g → URL бота
4. Первый cron tick → бот получает `api_token`, сохраняет
5. Бот начинает записывать сообщения

### При получении TG сообщения

1. Проверить `chat_id ∈ ALLOWED_CHAT_IDS`
2. Если нет — игнорировать
3. Добавить в буфер

### При получении webhook

1. Верифицировать HMAC подпись
2. Сохранить `api_token` в `/tmp/inboxbot_{id}`
3. Забрать буфер (flush)
4. Если буфер пуст → `{"status": "ok", "message": "No new messages"}`
5. Форматировать сообщения в markdown
6. Вернуть changes с find/replace

### Немедленная запись (async, опционально)

Если нужно записывать сразу (не ждать cron tick):

1. TG сообщение → буфер
2. Проверить есть ли сохранённый api_token
3. Если есть и не истёк → вызвать `updateNote` через API
4. Очистить буфер
5. Если токена нет → ждать следующий cron tick

---

## Безопасность

| Аспект | Решение |
|--------|---------|
| Доступ к боту | `ALLOWED_CHAT_IDS` — whitelist чатов |
| Webhook auth | HMAC-SHA256 верификация подписи |
| API token scope | `write_patterns: ["inbox/**"]` — бот пишет только в inbox |
| Token storage | `/tmp/` — не переживает reboot, новый токен придёт с cron |
| TG Bot Token | ENV переменная, не хранится в БД |

---

## План реализации

### Этап 1: Ядро

1. `agents/tginbox/main.go` — entry point, ENV config
2. `agents/tginbox/bot.go` — TG long polling, message buffer
3. `agents/tginbox/webhook.go` — HTTP handler, HMAC verify, changes response
4. `agents/tginbox/format.go` — TG message → markdown
5. `agents/tginbox/store.go` — token store

### Этап 2: Backend (updateNote)

6. Реализовать `updateNote` мутацию (см. [update_note_mutation.md](update_note_mutation.md))
7. Расширить agent response find/replace поддержкой

### Этап 3: Тестирование

8. Unit: форматирование разных типов TG сообщений
9. Unit: HMAC верификация
10. Integration: cron webhook → inboxbot → changes applied
11. Ручной тест: отправить сообщение в TG → увидеть в inbox

### Этап 4: MCP (future)

12. `include_mcp_tools` колонка в webhook таблицах
13. Генерация mcp_tools в payload
14. Фильтрация tools по read/write patterns
15. MCP SSE endpoint (отдельная фича)

---

## Решённые вопросы

1. **Sync vs Async?** Синхронный (через changes) как основной режим. Async через api_token как опция для немедленной записи.

2. **Как не потерять сообщения при перезапуске бота?** MVP: теряем (in-memory buffer). Future: SQLite или файловый буфер. При нормальной работе cron flush каждые 5 минут — потеря максимум 5 минут сообщений.

3. **Один файл или по дням?** Настраивается через ENV. Default: один файл `inbox/telegram.md`.

4. **Фото и файлы?** MVP: только текстовые описания (`📎 photo`, `📎 document: name`). Future: загрузка через uploadNoteAsset.

5. **Markdown formatting?** Telegram entities (bold, italic, code, links) конвертируются в markdown. Markdown-it compatible.

6. **Токен истёк?** Cron webhook обновляет токен при каждом срабатывании. TTL shortapitoken ≥ cron interval. Если бот долго не получал webhook — async режим отключается, sync продолжает работать.

---

## Открытые вопросы

1. **Загрузка медиа** — загружать фото/документы из TG как note assets? Требует uploadNoteAsset API через shortapitoken.

2. **Inline keyboard** — команды для бота через кнопки? (пометить как TODO, переместить в другую заметку)

3. **Multiple inboxes** — маршрутизация по чатам в разные заметки? (chat A → inbox/work.md, chat B → inbox/personal.md)

4. **Обратная связь** — отправлять в TG уведомление что сообщение записано? Или только при ошибке?
