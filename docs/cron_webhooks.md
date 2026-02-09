# Cron Webhooks: дизайн-документ

## Цель

Cron webhooks вызывают внешние агенты по расписанию. Агент получает POST-запрос с инструкцией, может вернуть данные (новые/обновленные заметки) в синхронном режиме или работать асинхронно через shortapitoken.

Предполагается что на стороне получателя — MCP-инструмент или агент, который может генерировать заметки, дайджесты, отчёты.

Общая инфраструктура (shortapitoken, HMAC, agent response, retry, debug endpoints) описана в [docs/shared_webhooks.md](shared_webhooks.md).

## Сценарий использования

1. Админ создает cron webhook: URL + cron schedule + instruction
2. По расписанию система запускает cron job
3. Job создает delivery запись, enqueue'ит в goqite
4. Background worker отправляет POST с инструкцией и опциональным shortapitoken
5. Агент обрабатывает запрос:
   - **Синхронно**: возвращает changes в response body -> сервер импортирует через InsertNote
   - **Асинхронно**: возвращает 202 Accepted -> работает через API, пушит изменения через shortapitoken
6. Сервер сохраняет результат delivery (статус, время выполнения, ответ)

---

## Таблицы

### `cron_webhooks`

```sql
create table cron_webhooks (
  id integer primary key autoincrement,
  url text not null,
  cron_schedule text not null,                -- cron expression: "0 9 * * *"
  instruction text not null default '',       -- инструкция для агента
  secret text not null,                       -- HMAC secret, автогенерируется если не задан
  pass_api_key boolean not null default false, -- генерировать shortapitoken
  timeout_seconds integer not null default 60, -- таймаут HTTP ответа
  max_depth integer not null default 1,       -- защита от рекурсии для агентных пушей
  max_retries integer not null default 0,     -- retry при ошибках agent response
  next_run_at datetime,                           -- следующее время запуска (вычисляется из cron_schedule)
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
- `cron_schedule` — стандартный cron expression: `"0 9 * * *"` (каждый день в 9:00)
- `instruction` — текстовая инструкция, которую получит агент. Например: "Сгенерируй дайджест за последний день"
- `secret` — **всегда** задан. Автогенерируется при создании если не указан вручную. Подробности HMAC подписи — см. [shared_webhooks.md](shared_webhooks.md).
- `pass_api_key` — если true, в payload включается shortapitoken (JWT, TTL из appconfig, read+write доступ к API, depth+1). Подробности shortapitoken — см. [shared_webhooks.md](shared_webhooks.md).
- `timeout_seconds` — таймаут ожидания HTTP ответа (default 60s, можно увеличить для тяжелых задач)
- `max_depth` — контроль рекурсии. Если агент пушит заметки через API, применяется depth checking (как в change webhooks). Подробности — см. [shared_webhooks.md](shared_webhooks.md).
- `max_retries` — unified retry: если > 0 и agent response вызвал ошибки, delivery переотправляется с информацией об ошибке в payload. Default: 0. goqite MaxReceive не используется для retry — только `max_retries`. Подробности — см. [shared_webhooks.md](shared_webhooks.md).
- `next_run_at` — следующее время запуска. Вычисляется из `cron_schedule` при создании и после каждого запуска. Системный cron проверяет `where next_run_at <= datetime('now')`.
- `read_patterns` — glob patterns для чтения. Default `["*"]`. Передаются в shortapitoken JWT.
- `write_patterns` — glob patterns для записи. Default `[]` (безопасный default). Передаются в shortapitoken JWT.

### `cron_webhook_deliveries`

```sql
create table cron_webhook_deliveries (
  id integer primary key autoincrement,
  cron_webhook_id integer not null references cron_webhooks(id) on delete cascade,
  status text not null default 'pending',    -- pending, success, failed
  response_status integer,                   -- HTTP status code ответа
  attempt integer not null default 1,        -- номер попытки
  duration_ms integer,                       -- время ответа в мс
  created_at datetime not null default (datetime('now')),
  completed_at datetime
);
```

**Заметки:**
- Структура идентична `webhook_deliveries` из change webhooks
- Тяжёлые данные (request_body, response_body, error_message) хранятся в `webhook_delivery_logs` — см. [shared_webhooks.md](shared_webhooks.md).
- Индексы: `(cron_webhook_id, created_at)` для истории

---

## Payload

### HTTP-запрос TO агента

```
POST {cron_webhook.url}
Content-Type: application/json
User-Agent: trip2g-webhooks/1.0
X-Webhook-ID: {delivery.id}
X-Webhook-Timestamp: {unix_timestamp}
X-Webhook-Signature: sha256={hmac_hex}
X-Webhook-Attempt: {attempt_number}
```

`X-Webhook-Signature` присутствует всегда (secret автогенерируется). Подробности HMAC — см. [shared_webhooks.md](shared_webhooks.md).

### Body

```json
{
  "version": 1,
  "id": 42,
  "timestamp": 1738000000,
  "attempt": 1,
  "instruction": "Сгенерируй дайджест за последний день",
  "response_schema": {
    "type": "object",
    "properties": {
      "status": {"type": "string"},
      "message": {"type": "string"},
      "changes": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "path": {"type": "string"},
            "content": {"type": "string"},
            "expected_hash": {"type": "string"}
          }
        }
      }
    }
  },
  "api_token": "eyJhbGc..."
}
```

**Поля:**
- `version` — версия формата payload (текущая: 1)
- `id` — ID доставки для дедупликации
- `timestamp` — Unix время создания
- `attempt` — номер попытки (1, 2, 3)
- `instruction` — текстовая инструкция из `cron_webhooks.instruction`
- `response_schema` — серверная константа (не хранится в БД), описывает ожидаемый формат ответа агента. Сервер включает её в каждый payload, чтобы агент знал какой формат возвращать
- `api_token` — shortapitoken JWT (только если `pass_api_key=true`). Read+write доступ (TTL из appconfig, по умолчанию 60 мин), depth+1 в claims

### Retry payload (attempt > 1)

При retry payload включает `previous_error`:

```json
{
  "version": 1,
  "id": 42,
  "timestamp": 1738000000,
  "attempt": 2,
  "instruction": "Сгенерируй дайджест за последний день",
  "previous_error": "expected_hash mismatch for digests/2026-02-09.md",
  "response_schema": {
    "type": "object",
    "properties": {
      "status": {"type": "string"},
      "message": {"type": "string"},
      "changes": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "path": {"type": "string"},
            "content": {"type": "string"},
            "expected_hash": {"type": "string"}
          }
        }
      }
    }
  },
  "api_token": "eyJhbGc..."
}
```

### Ответ агента (синхронный режим)

Подробности формата agent response, обработки changes, валидации и retry — см. [shared_webhooks.md](shared_webhooks.md).

```json
{
  "status": "ok",
  "message": "Generated 1 digest",
  "changes": [
    {
      "path": "digests/2026-02-09.md",
      "content": "# Дайджест за 2026-02-09\n\n...",
      "expected_hash": ""
    }
  ]
}
```

**Поля:**
- `status` — статус выполнения (`ok`, `error`, `partial`)
- `message` — сообщение для лога (опционально)
- `changes` — массив изменений (опционально). Если присутствует — сервер импортирует через InsertNote:
  - `path` — путь заметки (обязательное поле)
  - `content` — содержимое заметки (обязательное поле)
  - `expected_hash` — для обновлений — проверить против `latest_content_hash`. Пустая строка для новых файлов

**Валидация:**
- Сервер не выполняет runtime JSON Schema валидацию
- Валидация через ozzo-validation на Go struct (обязательные поля: `path`, `content`)
- `response_schema` в payload — серверная константа, только документация для агента

### Асинхронный режим

Агент может вернуть `202 Accepted` и работать через API:

```json
{
  "status": "accepted",
  "message": "Processing started, will push results via API"
}
```

Агент использует `api_token` из payload для вызова API (pushNotes, commitNotes).

---

## Режимы работы: Sync vs Async

### Синхронный режим

1. Агент получает POST
2. Обрабатывает запрос (генерация, AI, вычисления)
3. Возвращает `changes` в response body
4. Сервер импортирует изменения через InsertNote
5. Delivery записывается как `success`

**Плюсы:** Простота, атомарность
**Минусы:** Ограничен таймаутом (default 60s)

### Асинхронный режим

1. Агент получает POST
2. Возвращает `202 Accepted` сразу
3. Запускает фоновую обработку
4. Пушит изменения через API (`api_token`)
5. Delivery записывается как `success` (агент принял задачу)

**Плюсы:** Нет ограничения по времени, можно делать долгие задачи
**Минусы:** Нет гарантии что агент завершит работу

### Таймауты

- `timeout_seconds` — конфигурируемый (default 60s)
- Для тяжелых задач (AI генерация, большие вычисления) — можно увеличить до 300s или работать асинхронно

---

## Cron execution

### Архитектура: system cron + next_run_at

```
System cron (cmd/server/cronjobs.go) — каждую минуту
    │
    ├── select * from cron_webhooks
    │   where enabled = true
    │     and disabled_at is null
    │     and next_run_at <= datetime('now')
    │
    └── Для каждого cron_webhook (в транзакции):
        ├── Обновить next_run_at (следующее время по cron_schedule)
        ├── Создать delivery запись (status=pending)
        └── Enqueue goqite job: deliver_cron_webhook(delivery_id)
