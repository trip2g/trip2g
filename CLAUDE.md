# Project Instructions

## Цикл работы с задачами

```
/do_next → работа → /wrap_up → закрыть сессию → /do_next → ...
```

### Команды

| Команда | Что делает |
|---------|------------|
| `/do_next` | Продолжить задачу или создать новую |
| `/wrap_up` | Сохранить прогресс, коммит, выйти |

### Когда создавать задачу

Создавай задачу в `docs/current_tasks.md` если:
- Работа займёт больше одной сессии
- Нужен план из 3+ шагов
- Важно сохранить контекст между сессиями

### Формат задачи
```markdown
## [IN PROGRESS] Название задачи

### Контекст
Краткое описание что делаем и зачем (2-3 предложения).

### План
- [x] Шаг 1
- [x] Шаг 2
- [ ] Шаг 3 ← текущий
- [ ] Шаг 4

### Заметки
Важные решения, подходы, что выяснили.
```

### Правила
- Максимум 2-3 активных задачи
- `/wrap_up` перед закрытием сессии — сохраняет прогресс
- Закончил задачу → удали или помети `[DONE]`

---

## Первый шаг: прочитай документацию

**Перед началом работы определи тип задачи и прочитай релевантные доки:**

### Обязательно
| Документ | Зачем |
|----------|-------|
| [docs/principles.md](docs/principles.md) | Базовые принципы проекта |

### По типу задачи

| Задача | Документы |
|--------|-----------|
| **Новая фича (backend)** | [docs/instructions.md](docs/instructions.md) — GraphQL mutations, SQL, cases |
| **Новая фича (frontend)** | [docs/mol.md](docs/mol.md), [docs/frontend.md](docs/frontend.md) |
| **Admin CRUD интерфейс** | [docs/frontend_crud.md](docs/frontend_crud.md), [docs/admin_config_modules.md](docs/admin_config_modules.md) |
| **Интеграция (OAuth, платежи)** | [docs/admin_config_modules.md](docs/admin_config_modules.md) |
| **Telegram** | [docs/telegram.md](docs/telegram.md) |
| **Тесты** | [docs/TESTING.md](docs/TESTING.md) |
| **Понять архитектуру** | [docs/architecture.md](docs/architecture.md) |
| **Рефакторинг** | [docs/refactor.md](docs/refactor.md) |

---

## Критичные правила (всегда помни)

### Go: формат ошибок
```go
// Правильно — две строки
err := doSomething()
if err != nil {

// Неправильно — одна строка
if err := doSomething(); err != nil {
```

### Go: после изменений
```bash
gofmt -w .
go test ./...
make lint
```

### SQL: lowercase keywords
```sql
select * from users where id = ?;  -- правильно
SELECT * FROM Users WHERE ID = ?;  -- неправильно
```

### Commits: без AI-подписей
```bash
# Правильно
git commit -m "feat(oauth): add Google OAuth"

# Неправильно — не добавляй эти строки:
# Co-Authored-By: Claude ...
# 🤖 Generated with Claude Code
```

### Admin mutations: всегда проверяй авторизацию
```go
token, err := env.CurrentAdminUserToken(ctx)
if err != nil {
    return nil, fmt.Errorf("failed to get current user token: %w", err)
}
```

---

## Quick Reference

### Команды
```bash
make sqlc          # SQL → Go
make gqlgen        # GraphQL → Go
npm run graphqlgen # GraphQL → TypeScript
make db-new name=X # Новая миграция
make db-up         # Применить миграции
```

### Структура
```
internal/case/           # Business logic
internal/case/admin/     # Admin mutations
internal/graph/          # GraphQL schema + resolvers
assets/ui/admin/         # Admin frontend
db/migrations/           # SQL migrations
```
