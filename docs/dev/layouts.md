---
title: "Layouts API"
slug: layouts
free: true
---

Техническая документация для разработчиков шаблонов. Описывает API `internal/templateviews` — обёртки над моделями для использования в Jet-шаблонах.

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
| `HTMLString()` | `string` | Отрендеренный HTML контент |
| `ContentString()` | `string` | Сырой markdown |
| `PathID()` | `int64` | ID для data-атрибутов |
| `Permalink()` | `string` | URL страницы |
| `CreatedAt()` | `time.Time` | Дата создания |
| `ReadingTime()` | `int` | Время чтения в минутах |
| `ReadingComplexity()` | `int` | Сложность (0-2) |
| `IsHomePage()` | `bool` | Является ли домашней страницей подграфа |
| `Description()` | `string` | SEO-описание |
| `PartialRenderer()` | `NoteViewPartialRenderer` | Рендерер для разбивки контента |
| `TOC()` | `NoteViewHeadings` | Оглавление |
| `M()` | `*Meta` | Доступ к frontmatter |

### Пример

```jet
<article>
  <h1>{{ note.Title() }}</h1>
  <time>{{ note.CreatedAt().Format("02.01.2006") }}</time>
  <span>{{ note.ReadingTime() }} мин</span>

  {{ note.HTMLString() | unsafe }}
</article>
```

---

## NVS (NoteViewService)

Сервис доступа к заметкам. Доступен в шаблоне как `nvs`.

### Методы доступа

| Метод | Описание |
|-------|----------|
| `ByPath(path)` | Заметка по пути файла (`"/_sidebar.md"`, `"docs/intro.md"`) |
| `ByPermalink(url)` | Заметка по URL (`"/docs"`, `"/about"`) |
| `List()` | Все видимые заметки (без системных `/_*`) |

### Методы для навигации

| Метод | Описание |
|-------|----------|
| `Sidebars(note)` | Сайдбары для заметки |
| `HomePages(note)` | Домашние страницы подграфов |
| `BackLinks(note)` | Обратные ссылки (кто ссылается на эту заметку) |
| `ResolveURL(note)` | Полный URL с версией |

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
  {{ sidebar.HTMLString() | unsafe }}
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

Поддерживаемые glob-паттерны:
- `*` — любые символы кроме `/`
- `**` — любая вложенность
- `?` — один символ

### Сортировка

```jet
.SortBy("Title")       {* По заголовку *}
.SortBy("CreatedAt")   {* По дате создания *}
.SortBy("Permalink")   {* По URL *}
.SortBy("created_at")  {* snake_case тоже работает *}

.SortByMeta("order")   {* По полю frontmatter *}
.SortByMeta("weight")
```

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

### Терминальные методы

| Метод | Возвращает | Описание |
|-------|------------|----------|
| `All()` | `[]*Note` | Все результаты |
| `First()` | `*Note` | Первый результат или nil |
| `Last()` | `*Note` | Последний результат или nil |

### Полный пример

```jet
{* Последние 5 постов блога *}
{{ range i, post := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").Desc().Limit(5).All() }}
  <article>
    <h2><a href="{{ post.Permalink() }}">{{ post.Title() }}</a></h2>
    <time>{{ post.CreatedAt().Format("02.01.2006") }}</time>
  </article>
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
| `Get(key)` | Сырое значение (`interface{}`) |
| `GetString(key, default)` | Строка или default |
| `GetInt(key, default)` | Число или default |
| `GetBool(key, default)` | Булево или default |
| `GetStringSlice(key)` | Массив строк или nil |

### Приведение типов

`GetBool` понимает:
- `true`, `false` (bool)
- `"true"`, `"yes"`, `"1"` (string → true)
- `1`, `0` (int → bool)

`GetInt` понимает:
- `int`, `int64`, `float64`

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
{{ range i, tag := note.M().GetStringSlice("tags") }}
  <span class="tag">{{ tag }}</span>
{{ end }}
```

---

## PartialRenderer

Разбивает markdown на логические блоки. Возвращается методом `note.PartialRenderer()`.

### Методы

| Метод | Описание |
|-------|----------|
| `Introduce()` | Контент до первого заголовка |
| `Sections(level)` | Секции под заголовками уровня level |
| `Section(title)` | Секция по тексту заголовка |

### Структура секции

```go
type Section struct {
    TitleHTML   string  // Текст заголовка (без тега)
    ContentHTML string  // Контент до следующего заголовка
}
```

### Примеры

```jet
{* Вступление *}
{{ intro := note.PartialRenderer().Introduce() }}
<div class="lead">{{ intro.ContentHTML | unsafe }}</div>

{* FAQ из H3 *}
{{ range i, q := note.PartialRenderer().Sections(3) }}
  <details>
    <summary>{{ q.TitleHTML | unsafe }}</summary>
    <div>{{ q.ContentHTML | unsafe }}</div>
  </details>
{{ end }}

{* Конкретная секция *}
{{ faq := note.PartialRenderer().Section("FAQ") }}
{{ if faq }}
  {{ faq.ContentHTML | unsafe }}
{{ end }}
```

---

## Jet-синтаксис

Шаблонизатор на основе Go templates с расширениями.

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
```

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

### Переопределение блоков

```jet
{* page.html *}
{{ import "blocks" }}

{{ block header() }}
  <header>Custom header</header>
{{ end }}

{{ yield main_layout() content }}
  ...
{{ end }}
```

### Фильтры

```jet
{{ value | unsafe }}          {* Вывод HTML без экранирования *}
{{ value | html }}            {* Экранирование (по умолчанию) *}
```

---

## Исходный код

```
internal/templateviews/
├── note.go       # Note — обёртка заметки
├── nvs.go        # NVS — сервис доступа к заметкам
├── query.go      # NoteQuery — query builder
├── meta.go       # Meta — доступ к frontmatter
└── *_test.go     # Тесты
```
