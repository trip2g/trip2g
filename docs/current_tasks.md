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

## [IN PROGRESS] Внедрить site_title_template в рендер

### Контекст
Настройка `site_title_template` добавлена в БД и админку, но ещё не используется при рендере страниц. Нужно заменить hardcoded формат заголовка на шаблон из конфига.

### План
- [ ] Найти где формируется `<title>` в rendernotepage
- [ ] Добавить метод `SiteTitleTemplate() string` в Env (если нет)
- [ ] Использовать шаблон: `fmt.Sprintf(template, pageTitle)`
- [ ] Проверить что `%s` заменяется корректно
- [ ] Запустить make test && make lint

### Заметки
- Дефолт: `%s` (только название страницы)
- Валидация: шаблон должен содержать `%s`
