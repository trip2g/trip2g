# Job Statuses: дизайн-документ

## Цель

Единая таблица для отслеживания всех фоновых задач сервера. Frontend показывает пользователю что сейчас в работе: webhook deliveries, embedding generation, Telegram публикации — независимо от типа задачи. Прогресс (progress_value/expected_value), статус, и ссылка на релевантную страницу.

## Сценарий использования

1. Background job стартует → создаёт запись в `job_statuses` (status=pending)
2. Job начинает работу → обновляет status=running
3. Job обновляет прогресс → `progress_value`/`expected_value` (например, 3/7 заметок обработано)
4. Job завершается → status=finished или status=failed + error_message
5. Frontend подписан на SSE → получает обновления в реальном времени
6. UI показывает список активных задач с progress bar и ссылками

---

## Таблица

### `job_statuses`

```sql
create table job_statuses (
  id text primary key,                          -- UUID (из goqite job или сгенерированный)
  kind text not null,                            -- тип задачи: "change_webhook_delivery", "embedding", etc
  title text not null default '',                -- human-readable: "Webhook → https://agent.example.com"
  status text not null default 'pending',        -- pending, running, finished, failed
  progress_value integer not null default 0,     -- текущий прогресс
  expected_value integer not null default 0,     -- ожидаемый итог (0 = неизвестно, показывать spinner вместо %)
  error_message text,                            -- описание ошибки (только для failed)
  meta text not null default '{}',               -- JSON: произвольные данные для фронтенда (ссылки, IDs)
  created_at datetime not null default (datetime('now')),
  updated_at datetime not null default (datetime('now'))
);

create index idx_job_statuses_status on job_statuses(status);
create index idx_job_statuses_updated on job_statuses(updated_at);
```

**Заметки:**
- `id` — UUID. Для goqite jobs — используем существующий job UUID. Для не-goqite задач (системный cron check) — генерируем UUID.
- `kind` — строковый enum. Frontend по `kind` определяет иконку, текст и URL для ссылки "открыть".
- `title` — формируется job'ом при создании. Конвенция: краткое описание задачи.
- `status` — `pending` → `running` → `finished`/`failed`. Только forward transitions.
- `progress_value`/`expected_value` — для progress bar. Если `expected_value=0` — показывать spinner (неизвестная длительность). Процент: `progress_value / expected_value * 100`.
- `meta` — JSON с произвольными данными. Frontend использует `kind` + `meta` для построения ссылки.
- `updated_at` — обновляется при каждом `UpdateJobStatus`. Используется для фильтрации устаревших записей.

### Kind → Frontend routing

| Kind | Meta пример | Frontend URL |
|------|-------------|-------------|
| `change_webhook_delivery` | `{"webhook_id":42,"delivery_id":123}` | `/admin/webhooks/42` |
| `cron_webhook_delivery` | `{"webhook_id":5,"delivery_id":67}` | `/admin/cron-webhooks/5` |
| `embedding` | `{"path":"blog/post.md"}` | `/notes/blog/post.md` |
| `telegram_post` | `{"chat_id":"-100...","message_id":456}` | `/admin/telegram` |

Frontend хранит маппинг `kind → URL template`. При добавлении нового kind — добавить маппинг на фронте.

---

## Go модель и интерфейс

### model.JobStatus

```go
package model

type JobStatusState string

const (
    JobStatusPending  JobStatusState = "pending"
    JobStatusRunning  JobStatusState = "running"
    JobStatusFinished JobStatusState = "finished"
    JobStatusFailed   JobStatusState = "failed"
)

type JobStatus struct {
    ID            string         `json:"id"`
    Kind          string         `json:"kind"`
    Title         string         `json:"title"`
    Status        JobStatusState `json:"status"`
    ProgressValue int            `json:"progress_value"`
    ExpectedValue int            `json:"expected_value"`
    ErrorMessage  string         `json:"error_message,omitempty"`
    Meta          map[string]any `json:"meta"`
}
```

### Builders

Билдеры для частых операций — избегаем ручной сборки struct:

```go
package model

// NewJobStatus — создать новую запись (status=pending).
func NewJobStatus(id, kind, title string, meta map[string]any) JobStatus {
    return JobStatus{
        ID:     id,
        Kind:   kind,
        Title:  title,
        Status: JobStatusPending,
        Meta:   meta,
    }
}

// BuildRunningJobStatus — пометить job как running.
func BuildRunningJobStatus(id string) JobStatus {
    return JobStatus{ID: id, Status: JobStatusRunning}
}

// BuildProgressJobStatus — обновить прогресс.
func BuildProgressJobStatus(id string, progress, expected int) JobStatus {
    return JobStatus{ID: id, ProgressValue: progress, ExpectedValue: expected}
}

// BuildFinishedJobStatus — пометить job как finished.
func BuildFinishedJobStatus(id string) JobStatus {
    return JobStatus{ID: id, Status: JobStatusFinished}
}

// BuildFailedJobStatus — пометить job как failed с ошибкой.
func BuildFailedJobStatus(id string, err string) JobStatus {
    return JobStatus{ID: id, Status: JobStatusFailed, ErrorMessage: err}
}
```

### Env interface

```go
type Env interface {
    // CreateJobStatus — создать новую запись (status=pending).
    CreateJobStatus(ctx context.Context, js model.JobStatus) error

    // UpdateJobStatus — обновить существующую запись.
    // Обновляет только непустые/ненулевые поля (кроме ID).
    // Всегда обновляет updated_at.
    // Публикует SSE event.
    UpdateJobStatus(ctx context.Context, js model.JobStatus) error
}
```

Два метода вместо upsert — явнее и проще для sqlc.

### Использование в background job

```go
func (j *Job) Resolve(ctx context.Context, env Env, deliveryID string) error {
    // 1. Создать запись.
    env.CreateJobStatus(ctx, model.NewJobStatus(
        j.UUID,
        "change_webhook_delivery",
        fmt.Sprintf("Webhook → %s", webhook.URL),
        map[string]any{"webhook_id": webhook.ID, "delivery_id": delivery.ID},
    ))

    // 2. Начать работу.
    env.UpdateJobStatus(ctx, model.BuildRunningJobStatus(j.UUID))

    // 3. Прогресс (для батчей).
    for i, change := range changes {
        applyChange(change)
        env.UpdateJobStatus(ctx, model.BuildProgressJobStatus(j.UUID, i+1, len(changes)))
    }

    // 4. Завершение.
    env.UpdateJobStatus(ctx, model.BuildFinishedJobStatus(j.UUID))
    return nil
}
```

---

## SSE подписка

При каждом `UpdateJobStatus` — публиковать событие в SSE канал.

```
event: jobStatus
data: {"id":"uuid","kind":"change_webhook_delivery","status":"running","progress_value":3,"expected_value":7,"title":"Webhook → ..."}
```

Frontend подписывается на `jobStatus` events. При получении — обновляет список задач в UI.

### Когда публиковать

- `CreateJobStatus` → публиковать (новая задача появилась)
- `UpdateJobStatus` → публиковать (прогресс/статус изменился)
- Не публиковать при чтении/запросе списка

---

## API

### GraphQL

```graphql
type JobStatus {
  id: String!
  kind: String!
  title: String!
  status: String!              # pending, running, finished, failed
  progressValue: Int!
  expectedValue: Int!
  errorMessage: String
  meta: JSON!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type Query {
  # Активные и недавние задачи.
  # По умолчанию: pending + running + finished/failed за последний час.
  jobStatuses(status: [String!], limit: Int = 50): [JobStatus!]!

  # Счётчики для виджета в header/sidebar.
  jobStatusCounts: JobStatusCounts!
}

type JobStatusCounts {
  pending: Int!
  running: Int!
}

type Subscription {
  jobStatusUpdated: JobStatus!
}
```

### Фильтрация в query

```sql
-- name: ListActiveJobStatuses :many
select * from job_statuses
where (
  status in ('pending', 'running')
  or updated_at > datetime('now', '-1 hour')
)
order by created_at desc
limit ?;

-- name: CountActiveJobStatuses :one
select
  count(*) filter (where status = 'pending') as pending,
  count(*) filter (where status = 'running') as running
from job_statuses
where status in ('pending', 'running');
```

