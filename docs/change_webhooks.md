# Change Webhooks: дизайн-документ

## Цель

Вебхуки уведомляют внешние сервисы (агенты, автоматизации) об изменениях заметок. Агент получает POST-запрос с информацией о том какие заметки изменились и может отреагировать — например, пересобрать индекс, обновить кеш, запустить линтер, или вызвать AI-агента. Агент может вернуть изменения, которые сервер применит автоматически.

Предполагается что на стороне получателя — MCP-инструмент, который дёргает API через shortapitoken.

Общая инфраструктура (shortapitoken, HMAC, agent response, retry, debug endpoints) описана в [docs/shared_webhooks.md](shared_webhooks.md).

## Сценарий использования

1. Админ создает вебхук: URL + include/exclude паттерны (`blog/*`, `guides/**`, `*`)
2. Админ пушит/коммитит/скрывает заметки
3. Система определяет depth запроса (0 для прямых, из shortapitoken для агентных)
4. Система матчит изменённые пути по glob-паттернам активных вебхуков, пропускает если `depth >= max_depth`
5. Фильтрует по типу события: on_create/on_update/on_remove
6. Для каждого совпавшего вебхука — собирает batch изменений, создает change_webhook_delivery запись
7. Запускает фоновую задачу (goqite) для отправки POST с батчем всех изменений
8. Если включён `pass_api_key` — генерирует shortapitoken (JWT) с depth+1 и read/write patterns, передает в payload
9. Если агент вернул изменения в ответе — применяет их через InsertNote (см. shared_webhooks.md, раздел "Формат ответа агента")

---

## Таблицы

### `change_webhooks`

```sql
create table change_webhooks (
  id integer primary key autoincrement,
  url text not null,
  include_patterns text not null,            -- JSON array: ["blog/**","docs/*"]
  exclude_patterns text not null default '[]', -- JSON array: ["*.draft.md"]
  instruction text not null default '',       -- текстовая инструкция для агента
  secret text not null,                      -- HMAC secret, автогенерируется если не задан
  max_depth integer not null default 1,      -- макс. depth для триггера (1 = только прямые правки)
  pass_api_key boolean not null default false,-- генерировать shortapitoken
  include_content boolean not null default true, -- включать содержимое заметок в payload
  timeout_seconds integer not null default 60,  -- таймаут HTTP ответа
  max_retries integer not null default 0,       -- retry при ошибках agent response
  on_create boolean not null default true,       -- триггерить на create events
  on_update boolean not null default true,       -- триггерить на update events
  on_remove boolean not null default true,       -- триггерить на remove events
  read_patterns text not null default '["*"]',   -- JSON array glob patterns для чтения агентом
  write_patterns text not null default '[]',      -- JSON array glob patterns для записи агентом
  enabled boolean not null default true,
  description text not null default '',
  created_at datetime not null default (datetime('now')),
  created_by integer not null references admins(user_id) on delete restrict,
  updated_at datetime not null default (datetime('now')),
  disabled_at datetime,
  disabled_by integer references admins(user_id) on delete restrict
);
```

**Заметки:**
- `include_patterns` / `exclude_patterns` — JSON array строк. Заметка матчится если подходит хотя бы под один include паттерн И НЕ подходит ни под один exclude. Матчинг через `doublestar.Match` (уже зависимость проекта, см. `internal/templateviews/query.go`). Поддерживает `*` (один уровень) и `**` (рекурсивно).
- `secret` — **всегда** задан. Автогенерируется при создании вебхука если не указан вручную. Payload всегда подписывается HMAC-SHA256. Показывается один раз при создании (как API keys).
- `max_depth` — защита от рекурсии. 1 = вебхук триггерится только на прямые правки (depth=0). 2 = триггерится на правки от агентов первого уровня. Вебхук пропускается если `depth >= max_depth`. Default: **1** (только прямые правки).
- `pass_api_key` — если true, в payload включается shortapitoken (JWT, TTL из appconfig, read+write доступ к API).
- `instruction` — текстовая инструкция для агента. Позволяет использовать один endpoint для разных задач: "проверь орфографию", "обнови SEO описания". Передаётся в payload.
- `include_content` — если true, в payload включается полное содержимое заметок. Для remove event — `content: null`.
- `timeout_seconds` — таймаут ожидания HTTP ответа (default 60s). Для AI-агентов можно увеличить.
- `max_retries` — если > 0 и agent response вызвал ошибки (expected_hash mismatch, InsertNote failed), delivery переотправляется с информацией об ошибке в payload. Default: 0 (ошибки agent response не приводят к retry).
- `on_create`/`on_update`/`on_remove` — boolean фильтры по типу события. Все true по умолчанию. Webhook получает только события matching его фильтрам.
- `read_patterns` — glob patterns для чтения. Default `["*"]` (читать всё). Передаются в shortapitoken JWT.
- `write_patterns` — glob patterns для записи. Default `[]` (ничего не писать, безопасный default). Админ явно открывает запись. Передаются в shortapitoken JWT.

