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
  "condition": "note.M().GetBool(\"show_block\", false)",
  "content": [
    { "type": "note_content", "path": "_block.md" }
  ]
}
```

Результат Jet:
```jet
{{ if note.M().GetBool("show_block", false) }}
  {{ /* содержимое content */ }}
{{ end }}
```

### `note_content` - Контент заметки

Вставляет содержимое текущей заметки или другой заметки по пути.

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
{{ nvs.ByPath("_sidebar.md").HTMLString() | unsafe }}
```

| Поле | Тип | Описание |
|------|-----|----------|
| `path` | string? | Путь к заметке. Если не указан - контент текущей заметки |

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

### `include_note` - Включение заметки с fallback

Вставляет содержимое заметки по пути. Если заметка не найдена, показывает сообщение "Create file: путь".

```json
{
  "type": "include_note",
  "path": "/_sidebar.md"
}
```

Результат Jet:
```jet
{{ _note0 := nvs.ByPath("/_sidebar.md") }}
{{ if _note0 }}
  {{ _note0.HTMLString() | unsafe }}
{{ else }}
  Create file: /_sidebar.md
{{ end }}
```

| Поле | Тип | Описание |
|------|-----|----------|
| `path` | string | Путь к заметке (обязательно) |

> **Примечание**: Если файл не найден, показывается сообщение "Create file: путь" для удобства отладки в визуальном редакторе.

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
      "condition": "note.M().GetBool(\"show_block\", false)",
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
{{ if note.M().GetBool("show_block", false) }}
  {{ nvs.ByPath("_block.md").HTMLString() | unsafe }}
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
| `internal/layoutloader/loader.go` | Загрузка layouts, парсинг блоков и `arg_type` |
| `internal/layoutloader/loader_test.go` | Тесты парсера блоков |
| `internal/noteloader/loader.go` | Обработка `.html.json` расширения |
| `internal/model/layout.go` | Модели `LayoutBlock`, `LayoutBlockParam`, `LayoutBlocks` |
| `internal/graph/schema.graphqls` | GraphQL схема для `layoutBlocks` |
| `internal/graph/schema.resolvers.go` | Резолверы для GraphQL API |

### Статус реализации

✅ **Готово:**
- Конвертер `json_layout.go` с полной поддержкой всех типов блоков
- Обработка ошибок с путями (`body[2].content[0]`) для отладки
- Интеграция в `noteloader` - файлы `.html.json` конвертируются на лету
- Полное покрытие тестами (28 тестов)
- GraphQL query `admin.layoutBlocks` для автодополнения блоков
- Парсинг `arg_type` директивы для метаданных параметров
- Type inference из default values

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
| `note_content` (с path) | `{{ nvs.ByPath("path").HTMLString() \| unsafe }}` |
| `include_note` | `{{ _var := nvs.ByPath("path") }}{{ if _var }}{{ _var.HTMLString() \| unsafe }}{{ else }}Create file: path{{ end }}` |
| `import` | `{{ import "name" }}` |

## Метаданные параметров блоков (arg_type)

Для визуального редактора нужна информация о типах и описаниях параметров блоков. Jet-шаблоны не имеют статической типизации, поэтому используется директива `arg_type`:

```jet
{{ block card(title, subtitle, level=1, featured=false) }}
  {{ arg_type("title", "string", "Заголовок карточки") }}
  {{ arg_type("subtitle", "string", "Подзаголовок") }}
  {{ arg_type("level", "int", "Уровень заголовка (1-6)") }}
  {{ arg_type("featured", "bool", "Выделить карточку") }}

  <div class="{{ if featured }}featured{{ end }}">
    <h{{ level }}>{{ title }}</h{{ level }}>
    <p>{{ subtitle }}</p>
  </div>
{{ end }}
```

### Синтаксис

```jet
{{ arg_type("имя_параметра", "тип", "описание") }}
```

| Аргумент | Обязательный | Описание |
|----------|--------------|----------|
| имя_параметра | да | Должно совпадать с именем в сигнатуре блока |
| тип | да | `string`, `int`, `float`, `bool` |
| описание | нет | Человекочитаемое описание для UI |

### Поведение

- **Runtime**: функция возвращает пустую строку, не влияет на рендеринг
- **Parse time**: `layoutloader` извлекает метаданные и добавляет в `LayoutBlockParam`

### Определение типа (fallback)

Если `arg_type` не указан для параметра:

1. Если есть default value — тип определяется автоматически из AST:
   - `"text"` → `string`
   - `42` → `int`
   - `3.14` → `float`
   - `true`/`false` → `bool`
