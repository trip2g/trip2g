---
title: Локальный запуск — поднять trip2g и запушить контент
description: Поднять инстанс trip2g локально из Docker-образа (или бинаря) и опубликовать волт с контентом по API за пару минут.
---

# Локальный запуск

Поднимаем инстанс trip2g на своей машине и пушим в него волт с контентом по API. Без Git-remote и без `GIT_API_REPO_PATH` — только сервер и CLI синка.

Нужно: Docker-образ `trip2g` (или собранный бинарь) и MinIO (S3-совместимое хранилище) для ассетов.

## 1. Поднять сервер

trip2g'у нужно S3-совместимое хранилище (MinIO) для ассетов. Поднимаем его, затем приложение.

```bash
# MinIO (пропустить, если уже есть)
docker run -d --name trip2g-minio -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=trip2g -e MINIO_ROOT_PASSWORD=trip2g-secret \
  minio/minio:latest server /data --console-address ":9001"

# Приложение trip2g на порту 24081 (healthcheck на 24082), свежая локальная БД
# Скачивается актуальный опубликованный образ. Для сборки из исходников
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

Важно:
- `DEV=true` включает dev-код входа (реальная почта не нужна) — **никогда не использовать в проде**.
- `FEATURES` / векторный поиск для обычного контента **не нужен** — не задавайте, чтобы не тянуть embedding-сервер.
- `--network host` даёт приложению достучаться до MinIO на `localhost:9000` и публикует порт приложения напрямую.

Сборка из исходников вместо образа: `make build` даёт `./tmp/server`; запускайте с теми же переменными (и доступным MinIO).

## 2. Завести API-ключ (dev-флоу)

При `DEV=true` сервер принимает фиксированный код входа (`111111`, также работает `000000`). Логинимся владельцем, затем создаём ключ:

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

(Если задали свой `USER_TOKEN_COOKIE_NAME` — используйте это имя cookie вместо `trip2g_token`.)

## 3. Запушить контент

CLI синка публикует папку с нотами (`.md`), шаблонами `_layouts/` и ассетами. Бинарь (`obsidian-sync/dist/trip2g-sync.mjs`) находится в репозитории trip2g — запустите из корня исходников:

```bash
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /путь/к/вашему/волту \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/graphql \
  --verbose
```

### Непрерывная синхронизация: `--watch`

Флаг `--watch` (алиас `-w`) запускает CLI как долгоживущий процесс. При старте он выполняет полную двустороннюю сверку, затем удерживает два канала:

- **Сервер → локально**: подписывается на SSE-стрим `noteChanges` и сразу записывает серверные изменения в папку волта.
- **Локально → сервер**: следит за файловой системой и отправляет правки на сервер с дебаунсом ~500 мс.

```bash
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder /путь/к/вашему/волту \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/_system/graphql
```

Процесс работает на переднем плане. Ctrl-C завершает его корректно. При фатальной ошибке выходит с ненулевым кодом — удобно для перезапуска через политику контейнера или `systemd`.

#### Фильтрация: `--include` и `--exclude`

`--include <glob>` (`-i`) и `--exclude <glob>` (`-x`) управляют тем, какие пути отслеживает SSE-подписчик. Оба флага можно повторять.

```bash
# Следить только за journal/ и projects/
node obsidian-sync/dist/trip2g-sync.mjs --watch \
  --folder /путь/к/волту \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/_system/graphql \
  --include "journal/**" \
  --include "projects/**"
```

Приоритет: флаги CLI перекрывают `livePull`-паттерны из `data.json`, которые в свою очередь перекрывают встроенный дефолт (`**` — следить за всем). Если никакие паттерны не заданы, отслеживаются все пути.

## 4. Посмотреть

Откройте permalink ноты, напр. `http://localhost:24081/<путь>/<нота>`.

Нота с `route: yourdomain.com/` во frontmatter обслуживается на этом домене (локально нужен `Host:`-хедер или DNS). Для превью без DNS открывайте обычный permalink.

Используете этот инстанс как память для ИИ-агента? Смотрите [[ru/user/agent-memory]].

## Грабли

- **Dev-код входа** — `111111` (также `000000`). Только при `DEV=true`.
- **`route:` во frontmatter** привязывает ноту к домену. Локально смотрите по обычному permalink (`/путь/нота`) или шлите `Host:`-хедер.
- **Кастомные Jet-layout'ы**: нота с `layout: <тема>/<страница>` рендерится через `_layouts/<тема>/<страница>.html`. Если шаблон не парсится, сервер молча падает на дефолтный layout (HTTP 200) — страница «работает», но выглядит не так.
- **Блок-комменты `{{/* ... */}}` ломают шаблонизатор** в текущих сборках — не используйте их в `_layouts/*.html`.
- Нота публична (видна анониму) только с `free: true` во frontmatter или patch-правилом `**/*.md → { free: true }`.
