# gqlgen: фичи и возможности

## Что уже используем

- Schema-first подход (`schema.graphqls`)
- Autobind — маппинг Go типов на GraphQL
- Follow-schema resolver layout
- Кастомные директивы (`@skipTx`, `@goField`, `@goModel`, `@goExtraField`)
- Транспорты: SSE, Options, GET, POST, MultipartForm
- Query cache (LRU)
- Automatic persisted queries
- Introspection (отключена для неаутентифицированных)
- AroundOperations middleware (транзакции, логирование)
- Union types для ошибок (`CreateUserPayload | ErrorPayload`) — это best practice
- File uploads через MultipartForm

## Что можно добавить

### Field-level директивы — средний effort, гибкость

Помимо существующих директив можно добавить:
- `@auth(role: ADMIN)` — авторизация на уровне поля
- `@cacheControl(maxAge: 60)` — хинты для кэширования
- `@deprecated(reason: "use newField")` — встроенная в GraphQL

Определяешь в схеме, реализуешь в `DirectiveRoot`. Effort: средний.

Документация: [gqlgen.com/reference/directives](https://gqlgen.com/reference/directives/)

### AroundFields / AroundRootFields — middleware на уровне полей

Помимо `AroundOperations` (уже используем для транзакций) есть:
- `AroundFields` — вызывается для каждого поля (логирование, трейсинг)
- `AroundRootFields` — перед root resolver
- `AroundResponses` — вокруг каждого ответа (полезно для подписок)

### OpenTelemetry — средний effort, продакшн

Трейсинг GraphQL операций: какие поля медленные, сколько времени resolver.

Пакеты: `github.com/ravilushqa/otelgqlgen`, `github.com/zhevron/gqlgen-opentelemetry`.

Effort: средний — нужен бэкенд для трейсов (Jaeger, etc.).

### Modelgen Hooks — кастомизация генерации

Можно модифицировать сгенерированные модели: добавить теги, изменить поля.

```go
// BuildMutateHook — для всей модели
// FieldMutateHook — для отдельного поля
```

Полезно если хочется генерировать модели из GraphQL схемы вместо ручного маппинга. Сейчас не нужно — autobind справляется.

Документация: [gqlgen.com/recipes/modelgen-hook](https://gqlgen.com/recipes/modelgen-hook/)

## Что НЕ нужно

### Dataloaders

Классическое решение N+1 проблемы. **Не нужны** — используем SQLite. N+1 проблема про сетевой round-trip к БД серверу. С SQLite запрос — вызов функции в том же процессе, overhead минимален.

Если когда-нибудь переедем на PostgreSQL — dataloaders станут критичны. Библиотека: `vikstrous/dataloadgen`.

### Complexity / Depth Limits

Не нужны — используем whitelist для query (Automatic Persisted Queries). Только разрешённые query могут выполняться. Complexity и depth limiting защищают от произвольных query, а их у нас нет.

### Field Collection (look-ahead)

Resolver узнаёт какие поля запросил клиент через `graphql.CollectAllFields(ctx)`. Позволяет оптимизировать SQL: джойнить только нужные таблицы. На практике ведёт к динамическим query, которые в Go неудобно строить без рефлексии. Статические sqlc query проще, надёжнее и покрыты типами.

### Apollo Federation

Разбивает API на микросервисы. Нужно для больших команд. У нас монолит — не нужно.

### Ent ORM Integration

Тесная интеграция с ent ORM. Полная замена ORM — слишком высокий effort, а autobind + sqlc работают хорошо.

## Ссылки

- [gqlgen.com](https://gqlgen.com/) — официальная документация
- [gqlgen.com/reference/field-collection](https://gqlgen.com/reference/field-collection/) — field look-ahead
- [gqlgen.com/reference/complexity](https://gqlgen.com/reference/complexity/) — complexity limits
- [gqlgen.com/reference/directives](https://gqlgen.com/reference/directives/) — директивы
- [gqlgen.com/reference/dataloaders](https://gqlgen.com/reference/dataloaders/) — dataloaders (для PostgreSQL)
- [gqlgen.com/recipes/subscriptions](https://gqlgen.com/recipes/subscriptions/) — подписки
- [gqlgen.com/recipes/modelgen-hook](https://gqlgen.com/recipes/modelgen-hook/) — хуки генерации
- [ravilushqa/otelgqlgen](https://github.com/ravilushqa/otelgqlgen) — OpenTelemetry
