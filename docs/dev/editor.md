# Редактор файлов

> **Примечание**: Веб-редактор markdown/HTML для редактирования контента сайта без Obsidian. Интегрирован с $mol фреймворком и системой синхронизации vault.

## Обзор

Веб-редактор markdown/HTML файлов, открывается как модальное диалоговое окно на любой странице фронтенда. Построен на $mol фреймворке.

**Два режима использования:**
1. **Плавающая кнопка на странице** — быстрое редактирование текущей страницы
2. **Полноценный редактор `/admin/editor`** — навигация по файлам, создание/удаление

## Возможности

### Текущая реализация (MVP)

- Редактор markdown на базе текстового поля
- Открывается как модальное окно через кнопку в хедере
- Состояние сохраняется в URL через `$mol_state_arg` (ключ: "editor")
- Двухколоночный layout: список файлов (слева) и редактор (по центру)

### Планируется

#### Полная версия
- 3-колоночный layout: список файлов | редактор | превью
- Все колонки можно скрывать/показывать
- Изменяемая ширина колонок (drag-handle между колонками)
- Поддержка редактирования HTML файлов
- Автоопределение текущей страницы и открытие соответствующего файла

#### Milkdown интеграция
- Milkdown bundled как IIFE через esbuild (паттерн из obsidian-sync/esbuild.browser.mjs)
- Bundle размещается в `assets/ui/editor/milkdown.js`
- Загрузка через `require()` в mol namespace
- Кастомный `$mol_view` оборачивает Milkdown и монтирует его на `dom_node()`

#### CRUD операции
- Создание файлов
- Переименование файлов
- Удаление файлов
- Сохранение на сервер через GraphQL mutation

## Структура файлов

```
assets/ui/editor/
  editor.view.tree     — кнопка-переключатель + модальное окно
  editor.view.ts       — управление состоянием диалога через $mol_state_arg
  editor.view.css.ts   — стили диалога
  pane/
    pane.view.tree     — layout с колонками (sidebar, editor, preview)
    panel.view.css.ts  — стили колонок, resize handles
  milkdown/            (планируется)
    esbuild.browser.mjs — конфиг esbuild для IIFE bundle
    milkdown.js         — скомпилированный IIFE bundle
    milkdown.ts         — mol интеграция (require + export в namespace)
```

## Технические детали

### Модальное окно (dialog)

Используется нативный HTML `<dialog>` элемент:

**Структура** (`editor.view.tree`):
```tree
$trip2g_editor $mol_view
	sub /
		<= Open $mol_check_icon
			Icon <= Open_icon $mol_icon_application_edit
			hint @ \Open Editor
			checked? <=> opened? false
		<= Dialog $mol_view
			dom_name \dialog
			attr * open <= open_status null null|string
			event *
				click?event <=> close_click?event null
				close?event <=> close_event?event null
			sub /
				<= Pane $trip2g_editor_pane
				<= CloseButton $mol_button_major
```

**Логика** (`editor.view.ts`):
- `open_status()` — управляет состоянием через `$mol_state_arg.value('editor')`
- `dialog_dom().showModal()` / `.close()` — нативные методы `<dialog>`
- При клике вне диалога — автоматическое закрытие (проверка координат)
- Состояние открытия хранится в URL параметре `?editor=open`

### Layout колонок

**Структура** (`pane/pane.view.tree`):
```tree
$trip2g_editor_pane $mol_view
	content? \
	sub /
		<= Sidebar $mol_list
			rows /
				<= Sidebar_head $mol_view
					sub / <= sidebar_title @ \Files
		<= Body $mol_view
			sub /
				<= Editor $mol_textarea
					hint @ \Start typing...
					value? <=> content?
```

**Стили** (`pane/panel.view.css.ts`):
- Flexbox layout с `flex-direction: row`
- Sidebar: фиксированная ширина `16rem`, border справа
- Body: растягивается на оставшееся место (`flex-grow: 1`)
- Editor: занимает всю высоту родителя

### Сохранение состояния

Используется `$mol_state_arg` для синхронизации с URL:

```typescript
// Открыть редактор
this.$.$mol_state_arg.value('editor', 'open')

// Закрыть редактор
this.$.$mol_state_arg.value('editor', null)

// Проверить состояние
const isOpen = this.$.$mol_state_arg.value('editor') === 'open'
```

При изменении URL параметра автоматически вызывается `showModal()` или `close()`.

## Интеграция с Backend

### Существующие эндпоинты

Для работы с файлами vault используются эндпоинты из obsidian-sync:

| Query/Mutation | Что делает | Файл |
|----------------|------------|------|
| `notePaths(filter)` | Получить содержимое файлов | `obsidian-sync/src/operations.graphql` |
| `pushNotes` | Сохранить файлы | `obsidian-sync/src/operations.graphql` |
| `commitNotes` | Закоммитить изменения | `obsidian-sync/src/operations.graphql` |

