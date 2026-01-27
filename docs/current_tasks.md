# Current Tasks

<!--
Активные задачи. Максимум 2-3.
Формат см. в CLAUDE.md
-->

## [IN PROGRESS] Рефакторинг конфига — Фаза 1

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
- [ ] Frontend: новая страница `/admin/config` ← текущий
- [x] Тесты (backend)

### Заметки
- Старую логику `config_versions` пока не трогаем
- Дефолт для site_title_template: `%s` (только название страницы)
- Валидация: шаблон должен содержать `%s`
- Registry конфигов в `internal/configregistry/` хранит метаданные (id, description, type, default, validate)
- Пока реализован только `site_title_template`, остальные конфиги возвращают defaults
- GraphQL API готов для всех конфигов сразу
