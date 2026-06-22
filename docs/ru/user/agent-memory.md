---
title: "Долгосрочная память для ИИ-агентов с trip2g"
free: true
lang_redirect: "[[en/user/agent-memory]]"
---

Разверните trip2g в Docker, подключите MCP-клиент агента, загрузите заметки — и агент будет читать и искать их как долгосрочную память. Чтение конкретных секций экономит токены: в типичном случае примерно в 15 раз дешевле, чем загружать всю заметку целиком.

## 1. Запустить демон (Docker)

trip2g использует S3-совместимое хранилище для ассетов. Для локальной разработки сначала поднимите MinIO, затем приложение.

```bash
# MinIO (пропустить, если уже есть)
docker run -d --name trip2g-minio -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=trip2g -e MINIO_ROOT_PASSWORD=trip2g-secret \
  minio/minio:latest server /data --console-address ":9001"

# Приложение trip2g на порту 24081 (healthcheck на 24082), свежая локальная БД
# Скачивается актуальный опубликованный образ. Для сборки локально из исходников
# выполните `docker build -t trip2g:local .` в корне репозитория trip2g и замените
# тег образа ниже на `trip2g:local`.
mkdir -p /tmp/trip2g-local
docker run -d --name trip2g-local --network host \
  -e LISTEN_ADDR=0.0.0.0:24081 -e INTERNAL_LISTEN_ADDR=:24082 \
  -e DB_FILE=/data/local.sqlite3 \
  -e DEV=true \
  -e OWNER_EMAIL=hello@example.com \
  -e MINIO_ENDPOINT=localhost:9000 \
  -e MINIO_ACCESS_KEY_ID=trip2g -e MINIO_SECRET_KEY=trip2g-secret \
  -e MINIO_BUCKET=trip2g-local -e MINIO_USE_SSL=false \
  -e PUBLIC_URL=http://localhost:24081 \
  -e JWT_SECRET=dev-secret-not-for-prod \
  -e USER_TOKEN_INSECURE=true \
  -e GIT_API_REPO_PATH=/data/git -e GIT_API_BASE_PATH=/git \
  -e RESEND_API_KEY=dev -e MAIL_FROM=dev@example.com \
  -v /tmp/trip2g-local:/data \
  ghcr.io/trip2g/trip2g:latest

# ждём, пока поднимется — и проверяем, что контейнер действительно запущен
docker ps | grep trip2g-local
until curl -sf http://localhost:24082/healthz >/dev/null; do sleep 1; done; echo "up"
```

Демон слушает на **порту 24081**. MCP-эндпоинт: `http://localhost:24081/_system/mcp`.

Важно:
- `DEV=true` включает фиксированный код входа (`111111`) — реальная почта не нужна. **Не используйте в продакшене.**
- `--network host` даёт приложению доступ к MinIO на `localhost:9000` и публикует порт приложения напрямую.
- Для продакшена с Caddy, TLS и внешним хранилищем — см. [[ru/user/selfhosted]].

### Завести API-ключ

Ключ нужен для публикации заметок-памяти и аутентификации MCP-эндпоинта. При `DEV=true`:

```bash
GQL=http://localhost:24081/graphql

curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:RequestEmailSignInCodeInput!){requestEmailSignInCode(input:$i){__typename}}","variables":{"i":{"email":"hello@example.com"}}}' >/dev/null

TOKEN=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:SignInByEmailInput!){signInByEmail(input:$i){__typename ... on SignInPayload{token} ... on ErrorPayload{message}}}","variables":{"i":{"email":"hello@example.com","code":"111111"}}}' \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

API_KEY=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' -H "Cookie: trip2g_token=$TOKEN" \
  -d '{"query":"mutation($i:CreateApiKeyInput!){admin{createApiKey(input:$i){__typename ... on CreateApiKeyPayload{value} ... on ErrorPayload{message}}}}","variables":{"i":{"description":"local"}}}' \
  | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

echo "API-ключ: $API_KEY"
```

## 2. Подключить агента (MCP)

trip2g открывает MCP-эндпоинт по адресу `/_system/mcp`. Stdio-адаптер оборачивает три шага поиска — поиск, навигация по оглавлению, чтение секции — в **один инструмент**, который вызывает агент.

Скрипт адаптера — `docs/en/user/trip2g_mcp_stdio_adapter.py` в репозитории trip2g (также доступен на deployed docs сайте). Скачайте его оттуда и укажите абсолютный путь в конфигурации.

Зарегистрируйте его в своём MCP-клиенте (Claude Desktop, Cursor, Claude Code или любой агент с MCP по stdio):

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

