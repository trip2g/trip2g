---
free: true
title: "Шаблоны: запросы к заметкам"
slug: templates-query
---

Выборка заметок по паттерну, сортировка, пагинация — всё через `nvs.ByGlob()`.

### Быстрый старт

Вывести последние 5 постов блога:

```jet
{{ range i, post := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").Desc().Limit(5).All() }}
  <article>
    <h2>{{ post.Title() }}</h2>
  </article>
{{ end }}
```

### ByGlob — фильтрация по паттерну

Выбирает заметки по glob-паттерну пути файла:

```jet
{* Все .md в папке blog *}
{{ nvs.ByGlob("blog/*.md") }}

{* Все README.md в любой вложенности *}
{{ nvs.ByGlob("projects/**/README.md") }}

{* Все .md в корне *}
{{ nvs.ByGlob("*.md") }}
```

Поддерживаемые паттерны:
- `*` — любые символы кроме `/`
- `**` — любая вложенность папок
- `?` — один любой символ

### SortBy — сортировка по полям заметки

Сортировка по встроенным полям:

```jet
.SortBy("Title")       — по заголовку
.SortBy("CreatedAt")   — по дате создания
.SortBy("Permalink")   — по URL
```

Поддерживается snake_case:

```jet
.SortBy("created_at")  — то же что CreatedAt
```

По умолчанию — по возрастанию (A→Z, старые→новые).

### SortByMeta — сортировка по frontmatter

Сортировка по произвольному полю из frontmatter:

```jet
{* Если в заметках есть order: 1, order: 2... *}
{{ range i, post := nvs.ByGlob("docs/*.md").SortByMeta("order").All() }}
  {{ post.Title() }}
{{ end }}
```

Работает с числами и строками.

### Desc и Asc — направление сортировки

```jet
.SortBy("CreatedAt").Desc()   — новые первыми
.SortBy("Title").Asc()        — A→Z (по умолчанию)
```

`Desc()` и `Asc()` действуют на последний критерий.

### Множественная сортировка

Сначала по одному полю, потом по другому:

```jet
{* Сначала по категории, внутри категории — по заголовку *}
{{ nvs.ByGlob("blog/*.md").SortByMeta("category").SortBy("Title").All() }}
```

Порядок важен: первый критерий главный, второй — для одинаковых значений первого.

### Limit и Offset — пагинация

```jet
.Limit(10)             — первые 10 результатов
.Offset(5)             — пропустить первые 5
.Offset(10).Limit(10)  — вторая страница по 10
```

### All, First, Last — получение результатов

```jet
.All()    — список всех заметок []*Note
.First()  — первая заметка или nil
.Last()   — последняя заметка или nil
```

**Важно:** `range` требует `All()`:

```jet
{{ range i, post := nvs.ByGlob("...").All() }}
```

### Query — запрос без фильтра

Для работы со всеми заметками без glob-фильтра:

```jet
{* Все заметки, отсортированные по заголовку *}
{{ range i, note := nvs.Query().SortBy("Title").All() }}
  {{ note.Title() }}
{{ end }}
```

### Примеры

#### Список постов блога

```jet
<div class="posts">
  {{ range i, post := nvs.ByGlob("blog/*.md").SortBy("Title").All() }}
    <article>
      <h2><a href="{{ post.Permalink() }}">{{ post.Title() }}</a></h2>
    </article>
  {{ end }}
</div>
```

#### Документация с ручным порядком

Frontmatter:
```yaml
---
title: Установка
order: 1
---
```

Шаблон:
```jet
<nav class="docs-nav">
  {{ range i, doc := nvs.ByGlob("docs/*.md").SortByMeta("order").All() }}
    <a href="{{ doc.Permalink() }}">{{ doc.Title() }}</a>
  {{ end }}
</nav>
```

#### Последний пост на главной

```jet
{{ latest := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").Desc().First() }}
{{ if latest }}
  <section class="latest-post">
    <h2>Последнее в блоге</h2>
    <a href="{{ latest.Permalink() }}">{{ latest.Title() }}</a>
  </section>
{{ end }}
```

### Синтаксис range в Jet

**Внимание:** в Jet `range` возвращает индекс и значение:

```jet
{* Правильно *}
{{ range i, post := list }}

{* Неправильно — post будет индексом! *}
{{ range post := list }}
```

Если индекс не нужен, просто игнорируйте его:

```jet
{{ range i, post := nvs.ByGlob("blog/*.md").All() }}
  {{ post.Title() }}
{{ end }}
```
