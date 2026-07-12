---
free: true
title: Шаблоны
home_position: 70
---

Шаблон — HTML-файл, который определяет внешний вид страницы. Один файл в папке `_layouts/` — и готово.

> **Ключевая идея:** контент остаётся чистым markdown, а шаблон через [[templates-advanced|PartialRenderer]] получает доступ к AST-структуре документа. Это позволяет автору писать обычный markdown, а разработчику шаблона — произвольно компоновать секции, заголовки и блоки без загрязнения контента разметкой.

> **Пример вживую:** [[instaframes/_index|Instagram-фреймы]] — готовый шаблон, который собирает из markdown-файла скачиваемые карусели для соцсетей. Наглядно, как кастомный layout делает реальную работу.

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

#### htmlInjectionsHead / htmlInjectionsBodyEnd — HTML-инъекции из настроек сайта

Скрипты и теги, добавленные в настройках сайта (Google Analytics, пиксели, кастомный `<head>`), доступны в шаблоне через две переменные:

```jet
{{ range i, injection := htmlInjectionsHead }}{{ injection.Content | unsafe }}{{ end }}
```

```jet
{{ range i, injection := htmlInjectionsBodyEnd }}{{ injection.Content | unsafe }}{{ end }}
```

`htmlInjectionsHead` — вставлять перед `</head>`, `htmlInjectionsBodyEnd` — перед `</body>`.

> **Совет:** Если используете кастомный Jet-шаблон, добавьте обе переменные, чтобы скрипты из Admin → HTML Injections подключались автоматически:
> ```jet
> <head>
>   ...
>   {{ range i, injection := htmlInjectionsHead }}{{ injection.Content | unsafe }}{{ end }}
> </head>
> <body>
>   ...
>   {{ range i, injection := htmlInjectionsBodyEnd }}{{ injection.Content | unsafe }}{{ end }}
> </body>
> ```

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

### Asset-ы между layout-файлами

`asset()` ищет URL в общей таблице, объединяющей ассеты всех layout-файлов сайта. Это значит, что блок из `cases.html`, вызывающий `asset("topo.svg")`, корректно отдаст ссылку на S3, даже когда страница рендерится через `index.html` (например, по цепочке `yield`).

Ключи в таблице — абсолютные пути (`_layouts/mesh/topo.svg`), коллизий между layout-ами не бывает.

Движок сам обходит `import` и yield-цепочки и находит вызовы `asset()`. В редких случаях, когда зависимость прячется в неочевидном месте, добавьте комментарий-подсказку:

```jet
{{ import "blocks" }}

<!-- {{ asset("style.css") }} -->

{{ yield main_layout() content }}
  ...
{{ end }}
```

Комментарий не попадает в HTML, но зависимость гарантированно попадёт в обнаружение.

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

Выборка и сортировка заметок — в [[templates-advanced|запросах к заметкам]].

Продвинутые паттерны организации шаблонов — в [[templates-best-practices|лучших практиках]].
