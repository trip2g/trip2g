# Admin-Managed Configuration Modules

## Философия: Self-Hosted First

Система проектируется с расчетом на **максимальную автономность клиентов**:

- Клиент может в любой момент развернуть систему на своем сервере
- Не нужен доступ к SSH, Docker, CI/CD или ENV-переменным
- Всё управление через веб-интерфейс админки
- Один бинарник + база данных = полностью рабочая система

**Цель:** Клиент купил/скачал систему → запустил → настроил через браузер → работает.

## Концепция

Многие модули системы работают с внешними сервисами и требуют credentials/конфигурации. Вместо хранения настроек в ENV-переменных или CLI-флагах, используем паттерн **Admin-Managed Config** — CRUD в админке с хранением в БД.

### Сравнение подходов

| Подход | Self-Hosted | Проблемы |
|--------|-------------|----------|
| ENV/CLI флаги | ❌ Плохо | Требует доступа к серверу, передеплой для изменений |
| Конфиг файлы | ❌ Плохо | Требует SSH, знания формата, риск синтаксических ошибок |
| БД + Админка | ✅ Идеально | Всё через браузер, валидация, мгновенные изменения |

### Когда использовать

- Модуль интегрируется с внешним сервисом (OAuth, платежки, боты)
- Нужна возможность менять credentials без передеплоя
- Возможно несколько конфигов (dev/prod, разные аккаунты)
- Секреты должны быть зашифрованы в бэкапах

## Архитектура

```
┌─────────────────────────────────────────────────────────┐
│                    Admin Panel                           │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐    │
│  │ Create  │  │  List   │  │  Show   │  │ Delete  │    │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘    │
└───────┼────────────┼────────────┼────────────┼──────────┘
        │            │            │            │
        ▼            ▼            ▼            ▼
┌─────────────────────────────────────────────────────────┐
│              GraphQL Admin Mutations                     │
│  create{Module}Credentials                               │
│  delete{Module}Credentials                               │
│  setActive{Module}Credentials  (если один активный)      │
│  deactivate{Module}  (отключить модуль)                 │
└───────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│                   Database                               │
│  {module}_credentials                                    │
│  ├─ id, name                                            │
│  ├─ client_id, client_secret_encrypted                  │
│  ├─ active (boolean)                                    │
│  ├─ created_at, created_by                              │
│  └─ ...module-specific fields                           │
└─────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│              Runtime Usage                               │
│  GetActive{Module}Credentials() → credentials or nil    │
│  - Nil = модуль отключен                                │
│  - Credentials = модуль работает                        │
└─────────────────────────────────────────────────────────┘
```

## Реализованные модули

### Google OAuth — `google_oauth_credentials`

```
assets/ui/admin/oauth/google/
├── catalog/     — список credentials
├── create/      — создание
├── show/        — просмотр + Set Active
├── delete/      — удаление
└── disableall/  — отключение OAuth
```

**Документация:** [docs/google_github_auth.md](google_github_auth.md)

### GitHub OAuth — `github_oauth_credentials`

Аналогичная структура. Общий паттерн.

## Модули ожидающие рефакторинга

### NowPayments (Crypto) — TODO

**Текущее состояние:**
- Credentials в ENV: `NOWPAYMENTS_API_KEY`, `NOWPAYMENTS_IPN_SECRET`
- Один набор credentials
- Нет возможности отключить без передеплоя

**План рефакторинга:**

1. **Миграция:**
   ```sql
   create table nowpayments_credentials (
       id integer primary key,
       name text not null,
       api_key_encrypted blob not null,
       ipn_secret_encrypted blob not null,
       active boolean not null default false,
       created_at datetime not null default (datetime('now')),
       created_by integer references users(id)
   );
   ```

2. **GraphQL:**
   - `createNowPaymentsCredentials`
   - `deleteNowPaymentsCredentials`
   - `setActiveNowPaymentsCredentials`
   - `deactivateNowPayments`

3. **Frontend:**
   ```
   assets/ui/admin/payments/nowpayments/
   ├── catalog/
   ├── create/
   ├── show/
   ├── delete/
   └── disableall/
   ```

4. **Удалить:**
   - ENV переменные из документации
   - CLI флаги из `internal/appconfig`

## Шаблон реализации

### 1. Database Migration

```sql
-- migrate:up
create table {module}_credentials (
    id integer primary key,
    name text not null,

    -- Module-specific fields
    api_key_encrypted blob not null,
    -- другие зашифрованные секреты

    -- Common fields
    active boolean not null default false,
    created_at datetime not null default (datetime('now')),
    created_by integer references users(id)
);

create unique index {module}_credentials_active_unique
    on {module}_credentials(active) where active = true;
```