### `change_webhook_deliveries`

```sql
create table change_webhook_deliveries (
  id integer primary key autoincrement,
  webhook_id integer not null references change_webhooks(id) on delete cascade,
  status text not null default 'pending',    -- pending, success, failed
  response_status integer,                   -- HTTP status code ответа
  attempt integer not null default 1,        -- номер попытки
  duration_ms integer,                       -- время ответа в мс
  created_at datetime not null default (datetime('now')),
  completed_at datetime
);
```

**Заметки:**
- Тяжёлые данные (request_body, response_body, error_message) хранятся в `webhook_delivery_logs` — см. [shared_webhooks.md](shared_webhooks.md).
- Индексы: `(webhook_id, created_at)` для просмотра истории конкретного вебхука.

### Изменения в `api_keys`

```sql
-- добавить колонку
alter table api_keys add column skip_webhooks boolean not null default false;
```

API ключи с `skip_webhooks=true` не триггерят вебхуки при commitNotes/hideNotes. Для агентов-линтеров, которые пушат исправления и не должны создавать цепочку.

---

## Защита от рекурсии (depth)

Концептуальное описание depth-механизма — см. shared_webhooks.md, раздел "Depth / Защита от рекурсии".

### Сценарий цепочки agent1 -> agent2

```
webhook1 (линтер):     max_depth=1, include: ["blog/**"]
webhook2 (индексатор): max_depth=2, include: ["*"]

1. Админ пушит blog/post.md -> depth=0
2. depth(0) >= max_depth(1)? Нет -> webhook1 триггерится
   on_update=true? Да -> доставляем

3. Линтер получает webhook, правит файл, пушит через shortapitoken с depth=1
4. depth(1) >= max_depth(1)? Да -> webhook1 НЕ триггерится (нет рекурсии)
5. depth(1) >= max_depth(2)? Нет -> webhook2 триггерится

6. Индексатор получает webhook, обрабатывает и пушит метаданные через depth=2
7. depth(2) >= max_depth(2)? Да -> webhook2 НЕ триггерится (нет рекурсии)
```

**Итого:**
- `max_depth=1` — срабатывает на прямые правки (depth=0)
- `max_depth=2` — срабатывает на прямые + правки агентов первого уровня
- `max_depth=0` — вебхук отключён (никогда не срабатывает)
- Default: `max_depth=1` (срабатывает только на прямые правки)

---

## Точки интеграции

### Когда триггерить вебхуки

**Только после `commitNotes` и `hideNotes`. НЕ после `pushNotes`.**

`pushNotes` загружает сырые данные (заметки, ассеты) во временное хранилище. На этом этапе ассеты могут быть ещё недоступны, данные не финализированы. `commitNotes` завершает транзакцию: ассеты доступны, заметки в финальном состоянии, подграфы обновлены. Вебхуки триггерятся только на этом этапе.

### Где перехватывать изменения

| Событие | Место | Что меняется |
|---------|-------|-------------|
| **create/update** | `HandleLatestNotesAfterSave(ctx, changedPathIDs)` — вызывается из `commitNotes` | Path IDs — созданы или обновлены |
| **remove (hide)** | `hidenotes.Resolve(ctx, env, input)` | Пути — скрыты |