`TRIP2G_TOKEN` — это **персональный токен доступа** (формат `t2g_…`), а не admin API-ключ из шага «Завести API-ключ». Создайте его в разделе Пользователь → Токены в интерфейсе trip2g. Подробнее — [[ru/user/mcp]] (раздел «Персональные токены доступа»).

Admin API-ключ (полученный выше) используется для прямых curl-запросов через `X-API-Key` и для CLI синка — не для stdio-адаптера.

Важно: инструмент `expand` работает корректно только с актуальным образом. В старых локальных сборках он возвращает плоский `toc` вместо навигируемого дерева. Используйте `ghcr.io/trip2g/trip2g:latest`, чтобы избежать этого.

`pip install` не нужен — адаптер использует только стандартную библиотеку Python 3.

### Доступные инструменты

Полный MCP-эндпоинт (напрямую или через адаптер) предоставляет:

| Инструмент | Что делает |
|------------|------------|
| `search(query)` | Векторный или полнотекстовый поиск по всем заметкам-памяти. Возвращает краткие сниппеты — хлебную крошку заголовка и `toc_path` для каждого результата, а не всю заметку |
| `expand(pid, toc_path?)` | Послойная навигация по оглавлению заметки. Возвращает прямые дочерние узлы TOC для последовательного погружения |
| `note_html(path, toc_path?)` | Читает всю заметку или конкретную секцию по `toc_path` |
| `similar(path)` | Находит заметки, похожие на указанную |

Адаптер оборачивает search → expand → note_html в один составной инструмент и возвращает только самую релевантную секцию. Подробнее — [[ru/user/ai-agent-mcp-adapter]].

### Аутентификация через API-ключ

API-ключи принимаются напрямую на MCP-эндпоинте:

```bash
curl http://localhost:24081/_system/mcp \
  -H "X-API-Key: <ваш-api-ключ>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

API-ключ даёт доступ ко всем заметкам на уровне администратора. Для пользовательского доступа используйте персональный токен (`t2g_…`) из раздела Пользователь → Токены.

## 3. Загрузить память

CLI синка публикует папку с Markdown-заметками в запущенный инстанс trip2g через API. Это шаг ingestion-памяти: ваши заметки становятся базой знаний, которую агент может искать и читать.

CLI синка (`obsidian-sync/dist/trip2g-sync.mjs`) находится в репозитории trip2g и не включён в Docker-образ. Запустите следующее из корня исходников trip2g:

```bash
# из корня репозитория trip2g
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /путь/к/вашему/волту \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/graphql \
  --verbose
```

Запускайте эту команду повторно при добавлении или обновлении заметок. Повторно загружаются только изменённые файлы.

Заметка доступна агенту без аутентификации только при наличии `free: true` во frontmatter. Для приватной памяти, которую читает только ваш агент, не добавляйте `free: true` — аутентифицируйтесь API-ключом или токеном.

Полный справочник по CLI синка — [[ru/user/local-quickstart]].

## 4. Вспоминать: search → expand → note_html

После загрузки заметок агент извлекает память через MCP-инструменты в три шага:

```
1. search(query)
   → краткие результаты: хлебная крошка заголовка + toc_path для каждого совпадения

2. expand(pid=N)                     # обзор структуры верхнего уровня
   expand(pid=N, toc_path=[...])    # углубиться в нужную ветку
   → повторять, пока не достигнете листа (has_children: false)

3. note_html(pid=N, toc_path=[...])
   → прочитать только нужную секцию
```

Если `search` уже вернул точный `toc_path` для совпадения, пропустите `expand` и вызовите `note_html` напрямую с этим путём. Адаптер делает это автоматически.

### Почему это экономит токены

Чтение одной секции вместо всей заметки в типичном случае **примерно в 15 раз дешевле** — и ответ оказывается в начале контекста, где у модели лучшее recall, а не в хвосте, где оно деградирует. Цифры и воспроизводимый бенчмарк: [[ru/user/token-economy-bench]].

Выигрыш масштабируется с размером заметки. Длинные заметки (архитектурные доки, changelog, руководства) экономят больше всего. Короткие заметки (факт, сниппет конфига) — незначительно: они и так дёшевы.

Подробнее о механизме: [[Token Economy]].

## Смотрите также

- [[ru/user/local-quickstart]] — полный справочник по локальному запуску
- [[ru/user/selfhosted]] — продакшен-установка с Docker Compose, Caddy и TLS
- [[ru/user/ai-agent-mcp-adapter]] — stdio-адаптер: один инструмент, только нужная секция
- [[ru/user/mcp]] — все MCP-методы, управление доступом и именованные точки входа
- [[ru/user/expand]] — послойная навигация по оглавлению
- [[ru/user/token-economy-bench]] — измеренная экономия токенов, воспроизводимый бенчмарк
