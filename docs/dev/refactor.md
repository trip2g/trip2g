# Refactoring Roadmap

## Архитектурные рефакторинги

### 1. Assets: Проксирование через сервер [HIGH]

**Текущее состояние:**
- Ассеты отдаются напрямую из Object Storage (MinIO/S3)
- Нет проверки прав доступа на уровне ассетов
- Приватные изображения доступны по прямой ссылке

**Проблемы:**
- Пользователь без подписки может получить приватный контент по URL
- Нет кеширования на уровне сервера
- Зависимость от публичного доступа к storage

**План:**
1. Эндпоинт `/_assets/{hash}` проксирует запросы к storage
2. Проверка прав: ассет принадлежит заметке → проверить доступ к subgraph
3. Кеширование: Cache-Control headers, опционально локальный кеш
4. Подписанные URL с TTL для CDN (опционально)

**Файлы:**
- `internal/case/getasset/endpoint.go` — новый эндпоинт
- `internal/router/` — регистрация роута
- Обновить генерацию URL ассетов

---

### 2. Crypto Payments: Перенос в админку [HIGH]

**Текущее состояние:**
- NowPayments credentials в ENV: `NOWPAYMENTS_API_KEY`, `NOWPAYMENTS_IPN_SECRET`
- Нет возможности отключить/изменить без передеплоя
- Не соответствует паттерну Self-Hosted First

**План:**
См. [docs/admin_config_modules.md](admin_config_modules.md#nowpayments-crypto--todo)

**Файлы для создания:**
```
db/migrations/YYYYMMDD_create_nowpayments_credentials.sql
internal/case/admin/createnowpaymentscredentials/
internal/case/admin/deletenowpaymentscredentials/
internal/case/admin/setactivenowpaymentscredentials/
internal/case/admin/deactivatenowpayments/
assets/ui/admin/payments/nowpayments/
```

---

### 3. Git Sync: Синхронизация с хранилищем [MEDIUM]

**Текущее состояние:**
- Git репозиторий отдельно от основного хранилища заметок
- Изменения через Obsidian plugin не синхронизируются с git
- Нет версионирования на уровне git

**Проблемы:**
- Две разные системы хранения (БД + git)
- Нет единой истории изменений
- Сложно откатить изменения

**План:**
1. Git как источник правды для контента
2. Push в git → webhook → обновление БД
3. Или: БД как источник → автоматический коммит в git
4. Единая история версий

**Вопросы для решения:**
- Кто master: git или БД?
- Как обрабатывать конфликты?
- Нужен ли git вообще или достаточно версионирования в БД?

---

### 4. Config Versions: Разделение на таблицы [MEDIUM]

**Текущее состояние:**
- Одна таблица `config_versions` со всеми настройками
- Каждое изменение создает новую версию со всеми полями
- Много пустых/дублирующихся значений

**Проблемы:**
- Sparse table: 20 колонок, меняется 1 → 19 пустых дублей
- Сложно понять что именно изменилось
- Таблица растет с каждой новой настройкой (ALTER TABLE ADD COLUMN)

**Новый подход:**
```
Вместо:
config_versions (id, site_name, logo_url, theme, analytics_id, ...)

Делать:
config_site_name (id, value, created_at, created_by)
config_logo (id, value, created_at, created_by)
config_analytics (id, provider, tracking_id, created_at, created_by)
```

**Преимущества:**
- Каждая таблица — атомарная история одной настройки
- Нет пустых значений
- Легко добавлять новые группы настроек
- Понятно кто что менял

**Принцип:**
> Новая функциональность = новая таблица, не ALTER TABLE ADD COLUMN

**План миграции:**
1. Создать отдельные таблицы для групп настроек
2. Мигрировать данные из config_versions
3. Обновить GraphQL resolvers
4. Удалить старую таблицу

---

### 5. Router: Рефакторинг маршрутизации [LOW]

**Текущее состояние:**
- Работает стабильно
- Код генерируется автоматически
- Есть дублирование в обработке ошибок

**Проблемы (некритичные):**
- Большой handler в cmd/server/main.go
- Смешение middleware и роутинга
- Нет middleware chain

**План (когда дойдут руки):**
1. Извлечь middleware в отдельный пакет
2. Chain pattern для middleware
3. Упростить error handling

**Приоритет:** Низкий. Работает — не трогай.

---

## Технический долг (Code-Level)

  High Impact Refactoring Opportunities

  3. Extract Admin Authorization Helper (schema.resolvers.go:285-291)
  - Multiple admin resolvers duplicate identical auth checks
  - Create internal/auth/admin.go with RequireAdmin() helper

  4. Consolidate Active Offer Query Patterns (queries.sql:195-214, 727-778)
  - Repeated active offer conditions across multiple queries
  - Create SQL view or helper method

  5. Refactor Server Handler Function (cmd/server/main.go:886-995)
  - 100+ line request handler with nested conditionals
  - Extract middleware and route handling

  6. Extract Transaction Management (cmd/server/main.go:350-412)
  - AcquireTxEnvInRequest and ReleaseTxEnvInRequest should be in separate package
  - Complex transaction logic mixed with app logic

  7. Standardize Error Handling in Case Handlers
  - Inconsistent patterns between ozzo validation and manual validation
  - Create common error handling utilities

  8. Extract Asset Management (cmd/server/main.go:414-454)
  - Asset URL generation and filesystem setup should be separate package

  9. Consolidate User Ban Logic (cmd/server/main.go:660-695)
  - Complex caching logic should be extracted to internal/cache/userbans.go

  10. Simplify Database Hash Collision Logic (internal/db/queries.go:20-86)
  - InsertNote method has complex collision resolution
  - Extract to separate hash generation package

  Medium Impact Opportunities

  11. Create Nullable Type Helpers (internal/db/helpers.go:18-91)
  - Multiple ToNullable* functions with repetitive patterns
  - Use generics for single ToNullable[T] function

  12. Extract Purchase Notification System (cmd/server/main.go:486-535)
  - Complex subscription management mixed with app struct
  - Move to separate internal/notifications package

  13. Standardize resolveOne Pattern (schema.resolvers.go:37-42)
  - Repeated pattern throughout GraphQL resolvers
  - Extract to common resolver utilities

  14. Consolidate Active User Access Queries (queries.sql:98-105, 265-272)
  - Similar filtering patterns for user subgraph access
  - Create SQL view for active accesses

  15. Extract String Generation Utilities (cmd/server/main.go:743-782)
  - GenerateApiKey and GeneratePurchaseID use similar patterns
  - Create internal/generate package

  16. Simplify Router Implementation (internal/router/router.go:77-144)
  - Large handle method with repeated error marshaling
  - Extract middleware pattern

  17. Consolidate Time-Based Filters (queries.sql)
  - Repeated datetime filtering patterns
  - Create parameterized time filter helpers

  18. Extract GraphQL Resolver Boilerplate (schema.resolvers.go:516-646)
  - 30+ resolver factory methods
  - Generate or simplify resolver registration

  19. Separate Environment Variables (cmd/server/main.go:784-790)
  - Environment variable access scattered throughout
  - Create config package with typed getters

  20. Consolidate Note Asset Queries (queries.sql:288-303)
  - Two nearly identical note asset lookup queries
  - Merge or create one as alias of other
