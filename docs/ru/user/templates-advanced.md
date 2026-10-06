---
free: true
wide: true
title: "Шаблоны: API"
---

Техническая документация для разработчиков шаблонов. Описывает API `internal/templateviews` — обёртки моделей для Jet-шаблонов.

### Архитектура

```
Jet-шаблон
    ↓
templateviews (Note, NVS, Meta, NoteQuery)
    ↓
model.NoteView, model.NoteViews
    ↓
База данных
```

`templateviews` изолирует шаблоны от внутренних изменений модели. Шаблоны работают со стабильным API.

---

## Note

Обёртка над `model.NoteView`. Представляет одну заметку в шаблоне.

### Методы

| Метод | Возвращает | Описание |
|-------|------------|----------|
| `Title()` | `string` | Заголовок из frontmatter |
| `HasH1()` | `bool` | Контент начинается с H1, который служит заголовком; свой `<h1>` можно не выводить |
| `HTMLString()` | `string` | Отрендеренный HTML контент |
| `ContentString()` | `string` | Сырой markdown |
| `PathID()` | `int64` | ID для data-атрибутов |
| `VersionID()` | `string` | ID текущей версии заметки |
| `Path()` | `string` | Путь файла в хранилище, например `blog/post.md` |
| `Permalink()` | `string` | URL страницы |
| `PermalinkEncoded()` | `string` | `Permalink()`, где каждый сегмент пути закодирован процентами: `/привет мир` → `/%D0%BF…%20%D0%BC…`. Это кодирование URL, не HTML-экранирование; `Permalink()` в `href` тоже безопасен. Нужен для строго закодированного адреса, когда в пути есть не-ASCII символы или пробелы |
| `CreatedAt()` | `time.Time` | Дата создания; если в frontmatter есть `created_at` / `created_on` — она |
| `UpdatedAt()` | `time.Time` | Из `updated_at`, `updated` или `modified`; нулевое время, если не задано (проверка — `.IsZero()`) |
| `Author()` | `string` | `author` из frontmatter, `""` если не задан |
| `Tags()` | `[]string` | Из `tags`, иначе из `keywords`; список или строка через запятую. `nil`, если не заданы |
| `ReadingTime()` | `int` | Время чтения в минутах |
| `ReadingComplexity()` | `int` | Сложность (0-2) |
| `IsHomePage()` | `bool` | Является ли домашней страницей подграфа |
| `IsSystem()` | `bool` | Какая-то часть пути начинается с `_` |
| `Description()` | `string` | SEO-описание |
| `OGImageURL()` | `string` | URL картинки из `og_image` (иначе `cover`), `""` если не найдена |
| `FirstImageURL()` | `string` | URL первой картинки в заметке, `""` если нет |
| `FirstListHTML()` | `string` | HTML первого `<ul>` в заметке, `""` если нет |
| `Lang()` | `string` | Нормализованный код языка (`ru`, `en`) |
| `LangName()` | `string` | Название языка на нём самом (`Русский`) |
| `HasLangAlternatives()` | `bool` | Есть версии на других языках |
| `LangAlternative("en")` | `*Note` | Версия на этом языке или nil |
| `LangAlternativesList()` | `[]*Note` | Все языковые версии, по коду языка |
| `HasCodeLanguage("mermaid")` | `bool` | Есть блок кода на этом языке — чтобы подключать скрипт виджета только там, где он нужен |
| `HasAnyCodeBlock()` | `bool` | Есть хотя бы один блок кода |
| `HasCharts()` | `bool` | Есть блоки datachart |
| `HasTaskListItems()` | `bool` | Есть чекбоксы задач |
| `FormSpecJSON()` | `string` | JSON формы, см. [[ru/user/forms\|Формы]] |
| `SubgraphNamesJSON()` | `string` | JSON-список подграфов заметки |
| `LastEditedBy()`, `LastEditedByLabel()` | объект / `string` | Кто запушил текущую версию. **Только для админа:** оборачивайте в `currentUser.IsAdmin()` |
| `PartialRenderer()` | `NoteViewPartialRenderer` | Рендерер для разбивки контента |
| `TOC()` | `NoteViewHeadings` | Оглавление: список `{Text, Level, ID}`; пустой, если оглавление для заметки скрыто |
| `M()` | `*Meta` | Доступ к frontmatter |

### Пример

```jet
<article>
  <h1>{{ note.Title() }}</h1>
  <time>{{ note.CreatedAt().Format("02.01.2006") }}</time>
  <span>{{ note.ReadingTime() }} мин</span>

  {{ note.HTMLString() }}
</article>
```

