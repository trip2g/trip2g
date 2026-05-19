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
