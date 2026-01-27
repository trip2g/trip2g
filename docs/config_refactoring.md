# Рефакторинг конфига

Переход от монолитной таблицы `config_versions` к атомарным таблицам настроек.

## Проблема

Текущая архитектура — одна таблица со всеми настройками:

```sql
config_versions (
    id, created_at, created_by,
    show_draft_versions,  -- bool
    default_layout,       -- string
    timezone,             -- string
    robots_txt            -- string
)
```

**Недостатки:**
- Изменил одно поле → записалась копия всех полей
- История смешана — непонятно что именно менялось
- Добавление нового поля = ALTER TABLE
- Фронт отправляет все поля даже если изменилось одно

## Решение

Каждая настройка — отдельная таблица с историей:

```sql
config_site_title_templates (id, created_at, created_by, value text)
config_timezones           (id, created_at, created_by, value text)
config_default_layouts     (id, created_at, created_by, value text)
config_robots_txts         (id, created_at, created_by, value text)
config_show_draft_versions (id, created_at, created_by, value bool)
```

**Преимущества:**
- Чистая история каждой настройки
- Добавление настройки = CREATE TABLE (не ALTER)
- Фронт меняет только то что изменилось
- Легко откатить конкретную настройку

## GraphQL API

### Универсальные типы

```graphql
# Для string настроек
type ConfigStringEntry {
  id: ID!
  value: String!
  createdAt: DateTime!
  createdBy: AdminUser!
}

type ConfigString {
  current: String!
  history: [ConfigStringEntry!]!
}

# Для bool настроек
type ConfigBoolEntry {
  id: ID!
  value: Boolean!
  createdAt: DateTime!
  createdBy: AdminUser!
}

type ConfigBool {
  current: Boolean!
  history: [ConfigBoolEntry!]!
}

# Payload для мутаций
union SetConfigStringPayload = SetConfigStringSuccess | ErrorPayload
type SetConfigStringSuccess { entry: ConfigStringEntry! }

union SetConfigBoolPayload = SetConfigBoolSuccess | ErrorPayload
type SetConfigBoolSuccess { entry: ConfigBoolEntry! }
```

### Query

```graphql
extend type AdminQuery {
  # Новые атомарные настройки
  configSiteTitleTemplate: ConfigString!
  configTimezone: ConfigString!
  configDefaultLayout: ConfigString!
  configRobotsTxt: ConfigString!
  configShowDraftVersions: ConfigBool!

  # Legacy (deprecated, удалить в Фазе 6)
  latestConfig: ConfigVersion! @deprecated(reason: "Use atomic config fields")
}
```

### Mutations

```graphql
extend type AdminMutation {
  # Новые атомарные мутации
  setConfigSiteTitleTemplate(value: String!): SetConfigStringPayload!
  setConfigTimezone(value: String!): SetConfigStringPayload!
  setConfigDefaultLayout(value: String!): SetConfigStringPayload!
  setConfigRobotsTxt(value: String!): SetConfigStringPayload!
  setConfigShowDraftVersions(value: Boolean!): SetConfigBoolPayload!

  # Legacy (deprecated, удалить в Фазе 6)
  createConfigVersion(input: CreateConfigVersionInput!): ... @deprecated
}
```

## Frontend

Новая страница `/admin/config`:

```
┌─────────────────────────────────────────────────┐
│  Настройки сайта                                │
├─────────────────────────────────────────────────┤
│                                                 │
│  Site Title Template                            │
│  ┌─────────────────────────────────┐            │
│  │ %s | Мой сайт                   │  [История] │
│  └─────────────────────────────────┘            │
│  Формат заголовка страницы. %s = название.      │
│                                                 │
│  ─────────────────────────────────────────────  │
│                                                 │
│  Timezone                                       │
│  ┌─────────────────────────────────┐            │
│  │ Europe/Moscow              ▼    │  [История] │
│  └─────────────────────────────────┘            │
│                                                 │
│  ─────────────────────────────────────────────  │
│                                                 │
│  Default Layout                                 │
│  ┌─────────────────────────────────┐            │
│  │ default                         │  [История] │
│  └─────────────────────────────────┘            │
│                                                 │
│  ...                                            │
│                                                 │
└─────────────────────────────────────────────────┘
```

