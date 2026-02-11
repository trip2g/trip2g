# Shared Webhook Infrastructure: общая инфраструктура вебхуков

## Цель

Общая инфраструктура, разделяемая между change webhooks (`docs/change_webhooks.md`) и cron webhooks (`docs/cron_webhooks.md`).

---

## Short API Token (JWT)

Пакет: `internal/shortapitoken/`

### Структура
```go
package shortapitoken

type Data struct {
    Depth         int      `json:"d"`
    ReadPatterns  []string `json:"rp"` // glob patterns for read access
    WritePatterns []string `json:"wp"` // glob patterns for write access
}

func Sign(d Data, secret string) (string, error)
func Parse(token string, secret string) (Data, error)
```

### JWT Claims
- `d` (Depth) — счётчик глубины рекурсии. 0 = прямая правка, 1+ = агентная правка
- `rp` (ReadPatterns) — glob-паттерны для фильтрации чтения. `["*"]` = читать всё. `[]` = нет доступа на чтение
- `wp` (WritePatterns) — glob-паттерны для записи. `[]` = нет доступа на запись (безопасный дефолт). `["blog/**"]` = только blog
- TTL: `max(webhook.timeout_seconds, appconfig default TTL)`. Токен не должен истечь раньше таймаута ответа вебхука. По умолчанию 60 минут
- Подписывается тем же секретом, что и другие токены проекта

### Auth flow
Расширяем `checkapikey.Resolve` — проверяем два источника:
1. `X-API-Key` заголовок -> lookup в таблице api_keys, depth=0, без ограничений паттернов
2. `Authorization: Bearer {token}` заголовок -> parse shortapitoken JWT, depth из claims, паттерны из claims

Оба дают API-доступ. Различия:
- API key: постоянный, может иметь `skip_webhooks=true`, без ограничений паттернов
- shortapitoken: временный (TTL = max(webhook.timeout_seconds, appconfig default)), содержит depth + read/write паттерны

### Enforcement паттернов

| Контекст | Read | Write |
|----------|------|-------|
| Обычный API key | Без ограничений | Без ограничений |
| shortapitoken `rp:["blog/**"]` | Query возвращает только matching заметки (soft filter) | -- |
| shortapitoken `wp:[]` | -- | Любой push -> 403 |
| shortapitoken `wp:["blog/**"]` | -- | `blog/x.md` OK, `docs/y.md` -> 403 |

Read enforcement: фильтрация результатов запросов, без ошибок. Агент видит только разрешённые заметки.
Write enforcement: строгий 403 на pushNotes/commitNotes если путь не матчит ни один write pattern.

Реализация в checkapikey.Resolve:
```go
// After parsing shortapitoken
ctx = context.WithValue(ctx, readPatternsKey, token.ReadPatterns)
ctx = context.WithValue(ctx, writePatternsKey, token.WritePatterns)
```

Проверка в push/commit операциях:
```go
func checkWriteAccess(ctx context.Context, path string) error {
    patterns := ctx.Value(writePatternsKey)
    if patterns == nil {
        return nil // regular API key — no restrictions
    }
    for _, p := range patterns.([]string) {
        if doublestar.Match(p, path) {
            return nil
        }
    }
    return fmt.Errorf("write denied: path %q not in allowed patterns", path)
}
```

### Колонки таблиц вебхуков (и change_webhooks, и cron_webhooks)
```sql
read_patterns text not null default '["*"]',    -- JSON array, дефолт: читать всё
write_patterns text not null default '[]',      -- JSON array, дефолт: нет доступа на запись
```

Дефолт: читать всё, писать ничего. Админ явно открывает доступ на запись.

---

## HMAC-SHA256 подпись

Payload ВСЕГДА подписывается. Secret автогенерируется при создании вебхука, если не задан вручную.

```go
// Отправитель (trip2g)
mac := hmac.New(sha256.New, []byte(webhook.Secret))
mac.Write(requestBody)
signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
// Header: X-Webhook-Signature: sha256=abc123...

// Получатель (агент)
expectedSig := computeHMAC(secret, body)
if !hmac.Equal(expectedSig, receivedSig) {
    return 401
}
```

