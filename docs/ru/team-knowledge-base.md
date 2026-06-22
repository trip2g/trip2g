---
title: "Командная база знаний на голой VM"
free: true
lang_redirect: "[[en/team-knowledge-base]]"
---

Разверните trip2g на одной голой VM — без MinIO, без облачного хранилища — и у команды появится общая база знаний с поиском, доступная любому AI-агенту через единый MCP-эндпоинт. Заметки закрыты по умолчанию; федеративный поиск позволяет агентам на других инстансах запрашивать вашу базу без доступа ко всему хранилищу.

## 1. Развернуть на голой VM (локальное хранилище, без MinIO)

В trip2g есть встроенный бэкенд локального файлового хранилища — ассеты сохраняются на диск. S3-контейнер и MinIO не нужны: один Docker-контейнер и один том с данными.

**Сгенерируйте секреты до запуска:**

```bash
# DATA_ENCRYPTION_KEY — 32 байта (обязательно в проде — сервер паникует на значении по умолчанию)
openssl rand -hex 16   # возвращает 32-символьную hex-строку

# JWT_SECRET — любая длинная случайная строка
openssl rand -base64 32
```

**Запуск сервера:**

```bash
docker run -d --name trip2g-kb \
  --restart unless-stopped \
  -p 8081:8081 \
  -e LISTEN_ADDR=0.0.0.0:8081 \
  -e INTERNAL_LISTEN_ADDR=:8082 \
  -e DB_FILE=/data/kb.sqlite3 \
  -e STORAGE_BACKEND=local \
  -e STORAGE_LOCAL_DIR=/data/storage \
  -e OWNER_EMAIL=owner@yourteam.example \
  -e PUBLIC_URL=https://kb.yourteam.example \
  -e JWT_SECRET=<ваша-длинная-случайная-строка> \
  -e DATA_ENCRYPTION_KEY=<ваши-32-символа-hex> \
  -v /opt/trip2g-kb:/data \
  ghcr.io/trip2g/trip2g:latest
```

Проверьте запуск:

```bash
docker ps | grep trip2g-kb
until curl -sf http://localhost:8082/healthz >/dev/null; do sleep 1; done; echo "up"
```

**Обязательные переменные окружения:**

| Переменная | Назначение |
|-----------|-----------|
| `STORAGE_BACKEND=local` | Хранить ассеты на диске вместо S3/MinIO |
| `STORAGE_LOCAL_DIR=/data/storage` | Директория ассетов внутри смонтированного тома |
| `JWT_SECRET` | Подпись сессионных токенов. Обязательна — дефолтное значение в проде не работает |
| `DATA_ENCRYPTION_KEY` | 32-байтовый hex-ключ для шифрования. Должен отличаться от дефолтного — иначе сервер паникует с сообщением "in production, data encryption key must be changed from default" |
| `OWNER_EMAIL` | Email аккаунта администратора |
| `PUBLIC_URL` | Внешний URL — используется в ссылках и в auth-потоках |
| `DB_FILE` | Путь к SQLite-базе внутри контейнера |
| `LISTEN_ADDR` | Основной адрес для приёма HTTP |
| `INTERNAL_LISTEN_ADDR` | Внутренний адрес для healthcheck |

**Не указывайте:**
- `DEV=true` — dev-режим только для разработки, отключает продовую безопасность
- `RESEND_API_KEY` / `SMTP_PASSWORD` — нужны только для входа по email-коду
- `GIT_API_REPO_PATH` — нужен только при использовании встроенного git-зеркала

**Обратный прокси.** Поставьте Caddy или Nginx перед сервером для TLS:

```caddyfile
kb.yourteam.example {
    encode zstd gzip
    reverse_proxy localhost:8081
}
```

**Альтернатива — systemd.** Если хотите запускать бинарь напрямую (сборка через `make build`):

