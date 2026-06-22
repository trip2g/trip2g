---
title: "Командная база знаний на голой VM"
free: true
lang_redirect: "[[en/team-knowledge-base]]"
---

Разверните trip2g на одной голой VM — без MinIO, без облачного хранилища — и у команды появится общая база знаний с поиском, доступная любому AI-агенту через единый MCP-эндпоинт. Заметки закрыты по умолчанию; федеративный поиск позволяет агентам на других инстансах запрашивать вашу базу без доступа ко всему хранилищу.

## Деплой на голой VM (пошаговый сценарий)

Все шаги проверены на реальном деплое в Yandex Cloud. В каждом шаге указана ошибка, которую он предотвращает.

### Шаг 1. Создать VM — размер и сеть

Создайте VM с **не менее 8 ГБ RAM** (или настройте swap до сборки образа). Этап компиляции Go во время сборки Docker-образа молча потребляет 4–6 ГБ. На VM с 4 ГБ OOM killer срабатывает в процессе сборки — SSH-сессия зависает, и VM выглядит мёртвой.

- ОС: Ubuntu 22.04.
- Зарезервируйте **статический IP**. При пересоздании или изменении конфигурации VM эфемерный IP меняется — DNS и записи Cloudflare ломаются.
- Передайте SSH-ключ при создании VM через параметр метаданных (`--metadata ssh-keys=ubuntu:<ключ>` в Yandex Cloud). По умолчанию образ включает OS Login, который игнорирует ключи, добавленные после создания.
- Используйте **SSH-ключ без кодовой фразы**. Неинтерактивные шаги (pipe `git archive`, скрипты) не могут ввести кодовую фразу.

### Шаг 2. Установить Docker и собрать образ на VM

Установите Docker на VM, затем соберите образ **из исходников прямо на VM**. Не переносите локально собранный образ с ноутбука: если ваша машина arm64 (Apple Silicon), а VM — x86_64, образ откажется запускаться с ошибкой `exec format error`.

Передайте исходники на VM и соберите там:

```bash
# на вашей машине
git archive HEAD | ssh ubuntu@<ip-vm> 'mkdir -p ~/trip2g && tar x -C ~/trip2g'

# на VM
cd ~/trip2g
docker build -t trip2g:local .
```

Сборка занимает 5–10 минут. Этап компиляции Go не выводит логов — это нормально. Не прерывайте процесс.

### Шаг 3. Запустить сервер

Смонтируйте том с данными и запустите контейнер. Используйте `STORAGE_BACKEND=local` — MinIO и S3 не нужны.

**Сначала сгенерируйте секреты:**

```bash
# DATA_ENCRYPTION_KEY — ровно 32 hex-символа (16 байт)
openssl rand -hex 16

# JWT_SECRET — любая длинная случайная строка
openssl rand -base64 32
```

**Запуск:**

```bash
docker run -d --name trip2g-kb \
  --restart unless-stopped \
  -p 80:80 \
  -e LISTEN_ADDR=0.0.0.0:80 \
  -e INTERNAL_LISTEN_ADDR=:8081 \
  -e DB_FILE=/data/kb.sqlite3 \
  -e STORAGE_BACKEND=local \
  -e STORAGE_LOCAL_DIR=/data/storage \
  -e OWNER_EMAIL=owner@yourteam.example \
  -e PUBLIC_URL=https://kb.yourteam.example \
  -e JWT_SECRET=<ваш-jwt-secret> \
  -e DATA_ENCRYPTION_KEY=<ваши-32-символа-hex> \
  -v /opt/trip2g-kb:/data \
  trip2g:local
```

**Важно:**

- `DATA_ENCRYPTION_KEY` должен отличаться от значения по умолчанию. Если переменная не задана или содержит плейсхолдер, сервер **паникует** при запуске: `"in production, data encryption key must be changed from default"`.
- `PUBLIC_URL` должен быть финальным `https://`-доменом, а не `http://localhost`. Auth-потоки и URL перенаправлений формируются из этого значения.
- `LISTEN_ADDR=0.0.0.0:80` открывает публичный порт напрямую. Если используете обратный прокси (Caddy/Nginx), привяжите к внутреннему порту и проксируйте с него.
- Не указывайте `DEV=true` в продакшене — это отключает проверки безопасности и включает фиксированные коды входа.
- Не указывайте `RESEND_API_KEY` / `SMTP_PASSWORD`, если вход по email не нужен.
- Не указывайте `GIT_API_REPO_PATH`, если не используете встроенное git-зеркало.

**Дождитесь готовности:**

```bash
until curl -sf http://localhost:8081/readyz >/dev/null; do sleep 2; done && echo "ready"
```

Используйте `/readyz`, а не `/healthz` — этот эндпоинт возвращает 200 только после завершения прогрева базы данных.

**Обязательные переменные окружения:**