### `HandleLatestNotesAfterSave` — create/update

```
cmd/server/main.go:997
```

Вызывается из `commitNotes`, НЕ из `pushNotes`.

Текущий flow:
1. `updatesubgraphs.Resolve()` — обновить подграфы
2. `handletgpublishviews.Resolve()` — Telegram посты
3. Vector embeddings — если включены

**Добавить 4-й шаг:**
```go
// 4. Trigger webhook deliveries for changed notes
depth := depthFromCtx(ctx) // 0 для обычных запросов, из shortapitoken для агентных
err = a.HandleNoteWebhooks(ctx, changedPathIDs, "update", depth)
```

### `hidenotes.Resolve` — remove

```
internal/case/hidenotes/resolve.go:17
```

Path IDs нужно получить до скрытия (после скрытия заметка может быть недоступна в LatestNoteViews).

---

## Архитектура

### Flow: от изменения до доставки

```
commitNotes / hideNotes          (НЕ pushNotes — ассеты ещё недоступны)
    |
    +-- определить depth из auth context:
    |   +-- API key (skip_webhooks=true) -> return, не триггерить
    |   +-- API key (обычный) -> depth=0
    |   +-- shortapitoken (JWT) -> depth из claims
    |
    v
HandleLatestNotesAfterSave(pathIDs)  или  hidenotes.Resolve(pathIDs)
    |
    v
app.HandleNoteWebhooks(ctx, changedPathIDs, event, depth)
    |                      [синхронная часть — только вычисления]
    +-- Загрузить все enabled webhooks из БД (ListEnabledWebhooks)
    +-- Для каждого webhook:
    |   +-- depth >= max_depth? -> skip
    |   +-- event filtering: on_create/on_update/on_remove check -> skip если false
    |   +-- LatestNoteViews() -> получить paths по pathIDs
    |   +-- glob match: include/exclude patterns через doublestar.Match
    |   +-- Если есть совпадения -> собрать batch (сортировка по path)
    |   +-- Сохранить change_webhook_delivery (status=pending)
    |   +-- Enqueue goqite job: deliver_webhook(delivery_id)
    |
    v
[goqite worker — BackgroundDefaultQueue]
    |
    +-- Загрузить delivery + webhook из БД
    +-- Если pass_api_key -> создать shortapitoken JWT (TTL из appconfig, depth=current+1, read/write patterns из webhook)
    +-- Подписать payload HMAC-SHA256 (webhook.secret) — см. shared_webhooks.md
    +-- POST url с payload + headers
    +-- Сохранить результат (status, response, duration)
    +-- Если ответ 2xx и содержит changes[] -> импортировать через InsertNote — см. shared_webhooks.md
    +-- Если ошибка -> retry через max_retries — см. shared_webhooks.md
```

### Фильтрация по типу события

В HandleNoteWebhooks перед glob-матчингом проверяем boolean фильтры:

```go
func HandleNoteWebhooks(ctx, changedPathIDs, event, depth) {
    if apiKey.SkipWebhooks {
        return
    }

    webhooks := ListEnabledWebhooks()
    for _, wh := range webhooks {
        if depth >= wh.MaxDepth {
            continue
        }

        // Event type filtering
        switch event {
        case "create":
            if !wh.OnCreate {
                continue
            }
        case "update":
            if !wh.OnUpdate {
                continue
            }
        case "remove":
            if !wh.OnRemove {
                continue
            }
        }

        // ... glob matching, delivery, enqueue
    }
}
```

### Батчинг

Если за один коммит изменилось 10 заметок и 7 из них матчат glob `blog/*`:
- **Один** вызов вебхука с массивом из 7 изменений
- Не 7 отдельных вызовов

Changes сортируются по `path` в алфавитном порядке (детерминизм).

### Background Job

```
internal/case/backjob/deliverwebhook/
+-- job.go          — JobID, QueueID, Priority
+-- resolve.go      — HTTP POST логика
+-- resolve_test.go — тесты
```

Параметры job:
```go
const JobID = "deliver_webhook"
const QueueID = model.BackgroundDefaultQueue
const Priority = 100  // низкий приоритет, не блокировать основные задачи
```

