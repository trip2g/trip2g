---
title: Jsonnet в trip2g
free: true
lang_redirect: "[[en/user/jsonnet]]"
---

Три возможности trip2g принимают не готовую настройку, а короткую программу: frontmatter-патч решает, какие поля добавить, валидатор — что не так с заметкой, преобразование вебхука — что отправить. Такие программы пишутся на [Jsonnet](https://jsonnet.org) — небольшом языке, который выдаёт JSON. На этой странице — та часть Jsonnet, которая нужна для trip2g, и места, где trip2g его запускает.

### Что такое Jsonnet

Любой JSON — уже правильный Jsonnet. Поверх JSON в нём есть переменные, условия, функции и циклы по спискам, а результат программы — всегда JSON: объект, список, строка или число. Побочных эффектов у Jsonnet нет. Программа не меняет файлы, не ходит в сеть и не читает часы, она только вычисляет значение.

trip2g запускает каждую программу в песочнице: `import` запрещён, а рекурсия глубже 500 вызовов останавливается с ошибкой, а не зависает.

### Где trip2g его использует

| Возможность | Что получает программа | Что возвращает |
|---|---|---|
| [[ru/user/frontmatter_patches\|Frontmatter-патчи]] | `meta` (frontmatter заметки), `path` | объект, поля которого сливаются с frontmatter |
| [[ru/user/validators\|Валидаторы заметок]] | `meta` (после патчей), `path`, `nodes`, `std.native('regexMatch')` | список сообщений `{level, msg, line?}` |
| Преобразование тела [[ru/user/change_webhooks\|вебхука]] | `std.extVar('change')`, `std.extVar('attached_notes')` — JSON-строками | тело запроса, которое уйдёт вместо стандартного |

В патчах и валидаторах программа лежит в блоке ```` ```jsonnet ```` внутри обычной заметки. Преобразование вебхука — это поле `transformJsonnet` у change или cron webhook в GraphQL API. Оно заменяет всё тело запроса и не сочетается с **Pass API key**. `attached_notes` есть, только если вебхук прикладывает заметки; чтение отсутствующей переменной останавливает программу с ошибкой `Undefined external variable`.

### Синтаксис за десять минут

#### Значения и комментарии

```jsonnet
// Комментарий. Обычный JSON плюс запятые в конце и ключи без кавычек.
{
  title: 'Hello',
  "tags": ['a', 'b'],
  draft: false,
  views: 3,
}
// → {"draft": false, "tags": ["a", "b"], "title": "Hello", "views": 3}
```

Строки пишутся в одинарных или двойных кавычках. Ключам кавычки не нужны, если это простые слова.

#### Переменные

`local` даёт значению имя. Объявление заканчивается `;`, и имя видно во всём, что идёт дальше:

```jsonnet
local site = 'trip2g';
local year = 2026;
{ footer: site + ' © ' + year }
// → {"footer": "trip2g © 2026"}
```

`+` склеивает строки, складывает числа, объединяет списки и сливает объекты. Число, прибавленное к строке, становится текстом.

#### Поля объекта

```jsonnet
local meta = { title: 'Post', tags: ['go'] };
{
  title: meta.title,
  hasTags: std.objectHas(meta, 'tags'),
  author: std.get(meta, 'author', 'unknown'),
}
// → {"author": "unknown", "hasTags": true, "title": "Post"}
```

`meta.author` падает, если поля нет. `std.get(meta, 'author', default)` вместо этого возвращает значение по умолчанию, а `std.objectHas` проверяет, есть ли поле. В патчах и валидаторах берите эти две функции для полей, которых у заметки может не быть.

#### Условия

```jsonnet
local meta = { status: 'draft' };
{
  visible: if meta.status == 'published' then true else false,
  label: if meta.status == 'draft' then 'Draft' else '',
}
// → {"label": "Draft", "visible": false}
```

`if` — это выражение со значением, как тернарный оператор. **Всегда пишите `else`:** без него ложное условие даёт `null`. Валидатор, который вернул `null` вместо списка, получает сообщение `CRITICAL`.

#### Списки и генераторы

```jsonnet
local tags = ['go', 'Draft', 'web'];
{
  lower: [std.asciiLower(t) for t in tags],
  noDrafts: [t for t in tags if t != 'Draft'],
  count: std.length(tags),
}
// → {"count": 3, "lower": ["go", "draft", "web"], "noDrafts": ["go", "web"]}
```

`[выражение for x in список if условие]` строит новый список. Валидаторы в основном так и пишутся: одно сообщение на каждый элемент, который нарушил правило.

#### Текст

```jsonnet
local name = 'Notes – 2026-10-18 – Cursor';
local parts = std.split(name, ' – ');
{
  parts: parts,
  msg: 'date %s, tool %s' % [parts[1], parts[2]],
  isNotes: std.startsWith(name, 'Notes'),
}
// → {"isNotes": true, "msg": "date 2026-10-18, tool Cursor",
//    "parts": ["Notes", "2026-10-18", "Cursor"]}
```

`'…%s…' % значение` подставляет значения, как `printf`: `%s` — для текста, `%d` — для целого числа. Если значений несколько, передайте список.

#### Функции

```jsonnet
local err(msg) = { level: 'error', msg: msg };
local missing(meta, f) = !std.objectHas(meta, f);
local meta = { title: 'Post' };
[err('missing ' + f) for f in ['title', 'description'] if missing(meta, f)]
// → [{"level": "error", "msg": "missing description"}]
```

Функция — это `local` с параметрами. Маленькие помощники вроде `err` делают валидатор короче.

#### Обход списка с состоянием

Когда каждый элемент зависит от предыдущих, например «в каком разделе эта картинка», нужен `std.foldl`. Он проходит по списку и несёт с собой накопитель:

```jsonnet
local nodes = [
  { type: 'heading', text: 'Intro' },
  { type: 'image' },
  { type: 'heading', text: 'Steps' },
];
std.foldl(
  function(acc, n)
    if n.type == 'heading' then acc + [{ title: n.text, images: 0 }]
    else acc[:std.length(acc) - 1]
      + [acc[std.length(acc) - 1] + { images: super.images + 1 }],
  nodes,
  [],
)
// → [{"images": 1, "title": "Intro"}, {"images": 0, "title": "Steps"}]
```

`acc[:n]` — срез: все разделы, кроме последнего. `obj + { images: super.images + 1 }` копирует объект с одним изменённым полем.

### Функции, которые нужны чаще всего

| Функция | Что делает |
|---|---|
| `std.length(x)` | длина списка, строки или объекта |
| `std.objectHas(obj, 'f')`, `std.get(obj, 'f', default)` | есть ли поле; поле или значение по умолчанию |
| `std.isString(x)`, `std.isArray(x)`, `std.isNumber(x)` | тип значения |
| `std.member(list, x)` | есть ли `x` в списке |
| `std.startsWith(s, p)`, `std.endsWith(s, p)` | начинается или заканчивается ли текст на `p` |
| `std.split(s, sep)`, `std.join(sep, list)` | разбить текст на список; собрать список в текст |
| `std.asciiLower(s)`, `std.asciiUpper(s)` | сменить регистр (латиница) |
| `std.substr(s, from, len)` | часть текста |
| `std.toString(x)` | любое значение как текст |
| `std.foldl(f, list, init)` | обход списка с накопителем |
| `std.parseJson(s)` | превратить JSON-строку в значение |

Своих регулярных выражений в Jsonnet нет. В валидаторах trip2g добавляет `std.native('regexMatch')(pattern, text)`. Обратные слэши в шаблоне удваиваются: `'^\\d{4}$'`.

### Частые ошибки

- **`if` без `else`** даёт `null`, если условие ложно.
- **Отсутствующее поле, прочитанное как `meta.x`,** останавливает программу с ошибкой `Field does not exist: x`. Используйте `std.get`.
- **`local` без `;`.** Каждый `local` заканчивается `;` (или `,`, если несколько объявлены подряд).
- **Одинарный обратный слэш в шаблоне.** `'\d'` — недопустимая последовательность, пишите `'\\d'`.

### Где почитать больше

Полный справочник языка и онлайн-песочница — на [jsonnet.org](https://jsonnet.org): [учебник](https://jsonnet.org/learning/tutorial.html) и [стандартная библиотека](https://jsonnet.org/ref/stdlib.html). Чтобы проверить программу на своих заметках, оформите её как валидатор и запустите `trip2g lint` на папке: см. [[ru/user/lint|Линтер для хранилища]].