| Переменная | Назначение |
|-----------|-----------|
| `STORAGE_BACKEND=local` | Хранить ассеты на диске вместо S3/MinIO |
| `STORAGE_LOCAL_DIR=/data/storage` | Директория ассетов внутри смонтированного тома |
| `JWT_SECRET` | Подпись сессионных токенов. Дефолтное значение в проде не работает |
| `DATA_ENCRYPTION_KEY` | 32-символьный hex-ключ для шифрования. Без него или на значении по умолчанию — паника при старте |
| `OWNER_EMAIL` | Email аккаунта администратора |
| `PUBLIC_URL` | Внешний URL для ссылок и auth-потоков. Укажите финальный https:// домен при первом запуске |
| `DB_FILE` | Путь к SQLite-базе внутри контейнера |
| `LISTEN_ADDR` | Основной адрес для приёма HTTP |
| `INTERNAL_LISTEN_ADDR` | Внутренний адрес для health/readiness-проверок |

### Шаг 4. Привязать домен и настроить TLS

Привяжите домен к статическому IP VM через Cloudflare (проксируемый режим). Сервер слушает порт 80; Cloudflare завершает TLS.

Установите режим SSL в Cloudflare: **Flexible** (Cloudflare → origin по HTTP) или установите сертификат на VM и используйте **Full**.

`PUBLIC_URL` должен совпадать с финальным `https://`-доменом. При смене домена позже потребуется пересоздать базу или пропатчить сохранённые URL.

### Шаг 5. Получить API-ключ администратора (HAT, без DEV-режима)

Без `DEV=true` фиксированного кода входа нет. Используйте HAT (Hot Auth Token) — короткоживущий JWT, подписанный `JWT_SECRET`, который обменивается на сессионный cookie.

```bash
# 1. Подпишите JWT: payload {"e":"<owner-email>","ae":true,"exp":<now+300>}
#    Алгоритм: HS256, секрет: JWT_SECRET
#    Через PyJWT:
JWT=$(python3 -c "
import jwt, time, os
print(jwt.encode({'e':'owner@yourteam.example','ae':True,'exp':int(time.time())+300},
  os.environ['JWT_SECRET'], algorithm='HS256'))
")

# 2. Обменяйте на сессионный cookie
TOKEN=$(curl -s -c - -X POST https://kb.yourteam.example/_system/hat \
  -H "Authorization: Bearer $JWT" \
  | grep trip2g_token | awk '{print $NF}')

# 3. Создайте API-ключ
API_KEY=$(curl -s -X POST https://kb.yourteam.example/_system/graphql \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"mutation($i:CreateApiKeyInput!){admin{createApiKey(input:$i){__typename ... on CreateApiKeyPayload{value} ... on ErrorPayload{message}}}}","variables":{"i":{"description":"team-sync"}}}' \
  | python3 -c "import sys,json; d=json.load(sys.stdin); r=d['data']['admin']['createApiKey']; print(r['value'] if r['__typename']=='CreateApiKeyPayload' else r['message'])")

echo "API-ключ: $API_KEY"
```

Мутация `createApiKey` возвращает union — `CreateApiKeyPayload{value}` при успехе, `ErrorPayload{message}` при ошибке. Проверяйте `__typename` перед чтением `value`.

Полный справочник по HAT и выпуску ключей — [[ru/user/local-quickstart]].

### Шаг 6. Опубликовать контент через CLI синка

```bash
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /путь/к/командному-хранилищу \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/_system/graphql \
  --verbose
```

Для непрерывной синхронизации во время редактирования:

```bash
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder /путь/к/командному-хранилищу \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/_system/graphql
```

Все флаги CLI синка — [[ru/user/local-quickstart]].

### Шаг 7. Настроить главную страницу

Главная страница сайта по адресу `/` должна быть заметкой с именем **`index.md`** или **`_index.md`** в корне хранилища. Сегмент `index`/`_index` в имени файла отбрасывается, и permalink становится `/`.

Поле frontmatter `route: kb.yourteam.example/` **не делает** заметку главной страницей основного домена. Оно привязывает заметку к этому пути на кастомном домене, но слот `/` принадлежит заметке `index`/`_index`.

### Шаг 8. Кастомные HTML-страницы — используйте Jet-layout

Сырой HTML в теле заметки санируется конвейером Markdown. Блок вида:

```html
<div style="display:flex; height:100vh; ...">Не лезь.</div>
```

превращается в `<!-- raw HTML omitted -->` в отрендеренной странице.

Для полностью кастомной страницы — например, гостевого «заглушки» — используйте **Jet-layout**: серверный `.html`-шаблон, который не санируется.

1. Создайте `_layouts/guard.html` в хранилище — полный `<!DOCTYPE html>`-документ:

```html
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <title>Только авторизованным</title>
  <style>
    body { margin: 0; display: flex; align-items: center; justify-content: center;
           height: 100vh; font-family: system-ui, sans-serif; }
    .msg { font-size: 3rem; font-weight: bold; text-align: center; }
    a { display: block; margin-top: 1rem; font-size: 1rem; }
  </style>
</head>
<body>
  <div class="msg">
    Не лезь — убьёт.<br>
    <a href="/kb">Войти</a>
  </div>
</body>
</html>
```