Старые finished/failed записи исчезают из UI через час, но остаются в БД. Все записи старше 1 месяца удаляются независимо от статуса.

### Cleanup

Cron задача: удалять все записи старше 1 месяца (независимо от статуса).

```sql
-- name: CleanupOldJobStatuses :exec
delete from job_statuses
where updated_at < datetime('now', '-30 days');
```

---

## Структура кода

```
internal/model/job_status.go          — JobStatus struct, constants, builders
internal/case/createjobstatus/
└── resolve.go                         — CreateJobStatus (insert + SSE publish)
internal/case/updatejobstatus/
└── resolve.go                         — UpdateJobStatus (update + SSE publish)
internal/case/listjobstatuses/
└── resolve.go                         — ListJobStatuses query

cmd/server/case_methods.go
├── func (a *app) CreateJobStatus(ctx, model.JobStatus) error
└── func (a *app) UpdateJobStatus(ctx, model.JobStatus) error
```

### SQL-запросы (sqlc)

```sql
-- queries.write.sql

-- name: InsertJobStatus :exec
insert into job_statuses (id, kind, title, status, progress_value, expected_value, error_message, meta)
values (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateJobStatusProgress :exec
update job_statuses
set progress_value = ?, expected_value = ?, updated_at = datetime('now')
where id = ?;

-- name: UpdateJobStatusState :exec
update job_statuses
set status = ?, error_message = coalesce(?, error_message), updated_at = datetime('now')
where id = ?;

-- name: UpdateJobStatusFull :exec
update job_statuses
set status = coalesce(?, status),
    progress_value = coalesce(?, progress_value),
    expected_value = coalesce(?, expected_value),
    error_message = coalesce(?, error_message),
    updated_at = datetime('now')
where id = ?;

-- queries.read.sql

-- name: ListActiveJobStatuses :many
select * from job_statuses
where (
  status in ('pending', 'running')
  or updated_at > datetime('now', '-1 hour')
)
order by created_at desc
limit ?;

-- name: JobStatusByID :one
select * from job_statuses where id = ?;

-- name: CleanupOldJobStatuses :exec
delete from job_statuses
where updated_at < datetime('now', '-30 days');
```

---

## Интеграция с существующими jobs

| Job | Kind | Title пример | Progress |
|-----|------|-------------|----------|
| `deliverwebhook` | `change_webhook_delivery` | `Webhook → https://agent.example.com` | 1/1 (один HTTP запрос) |
| `delivercronwebhook` | `cron_webhook_delivery` | `Cron → https://agent.example.com` | 1/1 |
| embedding generation | `embedding` | `Embeddings: blog/post.md` | 3/7 (заметок) |
| telegram publish | `telegram_post` | `Telegram: My Post Title` | 1/1 |

Для jobs без батчинга (один HTTP запрос) — `expected_value=1`, `progress_value` переключается 0→1.

---

## UI

### Виджет (header/sidebar)

Компактный badge с количеством задач в очереди/в работе. Всегда виден в админке.

```
┌──────────────────────────────┐
│  ⚙ Jobs  3 pending · 1 running  │
└──────────────────────────────┘
```

- Показывает `pending` + `running` счётчики из `jobStatusCounts` query
- Обновляется через SSE подписку `jobStatusUpdated` (пересчёт при каждом событии)
- Если `pending + running == 0` — показывать пустое состояние или скрывать badge
- При клике → переход на страницу `/admin/jobs`

### Страница `/admin/jobs`

Таблица всех текущих и недавних задач:

```
┌─────────────────────────────────────────────────────────────────────┐
│ Jobs                                                                │
├──────┬──────────────────────────┬──────────┬────────────┬──────────┤
│ Kind │ Title                    │ Status   │ Progress   │ Action   │
├──────┼──────────────────────────┼──────────┼────────────┼──────────┤
│ 🔗   │ Webhook → agent.com      │ running  │ ████░░ 3/7 │ Open →   │
│ 🔗   │ Webhook → lint.com       │ pending  │ —          │ Open →   │
│ 📨   │ Telegram: My Post        │ finished │ ██████ 1/1 │ Open →   │
│ 🧮   │ Embeddings: blog/post.md │ failed   │ ██░░░░ 2/5 │ Open →   │
└──────┴──────────────────────────┴──────────┴────────────┴──────────┘
```