### HTTP таймауты

| Параметр | Значение |
|----------|----------|
| Connect timeout | 5s |
| Response timeout | `webhook.timeout_seconds` (default 60s) |
| Read body limit | 1MB |

---

## Payload вебхука

### HTTP-запрос

HTTP заголовки — см. shared_webhooks.md, раздел "HTTP заголовки".

### Body

```json
{
  "version": 1,
  "id": 42,
  "timestamp": 1738000000,
  "attempt": 1,
  "depth": 0,
  "instruction": "Проверь орфографию и грамматику",
  "changes": [
    {
      "path": "blog/my-post.md",
      "event": "update",
      "path_id": 123,
      "version": 5,
      "title": "My Post",
      "content": "# My Post\n\nContent here..."
    },
    {
      "path": "blog/new-post.md",
      "event": "create",
      "path_id": 456,
      "version": 1,
      "title": "New Post",
      "content": "# New Post\n\nMore content..."
    }
  ],
  "api_token": "eyJhbGc..."
}
```

**Поля:**
- `version` — версия формата payload (всегда `1`). См. shared_webhooks.md
- `id` — ID доставки (`change_webhook_deliveries.id`) для дедупликации
- `timestamp` — Unix время создания
- `attempt` — номер попытки (1, 2, 3)
- `depth` — текущий уровень глубины (0 = прямая правка, 1+ = правка от агента)
- `changes[]` — массив изменений, отсортированный по `path` (алфавит):
  - `path` — полный путь заметки
  - `event` — тип: `create`, `update`, `remove`
  - `path_id` — ID пути в БД
  - `version` — текущая версия (для create/update; для remove — последняя известная)
  - `title` — заголовок заметки
  - `content` — содержимое (если `include_content=true`; для remove — `null`)
- `api_token` — shortapitoken JWT (только если `pass_api_key=true`). Даёт read+write доступ к API (TTL из appconfig, по умолчанию 60 мин). Содержит depth+1 и read/write patterns в claims
- `previous_error` — (только при retry) описание ошибки предыдущей попытки. Агент может использовать для корректировки ответа

### Retry payload (attempt > 1)

При retry payload включает `previous_error`:

```json
{
  "version": 1,
  "id": 42,
  "timestamp": 1738000000,
  "attempt": 2,
  "depth": 0,
  "instruction": "Проверь орфографию и грамматику",
  "previous_error": "expected_hash mismatch for blog/my-post.md: expected abc123, got def456",
  "changes": [
    {
      "path": "blog/my-post.md",
      "event": "update",
      "path_id": 123,
      "version": 5,
      "title": "My Post",
      "content": "# My Post\n\nContent here..."
    }
  ],
  "api_token": "eyJhbGc..."
}
```

### Определение event type

| Ситуация | Event |
|----------|-------|
| `note_paths.version_count == 1` (новый путь) | `create` |
| `note_paths.version_count > 1` | `update` |
| Вызов из `hidenotes` | `remove` |

---

## Матчинг glob-паттернов

### Используем doublestar.Match

Проект уже использует `github.com/bmatcuk/doublestar/v4` в `internal/templateviews/query.go`.

**Возможности:**
- `*` (один уровень): `blog/*` матчит `blog/post.md`, **не** матчит `blog/drafts/post.md`
- `**` (рекурсивно): `blog/**` матчит `blog/post.md` и `blog/drafts/post.md`
- Совместим с ожиданиями пользователей (как в .gitignore)

**Логика матчинга:**
1. Заметка подходит, если она матчится хотя бы с одним `include_pattern`
2. И НЕ матчится ни с одним `exclude_pattern`

Пример:
```
include_patterns: ["docs/**", "blog/**"]
exclude_patterns: ["docs/internal/**", "*.draft.md"]

docs/guide.md        — матчится (docs/** и нет exclude)
blog/post.md         — матчится (blog/** и нет exclude)
docs/internal/dev.md — не матчится (exclude по docs/internal/**)
blog/new.draft.md    — не матчится (exclude по *.draft.md)
```

### Без кеширования (MVP)

