---
title: Справочник функций Jet
free: true
lang_redirect: "[[en/user/jet-functions]]"
---

Все функции, фильтры и конструкции, которые работают в своём Jet-шаблоне в `_layouts/`. Каждая проверена на той версии Jet, что стоит в trip2g (CloudyKit/jet v6.3.1), и на рендерере trip2g.

Страница из двух частей. Первая — полный справочник по **самому Jet**: встроенные функции, экранирование, конструкции языка. Вторая — **указатель того, что добавляет trip2g** (`note`, `nvs`, `asset()` и остальное), со ссылкой на страницу, где описан каждый пункт.

Если вы ещё не делали свой шаблон, начните с [[ru/user/templates|Шаблонов]]. Если что-то выводится не так — [[ru/user/jet-debugging|Отладка Jet-шаблонов]].

### Вывод по умолчанию не экранируется

Это главное. trip2g создаёт набор шаблонов Jet с выключенным автоэкранированием: шаблоны вставляют готовый HTML заметок. `{{ value }}` выводит значение **как есть**:

```jet
{{ "<b>жирный</b>" }}            {* → <b>жирный</b> *}
{{ "<b>жирный</b>" | html }}     {* → &lt;b&gt;жирный&lt;/b&gt; *}
```

Что из этого следует:

- `| unsafe` и `| raw` в trip2g ничего не делают. `{{ note.HTMLString() }}` и `{{ note.HTMLString() | unsafe }}` выводят одно и то же. В примерах документации `| unsafe` оставлен: так видно, что HTML выводится намеренно.
- Текст, в котором могут оказаться `<`, `&` или `"`, экранируйте сами: заголовки, значения frontmatter, всё, что пишет посетитель или другой автор. В HTML-тексте и атрибутах — `| html`, в параметре query-строки — `| url`.

### Встроенные функции Jet

| Функция | Сигнатура | Возвращает |
|---|---|---|
| `len` | `len(x)` | Длину строки (в байтах), среза, массива или карты; число полей структуры |
| `isset` | `isset(a, b, …)` | `true`, если все аргументы определены и не nil |
| `lower` / `upper` | `lower(s)` | Строку в нижнем / верхнем регистре |
| `hasPrefix` / `hasSuffix` | `hasPrefix(s, prefix)` | `bool` |
| `trimSpace` | `trimSpace(s)` | Строку без пробелов по краям |
| `repeat` | `repeat(s, n)` | `s`, повторённую `n` раз |
| `replace` | `replace(s, old, new, n)` | `s` с заменой первых `n` совпадений; `n = -1` — всех |
| `split` | `split(s, sep)` | `[]string` |
| `map` | `map(k1, v1, k2, v2, …)` | `map[string]interface{}` |
| `slice` / `array` | `slice(a, b, …)` | `[]interface{}` |
| `ints` | `ints(from, to)` | Последовательность `from … to-1` для `range` |
| `json` | `json(v)` | JSON значения `v`, компактный |
| `writeJson` | `writeJson(v)` | Пишет JSON значения `v` в вывод, с переводом строки в конце |
| `includeIfExists` | `includeIfExists(path, ctx?)` | Рендерит шаблон, если он есть; возвращает невидимый `true`/`false` |
| `exec` | `exec(path, ctx?)` | Выполняет шаблон, не выводя результат (см. оговорку ниже) |
| `dump` | `dump()`, `dump("name")` | Текстовый дамп контекста, переменных и глобальных имён |