```

### Как это работает

Существующий системный cron job (в `cmd/server/cronjobs.go`) запускается каждую минуту. Job `executecronwebhooks`:

1. Выполняет запрос `ListCronWebhooksDueForExecution` — все enabled вебхуки с `next_run_at <= datetime('now')`
2. Для каждого найденного вебхука (шаги 2a-2b в одной транзакции):
   a. Обновляет `next_run_at` на следующее запланированное время
   b. Создает delivery запись (status=pending)
3. После коммита транзакции — enqueue'ит goqite job `deliver_cron_webhook(delivery_id)`

Атомарное обновление `next_run_at` + создание delivery в одной транзакции предотвращает дублирование триггеров при краше процесса между enqueue и обновлением `next_run_at`.

### robfig/cron — только как парсер

`robfig/cron/v3` используется **только** как парсер (`cron.ParseStandard`) для вычисления следующего времени запуска из cron expression. Он **не** используется как scheduler — расписание хранится в БД в поле `next_run_at`.

```go
parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
schedule, err := parser.Parse(webhook.CronSchedule)
nextRun := schedule.Next(time.Now())
```

### Timezone

Timezone берётся из конфигурации проекта (`config.timezone`). Cron expressions интерпретируются в этой timezone.

### Поведение при перезапуске сервера

Пропущенные выполнения НЕ восстанавливаются — это стандартное поведение cron. При рестарте system cron просто проверяет `next_run_at <= now()` и запускает то, что просрочено. Если сервер был выключен несколько часов — каждый webhook выполнится один раз (не N раз за пропущенные интервалы).

### Регистрация cron job

В `cmd/server/cronjobs.go`:

```go
func getCronJobConfigs(app *app) []cronjobs.Job {
    jobs := []cronjobs.Job{
        // ... existing jobs
        &executecronwebhooks.Job{},
    }
    return jobs
}
```

### goqite worker

Background job `internal/case/backjob/delivercronwebhook/`:

1. Загрузить delivery + webhook из БД
2. Если `pass_api_key=true` -> создать shortapitoken JWT (TTL из appconfig, depth=current+1, read_patterns, write_patterns)
3. Подписать payload HMAC-SHA256
4. POST url с payload + headers (timeout = webhook.timeout_seconds)
5. Если response содержит `changes` -> импортировать через InsertNote
6. Сохранить результат (status, response, duration_ms)

---

## Защита от рекурсии (depth)

Depth tracking через shortapitoken JWT + max_depth. Подробности механизма — см. [shared_webhooks.md](shared_webhooks.md).

**Default `max_depth=1` для cron webhooks:**
- depth=0 (прямые правки) -> триггерит change webhooks с max_depth >= 1
- depth=1 (правки от cron webhook агента) -> НЕ триггерит change webhooks с max_depth=1

Чтобы change webhook срабатывал на правки от cron webhook агента — установить `max_depth=2`.

---

## Интеграция

### Точки интеграции

| Компонент | Файл | Что делает |
|-----------|------|------------|
| Cron job | `internal/case/cronjob/executecronwebhooks/` | Проверяет next_run_at, создает deliveries, enqueue |
| Background job | `internal/case/backjob/delivercronwebhook/` | HTTP POST + parse response + import changes |
| Admin mutations | `internal/case/admin/createcronwebhook/`, `updatecronwebhook/`, `deletecronwebhook/` | CRUD операции |
| Admin queries | `internal/graph/schema.resolvers.go` | `cronWebhooks`, `cronWebhookDeliveries` |
| Shared | `internal/shortapitoken/`, HMAC, agent response, retry | См. [shared_webhooks.md](shared_webhooks.md) |

---

## GraphQL схема

```graphql
input CreateCronWebhookInput {
  url: String!
  cronSchedule: String!
  instruction: String! = ""
  secret: String                  # автогенерируется если не задан
  passApiKey: Boolean! = false
  timeoutSeconds: Int! = 60
  maxDepth: Int! = 1
  maxRetries: Int! = 0
  enabled: Boolean! = true
  description: String! = ""
  readPatterns: [String!]! = ["*"]
  writePatterns: [String!]! = []
}

