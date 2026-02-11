# Agents

Агенты — внешние программы, которые подключаются к trip2g через вебхуки и работают с контентом. Получают уведомления об изменениях или срабатывают по расписанию, читают и модифицируют заметки.

Общие паттерны и примеры настройки: [webhook_bots.md](webhook_bots.md).

---

## Агенты

### [Subprocess Agent](subprocess_agent.md) — `agents/subprocess/`

Универсальная обёртка: запускает любой CLI как webhook-агент. Одна команда — и `claude -p`, `gemini`, или любой другой инструмент становится агентом trip2g.

```bash
go run ./cmd/subot -cmd 'claude -p' -listen :3334
```

Примеры конкретных конфигураций (Claude CLI, Gemini, Python-скрипты) см. в документе [subprocess_agent.md](subprocess_agent.md).

### [Telegram Inbox Agent](telegram_inbox_agent.md) — `agents/tginbox/`

Телеграм-бот, который слушает сообщения и записывает в заметку-inbox через cron webhook + find/replace.

---

## Future

### Email Inbox Agent

IMAP listener → заметка-inbox. Аналог tginbox но для почты.

- **Паттерн**: listener agent (буферизует между cron тиками)
- **Триггер**: cron webhook
- **Результат**: `inbox/email.md` через find/replace
- **ENV**: `IMAP_HOST`, `IMAP_USER`, `IMAP_PASS`, `ALLOWED_SENDERS`

### RSS Reader Agent

Подписка на внешние RSS-фиды, новые записи → заметки. trip2g сам поддерживает RSS на любую заметку (`/path/to/note.rss.xml`) — можно подписаться на другой trip2g инстанс и трекать чужие изменения.

- **Паттерн**: listener agent (cron polling фидов)
- **Триггер**: cron webhook (каждые 15-30 мин)
- **Результат**: `feeds/{source}/YYYY-MM-DD-title.md` или find/replace в `feeds/{source}.md`
- **ENV**: `FEEDS` (JSON: `[{"url": "...", "path": "feeds/source.md"}]`)
- **trip2g→trip2g**: подписка на RSS другого инстанса = зеркалирование контента между инстансами

### Subprocess примеры (prompt + config)

Не требуют отдельного кода — это конфигурации [subprocess agent](subprocess_agent.md):

| Агент | Промпт | Триггер |
|-------|--------|---------|
| Линтер | `prompts/linter.md` | change webhook, `blog/**` |
| Переводчик | `prompts/translator.md` | change webhook, `blog/**` |
| Тегировщик | `prompts/tagger.md` | change webhook, on_create only |
| Дайджест | `prompts/digest.md` | cron, раз в неделю |
| SEO | `prompts/seo.md` | change webhook, `blog/**` |

---

## Как подключить агента

1. Написать бот (или использовать `cmd/subot` как обёртку)
2. Создать webhook в админке: URL бота, паттерны, instruction
3. Бот получает POST при триггере
4. Бот возвращает `changes` (sync) или работает через API (async)

Подробности: [webhook_bots.md](webhook_bots.md), [change_webhooks.md](change_webhooks.md), [cron_webhooks.md](cron_webhooks.md).

---

## API для агентов

| Операция | Механизм |
|----------|----------|
| Вернуть изменения | `changes` в response body ([shared_webhooks.md](shared_webhooks.md)) |
| Атомарная вставка | find/replace в changes ([update_note_mutation.md](update_note_mutation.md)) |
| Читать/писать через API | `api_token` из webhook payload → GraphQL |
| MCP tools | Описание инструментов в payload ([telegram_inbox_agent.md](telegram_inbox_agent.md#mcp-tool-definitions-в-webhook-payload)) |

---

## Стейт агента через frontmatter

Заметки — не только контент, но и хранилище состояния агента. Агент может использовать frontmatter для своих метаданных:

```markdown
---
last_processed_at: 2026-02-10T15:30:00Z
processed_count: 42
last_rss_guid: "https://example.com/post-123"
agent: rss-reader
---

# Feed: Example Blog

$FEED$
```

Агент при следующем запуске читает frontmatter через API, видит `last_processed_at` / `last_rss_guid`, обрабатывает только новое, обновляет frontmatter через find/replace.

Преимущества:
- **Нет внешнего хранилища** — стейт живёт в самой заметке
- **Версионируется** — история изменений стейта бесплатно
- **Видим** — человек может посмотреть и поправить в Obsidian
- **Переносим** — стейт переезжает вместе с заметкой
