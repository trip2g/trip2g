---
title: Приложение поверх trip2g
free: true
lang_redirect: "[[en/user/spa]]"
---

Свой шаблон не обязан рисовать страницу. Он может отдать JavaScript-приложение, которое показывает заметку как что-то интерактивное и записывает изменения обратно. Заметка остаётся обычным markdown: Obsidian, агент и приложение правят один и тот же файл.

Так устроен [[ru/user/kanban|шаблон канбан-доски]]. Он превращает заметку в формате obsidian-kanban в доску с перетаскиванием карточек, и каждое перемещение сохраняется обратно в markdown заметки. Эта страница описывает схему, по которой он сделан, чтобы вы могли собрать своё приложение: чек-лист, редактор таблицы, конструктор форм, дашборд.

Если вы ещё не делали шаблоны, начните с [[ru/user/templates|Шаблонов]].

### Три части

| Часть | Где живёт | Что делает |
|---|---|---|
| Шаблон | `_layouts/app.html` в хранилище | Кладёт данные заметки в страницу, элемент для монтирования и `<script>`, который загружает приложение |
| Бандл приложения | Рядом с шаблоном или по любому URL | Читает данные со страницы, рисует интерфейс |
| GraphQL API | `/_system/graphql` на вашем сайте | Сохраняет отредактированный markdown обратно в заметку |

Заметка включает приложение строкой `layout: app` во frontmatter, как и любой другой шаблон.

### Шаблон: заметка внутри страницы

Шаблон даёт приложению три вещи: путь заметки в хранилище, её исходный markdown и то, может ли посетитель редактировать. `json()` кладёт их в элемент `<script type="application/json">`:

```jet
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ title }}</title>
</head>
<body>
  <div id="app"></div>

  <script type="application/json" id="app-data">
    {{ json(map(
      "path", note.Path(),
      "content", note.ContentString(),
      "editable", currentUser.IsAdmin()
    )) }}
  </script>
  <script src="{{ asset("app.js") }}"></script>
</body>
</html>
```

- `note.ContentString()` — исходный markdown заметки вместе с frontmatter. Это тот же текст, от которого сервер считает хеш для проверки конфликтов, поэтому приложение может его изменить и отправить обратно.
- `json()` экранирует `<`, `>` и `&` как `<`…, поэтому markdown с `</script>` внутри не закроет элемент. `JSON.parse` вернёт текст в точности.
- `currentUser.IsAdmin()` решает только, показывает ли приложение кнопки редактирования. Каждое сохранение сервер проверяет заново (см. ниже).
- Обращение к `currentUser` делает страницу персонализированной: она не отдаётся из анонимного кэша страниц, поэтому приложение всегда стартует с текущей версии заметки.

Шаблон канбана передаёт те же данные иначе: markdown — в скрытой `<textarea>`, путь — в скрытом `<span>`, оба выведены обычным `{{ … }}`. Это тоже безопасно, потому что вывод экранируется по умолчанию: заметка с `</textarea>` попадёт на страницу как `&lt;/textarea&gt;`, и браузер декодирует её обратно. Но у `<textarea>` две особенности: браузер приводит переводы строк к `\n` и выбрасывает перевод строки сразу после открывающего тега. У JSON-острова их нет.

Чтобы на странице приложения была кнопка входа trip2g, добавьте в `<head>` `{{ defaultTemplate.UserSpaceScripts() }}` и элемент для неё, как это делает шаблон канбана.

### Бандл

Отдать JavaScript можно двумя способами:

- **Рядом с шаблоном.** Положите `app.js` рядом с `_layouts/app.html` и подключите через `{{ asset("app.js") }}`, как выше. В URL будет хеш для сброса кэша. См. [[ru/user/yield_blocks|yield_blocks]], раздел про `asset()`.
- **Из релиза.** Шаблон канбана загружает бандл по ссылке на релиз GitHub, `…/releases/latest/download/kanban.js`. Пользователь ставит один HTML-файл и получает обновления, ничего не синхронизируя.

Бандл может быть на любом фреймворке или без него. Канбан — React-приложение, которое esbuild собирает в один самодостаточный файл.

Приложение читает данные при старте:

```js
const data = JSON.parse(
  document.getElementById('app-data').textContent
)
// data.path, data.content, data.editable
```