Secret показывается один раз при создании (как API keys). Формат по конвенции GitHub/Stripe.

Мутация `regenerateSecret` доступна для обоих типов вебхуков — генерирует новый secret, возвращает его один раз.

---

## Depth / Защита от рекурсии

### Проблема
Агент получает вебхук -> читает заметку -> пушит фикс -> триггерит тот же вебхук -> бесконечный цикл.

### Решение: счётчик depth

Depth передаётся через auth context:

| Источник запроса | Depth |
|------------------|-------|
| Обычный API key | 0 (прямая правка) |
| API key с `skip_webhooks=true` | вебхуки не триггерятся вообще |
| shortapitoken (JWT) | depth из JWT claims (1, 2, ...) |

При создании shortapitoken для delivery: depth = текущий depth + 1.

### Логика в HandleNoteWebhooks
```go
func HandleNoteWebhooks(ctx, changedPathIDs, event, depth) {
    if apiKey.SkipWebhooks {
        return
    }
    webhooks := ListEnabledWebhooks()
    for _, wh := range webhooks {
        if depth >= wh.MaxDepth {
            continue // depth слишком глубокий для этого вебхука
        }
        // ... matching, delivery, enqueue
    }
}
```

### Значения depth
- `max_depth=1` — триггерится только на прямых правках (depth=0)
- `max_depth=2` — триггерится на прямых + первый уровень агентных правок
- `max_depth=0` — вебхук отключён (никогда не триггерится)
- Дефолт: `max_depth=1`

### Изменение api_keys
```sql
alter table api_keys add column skip_webhooks boolean not null default false;
```
API keys с `skip_webhooks=true` не триггерят вебхуки при pushNotes/commitNotes/hideNotes.

---

## Формат ответа агента

После успешной доставки (HTTP 2xx) сервер парсит тело ответа. Если ответ содержит массив `changes[]`, изменения применяются через InsertNote.

### Формат ответа
```json
{
  "status": "ok",
  "message": "Linted 3 files",
  "changes": [
    {
      "path": "blog/post.md",
      "content": "# Fixed post\n\n...",
      "expected_hash": "abc123def456..."
    }
  ]
}
```

Поля:
- `status` (string, опционально) — статус обработки
- `message` (string, опционально) — человекочитаемое описание результата
- `changes[]` (array, опционально) — изменения для применения:
  - `path` (string, обязательно) — полный путь заметки
  - `content` (string, обязательно) — новое содержимое заметки
  - `expected_hash` (string, опционально) — SHA-256 хеш ожидаемого текущего содержимого (optimistic concurrency)

### Валидация (ozzo-validation)
```go
type AgentChange struct {
    Path         string `json:"path"`
    Content      string `json:"content"`
    ExpectedHash string `json:"expected_hash"`
}

func (c AgentChange) Validate() error {
    return validation.ValidateStruct(&c,
        validation.Field(&c.Path, validation.Required),
        validation.Field(&c.Content, validation.Required),
    )
}
```

JSON Schema не валидируется в рантайме. Валидация обязательных полей через ozzo-validation на Go struct.

### Логика обработки

1. **Parse:** После HTTP 2xx парсим `response_body` как JSON в Go struct.
2. **Validate:** Каждое изменение валидируется через ozzo (path и content обязательны).
3. **Проверка write access:** Каждый `change.path` проверяется по `WritePatterns` из shortapitoken. Если путь вне scope -> ошибка, откат транзакции.
4. **Проверка `changes`:** Если `changes` отсутствует или пустой массив -> ничего делать не нужно, нормальный ответ.
5. **Транзакция:** Все изменения применяются в одной транзакции:
   - **Optimistic concurrency:** Для каждого изменения с `expected_hash`:
     - Читаем текущий `note_paths.latest_content_hash`
     - Если хеши не совпадают -> откат всей транзакции
   - **Apply:** Вызываем `InsertNote(ctx, path, content)` с depth+1
   - Если любое изменение падает -> откат всей транзакции