input UpdateCronWebhookInput {
  id: Int!
  url: String
  cronSchedule: String
  instruction: String
  secret: String
  passApiKey: Boolean
  timeoutSeconds: Int
  maxDepth: Int
  maxRetries: Int
  enabled: Boolean
  description: String
  readPatterns: [String!]
  writePatterns: [String!]
}

type CronWebhook {
  id: Int!
  url: String!
  cronSchedule: String!
  instruction: String!
  hasSecret: Boolean!            # не раскрывать сам secret
  passApiKey: Boolean!
  timeoutSeconds: Int!
  maxDepth: Int!
  maxRetries: Int!
  enabled: Boolean!
  description: String!
  readPatterns: [String!]!
  writePatterns: [String!]!
  nextRunAt: DateTime
  createdAt: DateTime!
  lastDeliveryAt: DateTime       # удобно для UI
  lastDeliveryStatus: String     # success/failed
}

type CronWebhookDelivery {
  id: Int!
  cronWebhookId: Int!
  status: String!
  responseStatus: Int
  attempt: Int!
  durationMs: Int
  createdAt: DateTime!
  completedAt: DateTime
}

type TriggerCronWebhookPayload {
  deliveryId: Int!
}

type Query {
  cronWebhooks: [CronWebhook!]!
  cronWebhookDeliveries(cronWebhookId: Int!, limit: Int = 50): [CronWebhookDelivery!]!
}

