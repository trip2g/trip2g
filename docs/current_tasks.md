# Current Tasks

<!--
Активные задачи. Максимум 2-3.
Формат см. в CLAUDE.md
-->

## [DONE] Фиксы после merge feat/webhooks

### Контекст
Code review merge `feat/webhooks` выявил критические баги, HIGH проблемы и мелочи. Исправлено перед продакшн-деплоем.

### План

**CRITICAL**

- [x] **`skip_webhooks` флаг не работает** — добавлен `SkipWebhooks bool` в `appreq.Request`, заполняется в `checkapikey.resolveAPIKey`, проверяется в `HandleLatestNotesAfterSave`. `WebhookDepth` передаётся вместо hardcoded 0.
- [x] **Removed notes не доставляются** — добавлен `Path string` в `NoteChange`, remove events обрабатываются когда noteView nil. Webhook trigger интегрирован в HideNotes resolver (`schema.resolvers.go`).

**HIGH**

- [x] **Agent writes обходят write_patterns** — создан `webhookutil/patterns.go` с `ParseJSONStringArray` и `MatchesAny`. Валидация path против writePatterns перед InsertNote в обоих deliver*webhook.
- [x] **Delivery "success" при ошибке agent changes** — при applyErr без retries теперь mark as "failed" + return, без fallthrough к "success".
- [x] **Валидация: max_depth ≤ 999, timeout ≥ 1s, max_retries ≤ 100** — добавлена во все 4 CRUD resolver (create/update webhook/cronwebhook).

**MEDIUM**

- [x] **HTTP >= 300 = ошибка** — порог изменён с >= 500 на >= 300 в обоих deliver*webhook.
- [x] **Debug endpoint: JSON injection fix** — `json.Marshal(string(body))` вместо строковой конкатенации.
- [x] **SQL: `updated_at` триггеры** — миграция `20260211120000_add_updated_at_triggers_for_webhooks.sql`.

**Cleanup**

- [x] **Удалить `turbo.js`** — файл удалён, embed directive и закомментированная строка убраны.

- ~~Partial changes: обернуть в транзакцию~~ — требует рефакторинг InsertNote/Env interface, отложить
- ~~GraphQL: `createdAt` в AdminCronWebhook~~ — уже есть (`schema.graphqls:711`)
- Frontend: readPatterns/writePatterns в формах (отдельная сессия)

### Заметки
- SSRF — не фиксим, localhost обращения к соседним контейнерам — by design
- Missing indexes — не нужны пока, мало данных
- CHECK constraints — не критично
- `parseJSONStringArray` дублируется в 3 пакетах — вынести в `webhookutil` при фиксе write_patterns (нужен `matchesAny` тоже)
- `createdAt` в `AdminCronWebhook` — проверено, уже есть в schema.graphqls:711
- shortapitoken `SkipWebhooks: false` (checkapikey/resolve.go:116) — правильно, shortapitoken использует depth вместо skip

## [DONE] Рефакторинг конфига — Фаза 1

### Контекст
Переход от монолитной таблицы `config_versions` к атомарным таблицам настроек. Фаза 1: добавляем `site_title_template` + инфраструктуру для новых конфигов.

Подробный план: [docs/config_refactoring.md](config_refactoring.md)

### План
- [x] Миграция: `create table config_site_title_templates`
- [x] sqlc queries
- [x] GraphQL: interface `AdminConfigValue`, типы `AdminConfigStringValue`, `AdminConfigBoolValue`
- [x] GraphQL: query `configValues`, `configValue(id)`
- [x] GraphQL: mutation `setConfigStringValue`, `setConfigBoolValue`
- [x] Registry конфигов: `internal/configregistry/`
- [x] Resolver: `internal/case/admin/setconfigstringvalue/`
- [x] Env method: `SiteTitleTemplate() string`
- [x] rendernotepage: `formatTitle()`
- [x] Frontend: новая страница `/admin/config`
- [x] Тесты (backend)

### Заметки
- Таблица `config_versions` сохранена для обратной совместимости
- Все 5 конфигов мигрированы в атомарные таблицы
- `LatestConfig()` оставлен без изменений
- Фронтенд `/admin/config` полностью работает со всеми конфигами

## [DONE] Внедрить site_title_template в рендер + Удалить старый config UI

### Контекст
Настройка `site_title_template` теперь используется при рендере страниц. Также удалён старый UI и GraphQL код для config_versions.

