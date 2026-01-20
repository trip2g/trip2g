# Current Tasks

<!--
Активные задачи. Максимум 2-3.
Формат см. в CLAUDE.md
-->

## [IN PROGRESS] Onboarding страница для пустого сайта

### Контекст
Показывать страницу онбординга когда на сайте нет ни одной заметки. Для гостей — предложение войти. Для админов — ссылка на скачивание стартового архива.

### План
- [ ] Добавить поле `OnboardingMode` в Response (rendernotepage)
- [ ] Добавить проверку `notes.Size() == 0` в Resolve()
- [ ] Создать шаблон онбординга в view.html
- [ ] Добавить переводы (ru/en)
- [ ] Протестировать оба сценария (гость/админ)

### Заметки
- Документация: [docs/onboarding.md](onboarding.md)
- Существующий компонент: `assets/ui/admin/onboardingvault/`
- Endpoint архива: `/_system/onboarding-vault`
