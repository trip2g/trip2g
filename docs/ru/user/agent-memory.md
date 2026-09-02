---
title: "Долгосрочная память для ИИ-агентов с trip2g"
free: true
lang_redirect: "[[en/user/agent-memory]]"
---

Разверните trip2g в Docker, подключите MCP-клиент агента, загрузите заметки. Агент будет читать и искать их как долгосрочную память. Чтение конкретных секций экономит токены: в типичном случае примерно в 15 раз дешевле, чем загружать всю заметку целиком.

Вставьте это в ваш coding-агент (Claude Code, Cursor или аналогичный), чтобы он настроил trip2g как долгосрочную память:

```text
Follow https://trip2g.com/en/user/agent-memory to set up trip2g as your persistent memory.

Step 1 — clone the repo (memcli ships prebuilt, no separate install needed):
  git clone https://github.com/trip2g/trip2g

Step 2 — boot the local memory server:
  node trip2g/cli/memcli/dist/memcli.js up --folder ./memory-vault

  This starts a trip2g instance on http://localhost:24081, mints an admin key
  via HAT, launches the trip2g-sync --watch sidecar (any .md file you write to
  ./memory-vault is pushed to the server within ~500 ms), and writes a hub.md
  federation note so you can also search trip2g.com's knowledge base.

Step 3 — register the MCP server in your agent's MCP client config:
  {
    "mcpServers": {
      "trip2g-memory": {
        "command": "python3",
        "args": ["/absolute/path/to/trip2g/docs/en/user/trip2g_mcp_stdio_adapter.py"],
        "env": {
          "TRIP2G_MCP_URL": "http://localhost:24081/_system/mcp",
          "TRIP2G_TOKEN": "t2g_your-personal-access-token"
        }
      }
    }
  }
  TRIP2G_TOKEN is a personal access token (format t2g_…) — create one under
  User → Tokens in the trip2g UI at http://localhost:24081. The adapter uses
  Python 3 stdlib only; no pip install needed.

Step 4 — verify recall is working:
  Call search("trip2g memory") via the MCP tool, then call note_html() on one
  of the returned paths and show a result.

Once set up, follow these rules every session:
- REMEMBER: write durable facts as Markdown files into ./memory-vault. The
  watcher syncs them in ~500 ms, making them searchable immediately.
- RECALL: always use search → expand → note_html. Read only the section you
  need — it is ~15× cheaper than loading the whole note.
```

Два типа учётных данных:
- **Admin API-ключ**: используется CLI синхронизации и прямыми curl-запросами через `X-API-Key`. Создаётся автоматически через memcli или вручную через GraphQL-мутации (раздел ниже).
- **Персональный токен (`t2g_…`)**: используется stdio MCP-адаптером. Создайте его в разделе Пользователь → Токены в интерфейсе trip2g после запуска сервера.

Это разные сущности. Путаница чаще всего возникает при настройке MCP-адаптера. При ошибках аутентификации см. [[ru/user/mcp]].

## 1. Запуск за 30 секунд через memcli

**memcli** запускает сервер одной командой: поднимает локальное хранилище, создаёт admin-ключ через HAT (без email, без `DEV=true`), стартует сайдкар `trip2g-sync --watch` и записывает федерационную заметку `hub.md` в ваш волт. Это самый быстрый путь.

Скомпилированный бандл поставляется в репозитории, сборка из свежего клона не нужна:

```bash
node cli/memcli/dist/memcli.js up --folder ./memory-vault
```

Готово. По завершении увидите:

```
memory live — web: http://localhost:24081  read/write .md in ./memory-vault
```

Сервер на порту 24081. MCP-эндпоинт: `http://localhost:24081/_system/mcp`. Sync-вотчер запущен в фоне, любой `.md`-файл, добавленный в `./memory-vault`, появится на сервере в течение ~500 мс.

При первом запуске memcli генерирует случайные секреты (`JWT_SECRET`, `DATA_ENCRYPTION_KEY`) и сохраняет их в `./memory-vault/.trip2g-memory/env`. Повторные вызовы `up` переиспользуют те же секреты и идемпотентны.

**Другие команды memcli:**

```bash
node cli/memcli/dist/memcli.js status   # состояние контейнера и вотчера
node cli/memcli/dist/memcli.js logs     # логи сервера
node cli/memcli/dist/memcli.js down     # остановить всё
node cli/memcli/dist/memcli.js key      # перевыпустить API-ключ
```

**Флаги для `up`:**