type Mutation {
  createCronWebhook(input: CreateCronWebhookInput!): CreateCronWebhookPayload!
  updateCronWebhook(input: UpdateCronWebhookInput!): UpdateCronWebhookPayload!
  deleteCronWebhook(id: Int!): DeleteCronWebhookPayload!
  regenerateCronWebhookSecret(id: Int!): RegenerateSecretPayload!
  triggerCronWebhook(webhookId: Int!): TriggerCronWebhookPayload!
}
```

---

## Структура кода

### Новые пакеты

```
internal/case/admin/
├── createcronwebhook/
│   ├── resolve.go      — создание cron webhook (admin mutation)
│   └── resolve_test.go
├── updatecronwebhook/
│   ├── resolve.go      — обновление url/schedule/instruction/etc
│   └── resolve_test.go
├── deletecronwebhook/
│   ├── resolve.go      — soft delete (disabled_at)
│   └── resolve_test.go
└── listcronwebhookdeliveries/
    └── resolve.go      — история доставок

internal/case/cronjob/executecronwebhooks/
├── job.go              — cron job definition (Schedule, Execute)
├── resolve.go          — select due webhooks by next_run_at, create deliveries, enqueue, update next_run_at
└── resolve_test.go

internal/case/backjob/delivercronwebhook/
├── job.go              — JobID, QueueID, Priority
├── resolve.go          — HTTP POST, shortapitoken, HMAC, parse response, import changes
└── resolve_test.go