```ini
[Unit]
Description=trip2g командная база знаний
After=network.target

[Service]
EnvironmentFile=/etc/trip2g-kb.env
ExecStart=/usr/local/bin/trip2g
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Важно: образ должен включать бэкенд локального хранилища. Эта функция появилась в ветке `feat/filestorage` — используйте `ghcr.io/trip2g/trip2g:latest` или любой выпуск после неё.

## 2. Получить аккаунт администратора и API-ключ

Войдите как владелец, затем выпустите один API-ключ для CLI синка. Вся команда использует этот ключ для публикации контента.

**Способ A — HAT (Hot Auth Token), email не нужен.**

HAT — это беспарольный вход администратора. Сервер подписывает JWT с email владельца и `JWT_SECRET`. Именно этот способ используют memcli и другие headless-инструменты. Подробности — [[ru/user/agent-memory]] (раздел «Завести API-ключ»).

**Способ B — вход по email (продакшен).**

Укажите `RESEND_API_KEY`, `MAIL_FROM` и верифицированный домен отправителя в Resend. Сервер отправит одноразовый код на `OWNER_EMAIL`. После входа создайте ключ в Администрирование → API-ключи → Создать.

**Выпустить ключ через curl** (используя сессионный токен из любого из способов выше):

```bash
API_KEY=$(curl -s -X POST https://kb.yourteam.example/graphql \
  -H 'Content-Type: application/json' \
  -H "Cookie: trip2g_token=$TOKEN" \
  -d '{"query":"mutation($i:CreateApiKeyInput!){admin{createApiKey(input:$i){__typename ... on CreateApiKeyPayload{value} ... on ErrorPayload{message}}}}","variables":{"i":{"description":"team-sync"}}}' \
  | grep -o '"value":"[^"]*"' | cut -d'"' -f4)
echo "API-ключ: $API_KEY"
```

Полный процесс выпуска ключа — [[ru/user/local-quickstart]] (раздел «Завести API-ключ»).

## 3. Опубликовать контент через CLI синка

Публикуем папку хранилища на сервер. Запускайте из корня исходников trip2g:

```bash
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /путь/к/командному-хранилищу \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/graphql \
  --verbose
```

Для непрерывной синхронизации (заметки обновляются в реальном времени при редактировании в Obsidian):

```bash
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder /путь/к/командному-хранилищу \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/_system/graphql
```

Полный справочник по CLI синка — [[ru/user/local-quickstart]].

## 4. Подключить федеративный поиск

Федерация позволяет AI-агенту обратиться к единому MCP-эндпоинту вашей базы и выполнить поиск по всем подключённым базам знаний. Для этого нужна **KB-заметка** в хранилище.

### Важно: `free: true` обязательно

KB-заметка без `free: true` невидима для неаутентифицированных MCP-запросов. Без этого поля сканирование федерации (`accessibleKBNotes`) игнорирует заметку, а федеративные инструменты возвращают "Federation is not configured".

Создайте файл в хранилище (например, `hub/peer-name.md`):

```yaml
---
free: true
mcp_federation_kb_url: https://hub.example.com/_system/mcp
mcp_federation_kb_id: hub-name
---
Использовать для: поиска по общей командной базе знаний и публичным справочникам.
```

Нажмите Sync. Локальный `/_system/mcp` теперь предоставляет `federated_search`, `federated_similar`, `federated_note_html` и `federated_expand` — они запрашивают хаб-пира.

### Пример MCP-вызова

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "federated_search",
    "arguments": {
      "query": "чеклист деплоя",
      "kb_id": "hub-name"
    }
  }
}
```

Возвращает подходящие заметки из базы пира. Без `kb_id` вызов выполняется веерно по всем зарегистрированным KB-заметкам параллельно.

### SSRF и глубина

Публичные хабы (внешние URL) разрешены по умолчанию. Для приватных/внутрисетевых адресов требуется `MCP_FEDERATION_ALLOW_PRIVATE=true`. Веерный поиск останавливается на глубине 3 по умолчанию (`MCP_FEDERATION_MAX_DEPTH`). Таймаут на один пир — 2 секунды (`MCP_FEDERATION_FANOUT_TIMEOUT`).

