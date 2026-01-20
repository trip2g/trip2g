# Current Tasks

<!--
Активные задачи. Максимум 2-3.
Формат см. в CLAUDE.md
-->

## [IN PROGRESS] Onboarding страница для пустого сайта

### Контекст
Показывать страницу онбординга когда на сайте нет ни одной заметки. Для гостей — предложение войти. Для админов — ссылка на скачивание стартового архива. Заменит текущую страницу в админке.

### План
- [ ] Проверить что `/_system/onboarding-vault` требует авторизации админа ← следующий
- [ ] Добавить поле `OnboardingMode bool` в Response (rendernotepage)
- [ ] Добавить проверку `LatestNoteViews().Size() == 0` в Resolve() (после проверки /_system)
- [ ] Создать шаблон онбординга в view.html:
  - Текст взять из `assets/ui/admin/onboardingvault/`
  - Для гостя: ссылка на `/admin` для входа
  - Для админа: ссылка `/_system/onboarding-vault`
  - `<meta name="robots" content="noindex">`
  - `Cache-Control: no-store`
- [ ] Добавить переводы (ru/en) — или показывать оба языка
- [ ] Добавить e2e тест в начало setup.spec.js (до загрузки данных)
- [ ] Удалить страницу онбординга из админки

### Заметки
- Документация: [docs/onboarding.md](onboarding.md)
- Текст для копирования: `assets/ui/admin/onboardingvault/onboardingvault.view.tree`
- Endpoint архива: `/_system/onboarding-vault`
- E2E тесты: `e2e/` (см. `e2e/setup.spec.js` для примера авторизации)
- Тест онбординга: добавить в начало setup.spec.js до загрузки данных
