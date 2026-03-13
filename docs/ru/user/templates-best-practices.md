---
free: true
title: "Шаблоны: лучшие практики"
slug: templates-best-practices
---

Как организовать шаблоны в проекте с несколькими страницами.

### Когда нужна структура

Один шаблон на страницу работает для простых сайтов. Когда появляются общие элементы — шапка, подвал, стили — стоит вынести их в общие блоки.

### Паттерн: базовый layout + страницы

Структура:

```
_layouts/
└── my-theme/
    ├── blocks.html   — общие блоки (шапка, подвал, обёртка)
    ├── page.html     — шаблон обычной страницы
    └── landing.html  — шаблон лендинга
```

#### blocks.html — общие блоки

```jet
{{ block main_layout() }}
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <title>{{ note.Title() }} — Мой сайт</title>
  <link rel="stylesheet" href="{{ asset("style.css") }}">
</head>
<body>
  {{ yield header() }}
  <main>
    {{ yield content }}
  </main>
  {{ yield footer() }}
</body>
</html>
{{ end }}

{{ block header() }}
<header>
  <a href="/">Главная</a>
  <nav>...</nav>
</header>
{{ end }}

{{ block footer() }}
<footer>
  <p>2025 Мой сайт</p>
</footer>
{{ end }}
```

#### page.html — использует layout

```jet
{{ import "blocks" }}

{{ yield main_layout() content }}
  <article class="prose">
    <h1>{{ note.Title() }}</h1>
    {{ note.HTMLString() | unsafe }}
  </article>
{{ end }}
```

#### landing.html — другой контент в том же layout

```jet
{{ import "blocks" }}

{{ yield main_layout() content }}
  <div class="hero">
    {{ intro := note.PartialRenderer().Introduce() }}
    {{ intro.ContentHTML | unsafe }}
  </div>

  <div class="features">
    {{ range b := note.PartialRenderer().Sections(2) }}
      <section>
        <h2>{{ b.TitleHTML | unsafe }}</h2>
        {{ b.ContentHTML | unsafe }}
      </section>
    {{ end }}
  </div>
{{ end }}
```

### Паттерн: общие блоки

Выносите повторяющиеся элементы в блоки:

```jet
{{ block card(title, content) }}
<div class="card">
  <h3>{{ title }}</h3>
  <div>{{ content }}</div>
</div>
{{ end }}

{{ block button(text, href, type="primary") }}
<a href="{{ href }}" class="btn btn-{{ type }}">{{ text }}</a>
{{ end }}
```

Использование:

```jet
{{ yield card(title="Быстрый старт", content="Описание...") }}
{{ yield button(text="Попробовать", href="/signup") }}
{{ yield button(text="Узнать больше", href="/docs", type="secondary") }}
```

### Паттерн: условные блоки

Переопределяйте блоки для разных страниц:

```jet
{* В blocks.html — блок по умолчанию *}
{{ block sidebar() }}
  <aside>Стандартный сайдбар</aside>
{{ end }}
```

```jet
{* В page.html — переопределение *}
{{ import "blocks" }}

{{ block sidebar() }}
  <aside>Сайдбар для статей</aside>
{{ end }}

{{ yield main_layout() content }}
  ...
{{ end }}
```

### Паттерн: include для фрагментов

Небольшие повторяющиеся фрагменты — через `include`:

```
_layouts/
└── my-theme/
    ├── blocks.html
    ├── page.html
    └── partials/
        ├── social-links.html
        └── newsletter-form.html
```

```jet
{* partials/social-links.html *}
<div class="social">
  <a href="https://t.me/...">Telegram</a>
  <a href="https://twitter.com/...">Twitter</a>
</div>
```

```jet
{* В любом шаблоне *}
{{ include "partials/social-links" }}
```

С передачей данных:

```jet
{{ include "partials/user-card" user }}
```

### Организация стилей

Один CSS-файл на тему, подключается в `blocks.html`:

```jet
<link rel="stylesheet" href="{{ asset("style.css") }}">
```

Для сборки (Tailwind, PostCSS) — выходной файл:

```jet
<link rel="stylesheet" href="{{ asset("style.out.css") }}">
```

### Чеклист перед созданием структуры

- [ ] Больше двух страниц с общей шапкой/подвалом?
- [ ] Есть повторяющиеся элементы (карточки, кнопки)?
- [ ] Планируются разные типы страниц (статьи, лендинги)?

Если да хотя бы на один — используйте структуру с `blocks.html`.

Если нет — один файл шаблона на страницу проще и понятнее.

### Паттерн: sidebar из markdown-файла

Навигацию в сайдбаре удобно хранить в отдельном markdown-файле. Это позволяет редактировать меню без изменения шаблона.

Структура:
```
docs/
├── _sidebar.md      — навигация для документации
├── onboarding.md
└── ...

thoughts/
├── _sidebar.md      — навигация для статей
├── article-1.md
└── ...
```

Файл `_sidebar.md`:
```markdown
---
title: "Sidebar"
---

### Раздел 1

- [Страница 1](/docs/page-1)
- [Страница 2](/docs/page-2)

### Раздел 2

- [Страница 3](/docs/page-3)
```

В шаблоне:
```jet
{{ sidebar := nvs.ByPath("/docs/_sidebar.md") }}
{{ if sidebar }}
  {{ range i, section := sidebar.PartialRenderer().Sections(3) }}
  <div>
    <h3>{{ section.TitleHTML | unsafe }}</h3>
    {{ section.ContentHTML | unsafe }}
  </div>
  {{ end }}
{{ else }}
  <p>Создайте файл <code>docs/_sidebar.md</code> для навигации</p>
{{ end }}
```

Преимущества:
- Добавить страницу в меню — отредактировать markdown
- Порядок и группировка — в одном месте
- Подсказка при отсутствии файла — не нужно читать документацию

### Подводные камни Jet: блоки с параметрами

Jet требует особого внимания при работе с блоками. Два правила, которые легко нарушить:

#### 1. Параметры блока должны иметь значения по умолчанию

Если хотите вызывать блок с именованными параметрами — укажите дефолтные значения при объявлении:

```jet
{* Правильно — есть дефолты *}
{{ block cta_section(title="", subtitle="") }}
  <h2>{{ title }}</h2>
  <p>{{ subtitle }}</p>
{{ end }}

{{ yield cta_section(title="Заголовок", subtitle="Подзаголовок") }}
```

```jet
{* Неправильно — без дефолтов *}
{{ block cta_section(title, subtitle) }}
  <h2>{{ title }}</h2>
  <p>{{ subtitle }}</p>
{{ end }}

{{ yield cta_section(title="Заголовок", subtitle="Подзаголовок") }}
{* Результат: title и subtitle будут false *}
```

Без значений по умолчанию Jet не связывает именованные аргументы с параметрами. На выходе — `false` вместо текста.

#### 2. `content` — зарезервированное слово

Нельзя использовать `content` как имя параметра. Jet выдаст ошибку парсинга:

```
unexpected keyword 'content' (expected closing parenthesis)
```

`content` — это механизм передачи вложенного HTML в блок. Работает в паре:

```jet
{* Определение блока — yield content выводит вложенное содержимое *}
{{ block link(href="") }}
  <a href="{{ href }}">{{ yield content }}</a>
{{ end }}

{* Вызов — content после параметров открывает блок для вложенного содержимого *}
{{ yield link(href="https://example.com") content }}
  Текст ссылки
{{ end }}

{* Результат *}
<a href="https://example.com">Текст ссылки</a>
```

Если нужен параметр для текста — используйте другое имя:

```jet
{{ block card(title="", body="") }}
  <h3>{{ title }}</h3>
  <p>{{ body }}</p>
{{ end }}
```

#### Как отлаживать

Если блок выводит `false` вместо значений:

1. Проверьте, что у всех параметров есть `=""` или другой дефолт
2. Проверьте, что имена параметров не зарезервированы (`content`)
3. Убедитесь, что при вызове используете именованные параметры: `param="value"`