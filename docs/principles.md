# Архитектурные принципы

## 1. GraphQL как контракт

**GraphQL — это JSON View Layer между фронтендом и бэкендом.**

```
Frontend ←→ GraphQL Schema ←→ Backend
   ↓              ↓              ↓
  $mol         Контракт         Go
```

### Почему GraphQL, а не REST

- **Типизированный контракт** — схема описывает что можно запросить
- **JSON View из коробки** — не нужно писать отдельные DTO/presenters
- **Гибкость запросов** — фронт берет только нужные поля
- **Кодогенерация** — типы генерируются для обеих сторон

### gqlgen — code-first подход

gqlgen подстраивается под программиста, а не наоборот:

```go
// Ты пишешь свои типы
type User struct {
    ID    int64
    Email string
}

// gqlgen генерирует резолверы под них
func (r *queryResolver) User(ctx context.Context, id int) (*User, error)
```

**Преимущества:**
- Схема генерируется из Go-кода (или наоборот)
- Резолверы — обычные Go-функции
- Нет runtime reflection magic
- Compile-time проверка типов

### Принцип

> Любое изменение API начинается с изменения GraphQL схемы.
> Схема — это контракт. Сначала контракт, потом реализация.

---

## 2. Self-Hosted First

**Система должна быть полностью управляемой через веб-интерфейс.**

См. [docs/admin_config_modules.md](admin_config_modules.md)

### SQLite + минимум зависимостей

Архитектурный выбор в пользу self-hosting:

```
Традиционный стек:          Наш стек:
─────────────────          ──────────
App Server                  Single Binary
PostgreSQL                  └── SQLite (embedded)
Redis
Message Queue
```

**Почему SQLite:**
- Один бинарник = вся система (нет внешних зависимостей)
- Один процесс обрабатывает десятки тысяч RPS на хорошем сервере
- Простой бэкап (один файл)
- Нулевая конфигурация БД

