---
title: Запросы к заметкам
free: true
lang_redirect: "[[en/user/note-queries]]"
---

Странице на JavaScript часто нужен список заметок: посты с тегом, пять последних статей, задачи по статусам. Эти данные trip2g хранит во frontmatter заметок, но сырой frontmatter посетителям не отдаёт. Запрос к заметкам — мост между ними: вы пишете в заметке небольшую программу на [[ru/user/jsonnet|Jsonnet]], страница вызывает её по имени и получает ровно тот JSON, который программа собрала.

Код лежит в вашем хранилище, поэтому вы решаете, что уходит с сервера. Посетитель передаёт только параметры, например тег.

### Быстрый старт

1\. Создайте заметку `queries/by_tag.md` с `type: note-query` и одним блоком `jsonnet`:

````
---
type: note-query
free: true
---

Посты блога с тегом, сначала новые.

```jsonnet
local tagged = [
  n for n in notes
  if std.startsWith(n.path, 'blog/')
  && std.member(std.get(n.meta, 'tags', []), params.tag)
];
local byDate = std.sort(tagged, function(n) std.get(n.meta, 'date', ''));

[
  { title: n.title, url: n.permalink, date: n.meta.date }
  for n in std.reverse(byDate)
]
```
````

2\. Синхронизируйте хранилище и вызовите запрос со страницы. `makeRequest` — маленький помощник для `fetch` со страницы [[ru/user/spa#save|Приложение поверх trip2g]]:

```js
const noteQuery = makeRequest(`query ($input: NoteQueryInput!) {
  noteQuery(input: $input) {
    __typename
    ... on NoteQueryPayload { json }
    ... on ErrorPayload { message }
  }
}`)

const data = await noteQuery({
  input: { query: '[[by_tag]]', params: JSON.stringify({ tag: 'go' }) },
})
const posts = JSON.parse(data.noteQuery.json)
// [{ title: 'Go on the web', url: '/blog/go_on_the_web', date: '2026-10-10' }, …]
```

`query` называет заметку-запрос: путём от корня хранилища, например `queries/by_tag.md`, или вики-ссылкой, например `[[by_tag]]`. Голое имя ищется по всему хранилищу; если имя повторяется, побеждает заметка с самым коротким путём. `params` — JSON-строка до 16 КБ; без неё запрос получает `{}`. Результат приходит JSON-строкой в поле `json`.

### Что видит запрос

| Имя | Что в нём |
|---|---|
| `notes` | список заметок, отсортированный по пути: `{ path, title, permalink, meta }` |
| `params` | разобранные `params`, `{}`, если их не передали |

`meta` — frontmatter заметки после [[ru/user/frontmatter_patches|frontmatter-патчей]]. Служебные заметки (пути, начинающиеся с `_`) и файлы, которые не являются markdown-заметками (canvas, base, excalidraw), в список не попадают.

Программа может вернуть любой JSON: список, объект, число.

### Кто может запустить и что он увидит

Два правила не дают запросу ничего раскрыть:

1. **Посетитель может запустить запрос, только если может прочитать заметку-запрос.** Доступ к ней задаётся как к любой заметке: `free: true` — для всех, [[ru/user/subgraphs|подграф]] — для подписчиков. Запрос, который посетитель прочитать не может, отвечает `query not found`, так же как несуществующий.
2. **Программа видит только заметки, которые может прочитать посетитель.** Гость, запустивший публичный запрос, не получит в `notes` платных заметок, так что запрос не покажет их заголовки или поля по ошибке. Админ, запустивший тот же запрос, видит всё.

На примерах ниже гость, запросивший `by_tag` с тегом `go`, получит два поста, а админ — ещё и третий, только для подписчиков.

Заметка-запрос публикуется как любая другая, поэтому читатели, которые могут запустить запрос, могут прочитать и его код.

### Примеры

Эти запросы взяты из демо-хранилища, которое тесты trip2g прогоняют при каждой сборке.

#### Посты с тегом, сначала новые

Параметры: `{"tag": "go"}`.

```jsonnet
local tagged = [
  n for n in notes
  if std.startsWith(n.path, 'blog/')
  && std.member(std.get(n.meta, 'tags', []), params.tag)
];
local byDate = std.sort(tagged, function(n) std.get(n.meta, 'date', ''));

[
  { title: n.title, url: n.permalink, date: n.meta.date }
  for n in std.reverse(byDate)
]
```

#### Последние посты

Параметры: `{"limit": 2}`; без них — три поста.

```jsonnet
local limit = std.get(params, 'limit', 3);
local posts = [n for n in notes if std.startsWith(n.path, 'blog/')];
local byDate = std.sort(posts, function(n) std.get(n.meta, 'date', ''));

[n.title for n in std.reverse(byDate)][:limit]
```

Срез `list[:limit]` оставляет первые элементы.

#### Счётчики для дашборда

Сколько задач в каждом статусе. Результат — объект, например `{"done": 1, "todo": 2}`:

```jsonnet
local tasks = [n for n in notes if std.startsWith(n.path, 'tasks/')];
local status(n) = std.get(n.meta, 'status', 'none');

{
  [s]: std.length([n for n in tasks if status(n) == s])
  for s in std.set([status(n) for n in tasks])
}
```

`{ [key]: value for … }` строит объект с вычисляемыми ключами, а `std.set` убирает повторы и сортирует.

### Ошибки

Неудачный вызов возвращает `ErrorPayload` с сообщением:

| Сообщение | Почему |
|---|---|
| `query not found` | такой заметки нет, или посетитель не может её прочитать |
| `<path> is not a note-query` | у заметки нет `type: note-query` |
| `<path>: a note-query needs exactly one jsonnet block` | блоков ноль или несколько |
| `params must be JSON`, `params is larger than 16 KB` | неправильные `params` |
| `<path>: RUNTIME ERROR: …` | программа упала; первая строка ошибки Jsonnet |

### Запрос или шаблон

Список заметок умеет строить и [[ru/user/templates|свой шаблон]]: `nvs.ByGlob("blog/*.md")` и `SortByMeta` собирают его на сервере при рендере страницы, а `json()` передаёт странице. Такой список фиксируется в момент рендера. Запрос к заметкам нужен, когда странице нужен разный ответ на разный ввод посетителя, например на тег, выбранный в меню.

### Ограничения

- **Каждый вызов заново выполняет программу.** Кеша пока нет; на хранилище примерно из 850 заметок вызов занимает около 12 мс. Для вошедшего посетителя, который не админ, проверка того, какие заметки ему доступны, стоит запроса к базе на каждую заметку. Оба пункта — в [issue #441](https://github.com/trip2g/trip2g/issues/441).
- **Запрос работает, пока не закончит.** Остановить Jsonnet на полпути нельзя, поэтому не пишите циклы, размер которых зависит от `params`, например `std.range(1, params.n)`: любой посетитель сможет сделать их огромными.
- **Как следить за ценой.** В метриках инстанса есть `trip2g_note_query_duration_seconds` по каждой заметке-запросу, а вызов дольше секунды пишется в лог как `slow note query`.
