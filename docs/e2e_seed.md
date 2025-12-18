# E2E Test Database Seed

Процесс создания seed-файла базы данных для E2E тестов Telegram публикации.

## Предварительные требования

- Telegram API credentials (`TELEGRAM_API_ID`, `TELEGRAM_API_HASH`)
- Тестовый бот (создан через @BotFather)
- 4 тестовых канала в Telegram (см. [telegram_e2e.md](telegram_e2e.md))

## Процесс создания seed

### 1. Создать чистую базу и запустить сервер

```bash
rm -f e2e_test.db
go run ./cmd/server -db-file=e2e_test.db -dev
```

### 2. Добавить Telegram аккаунт

В админке: Telegram Accounts → Add Account → пройти авторизацию

### 3. Добавить бота

В админке: Telegram Bots → Add Bot → ввести токен

### 4. Перезапустить сервер

Чтобы бот подключился и увидел каналы. Дождаться пока появятся в списке чатов.

### 5. Создать publish tags

```bash
sqlite3 e2e_test.db "insert into telegram_publish_tags (label) values ('test_channel'), ('test_premium_channel')"
```

### 6. Связать теги с каналами

В админке:

**Для бота:**
- Telegram Bots → выбрать бота → Chats → выбрать тестовый канал
- Publish Tags → добавить `test_channel`

**Для аккаунта:**
- Telegram Accounts → выбрать аккаунт → Dialogs → выбрать тестовый канал
- Publish Tags → добавить `test_channel` и `test_premium_channel`

### 7. Остановить сервер и сделать дамп

```bash
sqlite3 e2e_test.db .dump > testdata/e2e_seed.sql
```

### 8. Заменить credentials на placeholders

В файле `testdata/e2e_seed.sql` заменить:

```sql
-- telegram_accounts: заменить phone, session_data, display_name на placeholders
-- Найти строку вида:
INSERT INTO telegram_accounts VALUES(1,'+79001234567',X'...',  'Name', ...);
-- Заменить на:
INSERT INTO telegram_accounts VALUES(1,'PHONE_PLACEHOLDER',X'00','NAME_PLACEHOLDER',0,1,'2025-01-01 00:00:00',1);

-- tg_bots: заменить token, name на placeholders
-- Найти строку вида:
INSERT INTO tg_bots VALUES(1,'123456:ABC...',1,'real_bot_name',...);
-- Заменить на:
INSERT INTO tg_bots VALUES(1,'TOKEN_PLACEHOLDER',1,'BOT_PLACEHOLDER','',1,'2025-01-01 00:00:00',1);
```

### 9. Извлечь credentials для .tg_e2e_session

```bash
./tge2e extract e2e_test.db
```

Теперь `.tg_e2e_session` содержит реальные credentials, а `testdata/e2e_seed.sql` — placeholders.

## Использование seed в тестах

```bash
# 1. Создать базу из seed
sqlite3 test.db < testdata/e2e_seed.sql

# 2. Подставить реальные credentials из .tg_e2e_session
./tge2e patch-db test.db

# 3. Запустить сервер
go run ./cmd/server -db-file=test.db -dev
```

## Структура данных в seed

| Таблица | Данные |
|---------|--------|
| `admins` | user_id=1 (создаётся автоматически) |
| `telegram_accounts` | id=1, placeholder credentials |
| `tg_bots` | id=1, placeholder token |
| `telegram_publish_tags` | test_channel, test_premium_channel |
| `telegram_publish_chats` | связь бот-канала с test_channel |
| `telegram_publish_account_chats` | связь аккаунт-канала с тегами |