2. Если нет default value — тип остаётся пустым, UI использует text input

### GraphQL API

```graphql
type AdminQuery {
  layoutBlocks: [LayoutBlock!]!
}

type LayoutBlock {
  name: String!           # "card"
  fullName: String!       # "blocks.html#card" — уникальный идентификатор
  sourceId: String!       # "blocks.html"
  hasContent: Boolean!    # true если блок использует {{ yield content }}
  params: [LayoutBlockParam!]!
}

type LayoutBlockParam {
  name: String!                   # "title"
  value: LayoutBlockParamValue    # типизированное значение или null
  comment: String                 # "Заголовок карточки"
}

# Union для типизированных значений параметров
union LayoutBlockParamValue = StringParamValue | IntParamValue | FloatParamValue | BoolParamValue

type StringParamValue {
  defaultValue: String
}

type IntParamValue {
  defaultValue: Int
}

type FloatParamValue {
  defaultValue: Float
}

type BoolParamValue {
  defaultValue: Boolean
}
```

### Пример запроса

```graphql
query {
  admin {
    layoutBlocks {
      name
      fullName
      sourceId
      hasContent
      params {
        name
        comment
        value {
          __typename
          ... on StringParamValue { defaultValue }
          ... on IntParamValue { defaultValue }
          ... on FloatParamValue { defaultValue }
          ... on BoolParamValue { defaultValue }
        }
      }
    }
  }
}
```

### Пример ответа API

```json
{
  "data": {
    "admin": {
      "layoutBlocks": [
        {
          "name": "card",
          "fullName": "blocks.html#card",
          "sourceId": "blocks.html",
          "hasContent": true,
          "params": [
            {
              "name": "title",
              "comment": "Заголовок карточки",
              "value": { "__typename": "StringParamValue", "defaultValue": null }
            },
            {
              "name": "level",
              "comment": "Уровень заголовка (1-6)",
              "value": { "__typename": "IntParamValue", "defaultValue": 1 }
            },
            {
              "name": "featured",
              "comment": "Выделить карточку",
              "value": { "__typename": "BoolParamValue", "defaultValue": false }
            }
          ]
        }
      ]
    }
  }
}
```

### Идентификация блоков

Блоки идентифицируются двумя способами:

| Поле | Формат | Пример | Использование |
|------|--------|--------|---------------|
| `name` | короткое имя | `"card"` | Для отображения в UI |
| `fullName` | `sourceId#name` | `"blocks.html#card"` | Уникальный ключ |

При наличии блоков с одинаковыми именами в разных файлах используй `fullName` для disambiguation.

## Визуальный редактор

Формат оптимизирован для визуального редактора:

- Каждый блок - отдельный перетаскиваемый элемент
- `content` массивы позволяют вложенность (drag into)
- `args` объекты редактируются через property panel
- `condition`/`collection` - текстовые поля для выражений

### Property Panel UI

На основе `LayoutBlockParam.value.__typename` редактор строит форму:

| `__typename` | UI компонент | Default value |
|--------------|--------------|---------------|
| `StringParamValue` | Text input | `defaultValue` или пустая строка |
| `IntParamValue` | Number input (целые) | `defaultValue` или 0 |
| `FloatParamValue` | Number input (дробные) | `defaultValue` или 0.0 |
| `BoolParamValue` | Checkbox / toggle | `defaultValue` или false |
| `null` (value отсутствует) | Text input (fallback) | пустая строка |

`comment` отображается как tooltip или hint под полем.

### Пример кода для фронтенда

```typescript
function renderParamInput(param: LayoutBlockParam) {
  const { name, comment, value } = param;

  if (!value) {
    // Тип неизвестен — fallback на текстовое поле
    return <TextInput name={name} hint={comment} />;
  }

  switch (value.__typename) {
    case 'StringParamValue':
      return <TextInput name={name} defaultValue={value.defaultValue} hint={comment} />;
    case 'IntParamValue':
      return <NumberInput name={name} defaultValue={value.defaultValue} step={1} hint={comment} />;
    case 'FloatParamValue':
      return <NumberInput name={name} defaultValue={value.defaultValue} step={0.1} hint={comment} />;
    case 'BoolParamValue':
      return <Checkbox name={name} defaultChecked={value.defaultValue} hint={comment} />;
  }
}
```

## Лучшие практики

### Определение блоков

1. **Всегда указывай default values** — это позволяет автоматически определить тип:
   ```jet
   {{ block card(title="", level=1, featured=false) }}
   ```

2. **Используй `arg_type` для параметров без defaults**:
   ```jet
   {{ block hero(title, subtitle) }}
     {{ arg_type("title", "string", "Главный заголовок") }}
     {{ arg_type("subtitle", "string", "Подзаголовок") }}
   ```