internal/shortapitoken/
├── token.go            — JWT sign/parse с depth + read_patterns + write_patterns в claims
└── token_test.go

cmd/server/case_methods.go
└── func (a *app) ImportNotesFromChanges(ctx, changes, depth)
```

### SQL-запросы (sqlc)

```sql
-- queries.read.sql

-- name: ListCronWebhooks :many
select * from cron_webhooks where disabled_at is null order by created_at;

-- name: ListEnabledCronWebhooks :many
select * from cron_webhooks where enabled = true and disabled_at is null;

-- name: CronWebhookByID :one
select * from cron_webhooks where id = ? and disabled_at is null;

-- name: ListCronWebhookDeliveries :many
select * from cron_webhook_deliveries
where cron_webhook_id = ?
order by created_at desc
limit ?;

-- name: ListCronWebhooksDueForExecution :many
select * from cron_webhooks
where enabled = true
  and disabled_at is null
  and next_run_at <= datetime('now');

-- queries.write.sql

-- name: InsertCronWebhook :one
insert into cron_webhooks (url, cron_schedule, instruction, secret, pass_api_key, timeout_seconds, max_depth, max_retries, next_run_at, read_patterns, write_patterns, description, created_by)
values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: UpdateCronWebhook :one
update cron_webhooks
set url = coalesce(?, url),
    cron_schedule = coalesce(?, cron_schedule),
    instruction = coalesce(?, instruction),
    secret = coalesce(?, secret),
    pass_api_key = coalesce(?, pass_api_key),
    timeout_seconds = coalesce(?, timeout_seconds),
    max_depth = coalesce(?, max_depth),
    max_retries = coalesce(?, max_retries),
    read_patterns = coalesce(?, read_patterns),
    write_patterns = coalesce(?, write_patterns),
    enabled = coalesce(?, enabled),
    description = coalesce(?, description),
    updated_at = datetime('now')
where id = ? and disabled_at is null
returning *;

-- name: UpdateCronWebhookNextRunAt :exec
update cron_webhooks
set next_run_at = ?, updated_at = datetime('now')
where id = ?;

-- name: RegenerateCronWebhookSecret :one
update cron_webhooks
set secret = ?, updated_at = datetime('now')
where id = ? and disabled_at is null
returning *;

-- name: DisableCronWebhook :exec
update cron_webhooks
set disabled_at = datetime('now'), disabled_by = ?, enabled = false
where id = ?;

-- name: InsertCronWebhookDelivery :one
insert into cron_webhook_deliveries (cron_webhook_id, attempt)
values (?, ?)
returning *;

-- name: UpdateCronWebhookDeliveryResult :exec
update cron_webhook_deliveries
set status = ?, response_status = ?, duration_ms = ?,
    completed_at = datetime('now')