---

## NVS (NoteViews)

Набор заметок сайта (`NoteViews`) в обёртке для шаблонов. Доступен в шаблоне как `nvs`.

### Методы доступа

| Метод | Описание |
|-------|----------|
| `ByPath(path)` | Заметка по пути файла (`"/_sidebar.md"`, `"docs/intro.md"`) |
| `ByPermalink(url)` | Заметка по URL (`"/docs"`, `"/about"`) |
| `ByWikilink(target)` | Заметка по тексту вики-ссылки, как в Obsidian: с `/` — путь, иначе побеждает самый короткий путь |
| `List()` | Все заметки, кроме системных `/_*`, по возрастанию URL. Платные и закрытые тоже входят |

Все методы возвращают `nil`, если заметки нет. Работают и `{{ if x }}`, и `{{ if x == nil }}`, а `x.Title()` на `nil` останавливает рендер. Удобнее объявить и проверить в одном теге: `{{ if about := nvs.ByPermalink("/about"); about }}…{{ end }}`.

### Методы для навигации

| Метод | Описание |
|-------|----------|
| `Sidebars(note)` | Сайдбары для заметки: заметка с URL из поля `sidebar` (`sidebar: false` — без сайдбара), иначе сайдбары подграфов, иначе `/_sidebar` |
| `HomePages(note)` | Домашние страницы подграфов |
| `BackLinks(note)` | Обратные ссылки (кто ссылается на эту заметку), без системных заметок, по заголовку без учёта регистра, затем по permalink |
| `OutLinks(note)` | Заметки, на которые ссылается эта |
| `ResolveURL(note)` | URL заметки — сейчас то же, что `Permalink()` |

### Методы запросов

| Метод | Описание |
|-------|----------|
| `ByGlob(pattern)` | Query builder с glob-фильтром |
| `Query()` | Query builder без фильтра |

### Примеры

```jet
{* Загрузить заметку по пути *}
{{ sidebar := nvs.ByPath("/docs/_sidebar.md") }}
{{ if sidebar }}
  {{ sidebar.HTMLString() }}
{{ end }}

{* Загрузить по URL *}
{{ about := nvs.ByPermalink("/about") }}

{* Обратные ссылки *}
{{ range i, link := nvs.BackLinks(note) }}
  <a href="{{ link.Permalink() }}">{{ link.Title() }}</a>
{{ end }}
```

---

## NoteQuery

Ленивый query builder. Операции накапливаются и выполняются при вызове терминального метода.

### Фильтрация

```jet
nvs.ByGlob("blog/*.md")           {* Все .md в папке blog *}
nvs.ByGlob("docs/**/*.md")        {* Рекурсивно все .md в docs *}
nvs.ByGlob("projects/**/README.md") {* Все README.md *}
nvs.Query()                        {* Все заметки без фильтра *}
```

Паттерн сравнивается с путём файла **без ведущего слэша**: `"blog/*.md"` работает, `"/blog/*.md"` не находит ничего.