Вебхуков будет мало (единицы-десятки). Читаем из БД каждый раз (`ListEnabledWebhooks`). SQLite быстрый, оптимизация не нужна для MVP.

---

## Структура кода

### Новые пакеты

```
internal/case/admin/
+-- createwebhook/
|   +-- resolve.go      — создание вебхука (admin mutation)
|   +-- resolve_test.go
+-- updatewebhook/
|   +-- resolve.go      — обновление url/patterns/enabled/pass_api_key/include_content
|   +-- resolve_test.go
+-- deletewebhook/
|   +-- resolve.go      — soft delete (disabled_at)
|   +-- resolve_test.go
+-- listwebhookdeliveries/
    +-- resolve.go      — история доставок для конкретного вебхука

internal/case/backjob/deliverwebhook/
+-- job.go              — JobID, QueueID, Priority, Enqueue
+-- resolve.go          — HTTP POST, shortapitoken, HMAC подпись, сохранение результата
+-- resolve_test.go

internal/case/handlenotewebhooks/
+-- resolve.go          — depth check, event type filtering, glob-матчинг, создание delivery записей, enqueue jobs
+-- resolve_test.go

internal/shortapitoken/
+-- token.go            — JWT sign/parse, содержит depth + read/write patterns
+-- token_test.go

internal/webhookutil/
+-- hmac.go             — HMAC-SHA256 sign/verify
+-- httpclient.go       — общий HTTP клиент (таймауты, body limit 1MB)
+-- agentresponse.go    — parse + validate agent response (ozzo)
+-- applychanges.go     — применение изменений через InsertNote с проверкой write access
+-- payload.go          — общие поля payload (version, id, timestamp)

cmd/server/case_methods.go
+-- func (a *app) HandleNoteWebhooks(ctx, changedPathIDs, event, depth)
```

### GraphQL схема

```graphql
# Admin mutations
type Mutation {
  createWebhook(input: CreateWebhookInput!): CreateWebhookPayload!
  updateWebhook(input: UpdateWebhookInput!): UpdateWebhookPayload!
  deleteWebhook(id: Int!): DeleteWebhookPayload!
  triggerChangeWebhook(input: TriggerWebhookInput!): TriggerWebhookPayload!
  regenerateWebhookSecret(id: Int!): RegenerateSecretPayload!
}

type RegenerateSecretPayload {
  secret: String!    # new secret, shown once
}

input TriggerWebhookInput {
  webhookId: Int!
  pathIds: [Int!]!            # ID путей для триггера
}

type TriggerWebhookPayload {
  matchedCount: Int!           # сколько путей прошли glob-матчинг
  ignoredCount: Int!           # сколько путей не прошли
  deliveryId: Int              # ID созданного delivery (null если matchedCount=0)
}

input CreateWebhookInput {
  url: String!
  includePatterns: [String!]!      # glob patterns: ["blog/**", "docs/*"]
  excludePatterns: [String!]       # exclude glob patterns (optional)
  instruction: String! = ""        # текстовая инструкция для агента
  secret: String                   # если не задан — автогенерируется
  maxDepth: Int! = 1               # 1 = только прямые правки
  passApiKey: Boolean! = false
  includeContent: Boolean! = true
  timeoutSeconds: Int! = 60
  maxRetries: Int! = 0
  description: String! = ""
  onCreate: Boolean! = true
  onUpdate: Boolean! = true
  onRemove: Boolean! = true
  readPatterns: [String!]! = ["*"]
  writePatterns: [String!]! = []
}

input UpdateWebhookInput {
  id: Int!
  url: String
  includePatterns: [String!]
  excludePatterns: [String!]
  instruction: String
  secret: String
  maxDepth: Int
  passApiKey: Boolean
  includeContent: Boolean
  timeoutSeconds: Int
  maxRetries: Int
  enabled: Boolean
  description: String
  onCreate: Boolean
  onUpdate: Boolean
  onRemove: Boolean
  readPatterns: [String!]
  writePatterns: [String!]
}

# Admin queries
type Query {
  webhooks: [Webhook!]!
  webhookDeliveries(webhookId: Int!, limit: Int = 50): [WebhookDelivery!]!
}

type Webhook {
  id: Int!
  url: String!
  includePatterns: [String!]!
  excludePatterns: [String!]!
  instruction: String!
  hasSecret: Boolean!          # не раскрывать сам secret
  maxDepth: Int!
  passApiKey: Boolean!
  includeContent: Boolean!
  timeoutSeconds: Int!
  maxRetries: Int!
  enabled: Boolean!
  description: String!
  createdAt: DateTime!
  lastDeliveryAt: DateTime     # удобно для UI
  lastDeliveryStatus: String   # success/failed
  onCreate: Boolean!
  onUpdate: Boolean!
  onRemove: Boolean!
  readPatterns: [String!]!
  writePatterns: [String!]!
}

type WebhookDelivery {
  id: Int!
  webhookId: Int!
  status: String!
  responseStatus: Int
  attempt: Int!
  durationMs: Int
  createdAt: DateTime!
  completedAt: DateTime
}
```