### План
- [x] Найти где формируется `<title>` в rendernotepage
- [x] SiteTitleTemplate() метод уже существует в Env
- [x] formatTitle() уже использует шаблон (resolve.go:156)
- [x] Исправить стандартный layout: использовать resp.Title вместо resp.Note.Title
- [x] Добавить title в переменные кастомного layout (endpoint.go:164)
- [x] Удалить старый admin UI: assets/ui/admin/configversion/
- [x] Удалить старый GraphQL: AdminConfigVersion, createConfigVersion, allConfigVersions, latestConfig
- [x] Удалить resolvers и case/admin/createconfigversion/
- [x] Написать тесты для title template
- [x] Запустить make test && make lint

### Заметки
- Стандартный и кастомный layout теперь оба используют отформатированный title
- В кастомных layout доступна переменная `{{ title }}` с отформатированным заголовком
- Старая система config_versions полностью удалена

## [IN PROGRESS] Фронтенд: SSE подписка currentTime

### Контекст
SSE подписки работают через fasthttp + fasthttpadaptor + gqlgen. Демо подписка `currentTime` отдаёт время каждую секунду. Нужно реализовать клиентскую часть, чтобы убедиться что весь pipeline работает end-to-end и можно использовать подписки для реальных задач.

В будущем подписки будут использоваться для: статусы задач, синхронизация файлов (страница обновляется при правках без кнопки обновить), live preview в редакторе.

### План
- [x] SSE клиент (`$trip2g_sse_host`) — fetch + ReadableStream, SSE парсер, автореконнект
- [x] Статус-виджет (`$trip2g_sse_status` + `$trip2g_sse_icon`) — по паттерну yuf/ws/status
- [x] `$trip2g_graphql_raw_subscription` — реализация вместо stub
- [x] Компонент `$trip2g_time_current` — показывает время + статус иконку
- [x] Codegen fix: запятая в subscription overloads, return type → `$trip2g_sse_host`
- [x] Тест SSE подписки (sse.test.ts)
- [x] Документация: TypeScript codegen секция в docs/graphql.md
- [ ] Проверить в браузере: подключение, получение events, автореконнект
- [ ] Удалить демо после проверки (или оставить как dev tool)

### Заметки
- SSE клиент написан с нуля (fetch + ReadableStream), без graphql-sse зависимости
- graphqlmol.js: исправлен баг — отсутствие запятой между query и variables в subscription overloads
- graphqlmol.js: subscription overloads теперь возвращают `$trip2g_sse_host` вместо data type
- `$mol_wait_timeout is not a function` — предсуществующий баг, не связан с SSE
- Бэкенд: `schema.graphqls` type Subscription + `schema.resolvers.go` CurrentTime resolver
- Endpoint: `POST /graphql` с `Accept: text/event-stream`
- Документация: [docs/graphql.md](graphql.md), [docs/gqlgen_fasthttp.md](gqlgen_fasthttp.md)
- **Data race fix**: `fasthttpadaptor.NewFastHTTPHandler` использует sync.Pool, writer возвращается в пул при ошибке записи, а SSE горутина ещё пишет → race. Решение: `internal/fastgql.NewSSEHandler` — обёртка без sync.Pool, с mutex и отменой контекста при disconnect

## [TODO] Добавить конфиг EnableNotFoundTracking

### Контекст
Сейчас система трекает все 404 ошибки (таблицы `not_found_paths`, `not_found_ip_hits`), что создаёт лишнюю нагрузку на БД. Нужно добавить булевый конфиг для включения/выключения этого трекинга. По умолчанию — выключен.

### План
- [ ] Добавить `EnableNotFoundTracking` (bool, default false) в `configregistry`
- [ ] Добавить поле в `model.SiteConfig`
- [ ] Загружать в `app.SiteConfig()`
- [ ] Обернуть трекинг 404 в проверку конфига
- [ ] Добавить в admin UI конфигов

## [DONE] RSS

### Контекст
Любая заметка — RSS лента. Markdown AST преобразуется в RSS feed: каждая ссылка → RSS item. Дизайн: [docs/rss.md](rss.md)

### План
- [x] Markdown AST → RSS: извлечение ссылок из заметки
- [x] Роутинг: `/path/to/note.rss.xml`
- [x] Метаданные из целевых заметок (description, created_at)
- [x] Frontmatter: `rss_title`, `rss_description`
- [x] Конфиг: `EnableRSS` (bool) в SiteConfig
- [x] Package `internal/rssfeed/` с генератором
- [x] Middleware `handleRSSFeed` в `cmd/server/main.go`
- [x] Тесты

### Заметки
- Поддерживаются wikilinks и markdown links
- Internal links обогащаются метаданными из целевых заметок
- AST walking пропускает embedded images и media files
- `created_at`/`created_on` из фронтматтера переопределяет значение из БД
- E2E тесты: `e2e/rss.spec.js`