| Флаг | По умолчанию | Назначение |
|------|-------------|------------|
| `--folder <path>` | `./memory-vault` | Директория волта |
| `--port <n>` | `24081` | Публичный порт |
| `--no-hub` | (нет) | Не создавать `hub.md` |
| `--hub-url <url>` | `https://trip2g.com/_system/mcp` | Другой федерационный хаб |
| `--image <ref>` | `ghcr.io/trip2g/trip2g:latest` | Docker-образ |

**Пересборка из исходников** нужна только при изменении исходного кода CLI:

```bash
cd cli/memcli && npm install && npm run codegen && npm run build
```

`codegen` читает схему из репозитория; запущенный сервер не нужен.

### Федерационная hub-заметка

По умолчанию `memcli up` записывает в волт `hub.md` с таким frontmatter:

```yaml
mcp_federation_kb_url: https://trip2g.com/_system/mcp
```

Это федерирует локальный инстанс с trip2g.com: всё, что подключается к вашему `/_system/mcp`, получает доступ к базе знаний trip2g.com через ваш инстанс (федерация, а не копия).

Передайте `--no-hub`, чтобы не создавать заметку. `--hub-url <url>` указывает другой хаб:

```bash
node cli/memcli/dist/memcli.js up --folder ./memory-vault --hub-url https://example.com/_system/mcp
```

Файл `hub.md` хранится как обычная Markdown-заметка: удалите его, и федерационная ссылка исчезнет при следующей синхронизации.

## 2. Ручная установка (без memcli или для существующего compose-стека)

Этот путь подходит, если нужен полный контроль над контейнером или интеграция с существующим Docker Compose.

### Запустить демон

```bash
# trip2g на порту 24081, healthcheck на 24082, ассеты хранятся на диске
# Замените ghcr.io/trip2g/trip2g:latest на trip2g:local, если собирали из исходников.
mkdir -p /tmp/trip2g-local
docker run -d --name trip2g-local \
  -p 24081:24081 -p 24082:24082 \
  -e LISTEN_ADDR=0.0.0.0:24081 \
  -e INTERNAL_LISTEN_ADDR=0.0.0.0:24082 \
  -e DB_FILE=/data/local.sqlite3 \
  -e STORAGE_BACKEND=local \
  -e STORAGE_LOCAL_DIR=/data/storage \
  -e DEV=true \
  -e OWNER_EMAIL=hello@example.com \
  -e PUBLIC_URL=http://localhost:24081 \
  -e JWT_SECRET=dev-secret-not-for-prod \
  -e USER_TOKEN_INSECURE=true \
  -e GIT_API_REPO_PATH=/data/git \
  -e GIT_API_BASE_PATH=/git \
  -e RESEND_API_KEY=dev \
  -e MAIL_FROM=dev@example.com \
  -v /tmp/trip2g-local:/data \
  ghcr.io/trip2g/trip2g:latest

# убеждаемся, что контейнер запустился, ждём готовности
docker ps | grep trip2g-local
until curl -sf http://localhost:24082/healthz >/dev/null; do sleep 1; done && echo "up"
```

Переменные окружения кратко:

```
LISTEN_ADDR / INTERNAL_LISTEN_ADDR   # порт приложения + порт healthcheck (внутри контейнера)
DB_FILE / STORAGE_BACKEND            # путь к SQLite + бэкенд ассетов (local = диск, без S3)
STORAGE_LOCAL_DIR                    # куда сохраняются ассеты внутри тома
DEV=true                             # фиксированный код входа 111111 — не использовать в продакшене
OWNER_EMAIL                          # email первого admin-аккаунта (нужен при DEV=true)
PUBLIC_URL                           # URL, по которому приложение считает себя доступным
JWT_SECRET                           # HMAC-секрет — в продакшене использовать случайное значение
USER_TOKEN_INSECURE                  # разрешает токены по HTTP (допустимо для localhost)
GIT_API_REPO_PATH / GIT_API_BASE_PATH  # git-зеркало для истории заметок
RESEND_API_KEY / MAIL_FROM           # транспорт email (stub-значения работают в DEV-режиме)
```

Важно: флаг `-p 24081:24081 -p 24082:24082` публикует порты на всех интерфейсах хоста и работает на Linux, Mac и Windows (Docker Desktop). `--network host` работает только на Linux. Docker Desktop запускает движок в Linux-VM, поэтому host-сеть не публикует порты на Mac и Windows, и `localhost:24081` не достигнет контейнера.

Для продакшен-установки с Caddy, TLS и S3-совместимым хранилищем см. [[ru/user/selfhosted]].

