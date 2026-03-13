---
free: true
title: Шаблоны
slug: templates
---

Шаблон — HTML-файл, который определяет внешний вид страницы. Один файл в папке `_layouts/` — и готово.

> **Ключевая идея:** контент остаётся чистым markdown, а шаблон через [[templates-advanced|PartialRenderer]] получает доступ к AST-структуре документа. Это позволяет автору писать обычный markdown, а разработчику шаблона — произвольно компоновать секции, заголовки и блоки без загрязнения контента разметкой.

### Быстрый старт

**Шаг 1.** Создайте файл `_layouts/my-page.html`:

```jet
<!DOCTYPE html>
<html>
<head>
  <title>{{ note.Title() }}</title>
</head>
<body>
  <h1>{{ note.Title() }}</h1>
  {{ note.HTMLString() | unsafe }}
</body>
</html>
```

**Шаг 2.** Укажите шаблон в заметке:

```yaml
---
layout: my-page
title: Моя страница
---

Текст страницы в markdown.
```

Готово. Страница использует ваш шаблон.

### Что доступно в шаблоне

#### note — текущая заметка

```jet
{{ note.Title() }}         — заголовок из frontmatter
{{ note.HTMLString() }}    — весь контент как HTML
{{ note.Permalink() }}     — URL страницы
{{ note.ReadingTime() }}   — время чтения в минутах
{{ note.PathID() }}        — уникальный ID для data-атрибутов
```

#### note.M() — доступ к frontmatter

```jet
{{ note.M().GetString("author", "Unknown") }}
{{ note.M().GetInt("version", 1) }}
{{ note.M().GetBool("featured", false) }}
{{ note.M().Has("custom_field") }}
```

#### nvs — доступ к другим заметкам

```jet
{{ sidebar := nvs.ByPath("/_sidebar.md") }}
{{ if sidebar }}
  {{ sidebar.HTMLString() | unsafe }}
{{ end }}

{{ about := nvs.ByPermalink("/about") }}
```

#### asset() — подключение файлов

```jet
<link rel="stylesheet" href="{{ asset("style.css") }}">
<script src="{{ asset("app.js") }}"></script>
```

### PartialRenderer — контент по частям

`PartialRenderer` разбирает markdown на логические блоки. Полезно для лендингов, FAQ, карточек.

#### Introduce() — вступление

Возвращает контент **до первого заголовка**:

```jet
{{ intro := note.PartialRenderer().Introduce() }}
<div class="intro">
  {{ intro.ContentHTML | unsafe }}
</div>
```

#### Sections(level) — секции по уровню заголовков

Собирает секции под заголовками нужного уровня. Каждая секция содержит:
- `Title` — текст заголовка (plain text)
- `TitleHTML` — текст заголовка с форматированием (без тега `<h3>`)
- `ContentHTML` — контент до следующего заголовка того же или выше уровня
- `Sections(level)` — вложенные секции
- `Section(title)` — поиск вложенной секции по заголовку

**Пример: FAQ из markdown**

```markdown
Часто задаваемые вопросы о сервисе.

### Как начать работу?

Зарегистрируйтесь и создайте первый проект.

### Сколько стоит?

Базовый тариф бесплатный.

### Есть ли API?

Да, документация на сайте.
```

Шаблон:

```jet
{{ intro := note.PartialRenderer().Introduce() }}
<p class="lead">{{ intro.ContentHTML | unsafe }}</p>

<div class="faq">
  {{ range i, s := note.PartialRenderer().Sections(3) }}
    <details>
      <summary>{{ s.TitleHTML | unsafe }}</summary>
      <div>{{ s.ContentHTML | unsafe }}</div>
    </details>
  {{ end }}
</div>
```

#### Section(title) — секция по заголовку

Находит конкретную секцию по тексту заголовка:

```jet
{{ faq := note.PartialRenderer().Section("FAQ") }}
{{ if faq }}
  <div class="faq-section">
    {{ faq.ContentHTML | unsafe }}
  </div>
{{ end }}
```

**Пример: карточки фич**

```jet
<div class="features-grid">
  {{ range i, s := note.PartialRenderer().Sections(3) }}
    <div class="feature-card">
      <h3>{{ s.TitleHTML | unsafe }}</h3>
      {{ s.ContentHTML | unsafe }}
    </div>
  {{ end }}
</div>
```

#### Вложенные секции

Каждая секция сама имеет методы `Sections(level)` и `Section(title)`. Это позволяет итерировать по вложенным уровням.

**Пример: презентация с категориями и слайдами**

Markdown:
```markdown
## Продукт

### Обзор
Краткое описание продукта.

### Возможности
Список ключевых функций.

## Команда

### Основатели
История создания.

### Вакансии
Открытые позиции.
```

Шаблон:
```jet
{{ range idx, category := note.PartialRenderer().Sections(2) }}
<section class="category">
  <h2>{{ category.TitleHTML | unsafe }}</h2>
  {{ range slideIdx, slide := category.Sections(3) }}
  <div class="slide" data-num="{{ slideIdx + 1 }}">
    <h3>{{ slide.TitleHTML | unsafe }}</h3>
    {{ slide.ContentHTML | unsafe }}
  </div>
  {{ end }}
</section>
{{ end }}
```

**Пример: найти конкретную подсекцию**

```jet
{{ features := note.PartialRenderer().Section("Продукт") }}
{{ if features }}
  {{ overview := features.Section("Обзор") }}
  {{ if overview }}
    <div class="product-overview">
      {{ overview.ContentHTML | unsafe }}
    </div>
  {{ end }}
{{ end }}
```

### Фильтр unsafe

HTML экранируется по умолчанию. Чтобы вывести разметку — добавьте `| unsafe`:

```jet
{{ note.HTMLString() | unsafe }}
{{ b.ContentHTML | unsafe }}
```

Без фильтра теги отобразятся как текст: `&lt;p&gt;...`

### Вложенные asset-зависимости

Если шаблон импортирует другой файл с `asset()`, движок может не увидеть зависимость. Обходное решение — добавить комментарий:

```jet
{{ import "blocks" }}

<!-- {{ asset("style.css") }} -->

{{ yield main_layout() content }}
  ...
{{ end }}
```

Комментарий не попадает в HTML, но зависимость подхватится.

### Синтаксис Jet

Шаблоны используют движок [[jet|Jet]]:

```jet
{{ переменная }}                       — вывод
{{ if условие }}...{{ end }}           — условие
{{ range item := список }}...{{ end }} — цикл
{{ block имя() }}...{{ end }}          — определение блока
{{ yield имя() }}                      — вызов блока
{{ include "путь" данные }}            — вставка шаблона
```

Подробнее — в [[jet|документации Jet]].

Выборка и сортировка заметок — в [[templates-query|запросах к заметкам]].

Продвинутые паттерны организации шаблонов — в [[templates-best-practices|лучших практиках]].