## [DONE] Sitemap.xml

### Контекст
Генерация sitemap.xml из всех опубликованных страниц. Генерируется вместе с заметками в NoteViews и отдаётся по запросу. Включаются только free: true заметки. Документация: [docs/sitemap.md](sitemap.md)

### План
- [x] Добавить поле `Sitemap []byte` в `NoteViews`
- [x] Генерировать sitemap при загрузке заметок в `mdloader`
- [x] Middleware `handleSitemap` для отдачи по `/sitemap.xml`
- [x] Конфиг `EnableSitemap` (bool, default true)
- [x] Тесты
- [x] make test && make lint

### Заметки
- Реализация завершена в `internal/sitemap/`
- Генерируется при загрузке заметок в `noteloader`
- Включаются только `free: true` заметки
- Системные страницы `/_*` исключаются
- `lastmod` берётся из `CreatedAt` или frontmatter `created_at`/`created_on`

## [DONE] Рефакторинг: ShowDraftVersions → LiveNoteViews()

### Контекст
Логика ShowDraftVersions дублировалась в `rendernotepage/resolve.go`. Перенесена в `app.LiveNoteViews()`.

### План
- [x] Перенести логику ShowDraftVersions в `app.LiveNoteViews()`
- [x] Убрать дополнительную логику из `rendernotepage/resolve.go`
- [x] Проверить все вызовы LiveNoteViews() — нигде не сломается
- [x] make test && make lint

### Заметки
- `LiveNoteViews()` теперь возвращает latest когда ShowDraftVersions включён
- RSS, sitemap и все middleware автоматически используют правильные заметки
- Админы по-прежнему могут переключаться через `?version=`

## [IN PROGRESS] Редактор файлов

### Контекст
Веб-редактор markdown файлов. Модалка на весь экран, доступна на любой фронт-странице через кнопку в хедере. Milkdown WYSIWYG редактор. Пока прикидываем интерфейс.

Подробный дизайн: [docs/editor.md](editor.md)

### План

**Интерфейс (текущий этап)**
- [x] Модалка на `<dialog>` с состоянием в `$mol_state_arg`
- [x] Кнопка открытия в `$trip2g_user_space`
- [x] Toolbar: заголовок, тогглы Files/Preview, кнопка закрытия
- [x] 3 колонки: navigator | editor | preview
- [x] Скрытие navigator и preview через тогглы
- [x] Milkdown бандл собран (esbuild IIFE, `assets/milkdown/`)
- [x] Async загрузка через `$mol_import.script`
- [x] `embed.go` и `Caddyfile` обновлены
- [x] Исправить закрытие модалки (raw CSS `display:none` для `dialog:not([open])`)
- [x] Добавить русские переводы (locale файлы)
- [x] Milkdown рендерится в content (`$mol_wire_sync` для async create)
- [x] Переход на Crepe (toolbar, block edit, link tooltip, theme)
- [x] Починить загрузку CSS Crepe — браузер пытается загрузить `prosemirror.css` и др. через mol paths

**Файловый навигатор (следующий этап)** ← текущий
- [ ] GraphQL query для списка файлов
- [ ] Дерево файлов в navigator
- [ ] Выбор файла → загрузка в редактор

**Загрузка и сохранение**
- [ ] Загрузка содержимого файла (notePaths GraphQL)
- [ ] Сохранение изменений (pushNotes GraphQL)
- [ ] Индикация несохраненных изменений
- [ ] Ctrl+S для сохранения

**Превью**
- [ ] Live preview рендеринг (renderNotePreview GraphQL)
- [ ] Синхронизация скролла

**Дополнительно**
- [ ] Автоопределение текущей страницы (meta tag trip2g:path)
- [ ] История версий файла
- [ ] Wikilinks автодополнение

### Заметки
- Tiptap бандл: ~990KB unminified (prosemirror внутри, без Vue.js)
- Загрузка async через `$mol_import.script('/assets/tiptap/tiptap.js')`
- Компоненты: `editor/navigator/`, `editor/content/`, `editor/preview/`, `editor/pane/`
- Raw CSS нужен для `dialog:not([open])` — mol `$mol_style_define` не поддерживает attribute selectors
- `$mol_wire_sync` для интеграции async tiptap create() в mol реактивную систему
- Wiki-links: markdown-it плагин в tiptap бандле
- CSS инлайнятся в JS через esbuild plugin (inline-css)
- Включены: StarterKit, TaskList, Link, Placeholder, slash menu, wiki-links
- **Tiptap**: заменил Milkdown. Бандл 990KB vs 5MB. Тот же интерфейс (create/destroy/getMarkdown/setMarkdown/onChange). Slash menu, task lists, wiki-links. Подключён в UI через `content.view.ts`