where id = ?;
```

---

## План реализации

### Этап 1: Инфраструктура

1. **Миграция**: таблицы `cron_webhooks` (с next_run_at, read_patterns, write_patterns) + `cron_webhook_deliveries`
2. **SQL-запросы** (sqlc): включая `ListCronWebhooksDueForExecution`, `UpdateCronWebhookNextRunAt`, `RegenerateCronWebhookSecret` + `make sqlc`
3. **`internal/shortapitoken/`** — JWT sign/parse с depth + read_patterns + write_patterns (если не существует)

### Этап 2: MVP (ядро)

4. **Admin mutations**: create/update/delete cron webhook + `regenerateCronWebhookSecret` + `triggerCronWebhook`
5. **Cron job**: `internal/case/cronjob/executecronwebhooks/` — select by `next_run_at <= now()`, create deliveries, update next_run_at (парсинг cron_schedule через `cron.ParseStandard`)
6. **Background job**: `internal/case/backjob/delivercronwebhook/` — HTTP POST + HMAC + shortapitoken (с read/write patterns) + parse response
7. **Import changes**: `ImportNotesFromChanges` метод — парсинг changes из ответа агента, вызов InsertNote
8. **Расширить auth**: `checkapikey` — поддержка `Authorization: Bearer` для shortapitoken (если еще не сделано)
9. **Регистрация cron job** в `cmd/server/cronjobs.go` — job `executecronwebhooks` каждую минуту
10. **Admin queries**: `cronWebhooks`, `cronWebhookDeliveries`
11. **job_statuses**: интеграция с таблицей job_statuses для отслеживания delivery
12. **Timezone**: cron_schedule парсится с timezone из `config.timezone`
13. **Debug endpoints** (`DEV_MODE=true`): общие с change_webhooks — см. [shared_webhooks.md](shared_webhooks.md)

### Этап 3: UI

14. Фронтенд: CRUD cron webhooks в админке
15. Фронтенд: просмотр истории deliveries
16. Кнопка "Run now" для ручного запуска (`triggerCronWebhook`)

### Этап 4: Улучшения (опционально)

17. Метрика: success rate за последние 24ч/7д
18. Автоотключение cron webhook после N последовательных failures
19. Include context notes в payload — опциональная фича
20. Alerting: уведомление в Telegram/email при N последовательных failures

---

## Решённые вопросы

1. **Response schema валидация?** Нет runtime JSON Schema валидации. `response_schema` — серверная константа (не хранится в БД, не задаётся админом), включается в payload чтобы агент знал формат ответа. Сервер валидирует agent response через ozzo-validation на Go struct (обязательные поля: path, content).

2. **Sync vs Async?** Оба режима поддержаны. Агент решает сам: вернуть changes в ответе или 202 Accepted и работать через API.

3. **Timeout?** Конфигурируемый `timeout_seconds` (default 60s). Можно увеличить для тяжелых задач или использовать async режим.

4. **Рекурсия?** depth tracking через shortapitoken JWT + max_depth в cron_webhooks. Default max_depth=1 (правки от агента НЕ триггерят change webhooks с max_depth=1).

5. **Secret обязательный?** Да, автогенерируется при создании. Payload всегда подписан HMAC-SHA256.

6. **Cron scheduling?** System cron каждую минуту + `next_run_at` в БД. `robfig/cron/v3` используется только как парсер (`cron.ParseStandard`) для вычисления следующего времени запуска. Не используется как scheduler.

7. **Token scope / write path restrictions?** Решено через `read_patterns` и `write_patterns`. Передаются в shortapitoken JWT. read_patterns default `["*"]`, write_patterns default `[]` (безопасный default).

8. **Timezone?** Берётся из конфигурации проекта (`config.timezone`).

9. **Retry?** Unified `max_retries` на уровне webhook. goqite MaxReceive не используется для retry. Подробности — см. [shared_webhooks.md](shared_webhooks.md).

---

## Открытые вопросы / Future

### Execution Group (future)

Группа последовательного выполнения. Колонка `execution_group text not null default ''`. Все вебхуки с одинаковым execution_group выполняются последовательно. Отложено на будущее.

### Include context notes

Как в change webhooks `include_patterns` — отправлять в payload контекстные заметки:

```json
{
  "instruction": "...",
  "context_notes": [
    {"path": "prompts/digest.md", "content": "..."}
  ]
}
```

**Решение**: future feature. Для MVP инструкция может содержать пути к нужным заметкам.