**Stateless-like поведение с [Litestream](https://litestream.io/):**
- Continuous replication SQLite → S3/MinIO
- При падении/миграции: восстановление < 1 минуты
- Приложение ведёт себя как stateless контейнер
- Можно мигрировать между серверами без даунтайма

```
┌─────────────┐     continuous      ┌─────────────┐
│   SQLite    │ ──────────────────▶ │  S3/MinIO   │
│   (WAL)     │     replication     │  (backup)   │
└─────────────┘                     └─────────────┘
       │                                   │
       │ crash/migrate                     │
       ▼                                   ▼
┌─────────────┐      restore        ┌─────────────┐
│  New Server │ ◀────────────────── │   Litestream│
│             │      < 1 min        │             │
└─────────────┘                     └─────────────┘
```

### Будущее: Managed Hosting

Хостинг-панель для решения:
- **Nomad cluster** — оркестрация контейнеров
- Генерируемые конфиги под каждого клиента
- Автоматический биллинг по потреблению
- Клиент в любой момент может забрать бинарник и уйти на свой сервер

### Правила

- Минимум CLI-флагов (только инфраструктура: порт, путь к БД, ключ шифрования)
- Все интеграции настраиваются через админку
- Credentials хранятся зашифрованными в БД
- Клиент может развернуть систему без доступа к серверу

### Антипаттерны

```bash
# Плохо — требует передеплоя
--google-client-id=xxx --google-client-secret=yyy

# Плохо — требует SSH
vim /etc/app/config.yaml

# Хорошо — через браузер
Admin Panel → OAuth → Add Google Credentials
```

---

## 3. Новая таблица вместо ALTER TABLE

**Новая функциональность = новая таблица, не ALTER TABLE ADD COLUMN.**

### Проблема широких таблиц

```sql
-- Плохо: sparse table, много NULL
config_versions (
    id, site_name, logo_url, theme, analytics_id,
    seo_title, seo_description, favicon, ...
)
-- Изменили только site_name → запись со всеми полями
```

### Решение: атомарные таблицы

```sql
-- Хорошо: каждая настройка — отдельная история
config_site_name (id, value, created_at, created_by)
config_analytics (id, provider, tracking_id, created_at, created_by)
```

### Преимущества

- Нет пустых значений
- Понятная история изменений каждой настройки
- Легко добавлять новые группы
- Миграции проще (CREATE TABLE vs ALTER TABLE)

### Когда всё-таки ALTER TABLE

- Добавление NOT NULL колонки с DEFAULT — ок
- Колонка логически принадлежит существующей сущности
- Нет версионирования этих данных

---

## 4. Env Interface Pattern

**Каждый use case объявляет только те зависимости, которые использует.**

```go
// Плохо — зависит от всего
func CreateUser(ctx context.Context, db *sql.DB, config *Config, ...)

// Хорошо — явные зависимости
type Env interface {
    InsertUser(ctx context.Context, params db.InsertUserParams) (db.User, error)
    SendEmail(ctx context.Context, to, subject, body string) error
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error)
```

### Преимущества

- Тестируемость — легко мокать интерфейс
- Явные зависимости — видно что нужно use case'у
- Изоляция — нет доступа к лишнему
- Compile-time проверка — забыл метод = ошибка компиляции

---

## 5. Validation → ErrorPayload, System Error → error

**Разделяй ошибки пользователя и системные ошибки.**

```go
func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
    // Валидация → ErrorPayload (пользователь увидит сообщение)
    if errPayload := validateRequest(&input); errPayload != nil {
        return errPayload, nil
    }

    // Системная ошибка → error (пользователь увидит "Internal Error")
    user, err := env.GetUser(ctx, input.UserID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return &SuccessPayload{User: user}, nil
}
```

### Правило

| Тип ошибки | Возврат | Что видит пользователь |
|------------|---------|------------------------|
| Валидация (неверный email) | `ErrorPayload, nil` | "Invalid email format" |
| Бизнес-логика (нет доступа) | `ErrorPayload, nil` | "Access denied" |
| Системная (БД упала) | `nil, error` | "Internal server error" |

### Ошибки с привязкой к полям

Если ошибка зависит от конкретного поля — указывай поле через `byFields`:

```go
// Ошибка конкретного поля
return model.NewFieldError("email", "Email already registered"), nil

// Валидация через ozzo — автоматически собирает ошибки по полям
err := ozzo.ValidateStruct(input,
    ozzo.Field(&input.Email, ozzo.Required, is.Email),
    ozzo.Field(&input.Password, ozzo.Required, ozzo.Length(8, 100)),
)
if err != nil {
    return model.NewOzzoError(err), nil
}
```

Хелперы в `internal/graph/model/extra_methods.go`:
- `NewFieldError(field, message)` — ошибка одного поля
- `NewOzzoError(err)` — преобразует ozzo.Errors в ErrorPayload с byFields

GraphQL ответ:
```json
{
  "message": "",
  "byFields": [
    {"name": "email", "value": "must be a valid email address"},
    {"name": "password", "value": "the length must be between 8 and 100"}
  ]
}
```

---

## 6. Read/Write разделение (SQLite)

**Читаем параллельно, пишем в один поток.**

SQLite в WAL режиме позволяет параллельное чтение, но запись — только один writer. Конкурентная запись = `SQLITE_BUSY` ошибки.

### Архитектура

```
HTTP Request
     │
     ▼
┌─────────────┐
│  GraphQL    │
│  Resolver   │
└─────────────┘
     │
     ├── Query (read)  ──▶  ReadDB  ──▶  любой goroutine
     │
     └── Mutation (write) ──▶  WriteDB  ──▶  один поток (очередь)
```

### Два пула соединений

```go
// ReadDB — много соединений, параллельное чтение
readDB.SetMaxOpenConns(10)

// WriteDB — одно соединение, сериализованная запись
writeDB.SetMaxOpenConns(1)
```

### Правила

| Операция | Соединение | Конкурентность |
|----------|------------|----------------|
| SELECT | ReadDB | Параллельно |
| INSERT/UPDATE/DELETE | WriteDB | Последовательно |
| Транзакция с записью | WriteDB | Последовательно |

### Антипаттерны

```go
// Плохо — запись через ReadDB
func (e *Env) CreateUser(ctx context.Context, params Params) error {
    return e.readDB.Insert(ctx, params)  // SQLITE_BUSY при нагрузке
}

// Плохо — чтение через WriteDB
func (e *Env) GetUsers(ctx context.Context) ([]User, error) {
    return e.writeDB.Select(ctx)  // блокирует очередь записи
}

// Хорошо — разделение
func (e *Env) CreateUser(ctx context.Context, params Params) error {
    return e.writeDB.Insert(ctx, params)
}

func (e *Env) GetUsers(ctx context.Context) ([]User, error) {
    return e.readDB.Select(ctx)
}
```

### Когда это критично

- Высокая нагрузка на запись (синхронизация vault, импорт)
- Долгие транзакции (batch-операции)
- Background jobs (очереди, cron)

---

## 7. SQL: Lowercase Keywords

**SQL ключевые слова пишем в lowercase.**

```sql
-- Хорошо
select * from users where id = ?;
create table users (id integer primary key);

-- Плохо
SELECT * FROM Users WHERE ID = ?;
CREATE TABLE Users (ID INTEGER PRIMARY KEY);
```

### Почему

- Меньше визуального шума
- Консистентность с Go-кодом (lowercase)
- Современный стиль (PostgreSQL docs, SQLite docs)

---

## 8. Commit Message Convention

```
type(scope): description

feat(oauth): add Google OAuth admin management
fix(db): resolve SQLITE_BUSY errors
refactor(ui): move components to namespaces
docs: update API documentation
```

### Типы

- `feat` — новая функциональность
- `fix` — исправление бага
- `refactor` — изменение кода без изменения поведения
- `docs` — документация
- `style` — форматирование
- `test` — тесты
- `chore` — поддержка, обновление зависимостей

---

## 9. Логирование для воспроизведения

**Лог должен содержать достаточно данных, чтобы воспроизвести ошибку.**

### Принцип

Когда случится ошибка в production — у тебя будет только лог. Ни дебаггера, ни доступа к состоянию. Лог должен ответить на вопросы:
- Что произошло?
- С какими данными?
- В каком контексте?

### Правила

```go
// Плохо — непонятно что случилось
log.Error().Err(err).Msg("failed")

// Плохо — нет контекста
log.Error().Err(err).Msg("failed to process user")

// Хорошо — можно воспроизвести
log.Error().
    Err(err).
    Int64("user_id", userID).
    Str("action", "sync_vault").
    Str("vault_path", path).
    Msg("failed to sync user vault")
```

### Что логировать

| Уровень | Что включать |
|---------|--------------|
| Error | ID сущностей, входные параметры, состояние |
| Warn | ID, причина предупреждения |
| Info | Ключевые события (старт/финиш операций) |
| Debug | Детали для отладки (в production выключено) |

### Чувствительные данные

Не логируй:
- Пароли, токены, ключи API
- Email, телефоны (можно хэш или маску: `a***@example.com`)
- Платёжные данные

```go
// Плохо
log.Info().Str("token", token).Msg("auth success")

// Хорошо
log.Info().Str("token_prefix", token[:8]+"...").Msg("auth success")
```

### Антипаттерны

```go
// Плохо — лог без ошибки
if err != nil {
    log.Error().Msg("something went wrong")
    return err
}

// Плохо — дублирование (логируем и возвращаем)
if err != nil {
    log.Error().Err(err).Msg("failed")
    return fmt.Errorf("failed: %w", err)  // ошибка залогируется выше ещё раз
}

// Хорошо — логируем на верхнем уровне, внизу только wrap
if err != nil {
    return fmt.Errorf("sync vault %s: %w", path, err)
}
```

---

## 10. Атомарные коммиты

**Одна фича = один коммит. Если работа перетекает в другую задачу — сначала закоммить текущую.**

### Проблема

```bash
# Плохо — один коммит на несколько несвязанных изменений
git commit -m "feat(oauth): add Google OAuth + fix typo in readme + refactor utils"
```

Такой коммит:
- Сложно ревьюить (что относится к OAuth, а что нет?)
- Невозможно откатить одну часть без другой
- Ломает git bisect и blame

### Правило

Когда замечаешь, что начинаешь делать что-то не относящееся к текущей задаче:

1. **Остановись**
2. **Закоммить текущую работу** (даже если она не закончена — используй WIP)
3. **Переключись на новую задачу**

```bash
# Работал над OAuth, заметил баг в utils
git add -A && git commit -m "wip: oauth in progress"

# Исправил баг
git commit -m "fix(utils): handle empty array case"

# Вернулся к OAuth
git commit -m "feat(oauth): add Google OAuth admin management"
```

### Признаки что пора коммитить

- Переключаешься на другой файл/модуль не связанный с задачей
- Исправляешь "попутный" баг
- Рефакторишь код который "заодно увидел"
- Добавляешь "небольшое улучшение" не из плана

### Чеклист при коммите

При каждом коммите спроси себя:

- [ ] **Changelog?** — Это изменение видно пользователям? → Добавь в `docs/changelog.md`

Changelog нужен для: новых фич, исправленных багов, изменений UI/UX.
Changelog НЕ нужен для: рефакторинга, тестов, внутренних оптимизаций.

### Антипаттерны

```bash
# Плохо — накопил изменения за день
git add -A && git commit -m "various fixes and improvements"

# Плохо — смешал фичи
git commit -m "feat: add OAuth and fix validation and update docs"
```