При клике на [История] — модалка со списком изменений.

## План миграции

### Фаза 1: site_title_template (новая настройка)

Добавляем новую настройку без миграции данных.

| Шаг | Описание |
|-----|----------|
| 1.1 | Миграция: `create table config_site_title_templates` |
| 1.2 | sqlc: queries для read/write |
| 1.3 | GraphQL: типы ConfigString, ConfigStringEntry |
| 1.4 | GraphQL: query `configSiteTitleTemplate` |
| 1.5 | GraphQL: mutation `setConfigSiteTitleTemplate` |
| 1.6 | Resolver: `internal/case/admin/setconfigsitetitletemplate/` |
| 1.7 | Env method: `SiteTitleTemplate() string` |
| 1.8 | rendernotepage: `formatTitle()` |
| 1.9 | Frontend: новая страница `/admin/config` |
| 1.10 | Тесты |

### Фаза 2: timezone

Миграция существующей настройки.

| Шаг | Описание |
|-----|----------|
| 2.1 | Миграция: `create table config_timezones` |
| 2.2 | Миграция данных: `insert ... select from config_versions` |
| 2.3 | sqlc queries |
| 2.4 | GraphQL: query + mutation |
| 2.5 | Resolver |
| 2.6 | Заменить `LatestConfig().Timezone` → `ConfigTimezone()` |
| 2.7 | Frontend: добавить на страницу |
| 2.8 | Тесты |

### Фаза 3: default_layout

Аналогично Фазе 2.

### Фаза 4: robots_txt

Аналогично Фазе 2.

### Фаза 5: show_draft_versions

Аналогично Фазе 2, но с типом bool.

### Фаза 6: Удаление legacy

| Шаг | Описание |
|-----|----------|
| 6.1 | Удалить `createConfigVersion` mutation |
| 6.2 | Удалить `latestConfig` query |
| 6.3 | Удалить `LatestConfig()` из Env |
| 6.4 | Удалить старую страницу настроек |
| 6.5 | Миграция: `drop table config_versions` |

## Точки изменений (Фаза 1)

| Файл | Изменение |
|------|-----------|
| `db/migrations/XXX_config_site_title_templates.sql` | Новая таблица |
| `db/queries/read.sql` | `GetConfigSiteTitleTemplateHistory`, `GetLatestConfigSiteTitleTemplate` |
| `db/queries/write.sql` | `InsertConfigSiteTitleTemplate` |
| `internal/graph/admin.graphqls` | Типы + query + mutation |
| `internal/graph/admin.resolvers.go` | Резолверы для query |
| `internal/case/admin/setconfigsitetitletemplate/resolve.go` | Бизнес-логика мутации |
| `cmd/server/main.go` | `SiteTitleTemplate() string` |
| `internal/case/rendernotepage/endpoint.go` | `formatTitle()` |
| `assets/ui/admin/config/` | Новая страница |

## Env методы

```go
// Текущие (legacy) — останутся до Фазы 6
func (a *app) LatestConfig() db.ConfigVersion

// Новые атомарные
func (a *app) SiteTitleTemplate() string      // Фаза 1
func (a *app) ConfigTimezone() string         // Фаза 2
func (a *app) ConfigDefaultLayout() string    // Фаза 3
func (a *app) ConfigRobotsTxt() string        // Фаза 4
func (a *app) ConfigShowDraftVersions() bool  // Фаза 5
```

## Дефолты

| Настройка | Дефолт | Описание |
|-----------|--------|----------|
| site_title_template | `%s` | Только название страницы |
| timezone | `UTC` | Часовой пояс |
| default_layout | `""` | Без кастомного layout |
| robots_txt | `open` | Разрешить индексацию |
| show_draft_versions | `true` | Показывать черновики админам |

## Валидация

| Настройка | Правила |
|-----------|---------|
| site_title_template | Должен содержать `%s` |
| timezone | Валидный timezone (`time.LoadLocation`) |
| default_layout | Существующий layout или пустая строка |
| robots_txt | `open`, `closed`, или произвольный текст |
| show_draft_versions | bool, без валидации |
