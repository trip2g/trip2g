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

---

## 6. SQL: Lowercase Keywords

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

## 7. Commit Message Convention

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