## [DONE] Change Webhooks для изменений заметок

### Контекст
Вебхуки уведомляют внешние сервисы (агенты, автоматизации) об изменениях заметок (create/update/remove). Агент получает POST с батчем изменений и может отреагировать — запустить линтер, пересобрать индекс, вызвать AI-агента. Агент может вернуть изменения в ответе (agent response).

Подробный дизайн: [docs/change_webhooks.md](change_webhooks.md)

### План

**Этап 1: Ядро (MVP)**
- [x] Миграция: таблицы `change_webhooks` + `change_webhook_deliveries` + `cron_webhooks` + `cron_webhook_deliveries` + alter `api_keys`
- [x] SQL-запросы (sqlc) + `make sqlc`
- [x] `internal/shortapitoken/` — JWT sign/parse с depth в claims
- [x] `internal/webhookutil/` — HMAC, HTTP client, payload, secret generation
- [x] Admin mutations: createWebhook/updateWebhook/deleteWebhook/regenerateSecret
- [x] `internal/case/handlenotewebhooks/` — depth check, glob-матчинг (doublestar), enqueue
- [x] `internal/case/backjob/deliverchangewebhook/` — HTTP POST + HMAC подпись + shortapitoken + парсинг agent response
- [x] Расширить `checkapikey` — поддержка `Authorization: Bearer` для shortapitoken
- [x] Интеграция в `HandleLatestNotesAfterSave` (create/update)
- [x] Admin queries: webhooks, webhookDeliveries
- [x] Debug endpoints (`DEV_MODE=true`): `/debug/test_webhook`, `/debug/test_webhook_calls`
- [x] Wiring в main.go: jobs, env methods, cron jobs
- [x] Cleanup cron jobs: delivery logs (7d), deliveries (30d)

**Этап 2: Admin UI**
- [x] Фронтенд: CRUD вебхуков в админке (catalog/show/create/update)
- [x] Фронтенд: просмотр истории доставок (встроено в show page)

### Заметки
- Батчинг: все совпавшие заметки → один POST на вебхук
- Include/exclude паттерны — JSON arrays, матчинг через doublestar.Match
- HMAC-SHA256 подпись **всегда** (secret автогенерируется при создании)
- `instruction` — текстовая инструкция для агента (один endpoint, разные задачи)
- `include_content` (default true) — содержимое заметок в payload (remove → null)
- shortapitoken (JWT, 30 мин TTL) — depth+1, read+write доступ к API
- Depth-based recursion protection: depth в JWT + max_depth в webhook + skip_webhooks в api_keys
- Agent response: опциональный `changes[]` с `expected_hash` для optimistic concurrency
- **X-Webhook-Chain-ID**: генерируется при depth=0, передаётся в headers всех вызовов в цепочке, записывается в delivery log для трейсинга полной цепочки обработки

## [TODO] Рефакторинг: обработка ошибок doublestar.Match в templateviews

### Контекст
В `internal/templateviews/query.go:82` ошибка `doublestar.Match` игнорируется (`match, _ := ...`). Нужно обрабатывать ошибку и pre-compile паттерны, чтобы невалидный glob обнаруживался раньше.

### План
- [ ] Обработать ошибку `doublestar.Match` в `query.go:82`
- [ ] Pre-compile glob паттерны (валидация при создании NoteQuery)
- [ ] make test && make lint

## [DONE] Cron Webhooks — вызов агентов по расписанию

### Контекст
Cron webhooks вызывают внешние агенты по расписанию (cron expression). Агент получает инструкцию + shortapitoken и может вернуть изменения (новые/обновлённые заметки) синхронно в ответе или async через API.

Подробный дизайн: [docs/cron_webhooks.md](cron_webhooks.md)

### План
- [x] Миграция: таблицы `cron_webhooks` + `cron_webhook_deliveries`
- [x] SQL-запросы (sqlc) + `make sqlc`
- [x] Admin mutations: create/update/delete cron webhook + regenerateSecret
- [x] Cron job registration в `cmd/server/cronjobs.go` (executecronwebhooks)
- [x] Background job: HTTP POST + парсинг ответа + импорт changes (delivercronwebhook)
- [x] Admin queries + фронтенд CRUD (catalog/show/create/update)
- [x] Delivery history view (встроено в show page)