6. **Ошибки:**
   - Ошибка парсинга JSON -> логируем warning, не фейлим delivery (агент может вернуть произвольный ответ)
   - Ошибка транзакции (expected_hash mismatch, InsertNote failed, write access denied) -> откат всех изменений
   - Если `max_retries > 0` -> retry с ошибкой в payload
   - Если `max_retries == 0` -> логируем warning, delivery помечается как success (HTTP 2xx получен)

---

## Стратегия retry

Единый retry через поле `max_retries`. Без goqite MaxReceive (ставим MaxReceive=1, без goqite retry).

### Логика
```go
func (j *Job) Resolve(ctx, deliveryID) {
    result := doHTTPPost(...)

    if result.err != nil || result.statusCode >= 500 {
        if delivery.Attempt < webhook.MaxRetries {
            enqueueRetry(deliveryID, attempt+1, result.err)
            return
        }
        markFailed(deliveryID, result.err)
        return
    }

    changes, err := parseAgentResponse(result.body)
    if err == nil && len(changes) > 0 {
        err = applyChanges(ctx, changes, depth)
    }
    if err != nil && delivery.Attempt < webhook.MaxRetries {
        enqueueRetry(deliveryID, attempt+1, err)
        return
    }

    markSuccess(deliveryID, result)
}
```

Единый `max_retries` + единый счётчик `attempt` — и для HTTP-ошибок, и для ошибок agent response.

### Retry payload
При retry payload включает поле `previous_error` с описанием ошибки. Агент может скорректировать ответ (обновить expected_hash, исправить content).

### HTTP клиент: fasthttp

Используем `valyala/fasthttp` клиент (проект уже использует fasthttp для сервера). Не использовать `net/http`.

| Параметр | Значение |
|----------|----------|
| Connect timeout | 5s |
| Response timeout | `webhook.timeout_seconds` (дефолт 60s) |
| Read body limit | 1MB |

---

## Версионирование payload

Все payload вебхуков включают поле `"version": 1`. Позволяет менять формат в будущем без поломки существующих агентов.

```json
{
  "version": 1,
  "id": 42,
  "timestamp": 1738000000,
  ...
}
```

---

## HTTP заголовки (общие для всех вебхуков)

```
POST {webhook.url}
Content-Type: application/json
User-Agent: trip2g-webhooks/1.0
X-Webhook-ID: {delivery.id}
X-Webhook-Timestamp: {unix_timestamp}
X-Webhook-Signature: sha256={hmac_hex}
X-Webhook-Attempt: {attempt_number}
```

`X-Webhook-Signature` всегда присутствует (secret автогенерируется).

---

## Таблица webhook_delivery_logs

Отдельная таблица для тяжёлых данных delivery (request/response body, ошибки). Основные таблицы deliveries хранят только метаданные. Логи чистятся агрессивнее.

### Таблица

```sql
create table webhook_delivery_logs (
  id integer primary key autoincrement,
  delivery_id integer not null,
  kind text not null,              -- "change" / "cron"
  request_body text,
  response_body text,
  error_message text,
  created_at datetime not null default (datetime('now'))
);

create index idx_wdl_delivery on webhook_delivery_logs(kind, delivery_id);
create index idx_wdl_created on webhook_delivery_logs(created_at);
```

- Одна таблица на оба типа deliveries, различаем по `kind`
- Без FK — чистим по времени, сиротские записи удаляются при cleanup
- Без лимита на размер полей — таблица регулярно чистится

### Cleanup

Cron задача: каждый час удалять записи старше 7 дней.

```sql
-- name: CleanupOldDeliveryLogs :exec
delete from webhook_delivery_logs
where created_at < datetime('now', '-7 days');
```

### Cleanup delivery таблиц

Cron задача: раз в день удалять deliveries старше 30 дней.

```sql
-- name: CleanupOldChangeWebhookDeliveries :exec
delete from change_webhook_deliveries
where created_at < datetime('now', '-30 days');

-- name: CleanupOldCronWebhookDeliveries :exec
delete from cron_webhook_deliveries
where created_at < datetime('now', '-30 days');
```