### Завести API-ключ

При `DEV=true` фиксированный код входа: `111111`. Три curl-запроса для получения ключа:

```bash
GQL=http://localhost:24081/graphql

# 1. запросить код входа (попадает в лог, не отправляется на настоящий email)
curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:RequestEmailSignInCodeInput!){requestEmailSignInCode(input:$i){__typename}}","variables":{"i":{"email":"hello@example.com"}}}' >/dev/null

# 2. войти с кодом 111111, получить токен сессии
TOKEN=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:SignInByEmailInput!){signInByEmail(input:$i){__typename ... on SignInPayload{token} ... on ErrorPayload{message}}}","variables":{"i":{"email":"hello@example.com","code":"111111"}}}' \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

# 3. создать API-ключ
API_KEY=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' -H "Cookie: trip2g_token=$TOKEN" \
  -d '{"query":"mutation($i:CreateApiKeyInput!){admin{createApiKey(input:$i){__typename ... on CreateApiKeyPayload{value} ... on ErrorPayload{message}}}}","variables":{"i":{"description":"local"}}}' \
  | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

echo "API-ключ: $API_KEY"
```

## 3. Подключить агента (MCP)

trip2g открывает MCP-эндпоинт по адресу `/_system/mcp`. Stdio-адаптер оборачивает три шага (поиск, навигация по оглавлению, чтение секции) в **один инструмент**, который вызывает агент.

Скрипт адаптера `docs/en/user/trip2g_mcp_stdio_adapter.py` находится в репозитории trip2g (также доступен на deployed docs сайте). Скачайте его и укажите абсолютный путь в конфигурации.

Зарегистрируйте его в MCP-клиенте (Claude Desktop, Cursor, Claude Code или любой агент с MCP по stdio):

```json
{
  "mcpServers": {
    "trip2g-memory": {
      "command": "python3",
      "args": ["/абсолютный/путь/к/trip2g_mcp_stdio_adapter.py"],
      "env": {
        "TRIP2G_MCP_URL": "http://localhost:24081/_system/mcp",
        "TRIP2G_TOKEN": "t2g_ваш-персональный-токен"
      }
    }
  }
}
```

`TRIP2G_TOKEN` содержит **персональный токен доступа** (формат `t2g_…`), а не admin API-ключ. Создайте его в разделе Пользователь → Токены в интерфейсе trip2g. Подробнее: [[ru/user/mcp]] (раздел «Персональные токены доступа»).

Admin API-ключ используется для прямых curl-запросов через `X-API-Key` и для CLI синхронизации, но не для stdio-адаптера.

Примечание: инструмент `expand` работает корректно только с актуальным образом. В старых локальных сборках он возвращает плоский `toc` вместо навигируемого дерева. Используйте `ghcr.io/trip2g/trip2g:latest`.

`pip install` не нужен: адаптер использует только стандартную библиотеку Python 3.

### Доступные инструменты

Полный MCP-эндпоинт (напрямую или через адаптер) предоставляет:

| Инструмент | Что делает |
|------------|------------|
| `search(query)` | Векторный или полнотекстовый поиск по всем заметкам. Возвращает краткие сниппеты: хлебную крошку заголовка и `toc_path`, а не всю заметку |
| `expand(pid, toc_path?)` | Послойная навигация по оглавлению заметки. Возвращает прямые дочерние узлы TOC для последовательного погружения; раздел без подразделов возвращается целиком |
| `note_html(path, toc_path?)` | Читает всю заметку или конкретную секцию по `toc_path` |
| `similar(path)` | Находит заметки, похожие на указанную |

Адаптер оборачивает search → expand → note_html в один составной инструмент и возвращает только самую релевантную секцию. Подробнее: [[ru/user/ai-agent-mcp-adapter]].

### Аутентификация через API-ключ

API-ключи принимаются напрямую на MCP-эндпоинте:

```bash
curl http://localhost:24081/_system/mcp \
  -H "X-API-Key: <ваш-api-ключ>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

API-ключ даёт доступ ко всем заметкам на уровне администратора. Для пользовательского доступа используйте персональный токен (`t2g_…`) из раздела Пользователь → Токены.

## 4. Загрузить память

CLI синхронизации публикует папку с Markdown-заметками в запущенный инстанс trip2g через API. Это шаг ingestion: ваши заметки становятся базой знаний, которую агент может искать и читать.

CLI синхронизации (`obsidian-sync/dist/trip2g-sync.mjs`) находится в репозитории trip2g и не включён в Docker-образ. Запустите из корня репозитория:

```bash
# из корня репозитория trip2g
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /путь/к/вашему/волту \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/graphql \
  --verbose
