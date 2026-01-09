# JSON Layouts Format

JSON-формат для описания страничных layout-ов. Используется визуальным редактором на фронтенде для drag-and-drop сборки страниц.

## Обзор

Файлы `*.html.json` хранятся в репозитории и конвертируются в Jet-шаблоны на лету при загрузке в `layoutloader`. HTML-файлы не генерируются на диск.

```
.html.json (source) → layoutloader → Jet template (in memory) → HTML output
```

## Структура файла

```json
{
  "meta": {},
  "body": []
}
```

| Поле | Тип | Описание |
|------|-----|----------|
| `meta` | object | Конфигурация layout-а (аналог YAML frontmatter). Зарезервировано для будущего использования |
| `body` | array | Массив блоков, составляющих layout |

## Типы блоков

### `block` - Вызов блока

Вызывает именованный блок с параметрами. Блоки определяются отдельно (в blocks.html или другом layout-файле).

```json
{
  "type": "block",
  "name": "cta_section",
  "args": {
    "title": "Готовы начать?",
    "subtitle": "Напишите нам"
  }
}
```

Результат Jet:
```jet
{{ yield cta_section(title="Готовы начать?", subtitle="Напишите нам") }}
```

**С вложенным контентом:**
```json
{
  "type": "block",
  "name": "card",
  "args": { "title": "Заголовок" },
  "content": [
    { "type": "html", "content": "<p>Текст карточки</p>" }
  ]
}
```

Результат Jet:
```jet
{{ yield card(title="Заголовок") content }}
  <p>Текст карточки</p>
{{ end }}
```

> **Примечание**: Блоки определяются в отдельных .html файлах через `{{ block name(param="default") }}...{{ end }}`. JSON-формат используется только для вызова блоков, не для их определения.

### `if` - Условный блок

Условный рендеринг содержимого.

```json
{
  "type": "if",
  "condition": "note.M().GetBool(\"show_block\")",
  "content": [
    { "type": "note_content", "path": "_block.md" }
  ]
}
```

Результат Jet:
```jet
{{ if note.M().GetBool("show_block") }}
  {{ /* содержимое content */ }}
{{ end }}
```

### `note_content` - Контент заметки

Вставляет содержимое текущей заметки или подключает другую заметку по пути.

**Текущая заметка:**
```json
{ "type": "note_content" }
```

Результат Jet:
```jet
{{ note.HTMLString() | unsafe }}
```

**Другая заметка по пути:**
```json
{ "type": "note_content", "path": "_sidebar.md" }
```

Результат Jet:
```jet
{{ _sidebar := nvs.ByPath("_sidebar.md") }}
{{ if _sidebar }}
  {{ _sidebar.HTMLString() | unsafe }}
{{ end }}
```

| Поле | Тип | Описание |
|------|-----|----------|
| `path` | string? | Путь к заметке для включения. Если не указан - контент текущей заметки |

### `asset` - Ссылка на ассет

Генерирует URL ассета (CSS, JS, изображения).

```json
{
  "type": "asset",
  "path": "style.css"
}
```

Результат Jet:
```jet
{{ asset("style.css") }}
```

### `html` - Сырой HTML

Вставляет HTML-разметку как есть.

```json
{
  "type": "html",
  "content": "<div class=\"container\">"
}
```

### `import` - Импорт блоков

Импортирует файл с определениями блоков.

```json
{
  "type": "import",
  "name": "blocks"
}
```

Результат Jet:
```jet
{{ import "blocks" }}
```

> Импортированные блоки становятся доступны для вызова через `block`.

### `range` - Цикл

Итерация по коллекции.

```json
{
  "type": "range",
  "iterator": "i, post",
  "collection": "nvs.ByGlob(\"blog/*.md\").SortBy(\"CreatedAt\").All()",
  "content": [
    { "type": "html", "content": "<li>" },
    {
      "type": "expr",
      "expr": "post.Title()"
    },
    { "type": "html", "content": "</li>" }
  ]
}
```

Результат Jet:
```jet
{{ range i, post := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").All() }}
<li>{{ post.Title() }}</li>
{{ end }}
```

### `expr` - Выражение

Вывод значения выражения.

```json
{
  "type": "expr",
  "expr": "post.Title()"
}
```

Результат Jet:
```jet
{{ post.Title() }}
```

## Полный пример

### JSON

```json
{
  "meta": {},
  "body": [
    {
      "type": "block",
      "name": "header",
      "args": {
        "level": 2
      }
    },
    {
      "type": "if",
      "condition": "note.M().GetBool(\"show_block\")",
      "content": [
        {
          "type": "note_content",
          "path": "_block.md"
        }
      ]
    },
    {
      "type": "note_content"
    }
  ]
}
```

### Эквивалент Jet

```jet
{{ yield header(level=2) }}
{{ if note.M().GetBool("show_block") }}
  {{ _block := nvs.ByPath("_block.md") }}
  {{ if _block }}
    {{ _block.HTMLString() | unsafe }}
  {{ end }}
{{ end }}
{{ note.HTMLString() | unsafe }}
```

## Реализация

### Архитектура

```
pushNotes (Obsidian) → noteloader → layoutloader → Jet template
     ↓                     ↓              ↓
 .html.json          case ".html.json"   ConvertJSONLayout()
```

### Файлы

| Файл | Назначение |
|------|------------|
| `internal/layoutloader/json_layout.go` | Конвертер JSON → Jet |
| `internal/layoutloader/json_layout_test.go` | Тесты конвертера |
| `internal/noteloader/loader.go` | Обработка `.html.json` расширения |

### Статус реализации

✅ **Готово:**
- Конвертер `json_layout.go` с полной поддержкой всех типов блоков
- Обработка ошибок с путями (`body[2].content[0]`) для отладки
- Интеграция в `noteloader` - файлы `.html.json` конвертируются на лету
- Полное покрытие тестами (28 тестов)

### Дальнейшие улучшения

**Визуальный редактор (frontend):**
- [ ] Компонент редактора с drag-and-drop
- [ ] Property panel для редактирования `args`
- [ ] Превью сгенерированного HTML
- [ ] Валидация JSON в реальном времени

**Расширения формата:**
- [ ] `else` для блоков `if`
- [ ] `else` для блоков `range` (пустая коллекция)
- [ ] Вложенные переменные в `expr` (например `{{ set var = value }}`)

**Интеграция:**
- [ ] GraphQL мутация для сохранения JSON layouts
- [ ] Автодополнение имён блоков из реестра блоков (`LayoutBlocks`)

### Порядок генерации Jet

| JSON type | Jet output |
|-----------|------------|
| `block` (без content) | `{{ yield name(args...) }}` |
| `block` (с content) | `{{ yield name(args...) content }}...{{ end }}` |
| `if` | `{{ if condition }}...{{ end }}` |
| `range` | `{{ range iterator := collection }}...{{ end }}` |
| `expr` | `{{ expr }}` |
| `html` | content as-is |
| `asset` | `{{ asset("path") }}` |
| `note_content` (без path) | `{{ note.HTMLString() \| unsafe }}` |
| `note_content` (с path) | `{{ _var := nvs.ByPath("path") }}{{ if _var }}{{ _var.HTMLString() \| unsafe }}{{ end }}` |
| `import` | `{{ import "name" }}` |

## Визуальный редактор

Формат оптимизирован для визуального редактора:

- Каждый блок - отдельный перетаскиваемый элемент
- `content` массивы позволяют вложенность (drag into)
- `args` объекты редактируются через property panel
- `condition`/`collection` - текстовые поля для выражений