### Изменения в delivery таблицах

Из `change_webhook_deliveries` и `cron_webhook_deliveries` убираем:
- `request_body` → переносим в `webhook_delivery_logs`
- `response_body` → переносим в `webhook_delivery_logs`
- `error_message` → переносим в `webhook_delivery_logs`

Оставляем в delivery таблицах: `response_status`, `attempt`, `duration_ms`, `status`, timestamps.

### UI

Если логи для delivery удалены (старше 7 дней) — UI показывает статус и duration, но тело помечается как "logs expired".

---

## Debug Endpoints (DEV_MODE)

Для оркестрации E2E тестов доступны вспомогательные endpoints (только при `DEV_MODE=true`):

### `/debug/wait_all_jobs`

Блокирует до завершения всех goqite фоновых задач. Используется в E2E тестах для синхронизации.

### `/debug/run_cron_job`

Принудительно запускает cron job. Используется для тестирования cron webhooks без ожидания.

### `POST /debug/test_webhook`

Mock-эндпоинт для приёма вебхуков. Сохраняет все вызовы в память для последующей инспекции.

```
POST /debug/test_webhook?status=200&delay=0s&body={...}
```

| Параметр | Описание |
|----------|----------|
| `status` | HTTP status code ответа (default: 200) |
| `delay` | Задержка перед ответом (Go duration, e.g. `2s`) |
| `body` | Кастомный response body (если пусто — echo mode, возвращает полученный body) |

### `GET /debug/test_webhook_calls`

Возвращает все сохранённые вызовы test_webhook. Каждый вызов содержит timestamp, headers и body.

```
GET /debug/test_webhook_calls        # все вызовы
GET /debug/test_webhook_calls?last=1 # только последний
```

### `DELETE /debug/test_webhook_calls`

Очищает список сохранённых вызовов.

---

## Интеграция с Job Status

Все delivery вебхуков создают записи в таблице `job_statuses` (см. `docs/job_statuses.md`).

```go
// В delivery job:
env.CreateJobStatus(ctx, model.NewJobStatus(
    jobUUID,
    "change_webhook_delivery", // или "cron_webhook_delivery"
    fmt.Sprintf("Webhook -> %s", webhook.URL),
    map[string]any{"webhook_id": webhook.ID, "delivery_id": delivery.ID},
))
env.UpdateJobStatus(ctx, model.BuildRunningJobStatus(jobUUID))
// ... работа ...
env.UpdateJobStatus(ctx, model.BuildFinishedJobStatus(jobUUID))
```

---

## Структура общего кода

```
internal/shortapitoken/
├── token.go            — JWT sign/parse с depth + read/write patterns
└── token_test.go

internal/webhookutil/
├── hmac.go             — HMAC-SHA256 sign/verify
├── httpclient.go       — общий HTTP клиент на fasthttp (таймауты, body limit 1MB)
├── agentresponse.go    — parse + validate agent response (ozzo)
├── applychanges.go     — применение изменений через InsertNote с проверкой write access
├── payload.go          — общие поля payload (version, id, timestamp)
└── deliverylog.go      — insert/query webhook_delivery_logs
```

---

## Тестирование

### Подход: fasthttputil.InMemoryListener

Для юнит-тестов delivery jobs используем `fasthttputil.InMemoryListener` (часть `valyala/fasthttp`, без дополнительных зависимостей). In-memory listener создаёт pipe без сети и портов.

```go
func TestDeliverWebhook(t *testing.T) {
    ln := fasthttputil.NewInMemoryListener()
    defer ln.Close()

    // Мок-сервер
    go fasthttp.Serve(ln, func(ctx *fasthttp.RequestCtx) {
        // Проверяем HMAC, headers, payload
        assert.Equal(t, "POST", string(ctx.Method()))
        assert.Contains(t, string(ctx.Request.Header.Peek("X-Webhook-Signature")), "sha256=")

        ctx.SetStatusCode(200)
        ctx.SetBody([]byte(`{"status":"ok","changes":[]}`))
    })

    // Клиент с in-memory dial
    client := &fasthttp.Client{
        Dial: func(addr string) (net.Conn, error) {
            return ln.Dial()
        },
    }

    // Тестируем delivery логику с этим клиентом
}
```

