# Current Tasks

<!--
Активные задачи. Максимум 2-3.
Формат см. в CLAUDE.md
-->

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
- [ ] Проверить в браузере: подключение, получение events, автореконнект ← текущий
- [ ] Удалить демо после проверки (или оставить как dev tool)

### Заметки
- SSE клиент написан с нуля (fetch + ReadableStream), без graphql-sse зависимости
- graphqlmol.js: исправлен баг — отсутствие запятой между query и variables в subscription overloads
- graphqlmol.js: subscription overloads теперь возвращают `$trip2g_sse_host` вместо data type
- `$mol_wait_timeout is not a function` — предсуществующий баг, не связан с SSE
- Бэкенд: `schema.graphqls` type Subscription + `schema.resolvers.go` CurrentTime resolver
- Endpoint: `POST /graphql` с `Accept: text/event-stream`
- Документация: [docs/graphql.md](graphql.md), [docs/gqlgen_fasthttp.md](gqlgen_fasthttp.md)

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

**Файловый навигатор (следующий этап)**
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
- Milkdown Crepe бандл: ~2.5MB minified / ~860KB gzipped (Vue.js, lodash-es, prosemirror внутри)
- Загрузка async через `$mol_import.script('/assets/milkdown/milkdown.js')`
- Компоненты: `editor/navigator/`, `editor/content/`, `editor/preview/`, `editor/pane/`
- Raw CSS нужен для `dialog:not([open])` — mol `$mol_style_define` не поддерживает attribute selectors
- `$mol_wire_sync` для интеграции async milkdown create() в mol реактивную систему
- `$remark` wikilink fix: передаём attacher как reference, options через 3-й аргумент `$remark`
- CSS Crepe темы инлайнятся в JS через esbuild plugin (inline-css)
- Отключены: Latex, CodeMirror, ImageBlock, Table (для уменьшения бандла)
- **CSS решено**: inline-css esbuild plugin теперь вызывает `esbuild.build()` с `bundle: true` для рекурсивного разрешения всех `@import`. KaTeX CSS пропускается через `skip-katex` sub-plugin (Latex отключён)
- **Tiptap альтернатива**: бандл собран рядом (`assets/tiptap/`), 990KB vs 5MB milkdown. Тот же интерфейс (create/destroy/getMarkdown/setMarkdown/onChange). Slash menu, task lists, wiki-links. Не подключён в UI — нужно протестировать и сравнить с milkdown

## [TODO] Webhooks для изменений заметок

### Контекст
Вебхуки уведомляют внешние сервисы (агенты, автоматизации) об изменениях заметок (create/update/remove). Агент получает POST с батчем изменений и может отреагировать. Include/exclude glob-паттерны через doublestar. Опциональный JWT webhook token для доступа к API.

Подробный дизайн: [docs/webhooks.md](webhooks.md)

### План

**Этап 1: Ядро (MVP)**
- [ ] Миграция: таблицы `webhooks` + `webhook_deliveries`
- [ ] SQL-запросы (sqlc) + `make sqlc`
- [ ] `internal/webhooktoken/` — JWT sign/parse (по аналогии с hotauthtoken)
- [ ] Admin mutations: createWebhook/updateWebhook/deleteWebhook
- [ ] `internal/case/handlenotewebhooks/` — glob-матчинг (doublestar) + enqueue delivery jobs
- [ ] `cmd/server/case_methods.go` — метод `HandleNoteWebhooks(ctx, changedPathIDs, event)`
- [ ] `internal/case/backjob/deliverwebhook/` — HTTP POST + JWT + сохранение результата
- [ ] Интеграция в `HandleLatestNotesAfterSave` (create/update) и `hidenotes.Resolve` (remove)
- [ ] Admin queries: webhooks, webhookDeliveries

**Этап 2: Безопасность**
- [ ] HMAC-SHA256 подпись payload (если задан secret)
- [ ] Auth middleware для валидации webhook JWT токенов

**Этап 3: Admin UI**
- [ ] Фронтенд: CRUD вебхуков в админке
- [ ] Фронтенд: просмотр истории доставок
- [ ] Кнопка retry для failed доставок

### Заметки
- Батчинг: все совпавшие заметки → один POST на вебхук
- Matching логика синхронная (просто compute), только HTTP POST уходит в goqite (BackgroundDefaultQueue)
- Event types: create (version_count == 1), update (version_count > 1), remove (hideNotes)
- Include/exclude паттерны (comma-separated), матчинг через doublestar.Match
- `include_content` (default true) — передавать полное содержимое заметок в payload
- JWT webhook token (30 мин TTL) вместо создания/удаления API-ключей

## [TODO] Рефакторинг: обработка ошибок doublestar.Match в templateviews

### Контекст
В `internal/templateviews/query.go:82` ошибка `doublestar.Match` игнорируется (`match, _ := ...`). Нужно обрабатывать ошибку и pre-compile паттерны, чтобы невалидный glob обнаруживался раньше.

### План
- [ ] Обработать ошибку `doublestar.Match` в `query.go:82`
- [ ] Pre-compile glob паттерны (валидация при создании NoteQuery)
- [ ] make test && make lint

## [TODO] UTM-метки для ссылок из заметок

### Контекст
Заметка — источник трафика (пост в TG канале, рассылка, лендинг). Нужна возможность указать UTM-метки у заметки, чтобы все исходящие ссылки автоматически получали UTM-параметры. Это позволяет видеть в аналитике, из какого конкретно поста/канала пришёл клик.

### План
- [ ] Frontmatter поля: `utm_source`, `utm_medium`, `utm_campaign`, `utm_content`, `utm_term`
- [ ] При рендере заметки — ко всем внешним ссылкам добавлять UTM-параметры
- [ ] TG посты: автоматически проставлять `utm_source=telegram`, `utm_medium=post`
- [ ] Wikilinks (внутренние) — не трогать, только внешние ссылки
- [ ] Конфиг или шаблон дефолтных UTM для TG канала