2. В корневой `index.md` укажите layout:

```yaml
---
free: true
layout: guard
---
```

Layout-файл рендерится на сервере — HTML не санируется. Ссылка `/kb` ведёт на любую закрытую заметку: поскольку у неё нет `free: true`, trip2g покажет анонимному посетителю встроенную страницу входа/пейвол. Отдельного URL `/login` нет — страница пейвола и есть точка входа.

### Шаг 9. Подключить федеративный поиск

Чтобы агенты могли искать по другому хабу через `/_system/mcp` вашего инстанса, добавьте KB-заметку в хранилище:

```yaml
---
free: true
mcp_federation_kb_url: https://hub.example.com/_system/mcp
mcp_federation_kb_id: hub-name
---
Использовать для: поиска по общей командной базе знаний и публичным справочникам.
```

**`free: true` обязательно.** Без этого поля сканирование федерации (`accessibleKBNotes`) игнорирует заметку, а `federated_search` возвращает "Federation is not configured". Это самая частая ошибка конфигурации.

После синхронизации `/_system/mcp` предоставляет `federated_search`, `federated_similar`, `federated_note_html` и `federated_expand`.

`memcli hub <url>` автоматизирует создание KB-заметки и синхронизацию. Подробнее — [[ru/memcli]].

---

## Подробный справочник

### Получить аккаунт администратора и API-ключ

Войдите как владелец, затем выпустите один API-ключ для CLI синка. Вся команда использует этот ключ для публикации контента.

**Способ A — HAT (Hot Auth Token), email не нужен.**

HAT — беспарольный вход администратора. Сервер проверяет короткоживущий JWT, подписанный `JWT_SECRET`. Именно этот способ используют memcli и другие headless-инструменты. Эндпоинт — `/_system/hat`. Полный процесс — в [[ru/user/local-quickstart]] (раздел «Завести API-ключ») и [[ru/user/agent-memory]].

**Способ B — вход по email (продакшен).**

Укажите `RESEND_API_KEY`, `MAIL_FROM` и верифицированный домен отправителя в Resend. Сервер отправит одноразовый код на `OWNER_EMAIL`. После входа создайте ключ в Администрирование → API-ключи → Создать.

### Опубликовать контент через CLI синка

Публикуем папку хранилища на сервер. Запускайте из корня исходников trip2g:

```bash
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /путь/к/командному-хранилищу \
  --api-key "$API_KEY" \
  --api-url https://kb.yourteam.example/_system/graphql \
  --verbose
```

Полный справочник по CLI синка — [[ru/user/local-quickstart]].

### Подключить федеративный поиск

Федерация позволяет AI-агенту обратиться к единому MCP-эндпоинту вашей базы и выполнить поиск по всем подключённым базам знаний. Для этого нужна **KB-заметка** в хранилище.

**Важно: `free: true` обязательно.**

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

Нажмите Sync. Локальный `/_system/mcp` теперь предоставляет `federated_search`, `federated_similar`, `federated_note_html` и `federated_expand`.

**Пример MCP-вызова:**

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

**SSRF и глубина.** Публичные хабы (внешние URL) разрешены по умолчанию. Для приватных/внутрисетевых адресов требуется `MCP_FEDERATION_ALLOW_PRIVATE=true`. Веерный поиск останавливается на глубине 3 (`MCP_FEDERATION_MAX_DEPTH`). Таймаут на один пир — 2 секунды (`MCP_FEDERATION_FANOUT_TIMEOUT`).

Полная настройка федерации — в [[ru/user/federation]].

### Управление доступом: кто что видит

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

Другой trip2g-инстанс может подключить вашу базу через федерацию. Это требует обмена HMAC-ключами (federation secrets), который точно определяет, какие подграфы видит пир. Подробнее — [[ru/user/federation]] (раздел «Добавить приватного пира»).

**Итоговая таблица доступа:**

| Запрашивающий | Токен | Видит |
|--------------|-------|-------|
| Анонимный агент / публичный хаб | Нет | Только заметки с `free: true` |
| Аутентифицированный участник | `t2g_<токен>` | Заметки в scope подписки |
| Admin API-ключ | `X-API-Key: <ключ>` | Все заметки |
| Федерированный пир с HMAC-секретом | Подписанный JWT | Заметки в scope секрета |

## Смотрите также

- [[ru/user/local-quickstart]] — полный справочник по локальному запуску, флаги CLI синка, mint-поток через HAT
- [[ru/user/agent-memory]] — память для одиночного агента; подробности выпуска ключа через HAT
- [[ru/memcli]] — сервер + API-ключ + sync watcher одной командой; подкоманда `hub` для федерации
- [[ru/user/federation]] — полная настройка федерации: публичные пиры, обмен HMAC-ключами, scope подграфов, граф федерации
- [[ru/user/selfhosted]] — Caddy + MinIO + TLS для продакшена (если нужен вход по email или внешнее хранилище объектов)
- [[ru/user/mcp]] — все MCP-инструменты, режимы аутентификации, персональные токены
