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

## [TODO] RSS

### Контекст
Любая заметка — RSS лента. Markdown AST преобразуется в RSS feed: каждая ссылка → RSS item. Дизайн: [docs/rss.md](rss.md)

### План
- [ ] Markdown AST → RSS: извлечение ссылок из заметки
- [ ] Роутинг: `/path/to/note.rss.xml`
- [ ] Метаданные из целевых заметок (description, created_at)
- [ ] Frontmatter: `rss_title`, `rss_description`
- [ ] Конфиг: `EnableRSS` (bool) в SiteConfig