Для инъекции клиента в delivery job — `*fasthttp.Client` передаётся как зависимость (через Env interface или параметр конструктора), не создаётся внутри job.

### Что тестировать (критичные пути)

Webhook delivery — критичная инфраструктура. Минимальный обязательный набор тестов:

**HMAC подпись (`internal/webhookutil/`):**
- Корректная подпись генерируется для payload
- Подпись верифицируется обратно
- Разные секреты дают разные подписи

**shortapitoken (`internal/shortapitoken/`):**
- Sign → Parse roundtrip (depth, read_patterns, write_patterns сохраняются)
- Expired token → ошибка
- Неверный секрет → ошибка
- TTL = max(timeout_seconds, appconfig TTL)

**Agent response parsing (`internal/webhookutil/agentresponse.go`):**
- Пустой ответ (нет changes) → ok, ничего не применять
- Валидные changes → парсятся корректно
- Невалидный JSON → warning, не фейлить delivery
- changes без обязательных полей (path/content) → ошибка ozzo-validation
- expected_hash mismatch → откат всей транзакции

**Delivery job (`internal/case/backjob/deliverwebhook/`):**
- HTTP 200 + пустой ответ → delivery success
- HTTP 200 + changes → changes применяются через InsertNote
- HTTP 200 + changes с ошибкой apply → retry если max_retries > 0
- HTTP 500 → retry если max_retries > 0, иначе failed
- HTTP 202 Accepted → delivery success (async режим)
- Timeout → retry или failed
- Retry payload содержит previous_error
- Headers: X-Webhook-ID, X-Webhook-Timestamp, X-Webhook-Signature, X-Webhook-Attempt

**Write access enforcement (`internal/webhookutil/applychanges.go`):**
- Путь в write_patterns → ok
- Путь вне write_patterns → 403, откат транзакции
- Пустые write_patterns `[]` → любая запись отклоняется

**Depth / рекурсия:**
- depth < max_depth → вебхук триггерится
- depth >= max_depth → вебхук пропускается
- depth инкрементируется в shortapitoken при delivery

**Cron execution (`internal/case/cronjob/executecronwebhooks/`):**
- Вебхук с next_run_at в прошлом → триггерится
- next_run_at обновляется атомарно с delivery (в одной транзакции)
- Disabled вебхук → не триггерится
- next_run_at в будущем → не триггерится

### E2E тесты: Playwright (`e2e/webhooks.spec.js`)

E2E тесты используют Playwright `request` API (без браузера) + debug endpoints (`/debug/test_webhook`). HMAC верификация через Node.js `crypto.createHmac`.

**Важно:** `pushNotes` только загружает данные. Вебхуки триггерятся после `commitNotes`. Каждый тест-сценарий вызывает оба: `pushNotes` → `commitNotes`.

Запуск: `npx playwright test e2e/webhooks.spec.js`
Интеграция в `test-e2e.sh`: после sync-тестов, перед Telegram-тестами.

**Сценарии:**

**1. Change webhook fires on commit**
- Создать change webhook: `url: /debug/test_webhook`, `includePatterns: ["blog/**"]`
- `DELETE /debug/test_webhook_calls` (очистить)
- `pushNotes` через GraphQL: `blog/test.md`
- `commitNotes` через GraphQL
- `GET /debug/wait_all_jobs`
- `GET /debug/test_webhook_calls?last=1`
- Проверить: ровно 1 вызов, `X-Webhook-Signature` присутствует, `changes[0].path == "blog/test.md"`, содержимое включено
- Верифицировать HMAC: `crypto.createHmac('sha256', secret).update(body)` совпадает с `X-Webhook-Signature`