3. **Добавляй описания** — они отображаются в UI редактора:
   ```jet
   {{ arg_type("level", "int", "Уровень заголовка от 1 до 6") }}
   ```

### Организация блоков

Рекомендуемая структура файлов:

```
_layouts/
├── blocks.html          # Общие переиспользуемые блоки
├── components.html      # Специфичные компоненты
└── main.html           # Основной layout
```

При дублировании имён блоков используй fullName:
- `blocks.html#card` — карточка из blocks
- `components.html#card` — карточка из components

### Блоки с вложенным контентом

Для блоков, принимающих вложенный HTML, используй `{{ yield content }}`:

```jet
{{ block wrapper(class="") }}
  <div class="{{ class }}">
    {{ yield content }}
  </div>
{{ end }}
```

В JSON это будет:
```json
{
  "type": "block",
  "name": "wrapper",
  "args": { "class": "container" },
  "content": [
    { "type": "html", "content": "<p>Вложенный контент</p>" }
  ]
}
```

`hasContent: true` в API указывает, что блок поддерживает вложенность.

## Live Preview API

Для визуального редактора необходим real-time preview — возможность видеть результат рендеринга при редактировании layout-а без сохранения на диск.

### Endpoint

```
POST /_system/layouts/render
Content-Type: application/json
```

### Request

```json
{
  "note_path": "/about",
  "layout": {
    "meta": {},
    "body": [
      { "type": "block", "name": "header", "args": { "level": 1 } },
      { "type": "note_content" },
      { "type": "block", "name": "footer" }
    ]
  }
}
```

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| `note_path` | string | да | Путь к заметке для рендеринга (например `/about`, `blog/hello.md`) |
| `layout` | object | да | JSON layout в формате `{ meta, body }` |

### Response

**Успех (200 OK):**
```json
{
  "html": "<html>...rendered content...</html>"
}
```

**Ошибка валидации layout (400 Bad Request):**
```json
{
  "error": "invalid layout",
  "details": "body[2]: unknown block type 'invalid'"
}
```

**Заметка не найдена (404 Not Found):**
```json
{
  "error": "note not found",
  "path": "/nonexistent"
}
```

### Как это работает

```
┌─────────────────┐     POST      ┌─────────────────┐
│  Visual Editor  │ ────────────> │  /_system/      │
│  (frontend)     │               │  layouts/render │
└─────────────────┘               └────────┬────────┘
                                           │
                                           ▼
                              ┌────────────────────────┐
                              │ 1. Валидация JSON      │
                              │ 2. ConvertJSONLayout() │
                              │ 3. Компиляция Jet      │
                              │ 4. Загрузка note       │
                              │ 5. Рендеринг HTML      │
                              └────────────────────────┘
                                           │
                                           ▼
                              ┌────────────────────────┐
                              │  { "html": "..." }     │
                              └────────────────────────┘
```

1. **Валидация JSON** — проверка структуры `{ meta, body }`
2. **ConvertJSONLayout()** — конвертация в Jet template
3. **Компиляция Jet** — парсинг и компиляция шаблона (временный, не сохраняется)
4. **Загрузка note** — получение заметки по `note_path` из текущих `NoteViews`
5. **Рендеринг HTML** — выполнение шаблона с контекстом заметки

### Особенности

- **Временный layout** — не сохраняется, используется только для этого запроса
- **Блоки из текущих layouts** — `{{ yield block() }}` работает с уже загруженными блоками
- **Полный контекст** — доступны `note`, `nvs`, `asset()` и другие функции шаблона
- **Авторизация** — требуется admin-доступ (endpoint под `/_system/`)

### Пример использования в редакторе

```typescript
async function previewLayout(notePath: string, layout: JSONLayout): Promise<string> {
  const response = await fetch('/_system/layouts/render', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ note_path: notePath, layout })
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.details || error.error);
  }

  const { html } = await response.json();
  return html;
}

// Использование с debounce для live preview
const debouncedPreview = debounce(async (layout) => {
  const html = await previewLayout('/about', layout);
  document.getElementById('preview-iframe').srcdoc = html;
}, 300);

// При каждом изменении в редакторе
editor.onChange((newLayout) => {
  debouncedPreview(newLayout);
});
```

### Partial Render (для отдельных узлов)

Для UI дерева каждый узел может рендерить только себя. Добавляем опциональное поле `node_path`:

```json
{
  "note_path": "/about",
  "layout": { "meta": {}, "body": [...] },
  "node_path": "body[1]"
}
```

