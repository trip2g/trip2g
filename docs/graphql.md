# GraphQL

## Зачем GraphQL

GraphQL — DSL для описания JSON-контрактов между сервером и клиентом.

Даже при REST API рано или поздно приходится делать JSON view, которые отличаются от моделей в БД — и это хорошая практика закладывать сразу. Разные страницы требуют разные наборы полей — появляются `/api/users?fields=...`, `/api/users/detailed`, вложенные includes. Многие компании рано или поздно приходят к своему DSL для описания этих view, и если его доработать — выйдет GraphQL. GraphQL решает это на уровне протокола: клиент запрашивает ровно то, что ему нужно.

GraphQL запрос выглядит как JSON без значений — описываешь форму ответа, получаешь данные ровно в этой форме:

```graphql
{                              →  {
  viewer {                     →    "viewer": {
    role                       →      "role": "ADMIN",
    user {                     →      "user": {
      email                    →        "email": "a@b.com"
    }                          →      }
  }                            →    }
}                              →  }
```

Один endpoint `/graphql` для всего. Схема — это граф типов, а `Query`, `Mutation`, `Subscription` — точки входа в этот граф. Клиент заходит через них и обходит ровно те узлы, которые ему нужны.

Главные бонусы:
- **Типизированные клиенты** — TypeScript типы генерируются из introspection сервера. Если фронт собрался — он может общаться с бэком. Контракт проверяется постоянно на этапе компиляции, ломающие изменения невозможно пропустить.
- **Один endpoint** — вместо десятков REST маршрутов.
- **Подписки** — SSE стриминг для real-time данных через тот же endpoint.
- **Introspection** — схема сама себя документирует, playground работает из коробки.

Технология просто выжила и хорошо работает.

## Ошибки через union, а не throw

Mutations возвращают `CreateUserPayload | ErrorPayload`, а не бросают ошибки в стандартный `errors` массив GraphQL.

**Стандартный механизм** (`errors` массив):
```json
{
  "data": { "createUser": null },
  "errors": [{ "message": "email already exists", "path": ["createUser"] }]
}
```

Проблемы:
- Ошибки нетипизированы — просто строки. Клиент парсит `message` регулярками.
- `data` становится `null` — клиент не знает что пошло не так без отдельного парсинга.
- Невозможно различить "email занят" и "сервер упал" без костылей в `extensions`.
- Кодген не может сгенерировать типы для ошибок — они вне схемы.

**Union подход** (`Payload | ErrorPayload`):
```json
{
  "data": {
    "createUser": {
      "__typename": "ErrorPayload",
      "message": "email already exists",
      "byFields": [{ "name": "email", "value": "already taken" }]
    }
  }
}
```

Почему лучше:
- **Ошибки — часть схемы.** Кодген генерирует типы, клиент получает discriminated union в TypeScript.
- **Типобезопасная обработка** — `if (result.__typename === 'ErrorPayload')` вместо парсинга строк.
- **Поле `byFields`** — ошибки привязаны к конкретным полям формы, клиент подсвечивает нужный input.
- **`data` никогда не null** — ответ всегда типизирован.
- **Стандартный `errors`** остаётся для настоящих ошибок: сервер упал, таймаут, невалидный GraphQL. То, что клиент не может обработать.

## Паттерны

### DB типы напрямую в схеме

```graphql
type AdminSubgraph @goModel(model: "trip2g/internal/db.Subgraph") {
  id: Int64!
  name: String!
}
```

Многие GraphQL типы привязаны к DB структурам через `@goModel` и `autobind` — без промежуточных DTO. С одной стороны это завязывает схему БД на выходные поля. С другой — экономит тонну времени на перекладывание структур. gqlgen отлично рефакторится когда разделение действительно понадобится: добавляешь отдельный тип, меняешь маппинг — resolvers обновляются при следующем `make gqlgen`.

### Вложенный admin namespace

```graphql
type Query {
  viewer: Viewer!
  admin: AdminQuery!   # все admin queries вложены
}
type Mutation {
  signOut: ...
  admin: AdminMutation! # все admin mutations вложены
}
```

Чистое разделение публичного API и админки. Разные правила авторизации, разный уровень introspection.

### Connection pattern без пагинации

```graphql
type AdminUsersConnection {
  nodes: [AdminUser!]! @goField(forceResolver: true)
}
```

46 Connection типов, все с одним полем `nodes`. Пока без `pageInfo`/`edges`/`cursor` — пагинация будет добавлена позже. `forceResolver: true` — resolver решает как именно грузить.

### @goExtraField для передачи контекста

```graphql
type AdminAuditLogsConnection
  @goExtraField(name: "Filter", type: "*...AdminAuditLogsFilterInput") {
  nodes: [AdminAuditLog!]! @goField(forceResolver: true)
}
```

