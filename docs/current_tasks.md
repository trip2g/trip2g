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
- [ ] Проверить Crepe в браузере, подтвердить что контролы работают ← текущий

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