Фильтры экранирования (`html`, `url`, `safeHtml`, `safeJs`, `raw`, `unsafe`) — в [[#Экранирование|отдельной таблице]].

#### Три способа вызвать функцию

Функцию можно вызвать напрямую или через пайп. В пайпе значение слева становится **первым** аргументом, остальные пишутся после двоеточия:

```jet
{{ upper("привет") }}                     {* → ПРИВЕТ *}
{{ "привет" | upper }}                    {* → ПРИВЕТ *}
{{ "a-b-c" | replace: "-", " ", -1 }}     {* replace("a-b-c", "-", " ", -1) → a b c *}
{{ "Hello" | hasPrefix: "He" }}           {* hasPrefix("Hello", "He") → true *}
{{ "a-b" | replace: "-", " ", -1 | upper }} {* пайпы цепляются: → A B *}
```

`{{ 3 | repeat: "ab" }}` падает: это вызов `repeat(3, "ab")`. Пишите `repeat("ab", 3)`.

#### Строки

```jet
{{ lower("ABC") }}                    {* → abc *}
{{ trimSpace("  x  ") }}              {* → x *}
{{ repeat("ab", 3) }}                 {* → ababab *}
{{ replace("aaa", "a", "b", 2) }}     {* → bba *}
{{ range i, part := split("a,b,c", ",") }}[{{ part }}]{{ end }}   {* → [a][b][c] *}
{{ if hasSuffix(note.Permalink(), "/faq") }}…{{ end }}
```

`len` считает байты, а не буквы: `len("héllo")` — это `6`, `len("привет")` — `12`.

#### `isset`

```jet
{{ m := map("a", 1) }}
{{ isset(m["a"]) }}       {* → true *}
{{ isset(m["z"]) }}       {* → false: ключа нет *}
{{ isset(undefinedVar) }} {* → false, без ошибки *}
```

`isset` — единственное место, где неопределённая переменная не ошибка. В остальных случаях `{{ undefinedVar }}` останавливает рендер с `identifier "undefinedVar" not available`.

#### `map`, `slice`, `ints`

```jet
{{ links := map("docs", "/docs", "blog", "/blog") }}
{{ links["blog"] }}  {{ links.blog }}      {* оба → /blog *}

{{ colors := slice("red", "green") }}
{{ colors[1] }}                            {* → green *}
{{ colors[0:1] }}                          {* срез: → [red] *}

{{ range i, n := ints(1, 4) }}{{ n }} {{ end }}   {* → 1 2 3 *}
```

- Ключи `map` — строки, аргументов должно быть чётное число.
- `ints(from, to)` не включает `to`, и `from` должен быть меньше `to`. `ints(3, 1)` — ошибка рендера.
- По числу `range` не ходит: `{{ range i := 3 }}` падает. Пишите `ints(0, 3)`.

#### `json` и `writeJson`

```jet
<script>window.pageData = {{ json(map("title", note.Title(), "tags", note.Tags())) }};</script>
<script type="application/json" id="data">{{ writeJson(note.M().Raw()) }}</script>
```

Обе экранируют `<`, `>` и `&` как `<`…, поэтому результат безопасен внутри `<script>`. `writeJson` добавляет перевод строки в конце, `json` — нет.

#### `includeIfExists` и `exec`

```jet
{{ includeIfExists("partials/banner") }}
{{ if !includeIfExists("partials/sidebar", note) }}<p>Боковой панели нет</p>{{ end }}
```

`includeIfExists` рендерит шаблон, если он существует, и возвращает булево значение, которое ничего не печатает. Второй аргумент, если он есть, становится `.` внутри вставленного шаблона.

`exec` выполняет шаблон, отбрасывает его вывод и возвращает значение из `return`. В trip2g шаблон с `{{ return }}` не загружается (см. [[#Чего нет в шаблонах trip2g]]), так что `exec` вернуть нечего. Используйте `include` или `block`.

#### `dump`

```jet
<pre>{{ dump() }}</pre>       {* контекст, переменные, глобальные имена, блоки *}
<pre>{{ dump("x") }}</pre>    {* одна переменная: "x:=5 // float64" *}
```

`dump` показывает *имена*. Чтобы увидеть тип, значение и методы объекта, используйте [[ru/user/jet-debugging|debug()]] из trip2g.

### Экранирование

| Фильтр | Что делает | Для чего |
|---|---|---|
| `html` | Экранирует `<`, `>`, `&`, `'`, `"` | Текст и значения атрибутов в HTML |
| `safeHtml` | То же экранирование, пишет прямо в вывод | То же, что `html` |
| `url` | Экранирование для query (`a b&c` → `a+b%26c`) | Одно значение параметра: `?q={{ term \| url }}` |
| `safeJs` | Экранирование JS-строки (`'` → `\'`, `<` → `<`) | Значение внутри строкового литерала JS |
| `raw` / `unsafe` | В trip2g ничего: вывод и так не экранируется | Пометить, что HTML выводится намеренно |

```jet
<a href="/search?q={{ term | url }}" title="{{ note.Title() | html }}">{{ note.Title() | html }}</a>
<script>var title = '{{ note.Title() | safeJs }}';</script>
```

`url` не для целого адреса: он экранирует и `/`, и `:`. Только для значения одного параметра.

### Конструкции языка

#### Вывод, комментарии, пробелы

```jet
{{ note.Title() }}               {* вывести выражение *}
{* комментарий, в HTML не попадает *}
<li>   {{- note.Title() -}}   </li>   {* {{- и -}} срезают пробелы с этой стороны *}
```

Методы вызывайте со скобками. `{{ note.Title }}` без `()` выведет адрес функции, а не заголовок.

#### Переменные: `:=` и `=`

```jet
{{ count := 0 }}                 {* объявить *}
{{ count = count + 1 }}          {* присвоить существующей *}
```

- `=` для переменной, которую не объявили, — ошибка рендера: `could not assign "x" … variable "x" is uninitialised`.
- `:=` внутри `if` или `range` объявляет **новую** переменную, которая до `{{ end }}` закрывает внешнюю. Чтобы изменить внешнюю, пишите `=`:

```jet
{{ x := 1 }}{{ if true }}{{ x := 2 }}{{ end }}{{ x }}   {* → 1 *}
{{ x := 1 }}{{ if true }}{{ x = 2 }}{{ end }}{{ x }}    {* → 2 *}
```

#### Операторы и значения

| Вид | Операторы |
|---|---|
| Арифметика | `+ - * / %` |
| Сравнение | `== != < > <= >=` |
| Логика | `&& \|\| !` |
| Тернарный | `условие ? a : b` |
| Индекс / срез | `m["key"]`, `m.key`, `list[0]`, `list[1:3]`, `str[1:3]` |

- Числовые литералы — дробные: `{{ 7 / 2 }}` выводит `3.5`.
- `+` со строкой склеивает: `{{ "страница " + 2 }}` → `страница 2`.
- `""`, `0`, `false` и `nil` в `if` и `?:` — ложь: `{{ s ? s : "нет" }}`. **Пустой список — истина**, поэтому списки проверяйте через `len(list) > 0`.
- `||` возвращает булево значение, а не первое непустое. Для значения по умолчанию есть [[#Глобальные функции|coalesce()]] из trip2g.

#### `if`

```jet
{{ if n == 1 }}один{{ else if n == 2 }}два{{ else }}много{{ end }}

{{ if v, ok := links["blog"]; ok }}<a href="{{ v }}">Блог</a>{{ end }}
```

#### `range`

```jet
{{ range i, tag := note.Tags() }}<span>{{ tag }}</span>{{ end }}
{{ range key, value := links }}{{ key }} → {{ value }}{{ end }}
{{ range i, post := posts }}…{{ else }}<p>Пока пусто.</p>{{ end }}
```

- **С одной переменной `range` отдаёт индекс, а не значение.** `{{ range tag := note.Tags() }}` даст `0, 1, 2…`. Всегда пишите `range i, value :=`.
- `{{ else }}` выводится, если коллекция пуста.
- `_` в качестве имени переменной ломает загрузку шаблона. Назовите индекс, даже если он не нужен.

#### `block` и `yield`

```jet
{{ block card(title="", url="") }}
  <a class="card" href="{{ url }}">{{ title | html }}</a>
{{ end }}

{{ yield card(title="Документация", url="/docs") }}
{{ yield card(url="/blog", title="Блог") }}
```

- Аргументы передаются **по имени**. Позиционный аргумент — `yield card("Документация")` — игнорируется, параметр остаётся со значением по умолчанию.
- Задавайте значение по умолчанию каждому параметру. Блок, объявленный в самой странице, ещё и выводится там, где объявлен, без аргументов. Параметр без значения по умолчанию тогда падает с `missing name for block parameter`.
- `content` — зарезервированное слово, параметр так называть нельзя.

Блок может оборачивать содержимое. Блок выводит то, что передал вызывающий, через `{{ yield content }}`:

```jet
{{ block panel(title="") }}
  <section><h2>{{ title }}</h2>{{ yield content }}</section>
{{ end }}

{{ yield panel(title="Связанное") content }}
  <p>Всё, что здесь, станет телом панели.</p>
{{ end }}
```

Значение после вызова становится `.` внутри блока: `{{ yield menu() items }}`.

#### `import`, `include`, `extends`

```jet
{{ import "blocks" }}                    {* загрузить блоки из _layouts/blocks.html, ничего не выводя *}
{{ include "partials/footer" }}          {* вывести здесь другой шаблон *}
{{ include "partials/card" note }}       {* … с `note` в качестве `.` внутри *}
```

```jet
{{ extends "base" }}
{{ block body() }}<p>Заменяет блок body из base.html</p>{{ end }}
```

- Пути — относительно `_layouts/`, без `.html`.
- `import` и `extends` должны стоять в начале файла.
- Кроме того, trip2g сам импортирует блоки, когда страница вызывает блок из другого файла шаблона. См. [[ru/user/yield_blocks|yield_blocks]] и [[ru/user/templates-best-practices|Лучшие практики]].

#### Чего нет в шаблонах trip2g

При загрузке trip2g обходит синтаксическое дерево каждого шаблона, чтобы найти вызовы `asset()` и блоки. Обходчик Jet не знает трёх конструкций. Шаблон с любой из них не загружается, ошибка — `layout panic: unexpected node …`:

| Конструкция | Вместо неё |
|---|---|
| `{{ try }} … {{ catch err }} … {{ end }}` | Проверка через `if`: `{{ if x }}{{ x.Title() }}{{ end }}` |
| `{{ return value }}` (а значит, и `exec`) | `block` или `include` |
| `_` как переменная `range` | Любое имя: `range i, v :=` |

Админ видит ошибку на странице и в `/_system/renderlayout`. Остальные посетители получают дефолтный шаблон.

### Что добавляет trip2g

trip2g регистрирует пять глобальных функций, передаёт каждому своему шаблону набор переменных и подставляет два плейсхолдера в исходник шаблона. У каждого пункта ниже — ссылка на страницу, где он описан. То, что нигде больше не описано, коротко описано здесь.

#### Глобальные функции

| Функция | Возвращает | Где описана |
|---|---|---|
| `asset("file.css")` | URL файла рядом с шаблоном; если такого файла нет — аргумент как есть | [[ru/user/yield_blocks|yield_blocks]], [[ru/user/templates|Шаблоны]] |
| `debug(expr)` | Go-тип, значение и список методов `expr` | [[ru/user/jet-debugging|Отладка Jet-шаблонов]] |
| `yield_blocks("prefix")` | Выводит все блоки, чьё имя начинается с префикса (или подходит под `/regex/`) | [[ru/user/yield_blocks|yield_blocks]] |
| `coalesce(a, b, …)` | Первый аргумент, который задан и не пуст; иначе последний | Ниже |
| `arg_type("param", "type", "comment")` | Ничего; описывает параметр блока для инструментов | Ниже |

`coalesce` считает пустыми отсутствующий ключ карты, `nil`, `""`, пустой список и пустую карту. `0` и `false` — значения:

```jet
{{ videos := map("en", "intro-en.mp4") }}
{{ coalesce(videos[note.Lang()], videos["en"]) }}
{{ coalesce(note.M().GetString("subtitle", ""), note.Description(), note.Title()) }}
```

`arg_type` ничего не выводит. trip2g читает его при загрузке и сообщает инструментам, которые показывают список блоков, тип и описание параметра:

```jet
{{ block hero(title="", image="") }}
  {{ arg_type("title", "string", "Текст заголовка") }}
  {{ arg_type("image", "asset", "Фоновая картинка") }}
  …
{{ end }}
```

#### Переменные в каждом своём шаблоне

| Переменная | Что это | Где описана |
|---|---|---|
| `note` | Текущая страница | [[ru/user/templates|Шаблоны]], полный список методов — [[ru/user/templates-advanced|Шаблоны: API]] |
| `nvs` | Все заметки сайта: поиск и выборки | [[ru/user/templates-advanced|Шаблоны: API]] |
| `title` | Заголовок страницы после шаблона заголовка сайта — готов для `<title>` | Ниже |
| `publicURL` | Адрес основного домена, например `https://example.com` | [[ru/user/templates|Шаблоны]], раздел про SEO-теги |
| `htmlInjectionsHead`, `htmlInjectionsBodyEnd` | HTML-инъекции из настроек сайта; выводите `injection.Content` | [[ru/user/templates|Шаблоны]] |
| `defaultTemplate.Header()`, `.Footer()` | HTML шапки / подвала дефолтного шаблона; `""`, если у страницы их нет | Ниже |
| `defaultTemplate.Styles()` | Теги `<link>` со стилями дефолтного шаблона | Ниже |
| `defaultTemplate.UserSpaceScripts()` | `<script>` с настройками и скрипты, нужные виджетам на клиенте | Ниже |
| `currentUser.IsAdmin()` | `true`, если страницу смотрит админ сайта | Ниже; используется в [[ru/user/themes|Темах]] |

Чтобы обернуть свой контент в шапку, подвал и стили дефолтного шаблона:

```jet
<head>
  <title>{{ title | html }}</title>
  {{ defaultTemplate.Styles() }}
  {{ defaultTemplate.UserSpaceScripts() }}
</head>
<body>
  {{ defaultTemplate.Header() }}
  <main>{{ note.HTMLString() }}</main>
  {{ defaultTemplate.Footer() }}
  {{ if currentUser.IsAdmin() }}<a href="/admin">Админка</a>{{ end }}
</body>
```

`UserSpaceScripts()` пишет объект настроек только при первом вызове, второй вызов его не дублирует. В превью шаблона (`/_system/renderlayout`) все четыре функции `defaultTemplate` возвращают `""`, а `currentUser.IsAdmin()` — `false`.

#### Плейсхолдеры в исходнике шаблона

`@lid` и `@did` в файле шаблона до разбора заменяются на id файла (`mesh/bar.html` → `mesh_bar` / `mesh-bar`), чтобы имена блоков и CSS-классов не пересекались. См. [[ru/user/yield_blocks|yield_blocks]] и [[ru/user/bem|BEM-именование]].

#### Методы `note`, `note.M()`, `nvs` и выборок

Полный список — в [[ru/user/templates-advanced|Шаблонах: API]]: методы заметки (`Title`, `HTMLString`, `Tags`, `Lang`, `LangAlternativesList`, `HasCodeLanguage`…), доступ к frontmatter (`GetString`, `GetInt`, `GetBool`, `GetStrings`…), `nvs` (`ByPath`, `ByPermalink`, `ByWikilink`, `BackLinks`…), выборки (`ByGlob`, `SortBy`, `SortByMeta`, `Public`, `Limit`…) и `PartialRenderer`. Там же — что выборка возвращает без фильтров и как ведёт себя сортировка. `FormSpecJSON()` описан в [[ru/user/forms|Формах]], `debug()` и `note.M().Debug()` — в [[ru/user/jet-debugging|Отладке]].

Две ловушки стоит знать заранее:

- Результат поиска проверяйте через `if`, а не через `== nil`. Ненайденная заметка — типизированный nil: `x == nil` даёт `false`, а `x.Title()` на нём останавливает рендер.
- `nvs.ByGlob(...)` без `.Public()` возвращает и платные, и закрытые, и системные (`_`) заметки.

```jet
{{ about := nvs.ByPermalink("/about") }}
{{ if about }}<a href="{{ about.PermalinkEncoded() }}">{{ about.Title() | html }}</a>{{ end }}
```

### Пример: последние посты со счётчиком

Список, который берёт пять самых новых публичных заметок из `blog/`, сортирует по полю `date` из frontmatter и показывает, сколько их всего:

```jet
{{ query := nvs.ByGlob("blog/*.md").Public() }}
{{ total := len(query.All()) }}
{{ posts := query.SortByMeta("date").Desc().SortBy("Title").Limit(5).All() }}
<h2>Latest posts</h2>
<p>Showing {{ len(posts) }} of {{ total }}</p>
<ul>
{{ range i, post := posts }}
  <li>
    <a href="{{ post.PermalinkEncoded() }}">{{ post.Title() | html }}</a>
    <time>{{ post.M().GetString("date", "undated") }}</time>
  </li>
{{ else }}
  <li>No posts yet.</li>
{{ end }}
</ul>
```

Как это работает:

- `.Public()` отбрасывает платные заметки и черновики (`_`), поэтому `total` считает только то, что посетитель может открыть.
- `query.All()` выборку не меняет, поэтому тот же `query` используется ещё раз для сортировки. Изменил бы её только `First()`.
- `date: 2024-05-10` приходит строкой, а даты в формате ISO сортируются как строки правильно. Заметки без `date` уходят в конец.
- `.SortBy("Title")` упорядочивает посты с одинаковой датой. Без него их порядок менялся бы от рендера к рендеру.
- `| html` экранирует заголовки вроде `Q&A`: сам trip2g вывод не экранирует.

Этот шаблон рендерит тест в репозитории trip2g (`internal/layoutloader/jet_functions_example_test.go`), поэтому пример не устареет незаметно. Текст в шаблоне английский, потому что тест сверяет вывод дословно.

### Смотрите также

- [[ru/user/templates|Шаблоны]] — как устроены шаблоны
- [[ru/user/templates-advanced|Шаблоны: API]] — методы `note`, `nvs`, выборок и frontmatter
- [[ru/user/jet|Синтаксис Jet]] — обзор синтаксиса
- [[ru/user/jet-debugging|Отладка Jet-шаблонов]]
- [[ru/user/yield_blocks|yield_blocks]] — компоненты, ассеты, `@lid` / `@did`
- [[ru/user/renderlayout|Превью лейаута]] — проверить шаблон без загрузки
- [Вики Jet по синтаксису](https://github.com/CloudyKit/jet/wiki/3.-Jet-template-syntax) — документация автора движка