**2. Exclude patterns фильтруют**
- Webhook: `include: ["*"]`, `exclude: ["_layouts/**"]`
- pushNotes + commitNotes: `_layouts/default.html` + `blog/post.md`
- `GET /debug/test_webhook_calls` → changes содержит только `blog/post.md`

**3. Agent response → changes applied**
- Webhook: `include: ["blog/**"]`
- Mock: `POST /debug/test_webhook?body={"changes":[{"path":"blog/auto.md","content":"# Auto"}]}`
- pushNotes + commitNotes: `blog/trigger.md`
- wait_all_jobs
- Проверить через GraphQL query: заметка `blog/auto.md` существует с содержимым `# Auto`

**4. Depth protection (no infinite loop)**
- Webhook: `include: ["blog/**"]`, `max_depth: 1`
- Mock: test_webhook возвращает changes с тем же путём (попытка цикла)
- pushNotes + commitNotes: `blog/post.md` (depth=0)
- wait_all_jobs
- `GET /debug/test_webhook_calls` → ровно 1 вызов (не бесконечный цикл)

**5. Cron webhook**
- Создать cron webhook: `url: /debug/test_webhook`, `pass_api_key: true`, `instruction: "Generate digest"`
- `DELETE /debug/test_webhook_calls`
- Мутация `triggerCronWebhook(webhookId)`
- wait_all_jobs
- `GET /debug/test_webhook_calls` → `payload.instruction == "Generate digest"`, `payload.api_token` присутствует

**6. Event type filtering**
- Webhook: `include: ["blog/**"]`, `on_create: true`, `on_update: false`
- pushNotes + commitNotes: новая `blog/new.md` (create)
- Проверить: вебхук вызван (`GET /debug/test_webhook_calls` → есть запрос)
- pushNotes + commitNotes: обновить `blog/new.md` (update)
- Проверить: вебхук НЕ вызван (нет нового запроса)

```bash
# test-e2e.sh — после sync тестов
echo "🔗 Running webhook E2E tests..."
npx playwright test e2e/webhooks.spec.js
```

---

## Решённые вопросы (общие)

1. **Secret обязателен?** Да, автогенерируется при создании. Payload всегда подписывается HMAC-SHA256.
2. **Read body limit?** 1MB.
3. **Стратегия retry?** Единый `max_retries`, без goqite retry (MaxReceive=1). Единый счётчик `attempt` для HTTP и agent response ошибок.
4. **Версионирование payload?** `"version": 1` в каждом payload.
5. **Scope токена?** ReadPatterns + WritePatterns в JWT. Дефолт: читать всё, писать ничего.
6. **Write enforcement?** Строгий 403. Read enforcement: soft filter.
7. **Валидация ответа агента?** ozzo-validation на Go struct. Без JSON Schema валидации в рантайме.
8. **Семантика `expected_hash`?** Если `expected_hash` не указан (отсутствует в JSON) — concurrency check не выполняется, перезапись без проверки. Если `expected_hash` задан (включая пустую строку `""`) — строгая проверка: hash должен совпасть с текущим `latest_content_hash`. Пустая строка `""` означает "файл должен быть новым" (не существовать). Если файл существует и hash не совпадает — ошибка, откат транзакции.

---

## Открытые вопросы

1. **Execution Group (future)** — последовательное выполнение вебхуков в группе. Пустая строка = дефолтная группа (все последовательно), разные непустые группы — параллельно. Группы изолированы по типу. Отложено: сложность реализации (потеря событий при заблокированной группе). Реализовать когда появится реальная потребность.

2. **`on delete cascade` vs soft delete** — FK deliveries → webhooks с `on delete cascade`, но delete делает soft delete (`disabled_at`). Каскад никогда не сработает при текущей логике. Задокументировать это решение или убрать cascade.

3. **Debug endpoint не проверяет HMAC** — для тестирования e2e flow добавить опциональный режим `verify_signature=true`.

4. **Secret UX** — secret показывается один раз, но нет confirm dialog перед закрытием. Если пользователь случайно закрыл — нужен `regenerateSecret`. Добавить "copy to clipboard" + confirm.