Полная настройка федерации — включая приватных пиров, обмен HMAC-ключами и scope подграфов — в [[ru/user/federation]].

## 5. Управление доступом: кто что видит

`/_system/mcp` открыт публично только для заметок с `free: true`. Всё остальное требует аутентификации.

**Для приватного или закрытого контента** запросы проходят с Bearer-токеном:

```
Authorization: Bearer t2g_<токен>
```

или через параметр URL:

```
https://kb.yourteam.example/_system/mcp?token=t2g_...
```

Формат токена — `t2g_<...>`. Персональные токены создаются в разделе Пользователь → Токены в интерфейсе trip2g. Каждый участник команды, которому нужен доступ к закрытым заметкам, получает свой токен.

**Подключение Claude Code или другого MCP-клиента:**

```json
{
  "mcpServers": {
    "team-kb": {
      "command": "python3",
      "args": ["/путь/к/trip2g_mcp_stdio_adapter.py"],
      "env": {
        "TRIP2G_MCP_URL": "https://kb.yourteam.example/_system/mcp",
        "TRIP2G_TOKEN": "t2g_токен-участника"
      }
    }
  }
}
```

**Федерирование приватной базы в другой хаб.**

Другой trip2g-инстанс может подключить вашу базу через федерацию. Это требует обмена HMAC-ключами (federation secrets), который точно определяет, какие подграфы видит пир. Подробнее — [[ru/user/federation]] (раздел «Добавить приватного пира»). Технический рецепт — в `docs/dev/federation_agent_setup.md`.

**Итоговая таблица доступа:**

| Запрашивающий | Токен | Видит |
|--------------|-------|-------|
| Анонимный агент / публичный хаб | Нет | Только заметки с `free: true` |
| Аутентифицированный участник | `t2g_<токен>` | Заметки в scope подписки |
| Admin API-ключ | `X-API-Key: <ключ>` | Все заметки |
| Федерированный пир с HMAC-секретом | Подписанный JWT | Заметки в scope секрета |

## 6. Главная страница для гостей: предупреждение на весь экран

Анонимным посетителям можно показать предупреждение на весь экран, а аутентифицированные участники попадают сразу в базу. Используются управление доступом trip2g и поле `route` во frontmatter.

Создайте индексную заметку (например, `_home.md`) и привяжите её к корню домена:

```yaml
---
route: kb.yourteam.example/
free: true
---
```

В теле заметки — HTML-блок на весь экран, центрированный через flexbox:

```html
<div style="
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  font-size: 3rem;
  font-weight: bold;
  text-align: center;
  font-family: system-ui, sans-serif;
">
  Не лезь — убьёт.
</div>
```

Эта заметка имеет `free: true`, поэтому предупреждение видно всем, кто обращается к корневому URL без авторизации. После входа участник переходит к реальному контенту через боковую панель или прямые ссылки.

Примечание о механизме. Точное поведение заметки с `route: domain/` и `free: true` описано в `docs/dev/multidomain.md` и `docs/dev/default_template.md`. Если нужно автоматическое перенаправление аутентифицированного пользователя на другую страницу — такой редирект в дефолтном шаблоне не предусмотрен и потребует кастомного Jet-layout. Подход выше (статическая страница предупреждения, участники переходят вручную) — самый простой и работающий.

## Смотрите также

- [[ru/user/local-quickstart]] — полный справочник по локальному запуску, флаги CLI синка
- [[ru/user/agent-memory]] — память для одиночного агента; описывает выпуск ключа через HAT
- [[ru/memcli]] — сервер + API-ключ + sync watcher одной командой
- [[ru/user/federation]] — полная настройка федерации: публичные пиры, обмен HMAC-ключами, scope подграфов, граф федерации
- [[ru/user/selfhosted]] — Caddy + MinIO + TLS для продакшена (если нужен вход по email или внешнее хранилище объектов)
- [[ru/user/mcp]] — все MCP-инструменты, режимы аутентификации, персональные токены
