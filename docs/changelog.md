# Changelog

## 2026-02-13

- **Change Webhooks**: исправлен баг с agent response
  - Заметки, созданные webhook'ом через agent response, теперь сразу доступны
  - Добавлен вызов `PrepareLatestNotes` после `InsertNote` для обновления кеша
  - E2E тесты для проверки полного workflow webhook delivery + agent response

## 2026-02-10

- **Change Webhooks**: уведомления внешних сервисов об изменениях заметок
  - Триггеры: create, update, remove — настраиваемые per webhook
  - Include/exclude glob-паттерны для фильтрации файлов
  - HMAC-SHA256 подпись payload
  - Short API Token (JWT) для авторизации агентов с depth-based recursion protection
  - Agent response: агент может вернуть изменения файлов в ответе
  - Admin UI: полный CRUD + история доставок

- **Cron Webhooks**: вызов внешних агентов по расписанию (cron expression)
  - Инструкция + API token в каждом вызове
  - Поддержка sync и async ответов
  - Admin UI: CRUD + история доставок

- **Debug endpoints**: `/debug/test_webhook` для тестирования webhook delivery

## 2026-02-02

- **RSS**: любая заметка доступна как RSS-лента по `*.rss.xml`
  - Каждая ссылка в заметке → RSS item
  - Внутренние ссылки обогащаются метаданными (description, дата)
  - Настройки: `rss_title`, `rss_description` в frontmatter
  - Конфиг: `enable_rss` (вкл/выкл через админку)

- **Sitemap**: автоматическая генерация `/sitemap.xml`
  - Включаются только бесплатные страницы (`free: true`)
  - Обновляется автоматически при загрузке заметок

- **Дата публикации из frontmatter**: поля `created_at`/`created_on`
  - Переопределяет дату из базы данных
  - Используется в RSS pubDate и sitemap lastmod

## 2026-01-28

- **Шаблон заголовка**: настройка `site_title_template` теперь применяется ко всем страницам
  - В стандартном layout используется отформатированный заголовок
  - В кастомных layout доступна переменная `{{ title }}`
  - Пример шаблона: `%s | Мой Сайт` → `Название страницы | Мой Сайт`

- **Админка**: удалён устаревший интерфейс Config Versions
  - Используйте новый раздел Config для управления настройками

## 2026-01-20

- **Onboarding**: страница онбординга для пустого сайта
  - Гости видят сообщение "Сайт в процессе настройки"
  - Админы видят кнопку скачивания стартового архива
  - Архив содержит настроенный плагин с API-ключом
  - Имя архива формируется из домена (например `trip2g-vault.zip`)
  - Удалена страница онбординга из админки

- **GitHub OAuth**: исправлена ошибка "small read buffer" при валидации credentials
  - fasthttp требует явно указывать `ReadBufferSize` для API с большими заголовками (GitHub CSP)