**Авторизация**: Эндпоинты поддерживают:
- `X-Api-Key` header для Obsidian плагина
- Admin cookie для веб-интерфейса (через `ResolveAPIKeyOrAdmin`)

### Preview markdown

```graphql
type AdminQuery {
  renderNotePreview(content: String!): RenderNotePreviewPayload!
}

type RenderNotePreviewPayload {
  html: String!
  warnings: [String!]!
}
```

Использует `mdloader` для рендера markdown → HTML с warnings (broken links и т.д.).

### Определение текущей страницы

Layout рендерит meta-тег с путем к файлу:

```html
<meta name="trip2g:path" content="about.md">
```

Frontend читает этот тег для автоматического открытия соответствующего файла в редакторе.

## UI Flow

### Режим 1: Плавающая кнопка

1. Админ заходит на любую страницу сайта
2. Видит плавающую кнопку "✏️" (или "Edit") в углу
3. Клик → открывается модалка с редактором текущей страницы
4. Редактирует markdown, видит live preview
5. Save → страница обновляется
6. Cancel → закрывает без сохранения

### Режим 2: Полноценный редактор

1. Админ открывает `/admin/editor`
2. Видит список файлов слева, редактор по центру, preview справа
3. Выбирает файл из списка → загружается в редактор
4. Редактирует, сохраняет
5. Может создавать/переименовывать/удалять файлы

## Roadmap

### Ближайшие шаги

1. **Список файлов**
   - GraphQL query для получения списка файлов
   - Рендеринг дерева файлов в Sidebar
   - Навигация по файлам (клик открывает файл)

2. **Загрузка/сохранение**
   - GraphQL query для чтения содержимого файла
   - GraphQL mutation для сохранения изменений
   - Индикация несохраненных изменений

3. **Milkdown интеграция**
   - Bundle Milkdown как IIFE через esbuild
   - Интеграция в mol namespace через `require()`
   - Обертка `$mol_view` для монтирования редактора

4. **Превью**
   - Третья колонка для preview
   - Live preview для markdown
   - Синхронизация скролла между редактором и превью

### Будущие улучшения

- Автоопределение текущей страницы из URL
- Drag-handles для изменения ширины колонок
- Кнопки показа/скрытия колонок
- Поддержка HTML редактирования
- Создание/переименование/удаление файлов
- Синтаксическая подсветка кода
- Поиск и замена
- История изменений (undo/redo)
- Ctrl+S для сохранения
- Подтверждение при закрытии с несохраненными изменениями

## Версии файлов

Каждое сохранение файла создаёт новую версию (`note_version`). Редактор может показывать историю версий:

- Список версий файла с датой и автором
- Просмотр содержимого конкретной версии
- Diff между версиями
- Откат к предыдущей версии

Версии уже хранятся в БД через систему синхронизации — нужно только UI.

## Wikilinks

Поддержка `[[wikilinks]]` через `remark-wiki-link`:

- Парсинг `[[page]]` и `[[page|alias]]` синтаксиса
- Ссылки на другие страницы vault
- Визуальное отличие существующих и несуществующих страниц
- Автодополнение при вводе `[[`

Подключен в бандле Milkdown через `$remark` утилиту.

## Связанные компоненты

- `$mol_state_arg` — синхронизация состояния с URL
- `$mol_textarea` — текущий редактор (временно, заменится на Milkdown)
- `$mol_list` — список файлов в sidebar
- `$mol_view` — базовый компонент для layout

## Примеры использования

### Открыть редактор программно

```typescript
// Установить URL параметр
this.$.$mol_state_arg.value('editor', 'open')
```

### Закрыть редактор

```typescript
// Удалить URL параметр
this.$.$mol_state_arg.value('editor', null)
```

### Получить содержимое редактора

```typescript
// Через property binding
const content = this.content()
```

## Важные заметки

### Синхронизация с Obsidian

Если админ редактирует файл в браузере, а затем изменяет его в Obsidian — при следующем push Obsidian перезапишет веб-изменения. Это документированное поведение, нужно предупреждать пользователей.

### Конфликты редактирования

Если два админа редактируют одну страницу одновременно:
- **MVP**: последний сохраненный выигрывает (last-write-wins)
- **Будущее**: optimistic locking с предупреждением о конфликте

## См. также

- [frontend.md](frontend.md) — общая документация по фронтенду
- [mol.md](mol.md) — документация по $mol фреймворку
- [Milkdown документация](https://milkdown.dev/)
- [obsidian-sync](../obsidian-sync/) — система синхронизации vault