`@goExtraField` добавляет поле в Go-структуру, невидимое в GraphQL. Родительский resolver записывает фильтр, дочерний `nodes` resolver читает его из `obj.Filter`. Передача контекста без загрязнения схемы.

Так же используется для инъекции API ключа в input:
```graphql
input PushNotesInput
  @goExtraField(name: "ApiKey", type: "trip2g/internal/db.ApiKey") {
  updates: [PushNoteInput!]!
}
```

### Case-layer: бизнес-логика в отдельных пакетах

```
internal/case/admin/createapikey/
  ├── resolve.go
  └── resolve_test.go
```

Каждая операция — отдельный пакет с `Resolve(ctx, env, input) (Payload, error)`. GraphQL resolver — тонкая обёртка:

```go
func (r *adminMutationResolver) CreateApiKey(ctx, input) {
    return createapikey.Resolve(ctx, r.env(ctx), input)
}
```

60+ пакетов. Тестируется без GraphQL, переиспользуется из CLI/REST.

### Env interface — единая точка зависимостей

```go
type Env interface {
    Logger() logger.Logger
    IsDevMode() bool
    // ... 200+ методов
    createapikey.Env  // встраивание case-интерфейсов
    signinbyemail.Env
    // ... 60+ embedded interfaces
}
```

Один интерфейс для всех resolvers, собранный из case-интерфейсов. `app` struct реализует его целиком.

### Автоматические транзакции

`AroundOperations` middleware оборачивает каждую mutation в транзакцию:
- Commit если `len(resp.Errors) == 0`
- Rollback если есть ошибки
- `@skipTx` — opt-out для операций вроде file upload

### resolveOne — generic helper для FK

```go
func resolveOne[T any](ctx, id, fetch) (*T, error) {
    row, err := fetch(ctx, id)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil // NotFound → nil
    }
    return &row, nil
}
```

DRY для всех `forceResolver: true` полей, которые грузят связанную сущность по FK.

## gqlgen — кодген

gqlgen — один из лучших кодгенов, которые я видел:

- **Подстраивается под существующий код** — не навязывает свою структуру. Указываешь `autobind` пакеты, и gqlgen маппит GraphQL типы на уже написанные Go структуры.
- **Resolvers не перетираются** — при повторной генерации добавляет только новые resolvers, не трогая реализованные.
- **Быстрая генерация** — `make gqlgen` отрабатывает за секунды даже на большой схеме.
- **Минимум boilerplate** — schema-first подход: пишешь `.graphqls`, получаешь типы и интерфейсы.

Workflow: правишь `schema.graphqls` → `make gqlgen` → реализуешь новые resolvers → готово.

## Архитектура: fasthttp + gqlgen

```
Клиент
  │
  ├── GET /page → fasthttp → pre-generated HTML (быстро, минимум ресурсов)
  │
  └── POST /graphql → fasthttp → fasthttpadaptor → gqlgen (queries, mutations)
      POST /graphql (Accept: text/event-stream) → gqlgen SSE (subscriptions)
```

fasthttp отдаёт заранее сгенерированный HTML — это основной трафик, и здесь fasthttp быстр: минимальное потребление памяти, переиспользование объектов.

gqlgen работает через `fasthttpadaptor` — обёртка, которая транслирует fasthttp запросы в `net/http`. Для GraphQL overhead адаптора незаметен: bottleneck в resolvers и SQL.

SSE подписки работают через тот же fasthttpadaptor — он реализует `http.Flusher`, что позволяет gqlgen стримить events. Подробности: [gqlgen_fasthttp.md](gqlgen_fasthttp.md).

Хороший фундамент: fasthttp для статики минимальными ресурсами + gqlgen для API со всеми фичами. При 30-50 процессах на одном сервере мелочи складываются.

## Подписки (SSE)