| Поле | Описание |
|------|----------|
| `node_path` | Путь к узлу в дереве (например `body[0]`, `body[2].content[1]`). Если указан — рендерится только этот узел. |

**Response при partial render:**
```json
{
  "html": "<div class='card'>...</div>"
}
```

Это позволяет каждой ячейке дерева независимо обновлять свой preview без перерендера всей страницы.

## Visual Editor: Tree UI

Визуальный редактор строится как дерево, где каждый узел JSON layout — это ячейка.

### Структура дерева

```
Layout
├── body[0]: block "header"        [config] [preview]
├── body[1]: if "show_sidebar"     [config] [preview]
│   └── content[0]: note_content   [config] [preview]
├── body[2]: block "card"          [config] [preview]
│   ├── args: { title, level }
│   └── content[0]: html           [config] [preview]
└── body[3]: note_content          [config] [preview]
```

### Режимы отображения ячейки

Каждая ячейка дерева имеет два режима:

| Режим | Описание | UI |
|-------|----------|-----|
| **Config** | JSON конфигурация узла | Property panel / JSON editor |
| **Preview** | Рендеренный HTML | iframe с результатом partial render |

### Пример компонента ячейки

```typescript
interface TreeNodeProps {
  node: LayoutNode;
  nodePath: string;      // "body[0]", "body[1].content[0]"
  notePath: string;      // "/about"
  fullLayout: JSONLayout;
}

function TreeNode({ node, nodePath, notePath, fullLayout }: TreeNodeProps) {
  const [mode, setMode] = useState<'config' | 'preview'>('config');
  const [previewHtml, setPreviewHtml] = useState<string>('');

  const loadPreview = async () => {
    const { html } = await fetch('/_system/layouts/render', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        note_path: notePath,
        layout: fullLayout,
        node_path: nodePath
      })
    }).then(r => r.json());

    setPreviewHtml(html);
  };

  useEffect(() => {
    if (mode === 'preview') loadPreview();
  }, [mode, node]);

  return (
    <div className="tree-node">
      <div className="node-header">
        <span className="node-type">{node.type}</span>
        {node.name && <span className="node-name">{node.name}</span>}
        <button onClick={() => setMode(mode === 'config' ? 'preview' : 'config')}>
          {mode === 'config' ? '👁 Preview' : '⚙ Config'}
        </button>
      </div>

      <div className="node-content">
        {mode === 'config' ? (
          <ConfigEditor node={node} onChange={...} />
        ) : (
          <iframe srcdoc={previewHtml} />
        )}
      </div>

      {/* Рекурсивный рендер вложенных узлов */}
      {node.content && (
        <div className="node-children">
          {node.content.map((child, i) => (
            <TreeNode
              key={i}
              node={child}
              nodePath={`${nodePath}.content[${i}]`}
              notePath={notePath}
              fullLayout={fullLayout}
            />
          ))}
        </div>
      )}
    </div>
  );
}
```

### Drag & Drop

Дерево поддерживает перетаскивание узлов:

- **Между siblings** — изменение порядка в `body` или `content`
- **В content** — перенос узла внутрь блока с `hasContent: true`
- **Из palette** — добавление нового блока из списка доступных

```typescript
function onDrop(draggedPath: string, targetPath: string, position: 'before' | 'after' | 'inside') {
  const newLayout = moveNode(layout, draggedPath, targetPath, position);
  setLayout(newLayout);
}
```

### Palette блоков

Боковая панель со списком доступных блоков (из `admin.layoutBlocks`):

```typescript
function BlockPalette({ blocks }: { blocks: LayoutBlock[] }) {
  return (
    <div className="palette">
      <h3>Блоки</h3>
      {blocks.map(block => (
        <div
          key={block.fullName}
          className="palette-item"
          draggable
          onDragStart={(e) => {
            e.dataTransfer.setData('block', JSON.stringify({
              type: 'block',
              name: block.fullName,
              args: defaultArgs(block.params)
            }));
          }}
        >
          <span>{block.name}</span>
          {block.hasContent && <span className="badge">+ content</span>}
        </div>
      ))}

      <h3>Примитивы</h3>
      <div className="palette-item" draggable>note_content</div>
      <div className="palette-item" draggable>html</div>
      <div className="palette-item" draggable>if</div>
      <div className="palette-item" draggable>range</div>
    </div>
  );
}
```

### Статус

- [ ] Реализация endpoint `/_system/layouts/render`
- [ ] Поддержка `node_path` для partial render
- [ ] Tree UI компонент на фронтенде
- [ ] Drag & drop между узлами
- [ ] Palette блоков с данными из GraphQL
