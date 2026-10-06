---
title: Справочник функций Jet
free: true
lang_redirect: "[[en/user/jet-functions]]"
---

Все функции, фильтры и конструкции, которые работают в своём Jet-шаблоне в `_layouts/`. Каждая проверена на той версии Jet, что стоит в trip2g (CloudyKit/jet v6.3.1), и на рендерере trip2g.

Страница из двух частей. Первая — полный справочник по **самому Jet**: встроенные функции, экранирование, конструкции языка. Вторая — **указатель того, что добавляет trip2g** (`note`, `nvs`, `asset()` и остальное), со ссылкой на страницу, где описан каждый пункт.

Если вы ещё не делали свой шаблон, начните с [[ru/user/templates|Шаблонов]], а страницы собирайте из компонентов, как советует страница [[ru/user/components|Компоненты шаблонов]]. Если что-то выводится не так — [[ru/user/jet-debugging|Отладка Jet-шаблонов]].

### На что похож Jet

Jet — движок шаблонов для Go. Синтаксис близок к `text/template` из Go, плюс то, что взято из Jinja2 и Twig: наследование шаблонов через `extends` и блоки, фильтры через `|`, тернарный оператор. Если вы знаете что-то из этого, читайте Jet как знакомый движок с отличиями ниже.

Как в шаблонах Go: скобки `{{ … }}`, `if` / `else if` / `else` / `end`, `range` по спискам и картам с веткой `else` для пустой коллекции, `if` с объявлением переменной.

Чем отличается от шаблонов Go:

| | `text/template` в Go | Jet |
|---|---|---|
| Переменные | `$x := .Title` | `x := note.Title()`, без `$` |
| Методы | `.Title`, вызывается сам | `note.Title()`, скобки обязательны |
| Операторы | функции: `eq`, `and`, `not` | `==`, `&&`, `!`, `+`, `условие ? a : b` |
| `if` с объявлением | `if $x := f`, проверяет `$x` | `if x := f(); условие`, любое условие |
| `range` с одной переменной | даёт элемент | даёт **индекс**; пишите `range i, x :=` |
| Пайп `a \| f b` | `a` становится **последним** аргументом | `a` становится **первым** аргументом |
| Повторяемый кусок | `define` + `template "name" .` | `block name(p="")` + `yield name(p=…)` |
| Наследование шаблонов | `extends` нет | `extends`; страница переопределяет блоки |
| Комментарии | `{{/* … */}}` | `{* … *}` |
| Экранирование | `html/template` учитывает контекст | одно и то же HTML-экранирование везде |

Как в Jinja2 и Twig: `extends` с блоками, которые страница переопределяет, `import` и `include`, фильтры с аргументами (`"a-b" | replace: "-", " ", -1`), `условие ? a : b` как в Twig. В отличие от них, все теги пишутся в `{{ }}` (`{% %}` нет), цикл — это `range`, а не `for x in list`, и блок принимает именованные параметры, как макрос в Jinja.

Экранирование не смотрит на контекст. Внутри `<script>` используйте `json` или `safeJs`, они описаны ниже.

### Вывод по умолчанию экранируется

Это главное. `{{ value }}` экранирует HTML, поэтому заголовок вроде `Q&A <черновик>` попадёт на страницу текстом, а не тегами:

```jet
{{ "<b>bold</b>" }}            {* → &lt;b&gt;bold&lt;/b&gt; *}
{{ "<b>bold</b>" | unsafe }}   {* → <b>bold</b> *}
```

Что из этого следует:

- Методы и поля, которые возвращают готовый HTML сервера, выводятся как есть, без фильтра: `note.HTMLString()`, `FirstListHTML()`, `FormSpecJSON()`, `SubgraphNamesJSON()`, `TitleHTML` / `ContentHTML` секции, `HTML` блока кода, функции `defaultTemplate.*` и `asset()`.
- `| unsafe` и `| raw` выводят обычную строку без экранирования. Нужны они только для разметки, которую собирает сам шаблон или которая лежит в строке, и для `injection.Content` [[ru/user/templates#HTML-инъекции|HTML-инъекций]] сайта.
- Для текста `| html` больше не нужен. Он экранирует один раз и помечает результат как безопасный, поэтому `{{ title | html }}` в старом шаблоне выводит то же, что `{{ title }}`, а не дважды экранированное `&amp;amp;`.
- Строковая функция или `+` над безопасным выводом возвращают обычную строку, и она экранируется: `{{ upper(note.HTMLString()) }}` выведет теги текстом.

Фильтры сравниваются ниже, в разделе [[#Экранирование]].

### Встроенные функции Jet

| Функция | Возвращает |
|---|---|
| `len(x)` | Длину строки (в байтах), среза, массива или карты; число полей структуры |
| `isset(a, b, …)` | `true`, если все аргументы определены и не nil |
| `lower(s)`, `upper(s)` | Строку в нижнем / верхнем регистре |
| `hasPrefix(s, prefix)`, `hasSuffix(s, suffix)` | `bool` |
| `trimSpace(s)` | Строку без пробелов по краям |
| `repeat(s, n)` | `s`, повторённую `n` раз |
| `replace(s, old, new, n)` | `s` с заменой первых `n` совпадений; `n = -1` — всех |
| `split(s, sep)` | `[]string` |
| `map(k1, v1, k2, v2, …)` | `map[string]interface{}` |
| `slice(a, b, …)`, `array(a, b, …)` | `[]interface{}` |
| `ints(from, to)` | Последовательность `from … to-1` для `range` |
| `json(v)` | JSON значения `v`, компактный, выводится без HTML-экранирования |
| `writeJson(v)` | Пишет JSON значения `v` в вывод, с переводом строки в конце |
| `includeIfExists(path, ctx?)` | Рендерит шаблон, если он есть; возвращает невидимый `true`/`false` |
| `exec(path, ctx?)` | Значение, которое другой файл шаблона отдал через `return`; его вывод отбрасывается |
| `dump()`, `dump("name")` | Текстовый дамп контекста, переменных и глобальных имён |

Фильтры экранирования (`html`, `url`, `safeHtml`, `safeJs`, `raw`, `unsafe`) — в [[#Экранирование|отдельной таблице]].

#### Три способа вызвать функцию

Функцию можно вызвать напрямую или через пайп. В пайпе значение слева становится **первым** аргументом, остальные пишутся после двоеточия:

```jet
{{ upper("hello") }}                {* → HELLO *}
{{ "hello" | upper }}               {* → HELLO *}
{{ "a-b-c" | replace: "-", " ", -1 }}
{* replace("a-b-c", "-", " ", -1) → a b c *}
{{ "Hello" | hasPrefix: "He" }}     {* → true *}
{{ "a-b" | replace: "-", " ", -1 | upper }}
{* pipes chain → A B *}
```

`{{ 3 | repeat: "ab" }}` падает: это вызов `repeat(3, "ab")`. Пишите `repeat("ab", 3)`.

#### Строки

```jet
{{ lower("ABC") }}                  {* → abc *}
{{ trimSpace("  x  ") }}            {* → x *}
{{ repeat("ab", 3) }}               {* → ababab *}
{{ replace("aaa", "a", "b", 2) }}   {* → bba *}

{{ range i, part := split("a,b,c", ",") }}
  [{{ part }}]
{{ end }}
{* → [a] [b] [c] *}

{{ if hasSuffix(note.Permalink(), "/faq") }}
  <a href="/faq/contact">Ask a question</a>
{{ end }}
```

`len` считает байты, а не буквы: `len("héllo")` — это `6`, `len("привет")` — `12`.

#### `isset`

```jet
{{ m := map("a", 1) }}
{{ isset(m["a"]) }}        {* → true *}
{{ isset(m["z"]) }}        {* → false: missing map key *}
{{ isset(undefinedVar) }}  {* → false, no error *}
```

`isset` — единственное место, где неопределённая переменная не ошибка. В остальных случаях `{{ undefinedVar }}` останавливает рендер с `identifier "undefinedVar" not available`.

#### `map`, `slice`, `ints`

```jet
{{ links := map("docs", "/docs", "blog", "/blog") }}
{{ links["blog"] }}  {{ links.blog }}   {* both → /blog *}

{{ range i, n := ints(1, 4) }}
  {{ n }}
{{ end }}
{* → 1 2 3 *}
```

- Ключи `map` — строки, аргументов должно быть чётное число.
- `ints(from, to)` не включает `to`, и `from` должен быть меньше `to`. `ints(3, 1)` — ошибка рендера.
- По числу `range` не ходит: `{{ range i := 3 }}` падает. Пишите `ints(0, 3)`.

#### Индекс и срез

`list[from:to]` берёт элементы с `from` до `to`, не включая `to`. Любую границу можно опустить, и обе могут быть переменными или выражениями:

```jet
{{ colors := slice("red", "green", "blue", "black") }}
{{ colors[1] }}             {* → green *}
{{ colors[1:3] }}           {* → [green blue] *}

{{ from := 1 }}
{{ to := len(colors) - 1 }}
{{ colors[from:to] }}       {* → [green blue] *}
{{ colors[from:] }}         {* → [green blue black] *}
{{ colors[:to] }}           {* → [red green blue] *}
```

Строки режутся так же, по байтам: `{{ word := "hello" }}{{ word[1:4] }}` выведет `ell`. Строковый литерал напрямую не режется: `"hello"[1:4]` не загрузится.

Ставьте пробелы вокруг `-` в индексе: `colors[len(colors) - 2:]` работает, а `colors[len(colors)-2:]` не загрузится — Jet прочитает `-2` как число.

#### `json` и `writeJson`

```jet
<script>
  window.pageData = {{ json(map(
    "title", note.Title(),
    "tags", note.Tags()
  )) }};
</script>
```

```jet
<script type="application/json" id="data">
  {{ writeJson(note.M().Raw()) }}
</script>
```

Обе экранируют `<`, `>` и `&` как `<`…, поэтому результат безопасен внутри `<script>`, и экранирование страницы его не трогает. `writeJson` добавляет перевод строки в конце, `json` — нет.

Тег может занимать несколько строк внутри скобок, как вызов `json(map(…))` выше: длинный список аргументов переносите после запятой. Цепочку методов перед `.` перенести нельзя. Длинную цепочку разбейте на переменные (см. [[#Пример: последние посты со счётчиком|пример]]).

#### `includeIfExists` и `exec`

```jet
{{ includeIfExists("partials/banner") }}

{{ if !includeIfExists("partials/sidebar", note) }}
  <p>No sidebar</p>
{{ end }}
```

`includeIfExists` рендерит шаблон, если он существует, и возвращает булево значение, которое ничего не печатает. Второй аргумент, если он есть, становится `.` внутри вставленного шаблона.

`exec("lib/featured", данные)` выполняет другой файл шаблона, отбрасывает всё, что он напечатал, и возвращает значение его `{{ return }}`. Внутри файла `данные` доступны как `.`, а `note`, `nvs` и другие переменные страницы работают. Путь отсчитывается от `_layouts/` и пишется **без расширения**: `exec("lib/featured")` работает, `exec("lib/featured.html")` падает с `template /lib/featured.html could not be found`. `exec` — для данных (список, словарь, число), для разметки — компоненты. Примеры и правила путей — в [[ru/user/jet#exec и return — данные из другого файла|Синтаксисе Jet: exec и return]].

#### `dump`

```jet
<pre>{{ dump() }}</pre>       {* контекст, переменные, глобальные, блоки *}
<pre>{{ dump("x") }}</pre>    {* одна переменная: "x:=5 // float64" *}
```

`dump` показывает *имена*. Чтобы увидеть тип, значение и методы объекта, используйте [[ru/user/jet-debugging|debug()]] из trip2g.

### Экранирование

| Фильтр | Что делает | Для чего |
|---|---|---|
| `html` | Функция: один раз экранирует `<`, `>`, `&`, `'`, `"` и возвращает безопасное значение | Показать HTML-код текстом; сохранить экранированное значение в переменную |
| `safeHtml` | То же экранирование, пишет прямо в вывод | Ничего такого, чего не делает `{{ value }}` |
| `url` | Функция: экранирование для query (`a b&c` → `a+b%26c`), возвращает безопасное значение | Одно значение параметра: `?q={{ term \| url }}` |
| `safeJs` | Экранирование JS-строки (`'` → `\'`, `<` → `<`) | Значение внутри строкового литерала JS |
| `raw` / `unsafe` | Без экранирования, пишет прямо в вывод | Разметка, которую собирает сам шаблон или которая лежит в обычной строке |

```jet
{{ s := "<b>Tom & Jerry</b>" }}
{{ s }}               {* → &lt;b&gt;Tom &amp; Jerry&lt;/b&gt; *}
{{ s | html }}        {* the same *}
{{ s | safeHtml }}    {* the same *}
{{ s | unsafe }}      {* → <b>Tom & Jerry</b> *}
{{ s | raw }}         {* the same as unsafe *}
```

**`html` или `safeHtml`.** Для текста — ни тот, ни другой: `{{ s }}` уже экранирует. Разница в том, что они такое. `html` — функция, она возвращает значение, поэтому работает везде, где работает выражение, и результат второй раз не экранируется: `{{ e := html(s) }}` можно вывести позже, в тексте или в атрибуте. `safeHtml`, `safeJs`, `raw` и `unsafe` — писатели: они работают только последним фильтром в теге вывода. `{{ t := s | unsafe }}` не загрузится, а `safeHtml(s)` упадёт при рендере. Единственная работа, которая осталась у `html`, — показать разметку текстом: `{{ note.HTMLString() }}` рендерит заметку, а `{{ html(note.HTMLString()) }}` печатает её теги, чтобы читатель их увидел.

```jet
{{ e := html(s) }}
<p title="{{ e }}">{{ e }}</p>
{* both escaped once: &lt;b&gt;Tom &amp; Jerry&lt;/b&gt; *}

<pre>{{ html(note.HTMLString()) }}</pre>
{* the note's HTML as visible tags *}
```

**`url`** экранирует одно значение параметра. Он экранирует и `/`, `:`, `&`, поэтому не годится для целого адреса. Как и `html`, это функция, и её результат второй раз не экранируется:

```jet
{{ term := "a b&c/d" }}
<a href="/search?q={{ term | url }}">Search</a>
{* → href="/search?q=a+b%26c%2Fd" *}
```

**`raw` и `unsafe`** — один и тот же фильтр. Используйте его только для разметки, которой доверяете: собственные строки шаблона, `injection.Content`. Никогда — для текста, который написал автор или посетитель.

**`safeJs`** экранирует значение для строкового литерала JavaScript:

```jet
<script>
  var title = '{{ note.Title() | safeJs }}';
</script>
```

### Конструкции языка

#### Вывод, комментарии, пробелы

```jet
{{ note.Title() }}                  {* print an expression *}
{* a comment, never rendered *}
<li>   {{- note.Title() -}}   </li>
{* {{- and -}} trim whitespace on that side *}
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
| Индекс / срез | `m["key"]`, `m.key`, `list[0]`, `list[1:3]`, `list[from:to]`, `str[1:3]` |

- Числовые литералы — дробные: `{{ 7 / 2 }}` выводит `3.5`.
- `+` со строкой склеивает: `{{ "страница " + 2 }}` → `страница 2`.
- `""`, `0`, `false` и `nil` в `if` и `?:` — ложь: `{{ s ? s : "нет" }}`. **Пустой список — истина**, поэтому списки проверяйте через `len(list) > 0`.
- `||` возвращает булево значение, а не первое непустое. Для значения по умолчанию есть [[#Глобальные функции|coalesce()]] из trip2g.

#### `if`

```jet
{{ if n == 1 }}
  one
{{ else if n == 2 }}
  two
{{ else }}
  many
{{ end }}

{{ if v, ok := links["blog"]; ok }}
  <a href="{{ v }}">Blog</a>
{{ end }}
```

#### `range`

```jet
{{ range i, tag := note.Tags() }}
  <span>{{ tag }}</span>
{{ end }}

{{ range key, value := links }}
  {{ key }} → {{ value }}
{{ end }}

{{ range i, post := posts }}
  <a href="{{ post.Permalink() }}">{{ post.Title() }}</a>
{{ else }}
  <p>Nothing here.</p>
{{ end }}
```

- **С одной переменной `range` отдаёт индекс, а не значение.** `{{ range tag := note.Tags() }}` даст `0, 1, 2…`. Всегда пишите `range i, value :=`.
- `{{ else }}` выводится, если коллекция пуста.
- Ненужную переменную назовите `_`: `range _, tag := note.Tags()`.

#### `block` и `yield`

```jet
{{ block card(title="", url="") }}
  <a class="card" href="{{ url }}">{{ title }}</a>
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
  <section>
    <h2>{{ title }}</h2>
    {{ yield content }}
  </section>
{{ end }}

{{ yield panel(title="Related") content }}
  <p>Anything here becomes the panel body.</p>
{{ end }}
```

Значение после вызова становится `.` внутри блока: `{{ yield menu() items }}`.

#### `import`, `include`, `extends`

```jet
{{ import "blocks" }}
{* load the blocks of _layouts/blocks.html, render nothing *}

{{ include "partials/footer" }}
{* render another template here *}

{{ include "partials/card" note }}
{* … with `note` as `.` inside it *}
```

`extends` наследует базовый шаблон и заполняет его блоки:

```jet
{{ extends "base" }}

{{ block main() }}
  <p>This replaces the main block of base.html</p>
{{ end }}
```

- Пути — относительно `_layouts/`, без `.html`.
- `extends` должен быть первым тегом файла, за ним — `import`, если они есть.
- Как база и страница складываются вместе, с проверенным примером, — в [[ru/user/templates#Наследование шаблонов: extends|Шаблонах: наследование]].
- Кроме того, trip2g сам импортирует блоки, когда страница вызывает блок из другого файла шаблона. Шаблон, до которого дошли через `include` или `extends`, этого не получает и импортирует компоненты сам через `{{ import }}`. См. [[ru/user/components#Автоимпорт|Компоненты шаблонов: автоимпорт]].

#### `return` и `try` / `catch`

```jet
{* lib/featured.html: the value exec gets *}
{{ return nvs.ByGlob(. + "/*.md").Public().Limit(3).All() }}
```

```jet
{{ try }}
  {{ stats := exec("lib/stats", "blog") }}
  <p>{{ stats.count }} posts</p>
{{ catch err }}
  {{ if currentUser.IsAdmin() }}
    <p>{{ err.Error() }}</p>
  {{ end }}
{{ end }}
```

- `return` отдаёт значение `exec`. В обычной странице он ничего не останавливает и ничего не выводит.
- `try` рендерит своё тело; если внутри что-то упало, вывод тела выбрасывается и рендерится `catch`. Без `catch` упавший `try` ничего не выводит. Шаблон, который не загрузился, `try` не ловит.

Примеры — в [[ru/user/jet#exec и return — данные из другого файла|exec и return]] и [[ru/user/jet#Обработка ошибок: try / catch|try / catch]].

### Что добавляет trip2g

trip2g регистрирует восемь глобальных функций, передаёт каждому своему шаблону набор переменных и подставляет два плейсхолдера в исходник шаблона. У каждого пункта ниже — ссылка на страницу, где он описан. То, что нигде больше не описано, коротко описано здесь.

#### Глобальные функции

| Функция | Возвращает | Где описана |
|---|---|---|
| `asset("file.css")` | URL файла рядом с шаблоном; если такого файла нет — аргумент как есть | [[ru/user/yield_blocks\|yield_blocks]], [[ru/user/templates\|Шаблоны]] |
| `debug(expr)` | Go-тип, значение и список методов `expr` | [[ru/user/jet-debugging\|Отладка Jet-шаблонов]] |
| `yield_blocks("prefix")` | Выводит все блоки, чьё имя начинается с префикса (или подходит под `/regex/`) | [[ru/user/yield_blocks\|yield_blocks]] |
| `coalesce(a, b, …)` | Первый аргумент, который задан и не пуст; иначе последний | Ниже |
| `arg_type("param", "type", "comment")` | Ничего; описывает параметр блока для инструментов | Ниже |
| `parseJSON(s)`, `parseYAML(s)`, `parseCSV(s)` | Текст в виде словарей и списков (CSV — список строк таблицы); `nil` на неверных данных | [[ru/user/templates#parse-data\|Шаблоны: разбор данных]] |

`coalesce` считает пустыми отсутствующий ключ карты, `nil`, `""`, пустой список и пустую карту. `0` и `false` — значения:

```jet
{{ videos := map("en", "intro-en.mp4") }}
{{ coalesce(videos[note.Lang()], videos["en"]) }}
{{ coalesce(
  note.M().GetString("subtitle", ""),
  note.Description(),
  note.Title()
) }}
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
| `note` | Текущая страница | [[ru/user/templates\|Шаблоны]], полный список методов — [[ru/user/templates-advanced\|Шаблоны: API]] |
| `nvs` | Все заметки сайта: поиск и выборки | [[ru/user/templates-advanced\|Шаблоны: API]] |
| `title` | Заголовок страницы после шаблона заголовка сайта — готов для `<title>` | Ниже |
| `publicURL` | Адрес основного домена, например `https://example.com` | [[ru/user/templates#SEO-теги в своём layout\|Шаблоны: SEO-теги]] |
| `htmlInjectionsHead`, `htmlInjectionsBodyEnd` | Фрагменты, которые админ добавил в Админке → SEO & URLs → HTML Injections; выводите `injection.Content` | [[ru/user/templates#HTML-инъекции\|Шаблоны: HTML-инъекции]] |
| `defaultTemplate.Header()`, `.Footer()` | HTML шапки / подвала дефолтного шаблона; `""`, если у страницы их нет | Ниже |
| `defaultTemplate.Styles()` | Теги `<link>` со стилями дефолтного шаблона | Ниже |
| `defaultTemplate.UserSpaceScripts()` | `<script>` с настройками и скрипты, нужные виджетам на клиенте | Ниже |
| `currentUser.IsAdmin()` | `true`, если страницу смотрит админ сайта | Ниже; используется в [[ru/user/themes\|Темах]] |

Чтобы обернуть свой контент в шапку, подвал и стили дефолтного шаблона:

```jet
<head>
  <title>{{ title }}</title>
  {{ defaultTemplate.Styles() }}
  {{ defaultTemplate.UserSpaceScripts() }}
</head>
<body>
  {{ defaultTemplate.Header() }}
  <main>{{ note.HTMLString() }}</main>
  {{ defaultTemplate.Footer() }}
  {{ if currentUser.IsAdmin() }}
    <a href="/admin">Admin</a>
  {{ end }}
</body>
```

`UserSpaceScripts()` пишет объект настроек только при первом вызове, второй вызов его не дублирует. В превью шаблона (`/_system/renderlayout`) все четыре функции `defaultTemplate` возвращают `""`, а `currentUser.IsAdmin()` — `false`.

#### Плейсхолдеры в исходнике шаблона

`@lid` и `@did` в файле шаблона до разбора заменяются на id файла (`mesh/bar.html` → `mesh_bar` / `mesh-bar`, `my-theme/card.html` → `my_theme_card` / `my-theme-card`), чтобы имена блоков и CSS-классов не пересекались. См. [[ru/user/yield_blocks|yield_blocks]] и [[ru/user/bem|BEM-именование]].

#### Методы `note`, `note.M()`, `nvs` и выборок

Полный список — в [[ru/user/templates-advanced|Шаблонах: API]]: методы заметки (`Title`, `HTMLString`, `Tags`, `Lang`, `LangAlternativesList`, `HasCodeLanguage`…), доступ к frontmatter (`GetString`, `GetInt`, `GetBool`, `GetStrings`…), `nvs` (`ByPath`, `ByPermalink`, `ByWikilink`, `BackLinks`…), выборки (`ByGlob`, `SortBy`, `SortByMeta`, `Public`, `Limit`…) и `PartialRenderer`. Там же — что выборка возвращает без фильтров и как ведёт себя сортировка. `FormSpecJSON()` описан в [[ru/user/forms|Формах]], `debug()` и `note.M().Debug()` — в [[ru/user/jet-debugging|Отладке]].

`PermalinkEncoded()` — это `Permalink()`, в котором каждый сегмент пути закодирован процентами: `/привет мир` → `/%D0%BF…%20%D0%BC…`. Это кодирование URL, а не HTML-экранирование. Раз вывод экранируется, `Permalink()` тоже безопасен в `href`. `PermalinkEncoded()` нужен, когда в пути есть не-ASCII символы или пробелы и нужен строго закодированный адрес: в sitemap, ленте или ссылке, которую читает другая программа.

Языковой переключатель:

```jet
{{ range i, alt := note.LangAlternativesList() }}
  <a href="{{ alt.PermalinkEncoded() }}" hreflang="{{ alt.Lang() }}">
    {{ alt.LangName() }}
  </a>
{{ end }}
```

Две ловушки стоит знать заранее:

- Поиск, который ничего не нашёл (`ByPath`, `ByPermalink`, `ByWikilink`, `First()` / `Last()` выборки, `LangAlternative()`, `Section()`, `FirstList()`), возвращает `nil`. Работают и `{{ if x }}`, и `{{ if x == nil }}`, а `x.Title()` на нём останавливает рендер. Удобнее всего объявить и проверить в одном теге, как в примере ниже.
- `nvs.ByGlob(...)` без `.Public()` возвращает и платные, и закрытые, и системные (`_`) заметки.

```jet
{{ if about := nvs.ByPermalink("/about"); about }}
  <a href="{{ about.PermalinkEncoded() }}">{{ about.Title() }}</a>
{{ end }}
```

### Пример: последние посты со счётчиком

Список, который берёт пять самых новых публичных заметок из `blog/`, сортирует по полю `date` из frontmatter и показывает, сколько их всего:

```jet
{{ query := nvs.ByGlob("blog/*.md").Public() }}
{{ total := len(query.All()) }}
{{ sorted := query.SortByMeta("date").Desc().SortBy("Title") }}
{{ posts := sorted.Limit(5).All() }}
<h2>Latest posts</h2>
<p>Showing {{ len(posts) }} of {{ total }}</p>
<ul>
{{ range i, post := posts }}
  <li>
    <a href="{{ post.PermalinkEncoded() }}">{{ post.Title() }}</a>
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
- Сортировка лежит в своей переменной `sorted`, потому что цепочку методов нельзя продолжить на следующей строке.
- `date: 2024-05-10` приходит строкой, а даты в формате ISO сортируются как строки правильно. Заметки без `date` уходят в конец.
- `.SortBy("Title")` упорядочивает посты с одинаковой датой. Без него их порядок менялся бы от рендера к рендеру.
- Заголовок вроде `Q&A` экранируется при выводе, фильтр не нужен.

Этот шаблон рендерит тест в репозитории trip2g (`internal/layoutloader/jet_functions_example_test.go`), поэтому пример не устареет незаметно. Короткие примеры на этой странице рендерит `internal/layoutloader/jet_functions_snippets_test.go`. Код примеров одинаков на русской и английской странице, поэтому текст в них английский: тест сверяет обе страницы.

### Смотрите также

- [[ru/user/templates|Шаблоны]] — как устроены шаблоны
- [[ru/user/components|Компоненты шаблонов]] — как устроить шаблон
- [[ru/user/spa|Приложение поверх trip2g]] — шаблон, который отдаёт JavaScript-приложение
- [[ru/user/templates-advanced|Шаблоны: API]] — методы `note`, `nvs`, выборок и frontmatter
- [[ru/user/jet|Синтаксис Jet]] — обзор синтаксиса
- [[ru/user/jet-debugging|Отладка Jet-шаблонов]]
- [[ru/user/yield_blocks|yield_blocks]] — компоненты, ассеты, `@lid` / `@did`
- [[ru/user/renderlayout|Превью лейаута]] — проверить шаблон без загрузки
- [Справочник синтаксиса Jet](https://github.com/CloudyKit/jet/blob/master/docs/syntax.md) — документация автора движка