Подписки работают через Server-Sent Events. SSE вместо WebSocket:
- Проще протокол (обычный HTTP, тестируется curl'ом)
- Для GraphQL подписок нужен только server→client поток
- Firewall-friendly
- Подробное сравнение: [gqlgen_fasthttp.md](gqlgen_fasthttp.md)

### Планы использования

**Статусы задач** — прогресс длительных операций (импорт, публикация) в реальном времени без поллинга.

**Синхронизация файлов** — файл изменён (через API, git push, редактор) → подписчики получают событие → страница обновляется автоматически. Без кнопки "обновить".

**Live preview в редакторе** — preview обновляется по мере редактирования.

### Демо подписка

```graphql
type Subscription {
  currentTime(format: String = "2006-01-02 15:04:05"): String!
}
```

```bash
curl -N --request POST --url http://localhost:8081/graphql \
  --data '{"query":"subscription { currentTime }"}' \
  -H "accept: text/event-stream" -H "content-type: application/json"
```

## Будущий рефакторинг

### Курсорная пагинация в Connection типах

Сейчас все 46 Connection типов грузят данные целиком. Нужно добавить `pageInfo` и cursor-based пагинацию:

```graphql
type AdminUsersConnection {
  nodes: [AdminUser!]!
  pageInfo: PageInfo!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}
```

Курсорная пагинация лучше offset-based: стабильна при вставках/удалениях, эффективнее на больших таблицах (`WHERE id > cursor LIMIT N` вместо `OFFSET N`).

---

## Техническая справка

### Файлы

```
internal/graph/
├── schema.graphqls        # Схема (типы, queries, mutations)
├── schema.resolvers.go    # Реализация резолверов (follow-schema layout)
├── generated.go           # Сгенерированный код (~60K строк)
├── model/models_gen.go    # Сгенерированные модели
├── resolver.go            # Resolver struct
├── handler.go             # HTTP handler
├── helpers.go             # Хелперы
└── config_builders.go     # Config builders
```

## Команды

```bash
make gqlgen          # Перегенерировать Go-код из схемы
make graphqlgen      # gqlgen + TypeScript клиент
```

## Конфигурация (gqlgen.yml)

### Оптимизации генерации

| Флаг | Значение | Зачем |
|------|----------|-------|
| `skip_mod_tidy: true` | Пропускает `go mod tidy` | Быстрее генерация |
| `omit_complexity: true` | Не генерирует ComplexityRoot | −6K строк в generated.go |

### Layout

- **exec**: `single-file` → один `generated.go`
- **resolver**: `follow-schema` → один `{name}.resolvers.go` на файл схемы

Если добавить новый файл схемы (например `admin.graphqls`), gqlgen автоматически создаст `admin.resolvers.go`.

### Autobind

gqlgen автоматически привязывает Go-типы из пакетов:
- `trip2g/internal/model`
- `trip2g/internal/db`

Если тип в схеме совпадает с Go-типом по имени — отдельная модель не генерируется.

## Разделение схемы на файлы

Конфиг уже поддерживает glob: `internal/graph/*.graphqls`. Можно добавлять файлы по доменам.

**Когда это делать:** для удобства навигации, когда схема вырастет за 3-4K строк.

**Не ускорит компиляцию:** Go компилирует на уровне пакета, а не файла. Пакет `graph` останется тем же размером.

## Complexity

`omit_complexity: true` — complexity estimation отключена. Если понадобится для отдельных полей, использовать whitelist через кастомный код, а не генерацию.

## TypeScript кодген

`npm run graphqlgen` генерирует типизированные TypeScript функции из GraphQL операций, найденных в `assets/ui/**/*.ts`.

### Как это работает

1. `graphql-codegen` с конфигом `graphqlgen.js` делает introspection запущенного сервера на `http://localhost:8081/graphql`
2. Сканирует все `.ts` файлы в `assets/ui/` на вызовы `$trip2g_graphql_request(...)` и `$trip2g_graphql_subscription(...)`
3. Кастомный плагин `graphqlmol.js` генерирует типизированные overloads в `assets/ui/graphql/queries.ts`

### Использование

**Queries/Mutations** — оборачиваются в `$trip2g_graphql_request`:

```typescript
const request = $trip2g_graphql_request(/* GraphQL */ `
    query MyQuery($id: Int64!) {
        admin { user(id: $id) { email } }
    }
`)

// request — функция (variables?) => типизированный результат
const data = request({ id: 123 })
```

**Subscriptions** — оборачиваются в `$trip2g_graphql_subscription`:

```typescript
const host = $trip2g_graphql_subscription(/* GraphQL */ `
    subscription CurrentTime($format: String) {
        currentTime(format: $format)
    }
`, { format: '15:04:05' })

// host — $trip2g_sse_host, реактивный объект
// host.data() — последние данные из SSE стрима
// host.ready() — true когда соединение установлено
// host.error_message() — текст ошибки или ''
```

### Требования

Для кодгена сервер должен быть запущен на `:8081` (introspection схемы).

### @exportType директива

Кастомная директива для экспорта вложенных типов:

```typescript
$trip2g_graphql_request(/* GraphQL */ `
    query AdminBackgroundQueue($id: String!) {
        admin {
            backgroundQueue(id: $id) {
                jobs @exportType(name: "Job", single: true) {
                    id
                    name
                }
            }
        }
    }
`)
// Генерирует: export type $trip2g_graphql_AdminBackgroundQueueJob = ...
```

- `@exportType(name: "Name")` — экспортирует тип массива
- `@exportType(name: "Name", single: true)` — экспортирует тип элемента массива
