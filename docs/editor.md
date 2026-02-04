# Редактор файлов в админке

## Цель

Возможность редактировать страницы без Obsidian. Админ видит плавающую кнопку "Редактировать" на любой странице.

## Концепция

```
┌─────────────────────────────────────────┐
│                                         │
│         Обычная страница                │
│                                         │
│    # About                              │
│                                         │
│    Some content here...                 │
│                                         │
│                                    [✏️] │  ← плавающая кнопка
│                                         │
└─────────────────────────────────────────┘
```

Клик на кнопку открывает редактор текущей страницы:

```
┌─────────────────────────┬─────────────────────────┐
│        Editor           │        Preview          │
│                         │                         │
│ ---                     │ ┌─────────────────────┐ │
│ title: About            │ │                     │ │
│ ---                     │ │   About             │ │
│                         │ │                     │ │
│ # About                 │ │   Some content...   │ │
│                         │ │                     │ │
│ Some content...         │ │                     │ │
│                         │ └─────────────────────┘ │
│                         │                         │
│    [Save]   [Cancel]    │                         │
└─────────────────────────┴─────────────────────────┘
```

## Scope

**В scope:**
- Плавающая кнопка "Редактировать" для админов
- Редактор текущей страницы (2 колонки: editor + preview)
- Сохранение изменений

**Вне scope (MVP):**
- Файловый навигатор (не нужен — редактируем текущую страницу)
- Загрузка ассетов
- Создание/удаление файлов

## UI Flow

1. Админ заходит на любую страницу сайта
2. Видит плавающую кнопку "✏️" (или "Edit") в углу
3. Клик → открывается модалка/overlay с редактором
4. Редактирует markdown
5. Save → страница обновляется
6. Cancel → закрывает без сохранения

## Реализация

### Скрипт для страниц

Подключается в layout для админов:

```html
<!-- Только для залогиненных админов -->
<script src="/_system/admin-edit.js"></script>
```

Скрипт:
1. Рендерит плавающую кнопку
2. При клике — открывает редактор
3. Загружает содержимое текущей страницы через GraphQL
4. Отправляет изменения через GraphQL mutation

### Существующие эндпоинты

Уже есть всё что нужно в [[operations.graphql|obsidian-sync/src/operations.graphql]]:

| Query/Mutation | Что делает |
|----------------|------------|
| `notePaths(filter)` | Получить содержимое файлов |
| `pushNotes` | Сохранить файлы |
| `commitNotes` | Закоммитить изменения |

**Проблема:** Эти эндпоинты требуют `X-Api-Key` header. Админ в браузере авторизован через cookie.

### Backend: API key ИЛИ admin auth

Добавить helper `ResolveAPIKeyOrAdmin` в [[checkapikey|internal/case/checkapikey/]]:

```go
func ResolveAPIKeyOrAdmin(ctx context.Context, env Env, action string) (*db.ApiKey, error) {
    // Попробовать API key
    apiKey, err := Resolve(ctx, env, action)
    if err == nil {
        return apiKey, nil
    }

    // Если нет API key — проверить admin cookie
    adminToken, err := env.CurrentAdminUserToken(ctx)
    if err != nil {
        return nil, errors.New("unauthorized: API key or admin auth required")
    }

    // Вернуть "виртуальный" API key для админа
    return &db.ApiKey{CreatedBy: adminToken.ID}, nil
}
```

Использовать в `PushNotes`, `NotePaths`, `CommitNotes` вместо `checkapikey.Resolve`.

### Backend: превью

Аналог [[renderlayoutpreview]] для заметок:

```graphql
type AdminQuery {
  renderNotePreview(content: String!): RenderNotePreviewPayload!
}

type RenderNotePreviewPayload {
  html: String!
  warnings: [String!]!
}
```

Реализация в `internal/case/admin/rendernotepreview/`:
- Использует `mdloader` для рендера markdown → HTML
- Возвращает warnings если есть (broken links и т.д.)

### Frontend: admin-edit.js

```
assets/ui/admin/
├── edit_button/
│   └── edit_button.view.tree    # плавающая кнопка
└── editor/
    └── editor.view.tree         # модалка с редактором
```

Или один bundle:
```
assets/ui/admin-edit.js          # самодостаточный скрипт
```

### Как определить текущую страницу

Варианты:
1. **Meta tag**: `<meta name="trip2g:path" content="about.md">`
2. **Data attribute**: `<body data-path="about.md">`
3. **По URL**: `/about` → ищем в API

**Решение**: Meta tag — layout рендерит `<meta name="trip2g:path" content="{{ path }}">`.

## Альтернатива: страница /admin/editor

Можно также сделать полноценную страницу `/admin/editor` с файловым навигатором для случаев когда нужно:
- Редактировать файл который не опубликован
- Смотреть все файлы
- Создавать новые файлы

Это можно добавить позже как отдельную фичу.

## План выполнения

### Этап 1: Backend
- [ ] `checkapikey.ResolveAPIKeyOrAdmin` — проверка API key ИЛИ admin cookie
- [ ] Использовать в `PushNotes`, `NotePaths`, `CommitNotes`
- [ ] GraphQL: `admin.renderNotePreview(content)` — превью markdown
- [ ] `internal/case/admin/rendernotepreview/` — рендер через mdloader
- [ ] Тесты backend

### Этап 2: Frontend
- [ ] Meta tag `trip2g:path` в layout (путь к файлу)
- [ ] Условие для показа скрипта только админам
- [ ] `admin-edit.js` — плавающая кнопка
- [ ] Модалка с редактором (textarea + preview)
- [ ] GraphQL: `notePaths` для загрузки, `pushNotes` + `commitNotes` для сохранения
- [ ] GraphQL: `admin.renderNotePreview` для превью

### Этап 3: UX улучшения
- [ ] Ctrl+S для сохранения
- [ ] Индикатор несохранённых изменений
- [ ] Подтверждение при закрытии с изменениями
- [ ] E2E тесты

## Вопросы

1. **Видимость кнопки**: Кнопка "Редактировать" видна только если пользователь — админ
   - Сервер рендерит скрипт только для админов, ИЛИ
   - Скрипт проверяет наличие админского токена

2. **Синхронизация с Obsidian**: Что если пользователь редактирует и в браузере, и в Obsidian?
   - Obsidian sync перезапишет при следующем push
   - Документируем это поведение

3. **Конфликты**: Два админа редактируют одну страницу?
   - MVP: последний выигрывает
   - Later: optimistic locking
