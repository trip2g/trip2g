---
free: true
title: Шаблоны
home_position: 70
---

Шаблон — HTML-файл, который определяет внешний вид страницы. Один файл в папке `_layouts/` — и готово.

> **Рекомендуемая структура:** собирайте свои шаблоны из компонентов: один `block` на файл, вызов через `yield`, имя через `@lid`, стили по BEM, импорт автоматически. См. [[ru/user/components|Компоненты, автоимпорт и лучшие практики]].

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
  {{ note.HTMLString() }}
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
{{ if sidebar := nvs.ByPath("/_sidebar.md"); sidebar }}
  {{ sidebar.HTMLString() }}
{{ end }}

{{ about := nvs.ByPermalink("/about") }}
```

Если поиск ничего не нашёл — `nvs.ByPath`, `nvs.ByPermalink`, `nvs.ByWikilink`, `.First()` и `.Last()` у запроса, `PartialRenderer().Section(...)`, `FirstList()`, — он возвращает `nil`. Проверять можно и `{{ if x }}`, и `{{ if x == nil }}`. Вызов метода у такого значения, например `x.Title()`, обрывает рендер с ошибкой, поэтому сначала проверьте. (Раньше эти методы возвращали значение, которое `{{ if x }}` считал пустым, а `x == nil` — нет; шаблоны с `{{ if x }}` продолжают работать.) Объявить и проверить переменную можно одним тегом, как в примере выше: см. [[ru/user/templates#Присваивание прямо в if (как в Go)|присваивание в if]].

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

### SEO-теги в своём layout

Шаблон по умолчанию сам пишет `<link rel="canonical">`, `og:url`, `hreflang` и `<meta name="robots">`. Свой layout не пишет ничего из этого. Единственное, что trip2g добавляет сам, — HTTP-заголовок `X-Robots-Tag: noindex` для заметок с `noindex: true`.

`publicURL` — адрес основного домена. На [[multidomains|кастомном домене]] это всё равно основной домен, поэтому canonical собирайте из маршрута заметки:

```jet
<head>
  {{ if note.M().GetBool("noindex", false) }}<meta name="robots" content="noindex">{{ end }}
  {{ canonicalRoute := note.M().GetString("route", "") }}
  {{ if canonicalRoute != "" }}
  <link rel="canonical" href="https://{{ canonicalRoute }}">
  {{ else }}
  <link rel="canonical" href="{{ publicURL }}{{ note.Permalink() }}">
  {{ end }}
</head>
```

Значение `noindex` по умолчанию оставляйте `false`. С `GetBool("noindex", true)` каждая страница без этого свойства помечается `noindex`, и весь сайт выпадает из поиска, а HTTP-заголовки при этом выглядят чистыми. Пример рассчитан на `route` в виде полного `домен/путь`; если используете `routes` или алиасы основного домена, доработайте его.

### PartialRenderer — контент по частям

`PartialRenderer` разбирает markdown на логические блоки. Полезно для лендингов, FAQ, карточек.

#### Introduce() — вступление

Возвращает контент **до первого заголовка**:

```jet
{{ intro := note.PartialRenderer().Introduce() }}
<div class="intro">
  {{ intro.ContentHTML }}
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
<p class="lead">{{ intro.ContentHTML }}</p>

<div class="faq">
  {{ range i, s := note.PartialRenderer().Sections(3) }}
    <details>
      <summary>{{ s.TitleHTML }}</summary>
      <div>{{ s.ContentHTML }}</div>
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
    {{ faq.ContentHTML }}
  </div>
{{ end }}
```

**Пример: карточки фич**

```jet
<div class="features-grid">
  {{ range i, s := note.PartialRenderer().Sections(3) }}
    <div class="feature-card">
      <h3>{{ s.TitleHTML }}</h3>
      {{ s.ContentHTML }}
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
  <h2>{{ category.TitleHTML }}</h2>
  {{ range slideIdx, slide := category.Sections(3) }}
  <div class="slide" data-num="{{ slideIdx + 1 }}">
    <h3>{{ slide.TitleHTML }}</h3>
    {{ slide.ContentHTML }}
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
      {{ overview.ContentHTML }}
    </div>
  {{ end }}
{{ end }}
```

#### Как Section(x) ищет заголовок

У секции кроме `Title`, `TitleHTML` и `ContentHTML` есть `ID` — якорь заголовка — и `Level` (2 для `##`). `Section(x)` пробует три способа по порядку и возвращает первый подошедший заголовок:

