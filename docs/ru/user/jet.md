---
free: true
title: Синтаксис Jet
---

Jet — движок шаблонов. Поддерживает наследование, блоки и фильтры.

[Документация разработчика](https://github.com/CloudyKit/jet/wiki)

### Вывод переменных

Двойные фигурные скобки выводят значение:

```jet
{{ note.Title }}
{{ user.Firstname }}
```

Доступ к полям структур, методам, элементам массивов и карт:

```jet
{{ user.Fullname() }}
{{ items[0] }}
{{ config["theme"] }}
```

### Объявление переменных

```jet
{{ item := items[0] }}
{{ name := "Иван" }}
```

### Комментарии

```jet
{* это комментарий, не попадёт в HTML *}
```

### Условия

#### if / else / else if

```jet
{{ if note.Title }}
  <h1>{{ note.Title }}</h1>
{{ else }}
  <h1>Без заголовка</h1>
{{ end }}
```

```jet
{{ if len(items) > 0 }}
  Есть элементы
{{ else if len(items) == 0 }}
  Пусто
{{ end }}
```

#### Объявление в условии

```jet
{{ if value, ok := myMap["key"]; ok }}
  {{ value }}
{{ end }}
```

#### Тернарный оператор

```jet
{{ note.Title ? note.Title : "Заголовок не задан" }}
```

### Циклы

#### range — перебор

> **Важно:** при одной переменной `range` отдаёт **индекс** (0, 1, 2…), а не значение.
> Чтобы получить значение, всегда указывайте две переменные.

```jet
{* item = 0, 1, 2 — это индексы, не значения! *}
{{ range item := items }}
  {{ item }}
{{ end }}
```

Чтобы получить значения — используйте две переменные:

```jet
{{ range idx, item := items }}
  {{ idx }}: {{ item }}
{{ end }}
```

Если индекс не нужен — поставьте `_` вместо первой переменной:

```jet
{{ range _, item := items }}
  {{ item }}
{{ end }}
```

Блок else — если коллекция пуста:

```jet
{{ range item := items }}
  {{ item }}
{{ else }}
  Список пуст
{{ end }}
```

Срезы (slice) — начало:конец, конец не включается:

```jet
{{ range item := items[1:3] }}
  {{ item }}
{{ end }}
```

### Операторы

#### Арифметика

`+`, `-`, `*`, `/`, `%`

```jet
{{ 1 + 2 * 3 }}
{{ (10 - 2) / 4 }}
```

#### Сравнение

`==`, `!=`, `<`, `>`, `<=`, `>=`

```jet
{{ if count > 0 }}...{{ end }}
```

#### Логические

`&&` (и), `||` (или), `!` (не)

```jet
{{ if isAdmin && isActive }}...{{ end }}
{{ if !isDeleted }}...{{ end }}
```

#### Конкатенация строк

```jet
{{ "Привет, " + user.Name + "!" }}
```

### Фильтры (пайплайны)

Фильтры преобразуют значения. Передаются через `|`:

```jet
{{ "ТЕКСТ" | lower }}
{{ name | upper }}
```

Цепочка фильтров:

```jet
{{ text | lower | trimSpace }}
```

#### Встроенные фильтры

| Фильтр | Описание |
|--------|----------|
| `lower` | В нижний регистр |
| `upper` | В верхний регистр |
| `trimSpace` | Убрать пробелы по краям |
| `split` | Разбить строку |
| `replace` | Заменить подстроку |
| `repeat` | Повторить строку |
| `hasPrefix` | Начинается с... |
| `hasSuffix` | Заканчивается на... |

#### Экранирование

| Фильтр | Описание |
|--------|----------|
| `html` | Экранировать HTML |
| `url` | Экранировать для URL |
| `unsafe` / `raw` | Без экранирования |
| `json` / `writeJson` | Преобразовать в JSON |
| `safeJs` | Безопасный вывод в JS |

### Функции

#### isset — проверка на существование

```jet
{{ isset(note.Title) ? note.Title : "Нет заголовка" }}
```

Работает и для проверки ключей в карте:

```jet
{{ if isset(config["theme"]) }}...{{ end }}
```

#### len — длина

```jet
{{ len(items) }}
{{ if len(text) > 100 }}...{{ end }}
```

### Блоки

Блоки — фрагменты, которые вызываете в разных местах.

#### Определение блока

```jet
{{ block card(title, content) }}
  <div class="card">
    <h3>{{ title }}</h3>
    <p>{{ content }}</p>
  </div>
{{ end }}
```

Параметры по умолчанию:

```jet
{{ block button(text, type="primary") }}
  <button class="btn-{{ type }}">{{ text }}</button>
{{ end }}
```

#### Вызов блока

**Важно:** параметры передаются по имени, не по позиции. Без имени — получите `false`.

```jet
{* Правильно — именованные параметры *}
{{ yield card(title="Заголовок", content="Текст") }}

{* Неправильно — позиционные параметры *}
{{ yield card("Заголовок", "Текст") }}  {* title и content будут false *}
```

Порядок параметров неважен:

```jet
{{ yield card(content="Текст", title="Заголовок") }}  {* работает *}
```

Параметры с дефолтами можно не передавать:

```jet
{{ yield button(text="Отправить") }}  {* type="primary" по умолчанию *}
{{ yield button(text="Отмена", type="secondary") }}
```

#### Блок с вложенным контентом

Определение:

```jet
{{ block link(href) }}
  <a href="{{ href }}">{{ yield content }}</a>
{{ end }}
```

Вызов:

```jet
{{ yield link(href="https://example.com") content }}
  Перейти на сайт
{{ end }}
```

#### Рекурсивные блоки

```jet
{{ block menu() }}
  <ul>
    {{ range item := . }}
      <li>
        {{ item.Name }}
        {{ if len(item.Children) > 0 }}
          {{ yield menu() item.Children }}
        {{ end }}
      </li>
    {{ end }}
  </ul>
{{ end }}

{{ yield menu() navItems }}
```

### Композиция шаблонов

#### import — импорт блоков

Загружает блоки из другого файла:

```jet
{{ import "blocks" }}
{{ yield main_layout() content }}
  ...
{{ end }}
```

#### include — вставка шаблона

Вставляет шаблон целиком с передачей данных:

```jet
{{ include "partials/user-card" user }}
```

Условная вставка:

```jet
{{ if ok := includeIfExists("sidebar"); !ok }}
  <p>Боковая панель не найдена</p>
{{ end }}
```

#### extends — наследование layout

Шаблон наследует layout и переопределяет блоки:

```jet
{{ extends "layouts/base" }}

{{ block title() }}Моя страница{{ end }}

{{ block content() }}
  <p>Контент страницы</p>
{{ end }}
```

`extends` должен быть первой строкой шаблона.

#### exec и return — данные из другого файла

`exec("путь", данные)` выполняет другой файл из `_layouts/` и возвращает значение, которое этот файл отдал через `return`. Всё, что файл напечатал, отбрасывается — назад приходит только значение. Внутри файла `данные` доступны как `.`, переменные страницы (`note`, `nvs`, …) работают как обычно.

`exec` — для **данных**, а не для разметки. Компонент (`{{ block }}` + `{{ yield }}`) выдаёт HTML; файл для `exec` выдаёт список, map или число, а вызывающий шаблон рисует их по-своему. Главный случай — универсальный способ задать выборку в одном месте: какие заметки раздела считаются «избранными», какая у раздела статистика. Файл вызывают несколько шаблонов, а правило живёт в одном месте.

**Избранные заметки раздела.** `_layouts/lib/featured.html`:

```jet
{{ return nvs.ByGlob(. + "/*.md").Public().SortByMeta("order").Limit(3).All() }}
```

Любой шаблон получает избранное нужного раздела:

```jet
<ul class="featured">
{{ range _, post := exec("lib/featured", "blog") }}
  <li><a href="{{ post.Permalink() }}">{{ post.Title() }}</a></li>
{{ end }}
</ul>
```

Поменять смысл «избранного» — другая сортировка, пять заметок вместо трёх, только с флагом во frontmatter — это одна правка в `lib/featured.html`.

**Статистика раздела в виде map.** `_layouts/lib/section_stats.html`:

```jet
{{ notes := nvs.ByGlob(. + "/*.md").Public().All() }}
{{ minutes := 0 }}
{{ range _, n := notes }}{{ minutes = minutes + n.ReadingTime() }}{{ end }}
{{ return map("count", len(notes), "minutes", minutes) }}
```

```jet
{{ stats := exec("lib/section_stats", "blog") }}
<p>{{ stats["count"] }} posts, {{ stats["minutes"] }} min of reading</p>
```

Что важно знать:

- **Путь отсчитывается от `_layouts/` и пишется без расширения.** `exec("lib/featured")` и `exec("/lib/featured")` загружают `_layouts/lib/featured.html` из любой папки. `exec("lib/featured.html")` файл *не найдёт*. Шаблонами считаются только файлы `.html` и `.html.json` внутри `_layouts/`.
- **Несуществующий файл роняет рендер** с ошибкой `template /lib/featured could not be found`. Если страница должна это пережить — оберните вызов в `try` (ниже).
- **`return` имеет смысл только в файле, который вызывают через `exec`.** В обычной странице он ничего не останавливает и ничего не выводит.
- **Компоненты внутри файла для `exec` нужно импортировать явно.** Автоимпорт работает только для самой страницы; файл, до которого дошли через `exec`, `include` или `extends`, должен сам написать `{{ import "/путь/к/компоненту" }}`.
- **Каждый вызов выполняет файл заново.** `exec` не кэшируется: `lib/section_stats` в цикле по двадцати разделам — это двадцать полных выборок на каждый рендер. Если значение зависит только от самой заметки, дешевле frontmatter; тяжёлые вычисления по многим заметкам лучше делать Go-хелпером.

### Обработка ошибок: try / catch

`{{ try }}…{{ catch err }}…{{ end }}` рендерит внутреннюю часть, а если в ней что-то упало — выбрасывает её вывод и рендерит часть `catch`. Остальная страница рендерится как обычно. Без `catch` упавший `try` не выводит ничего.

Оборачивайте виджет, входные данные которого вы не контролируете, — тогда одна плохая заметка покажет заглушку вместо страницы с ошибкой.

**Виджет графика из frontmatter.** Если `chart` нет или он кривой, читатель увидит маленькую карточку, а не сломанную страницу:

```jet
{{ try }}
  {{ chart := note.M().Get("chart") }}
  <figure class="chart">
    <figcaption>{{ chart["title"] }}</figcaption>
    {{ range _, v := chart["values"] }}<span class="bar" style="--v: {{ v }}"></span>{{ end }}
  </figure>
{{ catch err }}
  <div class="chart chart--broken">Chart unavailable: {{ err.Error() | html }}</div>
{{ end }}
```

**Необязательная связанная заметка.** Поле `related` во frontmatter может указывать на переименованную или удалённую заметку. Если поиск или любой вызов на ней упадёт, блока просто не будет:

```jet
{{ try }}
  {{ related := nvs.ByPath(note.M().GetString("related", "")) }}
  <aside class="related">See also: <a href="{{ related.Permalink() }}">{{ related.Title() }}</a></aside>
{{ end }}
```

**Строка статистики из файла-помощника.** Если `lib/section_stats` сломается, посетители ничего не увидят, а админ сайта увидит причину:

```jet
{{ try }}
  {{ stats := exec("lib/section_stats", "blog") }}
  <p>{{ stats["count"] }} posts, {{ stats["minutes"] }} min of reading</p>
{{ catch err }}
  {{ if currentUser.IsAdmin() }}<p class="admin-error">lib/section_stats: {{ err.Error() | html }}</p>{{ end }}
{{ end }}
```

Что держать в голове:

- **`catch` — для заглушек, а не чтобы прятать ошибки.** `try` вокруг половины страницы превращает любую ошибку в тишину. Оборачивайте минимальный кусок, который может законно упасть, и показывайте ошибку админам (как выше), чтобы сломанный виджет заметили. Обращение к `currentUser` делает страницу персонализированной — она не отдаётся из анонимного кэша страниц.
- **Экранируйте сообщение.** В `err.Error()` может попасть текст из заметки или URL — пропускайте его через `html`. Фильтр экранирует ровно один раз, включено экранирование вывода по умолчанию или нет.
- **`try` не ловит шаблон, который не загрузился.** Синтаксическая ошибка или `extends` не на своём месте останавливают шаблон до рендера; это видно как предупреждение шаблона, а не как `catch`.

### Отладка шаблонов

#### debug() — тип, значение и методы объекта

Глобальная функция `debug()` возвращает строку с Go-типом, значением и списком методов любого выражения:

```jet
{{ debug(note.M()) }}
{* → *templateviews.Meta: &{raw:map[title:My Page extra_content:[a b]]}
      methods: [Debug Get GetBool GetInt GetString GetStrings Has Raw] *}

{{ debug(note.Title()) }}
{* → string: My Page *}
```

> Всегда добавляйте скобки при передаче методов: `debug(note.Title())` — правильно, `debug(note.Title)` — вернёт ссылку на функцию.

#### Meta.Debug() — JSON frontmatter

```jet
{{ note.M().Debug() }}
{* → {"extra_content":["channels","prices"],"title":"My Page"} *}
```

Для полного гайда по отладке шаблонов через `/_system/renderlayout` — см. [[skills/check_templates]].

### Полезные ссылки

- [Официальная документация](https://github.com/CloudyKit/jet/wiki)
- [Синтаксис шаблонов](https://github.com/CloudyKit/jet/wiki/3.-Jet-template-syntax)
- [Встроенные функции](https://github.com/CloudyKit/jet/wiki/4.-Built-in-functions)