```

Повторяйте команду при добавлении или обновлении заметок. Повторно загружаются только изменённые файлы.

Заметка доступна агенту без аутентификации, только если во frontmatter есть `free: true`. Для приватной памяти, которую читает только ваш агент, не добавляйте `free: true`; аутентифицируйтесь API-ключом или токеном.

Полный справочник по CLI синхронизации: [[ru/user/local-quickstart]].

Если вы использовали memcli, сайдкар `--watch` уже запущен. Достаточно просто создавать `.md`-файлы прямо в папке волта.

## 4а. Непрерывная синхронизация: `--watch` как сайдкар

Разовый синк из раздела 4 загружает заметки один раз. Если агент пишет заметки и вы хотите, чтобы правки были доступны сразу, запустите CLI синхронизации в режиме watch как долгоживущий сайдкар рядом с демоном trip2g.

```bash
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder /путь/к/вашему/волту \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/_system/graphql
```

При старте выполняется полная двусторонняя сверка. Затем:

- Любая заметка, записанная агентом на диск, отправляется на сервер в течение ~500 мс (файловый вотчер + дебаунс).
- Любая заметка, изменённая на сервере, сразу записывается обратно в папку волта через SSE-подписку `noteChanges`.

Процесс работает на переднем плане и выходит с ненулевым кодом при фатальной ошибке. В Docker Compose добавьте его вторым сервисом, смонтировав тот же том волта, что и контейнер агента. Ctrl-C завершает работу корректно.

Чтобы ограничить, за какими путями следит демон синхронизации, передайте глобы `--include` и `--exclude`. Полный справочник: [[ru/user/local-quickstart]] (раздел «Фильтрация: --include и --exclude»).

### Смотрите за работой агента в браузере

Пока `--watch` запущен, откройте любую страницу сайта и добавьте к URL `?#!live_follow=1`:

```
http://localhost:24081/some-note?#!live_follow=1
```

Браузер войдёт в режим следования (cinema mode) и будет автоматически переходить на заметку, изменившуюся последней. Настройка сохраняется при автопереходах, браузер продолжает следовать, пока вы не отключите режим.

Подробнее о режиме следования и переключателе перезагрузки: [[ru/user/live-editing]].

## 5. Вспоминать: search → expand → note_html

После загрузки заметок агент извлекает память через MCP-инструменты в три шага:

```
1. search(query)
   → краткие результаты: хлебная крошка заголовка + toc_path для каждого совпадения

2. expand(pid=N)                     # обзор структуры верхнего уровня
   expand(pid=N, toc_path=[...])    # углубиться в нужную ветку
   → повторять, пока не достигнете листа (has_children: false); expand на листе
     возвращает его содержимое (как note_html), на родителе — список детей

3. note_html(pid=N, toc_path=[...])
   → прочитать секцию, путь к которой уже известен
```

Если `search` уже вернул точный `toc_path` для совпадения, пропустите `expand` и вызовите `note_html` напрямую с этим путём. Адаптер делает это автоматически.

### Почему это экономит токены

Чтение одной секции вместо всей заметки в типичном случае **примерно в 15 раз дешевле**. Ответ оказывается в начале контекста, где у модели лучший recall, а не в хвосте, где он деградирует. Цифры и воспроизводимый бенчмарк: [[ru/user/token-economy-bench]].

Выигрыш масштабируется с размером заметки. Длинные заметки (архитектурные доки, changelog, руководства) экономят больше всего. Короткие (факт, сниппет конфига) экономят незначительно: они и так дёшевы.

Подробнее о механизме: [[Token Economy]].

## Смотрите также

- [[ru/user/local-quickstart]]: полный справочник по локальному запуску, включая флаг `--watch`
- [[ru/user/selfhosted]]: продакшен-установка с Docker Compose, Caddy и TLS
- [[ru/user/ai-agent-mcp-adapter]]: stdio-адаптер: один инструмент, только нужная секция
- [[ru/user/mcp]]: все MCP-методы, управление доступом и именованные точки входа
- [[ru/user/expand]]: послойная навигация по оглавлению
- [[ru/user/token-economy-bench]]: измеренная экономия токенов, воспроизводимый бенчмарк
- [[ru/user/live-editing]]: режим следования (cinema mode) и переключатель перезагрузки для наблюдения за правками в реальном времени
- [[ru/user/agent-status]]: автоматическая трансляция статуса коллегам при завершении сессии