1. точный текст заголовка: `Section("Цены и тарифы")`;
2. текст без учёта регистра и лишних пробелов: `Section("цены  и тарифы")`;
3. якорь заголовка, с `#` или без: `Section("cenyi_i_tarifyi")`, `Section("#pricing")`.

С `#` поиск по якорю не путается с первыми двумя шагами: текст заголовка не начинается с `#`, поэтому `Section("#" + h.ID)` всегда находит заголовок с этим id. Без `#` вызов `Section("intro")` найдёт заголовок «Intro» раньше, чем заголовок с якорем `intro`.

### Заголовки, якоря и своё оглавление {#own-toc}

Откуда у заголовков id и как задать свой через `{#id}`, описано в [[ru/user/markdown#heading-anchors|Markdown]]. В шаблоне `note.TOC()` возвращает те же заголовки, что показало бы стандартное оглавление, у каждого есть `Text`, `ID` и `Level`. `Level` здесь относительный: верхний уровень заголовков в заметке — 1, следующий из использованных — 2 и так далее. Если поле `toc` во frontmatter скрывает оглавление (или `auto` решило его не показывать), список пустой.

Оглавление, а под ним секции с теми же якорями:

```jet
{{ pr := note.PartialRenderer() }}
<nav class="toc">
  {{ range i, h := note.TOC() }}
    <a class="toc__item toc__item--{{ h.Level }}" href="#{{ h.ID }}">{{ h.Text }}</a>
  {{ end }}
</nav>

{{ range i, s := pr.Sections(2) }}
  <section id="{{ s.ID }}">
    <h2>{{ s.TitleHTML }}</h2>
    {{ s.ContentHTML }}
  </section>
{{ end }}
```

Пункты `TOC()`, `Section(...).ID` и `id` заголовков в `note.HTMLString()` — одна и та же строка, поэтому ссылка, собранная из любого из них, ведёт на нужный заголовок.

#### Списки и задачи

`FirstList()` и `Lists()` отдают списки заметки как данные: у пункта есть `Text`, `URL` (если пункт — ссылка), `Children` и два поля задачи:

| Пункт | `Task` | `TaskMark` |
|---|---|---|
| `- обычный` | `""` | `""` |
| `- [ ] открыта` | `"todo"` | `" "` |
| `- [x] готово` или `- [X] готово` | `"done"` | `"x"` / `"X"` |
| `- [/] начата`, `- [-] отменена`, `- [>] перенесена` | `"done"` | `"/"`, `"-"`, `">"` |

Любой символ кроме пробела считается `done` — так Obsidian показывает такой чекбокс отмеченным. В `TaskMark` лежит сам символ, по нему шаблон различает пользовательские статусы Obsidian. Чекбоксами на опубликованной странице становятся только `[ ]`, `[x]` и `[X]`: `- [/] начата` на странице выглядит как текст «[/] начата», а в `Text` — только «начата».

```jet
{{ if list := note.PartialRenderer().FirstList(); list }}
  <ul class="tasks">
    {{ range i, item := list.Items }}
      <li class="tasks__item tasks__item--{{ item.Task }}" data-mark="{{ item.TaskMark }}">{{ item.Text }}</li>
    {{ end }}
  </ul>
{{ end }}
```

#### Images() — картинки заметки

`Images()` возвращает все картинки заметки по порядку: markdown-картинки `![alt](url "title")` и встраивания Obsidian `![[photo.png]]`, с тем же URL, что и на странице (вложение превращается в адрес загруженного файла). У каждой есть `URL`, `Alt` и `Title`. Размер вида `|300` или `|300x200` в `Alt` не попадает. Видео, аудио, документы и встраивания YouTube — не картинки, их в списке нет.

Галерея:

```jet
<div class="gallery">
  {{ range i, img := note.PartialRenderer().Images() }}
    <figure>
      <img src="{{ img.URL }}" alt="{{ img.Alt }}" loading="lazy">
      {{ if img.Title }}<figcaption>{{ img.Title }}</figcaption>{{ end }}
    </figure>
  {{ end }}
</div>
```

#### CodeBlocks(lang) — блоки кода

`CodeBlocks("lang")` возвращает блоки кода (в тройных обратных кавычках) этого языка по порядку, включая блоки внутри списков и каллаутов; `CodeBlocks("")` — все блоки. У каждого:

- `Lang` — первое слово после открывающих кавычек (`mychart`);
- `Info` — вся строка после них (`mychart {"height": 300}`);
- `Content` — сам код, как написан;
- `HTML` — блок в том виде, в каком его рендерит страница.

Вместе с `parseJSON` блок кода становится данными для своего виджета. Заметка:

````markdown
```mychart
{"labels": ["Q1", "Q2", "Q3"], "values": [3, 5, 8]}
```
````

Шаблон:

```jet
{{ range i, b := note.PartialRenderer().CodeBlocks("mychart") }}
  {{ if d := parseJSON(b.Content); d }}
    <ul class="bars">
      {{ range j, label := d.labels }}
        <li style="--value: {{ d.values[j] }}">{{ label }}</li>
      {{ end }}
    </ul>
  {{ else }}
    {{ b.HTML }}
  {{ end }}
{{ end }}
```

В `note.HTMLString()` блок остаётся кодом. Чтобы показать только виджет, выводите секции сами или спрячьте код через CSS.

### Разбор данных: parseJSON, parseYAML, parseCSV {#parse-data}

Три функции превращают строку в данные, по которым шаблон может пройти циклом:

- `parseJSON(s)` — объекты становятся словарями (`d.title` или `d["title"]`), массивы — списками, числа — дробными;
- `parseYAML(s)` — то же для YAML;
- `parseCSV(s)` — список строк таблицы, каждая строка — список строковых значений. Строки могут быть разной длины. Первая строка заголовком не считается.

На неверных данных функция возвращает `nil` и пишет ошибку в лог сервера, рендер продолжается. Проверяйте результат перед использованием: `{{ if d := parseJSON(b.Content); d }} … {{ else }} запасной вариант {{ end }}`.

Как фильтр они работают в теге вывода — `{{ b.Content | parseJSON }}`, но Jet разрешает фильтры только там: в присваивании, `if` и `range` нужен вызов — `d := parseJSON(b.Content)`.

Таблица из блока ` ```csv `:

```jet
{{ if b := note.PartialRenderer().CodeBlocks("csv"); len(b) > 0 }}
  {{ if rows := parseCSV(b[0].Content); rows }}
    <table>
      {{ range i, row := rows }}
        <tr>{{ range j, cell := row }}<td>{{ cell }}</td>{{ end }}</tr>
      {{ end }}
    </table>
  {{ end }}
{{ end }}
```

Значения из заметки — это текст её автора, и layout экранирует их при выводе: заголовок `Q&A <черновик>` попадёт на страницу текстом, а не тегами. HTML, в который заметка отрендерена (`HTMLString()`, `TitleHTML`, `ContentHTML`, `HTML` блока кода), выводится как есть.

### Фильтр unsafe

Вывод экранируется по умолчанию: строка с `<p>` отобразится как текст `&lt;p&gt;`. Методы, которые возвращают готовый HTML, — `note.HTMLString()`, `TitleHTML`, `ContentHTML`, `FirstListHTML()`, `FormSpecJSON()`, `asset()` — выводятся как есть, фильтр им не нужен.

`| unsafe` выводит без экранирования любую строку. Нужен он, только когда разметку собирает сам шаблон или она лежит в обычной строке:

```jet
{{ block hero(title="") }}<h2>{{ title | unsafe }}</h2>{{ end }}
{{ yield hero(title="Данные там,<br>где вы решите.") }}
```

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

HTML-комментарий остаётся в исходном коде страницы (в нём будет итоговый URL), но посетителю не виден. Зависимость гарантированно попадёт в обнаружение.

### Синтаксис Jet

Шаблоны используют движок [[jet|Jet]]:

```jet
{{ переменная }}                       — вывод
{{ if условие }}...{{ end }}           — условие
{{ range _, item := список }}...{{ end }} — цикл
{{ block имя() }}...{{ end }}          — определение блока
{{ yield имя() }}                      — вызов блока
{{ include "путь" данные }}            — вставка шаблона
{{ x := exec("lib/имя", данные) }}     — выполнить другой файл и взять значение из его return
{{ try }}...{{ catch err }}...{{ end }} — заглушка, если внутренняя часть упала
{{ d := parseJSON(текст) }}            — разбор JSON, YAML или CSV в данные
```

Подробнее, включая `exec`/`return` и `try`/`catch` с примерами, — в [[jet|документации Jet]]. Все функции и фильтры, включая добавленные trip2g, — в [[ru/user/jet-functions|справочнике функций Jet]].


`parseJSON`, `parseYAML` и `parseCSV` описаны в разделе [[ru/user/templates#parse-data|Разбор данных]].

### Присваивание прямо в if (как в Go)

`if` умеет объявить переменную и сразу её проверить — как `if x := f(); cond` в Go:

```jet
{{ if имя := выражение; условие }}
  ...
{{ else }}
  ...
{{ end }}
```

Переменная живёт только внутри этого `if`: в его ветках `else if` и `else`. После `{{ end }}` её нет — обращение к ней обрывает рендер с ошибкой `identifier "имя" not available in current … scope`. Если до `if` уже была переменная с тем же именем, внутри `if` её заслоняет новая, а после `{{ end }}` старое значение на месте.

Типичный случай — заметка, которой может не быть:

```jet
{{ if about := nvs.ByPermalink("/about"); about }}
  <a href="{{ about.Permalink() }}">{{ about.Title() }}</a>
{{ else }}
  <span>Страница «О нас» ещё не опубликована</span>
{{ end }}
```

Условие — любое выражение, не только сама переменная:

```jet
{{ if subtitle := note.M().GetString("subtitle", ""); subtitle != "" }}
  <p class="subtitle">{{ subtitle }}</p>
{{ else }}
  <p class="subtitle">{{ note.Title() }}</p>
{{ end }}

{{ if latest := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").Desc().First(); latest }}
  Свежий пост: <a href="{{ latest.Permalink() }}">{{ latest.Title() }}</a>
{{ end }}

{{ if faq := note.PartialRenderer().Section("FAQ"); faq }}
  {{ faq.ContentHTML }}
{{ end }}
```

Каждый `else if` может объявить свою переменную и видит объявленные до него:

```jet
{{ if header := nvs.ByPath("/blog/_header.md"); header }}
  {{ header.HTMLString() }}
{{ else if fallback := nvs.ByPath("/_header.md"); fallback }}
  {{ fallback.HTMLString() }}
{{ end }}
```

Работает и форма с двумя значениями: `{{ if value, ok := someMap["key"]; ok }}`.

С `=` вместо `:=` тег присваивает значение переменной, объявленной раньше, и оно остаётся после `{{ end }}`.

**`range` устроен иначе.** `{{ range i, post := список }}` тоже объявляет переменные, видимые только в цикле, но `; условие` не принимает — `{{ range p := список; p }}` не разбирается. Переменные — это индекс и элемент, а не результат выражения: с одной переменной по списку вы получите индекс. `{{ range … }}{{ else }}…{{ end }}` выполняет `else`, когда список пуст.

Подробнее — в [[jet|документации Jet]].

Выборка и сортировка заметок — в [[templates-advanced|запросах к заметкам]].

Как собирать шаблон из компонентов — в [[ru/user/components|Компонентах, автоимпорте и лучших практиках]].