**Колонки:**
- **Kind** — иконка по `kind`
- **Title** — `title` из job_statuses
- **Status** — badge: pending (серый), running (синий), finished (зелёный), failed (красный)
- **Progress** — progress bar если `expected_value > 0`, иначе spinner (running) или `—` (pending)
- **Action** — ссылка "Open →" ведёт на релевантную страницу по `kind` + `meta`

**Поведение:**
- По умолчанию показывает: pending + running + finished/failed за последний час
- SSE подписка: новые задачи появляются в реальном времени, прогресс обновляется live
- Сортировка: running первые, потом pending, потом finished/failed по `created_at desc`

---

## План реализации

### Этап 1: Ядро

1. Миграция: таблица `job_statuses` + индексы
2. SQL-запросы (sqlc): insert, update, list, count, cleanup + `make sqlc`
3. `internal/model/job_status.go` — struct, constants, builders
4. `internal/case/createjobstatus/resolve.go` — insert + SSE publish
5. `internal/case/updatejobstatus/resolve.go` — update + SSE publish
6. `internal/case/listjobstatuses/resolve.go` — query + count
7. `cmd/server/case_methods.go` — методы CreateJobStatus, UpdateJobStatus

### Этап 2: API

8. GraphQL query: `jobStatuses` — список задач
9. GraphQL query: `jobStatusCounts` — счётчики pending/running для виджета
10. SSE subscription: `jobStatusUpdated` — отдельная подписка
11. Cron задача: cleanup записей старше 1 месяца

### Этап 3: Интеграция

12. Добавить CreateJobStatus/UpdateJobStatus в `deliverwebhook`
13. Добавить в `delivercronwebhook`
14. Добавить в embedding generation (если применимо)
15. Добавить в telegram publish (если применимо)

### Этап 4: Frontend

16. Виджет в header/sidebar: badge с `pending` + `running` счётчиками, клик → `/admin/jobs`
17. Страница `/admin/jobs`: таблица задач с kind, title, status, progress bar, ссылка "Open"
18. SSE подписка: real-time обновление виджета и таблицы
19. Progress bar для задач с `expected_value > 0`, spinner для остальных running

### Этап 5: Улучшения (опционально)

20. Группировка по kind в UI
21. Фильтрация по kind/status в UI

---

## Оценка сложности

| Компонент | Сложность |
|-----------|-----------|
| Таблица + sqlc + model + builders | Простая |
| CreateJobStatus / UpdateJobStatus | Простая |
| GraphQL queries (jobStatuses, jobStatusCounts) | Простая |
| SSE subscription jobStatusUpdated | Средняя — подключение к существующей SSE инфраструктуре |
| Интеграция в existing jobs | Средняя — нужно добавить вызовы в каждый job |
| Frontend виджет (badge в header) | Простая — count query + SSE + ссылка |
| Frontend страница /admin/jobs | Средняя — таблица + progress bar + SSE live update + routing по kind |

Общая оценка: **средняя сложность**. Ядро простое, основная работа — интеграция в существующие jobs и frontend.

---

## Открытые вопросы

1. **Нет статуса `running` в deliveries** — deliveries имеют три статуса: `pending`, `success`, `failed`. Пока delivery обрабатывается worker'ом — она `pending`. Решение: не добавлять `running` в deliveries. Deliveries таблица хранит персистентный результат (pending/success/failed). `job_statuses` — live view для UI (pending/running/finished/failed). Дублирование минимально и оправдано разными целями: deliveries — историческая запись, job_statuses — текущее состояние для real-time отображения.

2. **Retry создаёт N записей job_statuses** — при retry создаётся новый goqite job с новым UUID → новая запись в job_statuses. Если `max_retries=5`, один delivery создаст до 6 записей. Рассмотреть группировку по `delivery_id` в meta для UI.