### 2. SQL Queries

```sql
-- queries.read.sql

-- name: GetActive{Module}Credentials :one
select * from {module}_credentials where active = true;

-- name: Get{Module}CredentialsById :one
select * from {module}_credentials where id = ?;

-- name: ListAll{Module}Credentials :many
select * from {module}_credentials order by created_at desc;

-- queries.write.sql

-- name: Insert{Module}Credentials :one
insert into {module}_credentials (name, api_key_encrypted, created_by)
values (?, ?, ?) returning *;

-- name: SetActive{Module}Credentials :exec
update {module}_credentials set active = (id = ?);

-- name: Deactivate{Module}Credentials :exec
update {module}_credentials set active = false;

-- name: Delete{Module}Credentials :exec
delete from {module}_credentials where id = ?;
```

### 3. GraphQL Schema

```graphql
# Types
type Admin{Module}Credentials {
  id: Int!
  name: String!
  # НЕ возвращаем расшифрованные секреты!
  active: Boolean!
  createdAt: DateTime!
  createdBy: User
}

type Admin{Module}CredentialsConnection {
  nodes: [Admin{Module}Credentials!]!
}

# Admin Query
type AdminQuery {
  all{Module}Credentials: Admin{Module}CredentialsConnection!
  {module}Credentials(id: Int!): Admin{Module}Credentials
}

# Admin Mutations
input Create{Module}CredentialsInput {
  name: String!
  apiKey: String!
  # другие секреты
}

type Create{Module}CredentialsPayload {
  credentials: Admin{Module}Credentials!
}

union Create{Module}CredentialsOrErrorPayload =
  Create{Module}CredentialsPayload | ErrorPayload

# Аналогично для delete, setActive, deactivate
```

### 4. Business Logic Cases

```
internal/case/admin/
├── create{module}credentials/
│   └── resolve.go
├── delete{module}credentials/
│   └── resolve.go
├── setactive{module}credentials/
│   └── resolve.go
└── deactivate{module}/
    └── resolve.go
```

### 5. Frontend Structure

```
assets/ui/admin/{category}/{module}/
├── catalog/
│   ├── catalog.view.tree
│   └── catalog.view.ts
├── create/
│   ├── create.view.tree
│   └── create.view.ts
├── show/
│   ├── show.view.tree
│   └── show.view.ts
├── delete/
│   ├── delete.view.tree
│   └── delete.view.ts
└── disableall/
    ├── disableall.view.tree
    ├── disableall.view.ts
    └── disableall.view.tree.locale=ru.json
```

### 6. Runtime Usage

```go
// В endpoint или case
creds, err := env.GetActiveNowPaymentsCredentials(ctx)
if err != nil {
    return nil, err
}
if creds == nil {
    // Модуль отключен
    return nil, fmt.Errorf("NowPayments not configured")
}

// Расшифровать секреты
apiKey, err := env.DecryptData(creds.ApiKeyEncrypted)
if err != nil {
    return nil, err
}

// Использовать
client := nowpayments.NewClient(string(apiKey))
```

## Шифрование секретов

Используем `internal/dataencryption/` (AES-256-GCM):

```go
// Шифрование при сохранении
encrypted, err := env.EncryptData([]byte(input.ApiKey))

// Расшифровка при использовании
decrypted, err := env.DecryptData(creds.ApiKeyEncrypted)
```

**Ключ шифрования:** `--data-encryption-key` (32 байта)

## Принципы

1. **Минимум в appconfig** — только то, что нужно до старта БД
2. **Секреты зашифрованы** — безопасные бэкапы
3. **Один активный** — unique index на active=true
4. **Graceful degradation** — nil credentials = модуль отключен
5. **Аудит** — created_by, created_at для истории
6. **Множественные конфиги** — dev/staging/prod в одной БД

## Что остается в appconfig

Только **инфраструктурные настройки**, которые задаются один раз при установке:

| Флаг | Почему в CLI |
|------|--------------|
| `--data-encryption-key` | Нужен до доступа к БД для расшифровки |
| `--db-path` | Путь к файлу базы данных |
| `--http-addr` | Адрес/порт сервера |
| `--assets-path` | Путь к статике |

**Правило:** Если настройку можно менять после старта системы → в админку.

Все интеграции с внешними сервисами → в БД через админку.