### Заметки
- Общая инфраструктура с change_webhooks: shortapitoken, HMAC, delivery log
- Агент может отвечать sync (changes в body) или async (202 + работа через API)
- response_schema передаётся агенту как документация, не валидируется сервером
- timeout настраиваемый (default 60s)

## [TODO] Рефакторинг: expected_hash в pushNotes API

### Контекст
Optimistic concurrency control для заметок. При pushNotes можно передать `expected_hash` — если `note_paths.latest_content_hash` не совпадает, сервер отклоняет изменение. Защита от перезаписи чужих правок агентами.

### План
- [ ] Добавить `expectedHash` опциональное поле в PushNotesInput (GraphQL)
- [ ] Проверка в InsertNote: если expected_hash задан и не совпадает → ошибка
- [ ] Добавить expected_hash в shortapitoken flow
- [ ] Тесты
- [ ] make test && make lint

### Заметки
- Аналог If-Match / ETag в HTTP, CAS в concurrent programming
- Используется в ответах change_webhooks и cron_webhooks (agent response)
- Для новых файлов expected_hash пустой

## [TODO] updateNote мутация (find/replace)

### Контекст
Атомарная операция find/replace для обновления части заметки без полной перезаписи. Фундамент для агентов (inbox, task toggle, AI-правки). Автокоммит, без отдельного commitNotes.

Подробный дизайн: [docs/update_note_mutation.md](update_note_mutation.md)

### План
- [ ] SQL-запрос `LatestNoteContentByPath` + `make sqlc`
- [ ] `internal/case/updatenote/` — resolve + тесты
- [ ] GraphQL schema: `updateNote` mutation
- [ ] Resolver в `schema.resolvers.go`
- [ ] Расширить `AgentChange` struct (find/replace поля)
- [ ] `applychanges.go` — добавить find/replace mode
- [ ] Тесты agent response с find/replace

## [TODO] Agents: subprocess agent + telegram inbox agent

### Контекст
Внешние агенты, подключаемые к trip2g через вебхуки. Два примера: subprocess agent (обёртка для CLI типа `claude -p`) и telegram inbox agent (TG → заметка-inbox).

Дизайн: [docs/agents.md](agents.md), [docs/subprocess_agent.md](subprocess_agent.md), [docs/telegram_inbox_agent.md](telegram_inbox_agent.md)

### План

**Структура проекта**
- [ ] `agents/subprocess/main.go` — entry point
- [ ] `agents/subprocess/Dockerfile`
- [ ] `agents/tginbox/main.go` — entry point
- [ ] `agents/tginbox/Dockerfile`
- [ ] `docker-compose.yml` — локальный запуск trip2g + агентов

**Subprocess agent (subot)**
- [ ] HTTP handler для webhook (HMAC verify)
- [ ] Запуск subprocess (`-cmd` флаг)
- [ ] Формирование промпта из instruction + changes
- [ ] MCP bridge (проксирование в trip2g API)
- [ ] Token store (`/tmp/subot_{id}`)

**Telegram inbox agent**
- [ ] TG bot long polling
- [ ] Message buffer
- [ ] Webhook handler (cron trigger → flush buffer → changes response)
- [ ] TG message → markdown форматирование
- [ ] Token store

**Docker Compose**
- [ ] trip2g сервер
- [ ] subprocess agent (с `claude -p` или mock)
- [ ] tginbox agent
- [ ] Общая сеть, ENV конфигурация

## [TODO] Рефакторинг: pushNotes → единый API для записи заметок

### Контекст
Текущий pushNotes требует отдельного commitNotes, не поддерживает find/replace, и дублирует часть логики с updateNote. Нужно унифицировать API записи заметок.

### План
- [ ] Проанализировать текущее использование pushNotes + commitNotes
- [ ] Определить единый API (поддержка full replace и find/replace)
- [ ] Рефакторинг с обратной совместимостью
- [ ] Тесты

## [TODO] UTM-метки для ссылок из заметок

### Контекст
Заметка — источник трафика (пост в TG канале, рассылка, лендинг). Нужна возможность указать UTM-метки у заметки, чтобы все исходящие ссылки автоматически получали UTM-параметры. Это позволяет видеть в аналитике, из какого конкретно поста/канала пришёл клик.

### План
- [ ] Frontmatter поля: `utm_source`, `utm_medium`, `utm_campaign`, `utm_content`, `utm_term`
- [ ] При рендере заметки — ко всем внешним ссылкам добавлять UTM-параметры
- [ ] TG посты: автоматически проставлять `utm_source=telegram`, `utm_medium=post`
- [ ] Wikilinks (внутренние) — не трогать, только внешние ссылки
- [ ] Конфиг или шаблон дефолтных UTM для TG канала