### SQL-запросы (sqlc)

```sql
-- queries.read.sql

-- name: ListWebhooks :many
select * from change_webhooks where disabled_at is null order by created_at;

-- name: ListEnabledWebhooks :many
select * from change_webhooks where enabled = true and disabled_at is null;

-- name: WebhookByID :one
select * from change_webhooks where id = ? and disabled_at is null;

-- name: ListWebhookDeliveries :many
select * from change_webhook_deliveries
where webhook_id = ?
order by created_at desc
limit ?;

-- queries.write.sql

-- name: InsertWebhook :one
insert into change_webhooks (url, include_patterns, exclude_patterns, instruction, secret, max_depth, pass_api_key, include_content, timeout_seconds, max_retries, description, on_create, on_update, on_remove, read_patterns, write_patterns, created_by)
values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: UpdateWebhook :one
update change_webhooks
set url = coalesce(?, url),
    include_patterns = coalesce(?, include_patterns),
    exclude_patterns = coalesce(?, exclude_patterns),
    instruction = coalesce(?, instruction),
    secret = coalesce(?, secret),
    max_depth = coalesce(?, max_depth),
    pass_api_key = coalesce(?, pass_api_key),
    include_content = coalesce(?, include_content),
    timeout_seconds = coalesce(?, timeout_seconds),
    max_retries = coalesce(?, max_retries),
    enabled = coalesce(?, enabled),
    description = coalesce(?, description),
    on_create = coalesce(?, on_create),
    on_update = coalesce(?, on_update),
    on_remove = coalesce(?, on_remove),
    read_patterns = coalesce(?, read_patterns),
    write_patterns = coalesce(?, write_patterns),
    updated_at = datetime('now')
where id = ? and disabled_at is null
returning *;

-- name: DisableWebhook :exec
update change_webhooks
set disabled_at = datetime('now'), disabled_by = ?, enabled = false
where id = ?;

-- name: RegenerateWebhookSecret :one
update change_webhooks
set secret = ?, updated_at = datetime('now')
where id = ? and disabled_at is null
returning *;

-- name: InsertWebhookDelivery :one
insert into change_webhook_deliveries (webhook_id, attempt)
values (?, ?)
returning *;

-- name: UpdateWebhookDeliveryResult :exec
update change_webhook_deliveries
set status = ?, response_status = ?, duration_ms = ?,
    completed_at = datetime('now')
where id = ?;
```

---

## План реализации

### Этап 1: Ядро (MVP)