### Сохранение: мутация `updateNotes`

Приложение записывает заметку мутацией GraphQL `updateNotes` на `/_system/graphql`. `fetch` с того же домена с `credentials: 'include'` отправляет cookie сессии посетителя. Писать может вошедший админ сайта, остальные получат ошибку, поэтому браузер посетителя не изменит заметку, даже если приложение по ошибке покажет кнопки.

```js
const UPDATE = `mutation ($i: UpdateNotesInput!) {
  updateNotes(input: $i) {
    __typename
    ... on UpdateNotesHashMismatchPayload { actualHash }
    ... on ErrorPayload { message }
  }
}`

async function save(path, content, expectedHash) {
  const change = { upsert: { path, content, expectedHash } }
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      query: UPDATE,
      variables: { i: { changes: [change] } },
    }),
  })
  const body = await res.json()
  if (body.errors) throw new Error(body.errors[0].message)
  return body.data.updateNotes
}
```

Результат показывает `__typename`:

| `__typename` | Что значит |
|---|---|
| `UpdateNotesSuccessPayload` | Сохранено |
| `UpdateNotesHashMismatchPayload` | Кто-то изменил заметку после того, как приложение её загрузило; `actualHash` — текущий хеш |
| `UpdateNotesPatchNotFoundPayload` | Изменение `patch` не нашло свой текст `find` |
| `ErrorPayload` | Отказ, причина в `message` |

Посетитель, который не админ, до результата не доходит: в ответе будет запись в `errors` и не будет `data.updateNotes`, и `save` выше превратит это в исключение.

**Не затирайте чужую правку.** `expectedHash` делает сохранение условным: сервер сохраняет, только если заметка всё ещё даёт этот хеш. Хеш — SHA-256 от markdown в URL-safe base64 с дополнением `=`:

```js
async function contentHash(text) {
  const bytes = new TextEncoder().encode(text)
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  return btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
}

const hash = await contentHash(data.content)
const result = await save(data.path, newContent, hash)
```

На `UpdateNotesHashMismatchPayload` перезагрузите страницу или получите новое содержимое и примените своё изменение к нему. Канбан перечитывает последнюю версию админскими запросами `noteVersionHistory` и `noteVersion` и повторяет перемещение.

**Маленькие правки.** Вместо `upsert`, который заменяет заметку целиком, изменение может быть `patch`: `{ patch: { path, find, replace, expectedHash } }` заменяет один точный кусок текста. Так канбан переключает чекбокс, и текст, который он не понимает, никогда не переписывается. Обе формы, пакеты изменений и ошибки — в [[ru/user/update_notes|updateNotes]].

### Живые обновления

Чтобы видеть правки из Obsidian или от агента, пока приложение открыто, подпишитесь на `noteChanges` с фильтром по пути заметки. Подписка идёт через server-sent events на том же `/_system/graphql` и принимает cookie сессии админа. Канбан так обновляет доску без перезагрузки; рабочий клиент — в его `src/api.ts`.

### Как сделать своё

1. Сначала решите формат markdown. Заметка должна читаться и правиться без вашего приложения: список, таблица, заголовки. Канбан взял формат плагина obsidian-kanban, поэтому та же заметка — доска и в Obsidian.
2. Напишите парсер и сериализатор, которые возвращают формат в точности, и проверьте тестом, что `serialize(parse(text)) === text`. Всё, что приложение не понимает, должно пережить сохранение без изменений.
3. Напишите шаблон как выше, со своим элементом для монтирования и своим бандлом.
4. Прочитайте данные со страницы, нарисуйте их и сохраняйте через `updateNotes` с `expectedHash`.
5. Показывайте кнопки редактирования, только когда `editable` — `true`. Сервер всё равно проверит.

### Смотрите также

- [[ru/user/kanban|Шаблон канбан-доски]] и его исходники, [github.com/trip2g/kanban_template](https://github.com/trip2g/kanban_template)
- [[ru/user/update_notes|updateNotes]] — API записи целиком
- [[ru/user/graphql|GraphQL API]]
- [[ru/user/jet-functions#json и writeJson|json() и writeJson()]]
- [[ru/user/templates|Шаблоны]]
