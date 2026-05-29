---
title: "Страница на чистом HTML"
free: true
home_position: 995
lang_redirect: "[[en/user/One HTML Page]]"
---

Иногда страница должна быть полностью на HTML — лендинг, кастомная главная или демо. В trip2g это делается через layout: весь HTML живёт в файле шаблона, заметка только указывает на него.

> **Внимание:** сырой HTML в теле заметки **не работает**. goldmark (markdown-движок trip2g) вырезает блочный HTML и заменяет его на `<!-- raw HTML omitted -->`. HTML пишите только в layout-файле.

### Паттерн 1 — единичная страница

Весь HTML в layout, тело заметки пустое. Подходит для кастомной главной, лендинга, демо.

**Шаг 1.** Создайте `_layouts/my-page.html` со всем HTML:

```jet
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ note.Title() }}</title>
  <style>/* весь CSS здесь */</style>
</head>
<body>
  <section class="hero">
    <h1>Привет, мир</h1>
    <p>Полностью кастомная HTML-страница.</p>
    <a href="/signup" class="btn">Начать</a>
  </section>
</body>
</html>
```

**Шаг 2.** Создайте `my-page.md` — только frontmatter, тело пустое:

```yaml
---
title: Моя страница
layout: my-page
free: true
---
```

Готово. После sync заметка публикуется как страница, которую целиком рендерит layout.

### Паттерн 2 — один layout на много заметок

Один layout, десятки заметок с разным контентом. Вариативность берётся из frontmatter и markdown-тела, **не из raw HTML в теле**.

Данные из frontmatter через `note.M()`:

```jet
{{ note.M().GetString("hero_title", "Заголовок") }}
{{ note.M().GetString("cta_text", "Начать") }}
{{ note.M().GetBool("show_video", false) }}
```

Markdown-контент заметки целиком:

```jet
{{ note.HTMLString() | unsafe }}
```

Разбивка markdown на секции через `note.PartialRenderer()`:

```jet
{{ range i, section := note.PartialRenderer().Sections(2) }}
  <section>
    <h2>{{ section.TitleHTML | unsafe }}</h2>
    {{ section.ContentHTML | unsafe }}
  </section>
{{ end }}
```

Полный API — в [[ru/user/templates-advanced|API шаблонов]].

### Паттерн 3 — многофайловый layout

Для сайтов с общей шапкой, подвалом и несколькими типами страниц: вынесите повторяющееся в `blocks.html`.

```
_layouts/
└── my-theme/
    ├── blocks.html   — шапка, подвал, базовая обёртка
    ├── page.html     — шаблон обычной страницы
    └── landing.html  — шаблон лендинга
```

`blocks.html`:

```jet
{{ block main_layout() }}
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <title>{{ note.Title() }}</title>
  <link rel="stylesheet" href="{{ asset("style.css") }}">
</head>
<body>
  {{ yield header() }}
  <main>{{ yield content }}</main>
  {{ yield footer() }}
</body>
</html>
{{ end }}

{{ block header() }}
<header><a href="/">Главная</a></header>
{{ end }}

{{ block footer() }}
<footer><p>2025 Мой сайт</p></footer>
{{ end }}
```

`page.html`:

```jet
{{ import "blocks" }}

{{ yield main_layout() content }}
  <article>
    <h1>{{ note.Title() }}</h1>
    {{ note.HTMLString() | unsafe }}
  </article>
{{ end }}
```

Подробнее — в [[ru/user/templates-best-practices|Лучших практиках]].

### Отладка: превью layout до sync

`/_system/renderlayout` рендерит layout с предупреждениями, **не записывая ничего в vault**:

```
GET /_system/renderlayout?note=<slug>&layout=<layout-name>
```

Используйте для проверки перед первым sync. Подробнее — в [[ru/user/renderlayout|renderlayout]].

### Примечания

URL страницы формируется из имени файла. Чтобы сделать страницу корнем сайта, добавьте `slug: /` в frontmatter.

CSS и JS размещайте в `_assets/` и подключайте через `asset()` в шаблоне:

```jet
<link rel="stylesheet" href="{{ asset("landing.css") }}">
```

---

Смотрите также: [[ru/user/templates|Шаблоны]] · [[ru/user/templates-advanced|API шаблонов]] · [[ru/user/templates-best-practices|Лучшие практики]] · [[ru/user/jet|Синтаксис Jet]] · [[ru/user/default-template|Дефолтный шаблон]] · [[ru/user/renderlayout|renderlayout]]