Поддерживаемые [glob-паттерны](https://github.com/bmatcuk/doublestar#patterns):
- `*` — любые символы кроме `/`
- `**` — любая вложенность
- `?` — один символ

### Что попадает в выборку

Всё, что подходит под паттерн: платные заметки, заметки за входом и системные (`_`) тоже. Для публичной страницы добавьте `.Public()`:

```jet
{* только то, что может прочитать анонимный посетитель *}
nvs.ByGlob("blog/*.md").Public()
```

`.Public()` оставляет заметки с `free`, не закрытые входом, не системные и без `noindex`. Фильтр срабатывает до `Offset` и `Limit`.

### Сортировка

```jet
.SortBy("Title")       {* По заголовку *}
.SortBy("CreatedAt")   {* По дате создания *}
.SortBy("Permalink")   {* По URL *}
.SortBy("created_at")  {* snake_case тоже работает *}

.SortByMeta("order")   {* По полю frontmatter *}
.SortByMeta("weight")
```

- `SortBy` принимает любой метод `Note` без аргументов: `Title`, `CreatedAt`, `Permalink`, `PathID`, `ReadingTime`, `UpdatedAt`, `Author`… Неизвестное имя порядок не меняет, ошибки не будет.
- `SortByMeta` сравнивает строки, целые и дробные числа, время. Заметки без значения идут первыми (с `Desc()` — последними). Значения разных типов, например `1` и `1.5`, считаются равными.
- Даты в frontmatter без кавычек (`date: 2024-05-10`) приходят строками. Даты в формате ISO как строки сортируются правильно.
- **Без сортировки порядок случайный** и меняется между рендерами. Сортировка стабильная, но равные элементы сохраняют этот случайный порядок, поэтому добавляйте второй критерий: `.SortByMeta("date").Desc().SortBy("Title")`.

### Направление

```jet
.Desc()   {* Последний критерий — по убыванию *}
.Asc()    {* Последний критерий — по возрастанию (по умолчанию) *}
```

### Множественная сортировка

```jet
{* Сначала по категории, внутри — по заголовку *}
nvs.ByGlob("blog/*.md").SortByMeta("category").SortBy("Title")
```

### Пагинация

```jet
.Limit(10)              {* Первые 10 *}
.Offset(5)              {* Пропустить 5 *}
.Offset(10).Limit(10)   {* Вторая страница *}
```

`Limit(0)` — без ограничения. `Offset` дальше конца даёт пустой список.

### Терминальные методы

| Метод | Возвращает | Описание |
|-------|------------|----------|
| `All()` | `[]*Note` | Все результаты |
| `First()` | `*Note` | Первый результат или nil |
| `Last()` | `*Note` | Последний результат или nil |

`All()` выборку не меняет, её можно выполнить ещё раз. `First()` навсегда ставит выборке лимит 1 — после него ту же выборку не используйте.

### Полный пример

```jet
{* Последние 5 публичных постов блога *}
{{ blog := nvs.ByGlob("blog/*.md").Public() }}
{{ posts := blog.SortBy("CreatedAt").Desc().Limit(5).All() }}
{{ range i, post := posts }}
  <a href="{{ post.Permalink() }}">{{ post.Title() }}</a>
{{ end }}

{* Документация с ручным порядком *}
{{ range i, doc := nvs.ByGlob("docs/*.md").SortByMeta("order").All() }}
  <a href="{{ doc.Permalink() }}">{{ doc.Title() }}</a>
{{ end }}

{* Последний пост *}
{{ latest := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").Desc().First() }}
{{ if latest }}
  <a href="{{ latest.Permalink() }}">{{ latest.Title() }}</a>
{{ end }}
```

---

## Meta

Типобезопасный доступ к frontmatter.

### Методы

| Метод | Описание |
|-------|----------|
| `Has(key)` | Проверка наличия ключа |
| `Get(key)` | Сырое значение (`interface{}`), nil если ключа нет. Арифметика с ним не работает: `Get("order") + 1` падает, используйте `GetInt` |
| `GetString(key, default)` | Строка; default, если ключа нет **или значение не строка** (`order: 3` даст default) |
| `GetInt(key, default)` | Число или default |
| `GetBool(key, default)` | Булево или default |
| `GetStrings(key)` | Список строк (не-строки отбрасываются); одна строка — список из одного элемента; если ключа нет — пустой список, не nil |
| `Raw()` | Весь frontmatter как карта, например для `json(note.M().Raw())` |
| `Debug()` | Frontmatter одной JSON-строкой, см. [[ru/user/jet-debugging\|Отладка]] |

### Приведение типов

`GetBool` понимает:
- `true`, `false` (bool)
- `"true"`, `"yes"`, `"1"` (string → true); любая другая строка — false
- `1`, `0` (число → bool: не ноль — true)

`GetInt` понимает:
- `int`, `int64`, `float64` (дробная часть отбрасывается: `2.7` → `2`). Строка `"3"` даёт default

### Примеры

```jet
{* Проверка наличия *}
{{ if note.M().Has("featured") }}
  <span class="badge">Featured</span>
{{ end }}

{* Получение значений *}
{{ author := note.M().GetString("author", "Anonymous") }}
{{ order := note.M().GetInt("order", 999) }}
{{ published := note.M().GetBool("published", false) }}

{* Теги *}
{{ range i, tag := note.M().GetStrings("tags") }}
  <span class="tag">{{ tag }}</span>
{{ end }}
```

---

## PartialRenderer

Разбивает markdown на блоки. Доступен через `note.PartialRenderer()`.

### Методы

| Метод | Описание |
|-------|----------|
| `Introduce()` | Контент до первого заголовка |
| `Sections(level)` | Секции под заголовками уровня level |
| `Section(x)` | Секция по тексту заголовка или по якорю (`"pricing"`, `"#pricing"`), иначе `nil` |
| `FirstList()` | Первый список заметки или `nil` |
| `Lists()` | Все списки верхнего уровня |
| `Images()` | Все картинки заметки по порядку |
| `CodeBlocks(lang)` | Блоки кода языка `lang`, все блоки для `""` |
| `FirstImageURL()` | URL первой картинки |

Подробнее с примерами — в [[ru/user/templates#own-toc|Шаблонах]].

### Структуры

```go
type Section struct {
    ID          string  // Якорь заголовка, тот же, что в TOC() и в HTML
    Level       int     // Уровень заголовка: 2 для ##
    Title       string  // Текст заголовка без разметки
    TitleHTML   string  // Текст заголовка (без тега)
    ContentHTML string  // Контент до следующего заголовка не ниже уровнем
}

type ListItem struct {
    Text     string
    URL      string
    Task     string  // "", "todo" для [ ], "done" для любого другого символа
    TaskMark string  // Символ в скобках: " ", "x", "/"…; "" без чекбокса
    Children []ListItem
}

type Image struct {
    URL   string  // Адрес, по которому картинку отдаёт страница
    Alt   string
    Title string
}

type CodeBlock struct {
    Lang    string  // Первое слово после ```
    Info    string  // Вся строка после ```
    Content string  // Код как написан
    HTML    string  // Блок как его рендерит страница
}
```

У секции есть свои `Sections(level)` и `Section(title)` — для вложенных подсекций.

### Примеры

```jet
{* Вступление *}
{{ intro := note.PartialRenderer().Introduce() }}
<div class="lead">{{ intro.ContentHTML }}</div>

{* FAQ из H3 *}
{{ range i, q := note.PartialRenderer().Sections(3) }}
  <details>
    <summary>{{ q.TitleHTML }}</summary>
    <div>{{ q.ContentHTML }}</div>
  </details>
{{ end }}

{* Конкретная секция *}
{{ faq := note.PartialRenderer().Section("FAQ") }}
{{ if faq }}
  {{ faq.ContentHTML }}
{{ end }}
```

---

## Jet-синтаксис

Краткая справка. Подробнее — в [[ru/user/jet|Синтаксисе Jet]] и [[ru/user/jet-functions|Справочнике функций Jet]].

### Переменные

```jet
{{ x := "value" }}              {* Объявление *}
{{ x = "new value" }}           {* Присваивание *}
{{ x }}                         {* Вывод *}
```

### Условия

```jet
{{ if condition }}
  ...
{{ else if other }}
  ...
{{ else }}
  ...
{{ end }}
```

### Циклы

```jet
{* range возвращает индекс и значение *}
{{ range i, item := list }}
  {{ i }}: {{ item }}
{{ end }}

{* Только значение — НЕПРАВИЛЬНО, item будет индексом! *}
{{ range item := list }}  {* item = 0, 1, 2... *}

{* Индекс не нужен — _ вместо первой переменной *}
{{ range _, item := list }}
  {{ item }}
{{ end }}
```

`exec`/`return` (данные из другого файла) и `try`/`catch` (заглушка вместо упавшего виджета) — в [[jet|документации Jet]].

### Блоки и наследование

```jet
{* blocks.html *}
{{ block header() }}
  <header>Default header</header>
{{ end }}

{* page.html *}
{{ import "blocks" }}

{{ yield header() }}  {* Вызов блока *}
```

### Наследование

Базовый шаблон отмечает места блоками с содержимым по умолчанию, страница начинается с `{{ extends "base" }}` и переопределяет нужные блоки. Как это устроено, с проверенным примером, — в [[ru/user/templates#Наследование шаблонов: extends|Шаблонах: наследование]].

### Фильтры

```jet
{{ value }}                   {* Экранируется по умолчанию *}
{{ value | unsafe }}          {* Вывод строки без экранирования *}
```

Вывод экранируется. Методы, которые возвращают готовый HTML (`HTMLString()`, `TitleHTML`, `ContentHTML`, `FirstListHTML()`, `FormSpecJSON()`, `asset()`), имеют тип `model.SafeHTML` и выводятся как есть. `| html` экранирует один раз и тоже возвращает `SafeHTML`, поэтому старые шаблоны с `| html` не экранируют дважды.

### Разбор данных

```jet
{{ d := parseJSON(text) }}    {* JSON в словари и списки, nil при ошибке *}
{{ d := parseYAML(text) }}    {* То же для YAML *}
{{ rows := parseCSV(text) }}  {* Список строк, строка — список значений *}
```

Подробнее — в разделе [[ru/user/templates#parse-data|Разбор данных]]. Все функции и фильтры — в [[ru/user/jet-functions|Справочнике функций Jet]].