1. Миграция: таблицы `change_webhooks` (включая on_create/on_update/on_remove, read_patterns, write_patterns) + `change_webhook_deliveries` + alter `api_keys`
2. SQL-запросы (sqlc) + `make sqlc`
3. `internal/shortapitoken/` — JWT sign/parse с depth + read/write patterns в claims
4. `internal/webhookutil/` — HMAC, HTTP client, agent response parsing, apply changes
5. Admin mutations: create/update/delete webhook (secret автогенерируется)
6. Admin mutation: `regenerateWebhookSecret` — перегенерация secret, возвращает новый один раз
7. `internal/case/handlenotewebhooks/` — depth check, event type filtering (on_create/on_update/on_remove), glob-матчинг через doublestar, enqueue
8. `cmd/server/case_methods.go` — метод `HandleNoteWebhooks(ctx, changedPathIDs, event, depth)`
9. `deliverwebhook` background job — HTTP POST + HMAC подпись + shortapitoken + результат + парсинг agent response
10. Расширить `checkapikey` — поддержка `Authorization: Bearer` для shortapitoken с read/write patterns enforcement
11. Интеграция в `HandleLatestNotesAfterSave` и `hidenotes.Resolve`
12. Admin query: `webhooks`, `webhookDeliveries`
13. Интеграция с job_statuses — записи в таблицу при delivery
14. Debug endpoints для e2e тестов (см. shared_webhooks.md)

### Этап 2: UI

15. Фронтенд: CRUD вебхуков в админке
16. Фронтенд: просмотр истории доставок
17. Кнопка "retry" для failed доставок

### Этап 3: Улучшения (опционально)

18. Метрика: success rate за последние 24ч/7д
19. Автоотключение вебхука после N последовательных failures
20. Debounce: галка в настройках вебхука, аккумулировать изменения за N секунд в один delivery
21. Alerting: уведомление в Telegram/email при N последовательных failures вебхука

---

## Run Now (ручной триггер)

Мутация `triggerChangeWebhook` позволяет вручную отправить webhook для заданных путей. Полезно для тестирования и отладки.

### Логика

1. Загрузить webhook по ID
2. Получить заметки по `pathIds` через `LatestNoteViews()`
3. Применить include/exclude glob-матчинг
4. Если есть совпадения -> создать delivery, enqueue job
5. Вернуть кол-во совпавших и проигнорированных путей

### UI

Фронтенд форма:
- Поле для ввода path IDs (можно выбрать из списка заметок)
- Форма запоминает последние введённые ID в `localStorage`
- Триггер per webhook — можно проверить что конкретный webhook игнорирует определённые пути

---

## Решённые вопросы

1. **Содержимое заметки в payload?** Флаг `include_content` (default true). Для remove — `content: null`.

2. **Несколько glob-паттернов?** `include_patterns` + `exclude_patterns` как JSON array.

3. **Рекурсия webhook -> agent push -> webhook?** Три механизма: `depth` в shortapitoken JWT, `max_depth` в change_webhooks таблице, `skip_webhooks` в api_keys.

4. **Secret обязательный?** Да, автогенерируется при создании. Payload всегда подписан HMAC-SHA256.

5. **Кеширование вебхуков?** Нет для MVP. Читаем из БД.

6. **Порядок changes?** Алфавитный по path.

7. **Debounce concurrent commits?** Пока нет. Два delivery за 100ms — допустимо. В будущем — опциональная галка.

8. **Дедупликация.** `X-Webhook-ID` + `attempt` — получатель решает сам.

9. **Agent response формат.** Опциональный JSON с массивом `changes[]`, каждый элемент содержит `path`, `content`, `expected_hash`. Применяется через InsertNote с optimistic concurrency check. Парсинг без JSON Schema validation. Ошибки не фейлят delivery.

10. **Фильтрация по типу события.** Boolean поля `on_create`/`on_update`/`on_remove` — все true по умолчанию. Webhook получает только matching события.

11. **Scope токена.** `read_patterns`/`write_patterns` хранятся в таблице webhook и передаются в shortapitoken JWT. Дефолт: читать всё, писать ничего.

12. **Retry.** Единый `max_retries`, без goqite MaxReceive (MaxReceive=1). Единый счётчик `attempt` для HTTP и agent response ошибок.

---

## Открытые вопросы / Future

1. **Execution Group** — группа последовательного выполнения. Колонка `execution_group text not null default ''`. Все вебхуки с одинаковым execution_group выполняются последовательно. Отложено на будущее.
2. **Debounce** — аккумулировать изменения за N секунд в один delivery. Галка в настройках вебхука.
3. **Автоотключение** — автоотключение вебхука после N последовательных failures.
4. **Alerting** — уведомление в Telegram/email при N последовательных failures.
