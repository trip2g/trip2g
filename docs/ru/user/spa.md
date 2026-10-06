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

### Как приложение получает доступ: cookie сессии {#auth}

Своего входа у приложения нет. Оно пользуется сессией посетителя в trip2g:

1. **Посетитель входит кнопкой в стандартной шапке.** Шаблон выводит шапку trip2g на страницу, см. [[ru/user/spa#chrome|Обернуть приложение в стандартную шапку и подвал]]. После успешного входа страница перезагружается.
2. **Сервер ставит cookie сессии.** Она `HttpOnly` (JavaScript страницы её не прочитает), `SameSite=Lax` и принадлежит хосту, на котором посетитель вошёл.
3. **Запросы самого приложения несут её.** `fetch('/_system/graphql', …)` со страницы идёт на тот же origin, и браузер сам прикладывает cookie. Сервер читает её и выполняет запрос от имени этого пользователя: вошедший админ может читать и записывать заметки.

Поэтому перезагруженная страница рисуется уже для вошедшего пользователя: у админа `currentUser.IsAdmin()` теперь `true`, а с ним и флаг `editable` приложения.

**Делайте так:**

- Обращайтесь к API по относительному адресу `/_system/graphql` со страницы, которую отдаёт сам сайт: шаблон и его файлы из `asset()` или бандл, который этот шаблон загружает.
- Оставьте у `fetch` credentials по умолчанию (`same-origin`) или укажите `credentials: 'include'`, как в функциях на этой странице; для запроса на тот же origin это одно и то же. Никогда не ставьте `credentials: 'omit'`.
- Отправляйте операции методом `POST` с `Content-Type: application/json`; загрузка файла идёт как `multipart/form-data`. Отправленное иначе эндпоинт не выполняет.
- Решайте, показывать ли кнопки редактирования, по `currentUser.IsAdmin()` в шаблоне и передавайте это приложению флагом, например `editable`.

**Не делайте так:**

- **Не кладите в страницу API-ключ или личный токен.** Их прочитает любой, кто откроет страницу. Учётные данные — это cookie.
- **Не делайте свой экран входа.** Вход — работа шапки; после него страница перезагружается уже с сессией.
- **Не отдавайте приложение с другого origin,** например с отдельного dev-сервера. Браузер не отправляет cookie `SameSite=Lax` с кросс-сайтовым `POST`, а запросы с другого origin сервер разрешает только плагину Obsidian.

**Ни CSRF-токена, ни особого заголовка.** Кроме cookie эндпоинт ничего не требует: ни CSRF-токена, ни своего заголовка. Форма на чужом сайте сессией не воспользуется, потому что браузер не прикладывает cookie `SameSite=Lax` к кросс-сайтовому `POST`.

**Посетитель, который не вошёл,** открывает страницу и приложение как обычно. `currentUser.IsAdmin()` — `false`, значит и `editable` — `false`: покажите заметку только для чтения и укажите на кнопку входа в шапке, например строкой «Войдите, чтобы редактировать». В API по-прежнему работает `search`; попытка записи вернёт запись в `errors`. Вошедший читатель, который не админ, получит на запись тот же отказ. Кто до чего дотягивается — в таблице раздела [[ru/user/spa#Готовые GraphQL-запросы|Готовые GraphQL-запросы]].

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

- `title` — переменная, которую получает каждый свой шаблон: заголовок заметки, пропущенный через шаблон заголовка сайта, готовый для `<title>`. С шаблоном по умолчанию, `%s`, это сам заголовок. Голый заголовок в другом месте страницы даёт `note.Title()`.
- `note.ContentString()` — исходный markdown заметки вместе с frontmatter. Это тот же текст, от которого сервер считает хеш для проверки конфликтов, поэтому приложение может его изменить и отправить обратно.
- `json()` экранирует `<`, `>` и `&` как `<`…, поэтому markdown с `</script>` внутри не закроет элемент. `JSON.parse` вернёт текст в точности.
- `currentUser.IsAdmin()` решает только, показывает ли приложение кнопки редактирования. Каждое сохранение сервер проверяет заново (см. ниже).
- Обращение к `currentUser` делает страницу персонализированной: она не отдаётся из анонимного кэша страниц, поэтому приложение всегда стартует с текущей версии заметки.

Шаблон канбана передаёт те же данные иначе: markdown — в скрытой `<textarea>`, путь — в скрытом `<span>`, оба выведены обычным `{{ … }}`. Это тоже безопасно, потому что вывод экранируется по умолчанию: заметка с `</textarea>` попадёт на страницу как `&lt;/textarea&gt;`, и браузер декодирует её обратно. Но у `<textarea>` две особенности: браузер приводит переводы строк к `\n` и выбрасывает перевод строки сразу после открывающего тега. У JSON-острова их нет.

### Настройки приложения во frontmatter {#config}

Frontmatter заметки — удобное место для настроек приложения. Кто правит заметку, в Obsidian или агентом, меняет их вместе с содержимым, а один бандл обслуживает много заметок, у каждой свои настройки:

```yaml
---
title: Todo
layout: app
columns: [Todo, Doing, Done]
wip_limit: 3
show_archive: false
---
```

Шаблон читает ключи через `note.M()` и добавляет их в блок данных как `config`:

```jet
<script type="application/json" id="app-data">
  {{ json(map(
    "path", note.Path(),
    "content", note.ContentString(),
    "editable", currentUser.IsAdmin(),
    "config", map(
      "columns", note.M().GetStrings("columns"),
      "wipLimit", note.M().GetInt("wip_limit", 0),
      "showArchive", note.M().GetBool("show_archive", true)
    )
  )) }}
</script>
```

Приложение находит их в `data.config`:

```js
const { columns, wipLimit, showArchive } = data.config
```

Каждый метод возвращает значение по умолчанию, если ключа нет или у него не тот тип, поэтому заметка без настроек тоже откроется; `GetStrings` вернёт пустой список. Чтобы отдать приложению весь frontmatter, напишите `"config", note.M().Raw()`. Методы описаны в [[ru/user/templates#Что доступно в шаблоне|Шаблонах]]. Настройки — тоже часть `content`, поэтому сохранение, которое не трогает frontmatter, их сохраняет.

### Обернуть приложение в стандартную шапку и подвал {#chrome}

Шаблон выше — голая страница. Чтобы у приложения была шапка сайта с кнопкой входа и подвал, выведите части дефолтного шаблона вокруг элемента для монтирования:

```jet
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ title }}</title>
  {{ defaultTemplate.Styles() }}
  {{ defaultTemplate.UserSpaceScripts() }}
</head>
<body>
  {{ defaultTemplate.Header() }}
  <main id="app"></main>
  <script type="application/json" id="app-data">
    {{ json(map(
      "path", note.Path(),
      "content", note.ContentString(),
      "editable", currentUser.IsAdmin()
    )) }}
  </script>
  <script src="{{ asset("app.js") }}"></script>
  {{ defaultTemplate.Footer() }}
</body>
</html>
```

| Вызов | Что выводит |
|---|---|
| `defaultTemplate.Styles()` | Теги `<link>` со стилями дефолтного шаблона, чтобы шапка и подвал выглядели как на других страницах |
| `defaultTemplate.UserSpaceScripts()` | `<script>` с настройками бандла, затем бандл user space. Бандл рисует в шапке кнопку входа и поиск. Настройки выводятся только при первом вызове |
| `defaultTemplate.Header()` | Шапку сайта: логотип, навигацию, кнопку входа и поиск |
| `defaultTemplate.Footer()` | Подвал сайта |

`Header()` и `Footer()` ничего не выводят, если у страницы нет заметки-шапки или заметки-подвала. Заметка берётся из `header:` / `footer:` во frontmatter страницы, подходящей layout-секции или заметки `_header` / `_footer` в хранилище: см. [[ru/user/default-template#Функциональные заметки: шапка, подвал и боковые панели|Функциональные заметки]]. Без шапки нет и кнопки входа. Чтобы она осталась, поставьте элемент монтирования бандла там, где нужна кнопка, и оставьте `UserSpaceScripts()` в `<head>`:

```html
<div mol_view_root="$trip2g_user_space"></div>
```

В превью шаблона (`/_system/renderlayout`) все четыре вызова ничего не выводят. Полный список того, что получает шаблон, — в [[ru/user/jet-functions#Переменные в каждом своём шаблоне|Функции Jet → Переменные в каждом своём шаблоне]].

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

### Сохранение: мутация `updateNotes` {#save}

Приложение записывает заметку мутацией GraphQL `updateNotes` на `/_system/graphql`. Писать может вошедший админ сайта, остальные получат ошибку, поэтому браузер посетителя не изменит заметку, даже если приложение по ошибке покажет кнопки.

Любая операция — один и тот же `POST` с JSON-телом `{ query, variables }`, поэтому хватит одной маленькой функции. `makeRequest` принимает операцию и возвращает функцию от её переменных:

```js
const makeRequest = (query) => (variables) =>
  fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
    .then((res) => res.json())
    .then((body) => {
      if (body.errors) throw new Error(body.errors[0].message)
      return body.data
    })

const updateNotes = makeRequest(`mutation ($input: UpdateNotesInput!) {
  updateNotes(input: $input) {
    __typename
    ... on UpdateNotesSuccessPayload { updated { path versionId } }
    ... on UpdateNotesHashMismatchPayload { path actualHash }
    ... on UpdateNotesPatchNotFoundPayload { path find }
    ... on ErrorPayload { message }
  }
}`)

const change = { upsert: { path: data.path, content: newContent, expectedHash } }
const { updateNotes: result } = await updateNotes({ input: { changes: [change] } })
```

`newContent` — отредактированный markdown, `expectedHash` разобран ниже. Запрос идёт на тот же origin, поэтому браузер сам прикладывает cookie сессии; `credentials: 'include'` лишь говорит это явно. Результат показывает `__typename`:

| `__typename` | Что значит |
|---|---|
| `UpdateNotesSuccessPayload` | Сохранено; в `updated` каждая заметка с `versionId`, который записало сохранение |
| `UpdateNotesHashMismatchPayload` | Ничего не записано: заметка больше не соответствует `expectedHash`; `actualHash` — её текущий хеш |
| `UpdateNotesPatchNotFoundPayload` | Изменение `patch` не нашло свой текст `find` или нашло его больше одного раза |
| `ErrorPayload` | Отказ, причина в `message` |

Посетитель, который не админ, до результата не доходит: в ответе будет запись в `errors` и не будет `data`, и `makeRequest` превратит это в исключение.

#### `expectedHash`: сохранение применяется один раз и к тому тексту, от которого сделано

`expectedHash` — поле изменений `upsert` и `patch` в `updateNotes`. Оно привязывает изменение к версии заметки, от которой оно сделано. Перед записью сервер считает хеш текущего markdown заметки и сравнивает:

- **Совпал:** изменение записывается.
- **Не совпал:** ничего не записывается, а результат — `UpdateNotesHashMismatchPayload` с путём заметки `path` и её текущим `actualHash`.

Поэтому изменение применяется не больше одного раза. То же сохранение, отправленное дважды — повтором после таймаута, двойным кликом или из второй открытой вкладки, — несёт тот же `expectedHash`. Первое меняет заметку, второе уже не совпадает и получает отказ, а не применяется ещё раз. Эта же проверка не даёт приложению затереть правку, которую кто-то успел сделать в Obsidian или агентом.

Передавать нужно хеш того markdown, с которого начало приложение: `latestContentHash` из запроса `ReadNote` ниже или хеш `data.content`, который отдал шаблон. Хеш — SHA-256 от markdown в URL-safe base64 с дополнением `=`:

```js
async function contentHash(text) {
  const bytes = new TextEncoder().encode(text)
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  return btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
}

const expectedHash = await contentHash(data.content)
```

Новый хеш успешное сохранение не возвращает. После `upsert` в заметке ровно тот markdown, который вы отправили, поэтому `expectedHash` следующего сохранения — `contentHash(newContent)`; после `patch` перечитайте `latestContentHash`. Два значения особые: пустой `expectedHash` значит «только создать», и на существующей заметке сохранение не пройдёт, а изменение без `expectedHash` записывается без проверки.

На `UpdateNotesHashMismatchPayload` перезагрузите страницу или получите новое содержимое и примените своё изменение к нему. Канбан перечитывает последнюю версию админскими запросами `noteVersionHistory` и `noteVersion` и повторяет перемещение.

**Маленькие правки.** Вместо `upsert`, который заменяет заметку целиком, изменение может быть `patch`: `{ patch: { path, find, replace, expectedHash } }` заменяет один точный кусок текста. Так канбан переключает чекбокс, и текст, который он не понимает, никогда не переписывается. Обе формы, пакеты изменений и ошибки — в [[ru/user/update_notes|updateNotes]].

### Живые обновления {#live}

Чтобы видеть правки из Obsidian или от агента, пока приложение открыто, подпишитесь на `noteChanges` с фильтром по пути заметки. Подписки эндпоинт отдаёт только через server-sent events, на том же `/_system/graphql`: `POST` с `Accept: text/event-stream`. Транспорта WebSocket у него нет, так что клиент `graphql-ws` не подключится. `EventSource` не умеет `POST`, поэтому функция сама читает поток ответа:

```js
const makeSubscription = (query) => async (variables, onData) => {
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    },
    body: JSON.stringify({ query, variables }),
  })
  const reader = res.body
    .pipeThrough(new TextDecoderStream())
    .getReader()
  let buffer = ''
  for (;;) {
    const { done, value } = await reader.read()
    if (done) return
    buffer += value
    const frames = buffer.split('\n\n')
    buffer = frames.pop()
    for (const frame of frames) {
      const line = frame.match(/^data: (.*)$/m)
      if (!frame.startsWith('event: next') || !line) continue
      const body = JSON.parse(line[1])
      if (body.errors) throw new Error(body.errors[0].message)
      onData(body.data)
    }
  }
}

const watchNotes = makeSubscription(`subscription ($filter: NoteChangesFilter!) {
  noteChanges(filter: $filter) {
    changes {
      __typename
      ... on NoteUpsertEvent { path versionId }
      ... on NoteHideEvent { path }
    }
  }
}`)

async function follow(path, onChange) {
  for (;;) {
    await watchNotes({ filter: { includePatterns: [path] } }, onChange)
      .catch(() => {})
    await new Promise((resolve) => setTimeout(resolve, 5000))
  }
}

const ownVersions = new Set()

follow(data.path, ({ noteChanges }) => {
  const foreign = noteChanges.changes
    .some((change) => !ownVersions.has(change.versionId))
  if (foreign) location.reload()
})
```

Сервер шлёт `event: next` с результатом, `event: complete` в конце и комментарий `: ping` каждые 30 секунд. Поток может оборваться, например при перезапуске сервера, поэтому `follow` после паузы подписывается снова.

Ваше собственное сохранение тоже придёт событием. Добавляйте каждый `versionId` из `updated` сохранения в `ownVersions`, и приложение пропустит своё эхо. `location.reload()` — самая простая реакция на чужую правку. Чтобы сохранить состояние приложения, перечитайте заметку через `ReadNote` и слейте изменения, как делает [src/api.ts](https://github.com/trip2g/kanban_template/blob/main/src/api.ts) канбана. Подписке нужен вошедший посетитель: админ получает все подходящие заметки, другой вошедший читатель — только те, что ему можно читать. Glob-шаблоны фильтра разобраны в разделе [[ru/user/spa#Следить за изменениями|Следить за изменениями]].

### Готовые GraphQL-запросы

Эти запросы можно копировать в приложение. Каждый уходит на `/_system/graphql` как `POST` с JSON-телом `{ query, variables }`: операция — это `query`, JSON-блок под ней — `variables`. Оберните операцию в `makeRequest` из раздела [[ru/user/spa#save|Сохранение]] и вызовите с переменными. Здесь в `READ_NOTE` лежит операция `ReadNote` ниже:

```js
const readNote = makeRequest(READ_NOTE)
const { notePaths } = await readNote({ paths: [data.path] })
```

Большинство результатов — union-типы: запросите `__typename` и по фрагменту `... on` на каждый тип, потом ветвитесь по `__typename`. Отказ, который приложение должно показать, приходит типом результата, например `ErrorPayload`. Вызов без доступа получает запись в `errors`, и `makeRequest` превращает её в исключение.

**Кто что может вызвать.** Подробности — в [[ru/user/update_notes#Авторизация|updateNotes → Авторизация]].

| Кто | Что отправляет | Что доступно |
|---|---|---|
| Вошедший админ | Cookie сессии или `Authorization: Bearer t2g_…` | Всё ниже, включая `admin { … }` |
| API-ключ | `X-Api-Key: …` | Всё, кроме `admin { … }`; запись только в пределах write patterns ключа |
| Токен вебхука | `Authorization: Bearer eyJ…` | То же, что API-ключу, в пределах read и write patterns токена, кроме `hideNotes`, `pushNotes` и `commitNotes` |
| Любой посетитель | Ничего | `search`, а после входа — `noteChanges` |

Приложение на странице, которую открывают посетители, авторизуется cookie сессии админа, её отправляет `credentials: 'include'`. Не кладите API-ключ в бандл: его прочитает любой, кто загрузит страницу.

#### Прочитать markdown заметки

```graphql
query ReadNote($paths: [String!]) {
  notePaths(filter: { paths: $paths }) {
    value
    content
    latestContentHash
  }
}
```

```json
{ "paths": ["boards/todo.md"] }
```

`value` — путь в хранилище, `content` — markdown вместе с frontmatter. `latestContentHash` — тот самый хеш, который принимает `expectedHash`, так что считать его в приложении не нужно. Несуществующего пути в списке просто не будет.

#### Список заметок

```graphql
query ListNotes($filter: NotePathsFilter) {
  notePaths(filter: $filter) {
    value
    latestContentHash
    latestNoteView {
      title
      url
    }
  }
}
```

```json
{ "filter": { "like": "boards/%" } }
```

```json
{ "filter": { "frontmatter": [{ "key": "layout", "equals": "app" }] } }
```

`like` — шаблон SQL `LIKE`: `%` — любая последовательность символов, `_` — один символ. Фильтр принимает ещё `search` (полнотекстовый поиск по заметкам) и `paths`. Если задано несколько, `paths` важнее `search`, а `search` важнее `like`. `frontmatter` сужает любой из них. Без фильтра вернутся все пути хранилища, кроме скрытых.

#### Поиск

```graphql
query Search($input: SearchInput!) {
  search(input: $input) {
    totalCount
    nodes {
      url
      highlightedTitle
      highlightedContent
      document {
        ... on PublicNote {
          path
          title
        }
      }
    }
  }
}
```

```json
{ "input": { "query": "release plan" } }
```

Это поиск сайта, он открыт любому посетителю. Заметки, которые посетителю читать нельзя, идут в конце: с заголовком и URL, но без `document`. Админ ищет по последним версиям, остальные — по опубликованным, если сайт не показывает черновики всем.

#### Сохранить заметку целиком

```graphql
mutation SaveNotes($input: UpdateNotesInput!) {
  updateNotes(input: $input) {
    __typename
    ... on UpdateNotesSuccessPayload {
      updated {
        path
        versionId
      }
    }
    ... on UpdateNotesHashMismatchPayload {
      path
      actualHash
    }
    ... on UpdateNotesPatchNotFoundPayload {
      path
      find
    }
    ... on ErrorPayload {
      message
    }
  }
}
```

```json
{
  "input": {
    "changes": [
      {
        "upsert": {
          "path": "boards/todo.md",
          "content": "# Todo\n\n- [ ] Ship it\n",
          "expectedHash": "latestContentHash from ReadNote"
        }
      }
    ]
  }
}
```

`upsert` заменяет заметку или создаёт её. С `expectedHash` сохранение пройдёт, только если с момента чтения заметку никто не менял, иначе придёт `UpdateNotesHashMismatchPayload`. Пустой `expectedHash` значит «только создать»: если заметка уже есть, сохранения не будет. Без `expectedHash` сохранение всегда перезаписывает. `updated[].versionId` — версия, которую записало сохранение.

#### Заменить один кусок текста

Та же мутация `SaveNotes`, но с изменением `patch`:

```json
{
  "input": {
    "changes": [
      {
        "patch": {
          "path": "boards/todo.md",
          "find": "- [ ] Ship it",
          "replace": "- [x] Ship it",
          "expectedHash": "latestContentHash from ReadNote"
        }
      }
    ]
  }
}
```

`find` должен встречаться в заметке ровно один раз. Если его нет или он встречается больше одного раза, заметка остаётся как была, а результат — `UpdateNotesPatchNotFoundPayload`. Один вызов может нести несколько изменений для нескольких заметок, см. [[ru/user/update_notes|updateNotes]].

#### Скрыть заметки

```graphql
mutation HideNotes($input: HideNotesInput!) {
  hideNotes(input: $input) {
    __typename
    ... on HideNotesPayload {
      success
    }
    ... on ErrorPayload {
      message
    }
  }
}
```

```json
{ "input": { "paths": ["boards/old.md"] } }
```

Скрытая заметка пропадает с сайта. Новое сохранение через `updateNotes` возвращает её. Изменение `hide` в `updateNotes`, `{ "hide": { "path": "boards/old.md" } }`, делает то же внутри пакета.

Скрытие записывается на админа: на вошедшего админа, на админа, создавшего API-ключ, или на админа, создавшего вебхук, который выдал токен. Токен вебхука получает `ErrorPayload` от `hideNotes`; скрывает он изменением `hide`, в пределах своих write patterns.

#### Загрузить файл

Файл уходит запросом `multipart/form-data` в формате [GraphQL multipart request](https://github.com/jaydenseric/graphql-multipart-request-spec): часть `operations` с запросом и переменными, часть `map`, которая говорит, какую переменную заполняет файл, и сам файл.

```graphql
mutation UploadAsset($input: UploadNoteAssetInput!) {
  uploadNoteAsset(input: $input) {
    __typename
    ... on UploadNoteAssetPayload {
      uploadSkipped
    }
    ... on ErrorPayload {
      message
    }
  }
}
```

```js
async function sha256Hex(blob) {
  const bytes = await blob.arrayBuffer()
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

async function upload(file, noteId, path, absolutePath) {
  const input = {
    file: null,
    noteId,
    sha256Hash: await sha256Hex(file),
    path,
    absolutePath,
  }
  const form = new FormData()
  form.append('operations', JSON.stringify({
    query: UPLOAD_ASSET,
    variables: { input },
  }))
  form.append('map', JSON.stringify({ 0: ['variables.input.file'] }))
  form.append('0', file, file.name)
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    body: form,
  })
  const body = await res.json()
  if (body.errors) throw new Error(body.errors[0].message)
  return body.data.uploadNoteAsset
}
```

`UPLOAD_ASSET` — операция выше. Не ставьте `Content-Type` сами: браузер добавит его вместе с границей multipart.

Файл принадлежит версии заметки, поэтому сначала сохраните markdown, который на него ссылается:

1. Сохраните заметку через `SaveNotes`, например с `![Plan](plan.png)` в markdown.
2. Возьмите `versionId` из `updated` и передайте его как `noteId`.
3. Передайте `path` ровно так, как ссылка записана в markdown (`plan.png`), а `absolutePath` — как путь файла в хранилище (`boards/plan.png`).

`sha256Hash` — SHA-256 файла в hex, а не в base64, как у `expectedHash`; сервер сверяет его после загрузки. `path`, на который заметка не ссылается, отклоняется с `ErrorPayload`, где перечислены ссылки заметки. `uploadSkipped: true` значит, что такой файл на сервере уже был и его только привязали к новой версии.

#### Следить за изменениями

```graphql
subscription WatchNotes($filter: NoteChangesFilter!) {
  noteChanges(filter: $filter) {
    changes {
      __typename
      ... on NoteUpsertEvent {
        path
        eventType
        versionId
      }
      ... on NoteHideEvent {
        path
      }
    }
  }
}
```

```json
{ "filter": { "includePatterns": ["boards/**"] } }
```

`includePatterns` (обязателен) и `excludePatterns` — glob-шаблоны: `*` не выходит за пределы папки, `**` проходит через папки, обычный путь соответствует одной заметке. Одно событие может нести несколько изменений, если одно сохранение записало несколько заметок.

Клиент, который выполняет эту подписку и переподключается, — в разделе [[ru/user/spa#live|Живые обновления]].

#### Прочитать старую версию (админ)

Этими двумя запросами канбан повторяет перемещение поверх чужой правки. Версии идут от новых к старым, поэтому `limit: 1` даёт последнюю.

```graphql
query NoteVersions($filter: AdminNoteVersionHistoryFilter!) {
  admin {
    noteVersionHistory(filter: $filter) {
      nodes {
        versionId
        version
        createdAt
      }
    }
  }
}
```

```json
{ "filter": { "path": "boards/todo.md", "limit": 1 } }
```

```graphql
query NoteVersion($id: Int64!) {
  admin {
    noteVersion(versionId: $id) {
      path
      content
      createdAt
    }
  }
}
```

```json
{ "id": 42 }
```

### Вызовы API и админские вызовы {#api-vs-admin}

Эндпоинт несёт операции двух видов, и обещают они разное:

| | Вызовы API | Админские вызовы |
|---|---|---|
| Где находятся | На верхнем уровне операции: `notePaths`, `search`, `updateNotes`, `noteChanges` | Внутри `admin { … }`, в типах `AdminQuery` и `AdminMutation` схемы |
| Кто на них опирается | Плагин Obsidian, агенты с API-ключом, приложения вроде этого | Собственная админка trip2g |
| Стабильность | Поверхность, на которой построены эти клиенты | Гарантий меньше: они меняются вместе с админкой и могут поменяться в любом релизе |
| Кто может вызвать | См. таблицу в разделе [[ru/user/spa#Готовые GraphQL-запросы\|Готовые GraphQL-запросы]] | Вошедший админ: cookie сессии или личный токен |

Приложению можно пользоваться админскими вызовами — канбан читает ими историю версий. Оставляйте их для функций, которые видят только админы, и проверяйте заново после обновления trip2g. Всё остальное на этой странице, кроме раздела с пометкой «(админ)», — API.

Поверхность API описана в [[ru/user/update_notes|updateNotes]], [[ru/user/graphql|GraphQL API]] и в запросах выше, а файл операций плагина Obsidian ниже показывает, что он выполняет.

### Где искать остальное

Запросы выше проверены тестом по схеме. Для всего остального:

- **Схема.** В [internal/graph/schema.graphqls](https://github.com/trip2g/trip2g/blob/main/internal/graph/schema.graphqls) перечислены все запросы, мутации и подписки с типами входа и результата. Войдя как админ, откройте `/_system/graphql` в браузере, чтобы изучать её в GraphiQL.
- **Клиент синхронизации Obsidian.** Его файл операций, [src/operations.graphql](https://github.com/trip2g/obsidian-sync/blob/master/src/operations.graphql) в [github.com/trip2g/obsidian-sync](https://github.com/trip2g/obsidian-sync), содержит запросы с API-ключом, которые он выполняет: чтение заметок и их файлов, `pushNotes`, `hideNotes`, `uploadNoteAsset`, `commitNotes`. В репозитории trip2g клиент лежит сабмодулем `obsidian-sync`; файлы сабмодуля GitHub внутри trip2g не показывает, поэтому вот [тот же файл на коммите, который закреплён в trip2g](https://github.com/trip2g/obsidian-sync/blob/05b385ae943a5f6b5f162c6b3b2ed014d51ee66c/src/operations.graphql).
- **Админка.** Каждый экран хранит свои операции в файлах `.graphql` рядом с кодом, в [assets/ui/](https://github.com/trip2g/trip2g/tree/main/assets/ui): `admin/` — админка, `editor/` — редактор заметок, `user/` — сторона читателя. Нужный файл ищите по полю, которое он вызывает, [поиском GitHub по этим файлам](https://github.com/search?q=repo%3Atrip2g%2Ftrip2g+path%3Aassets%2Fui+extension%3Agraphql&type=code) (допишите имя поля в запрос) или в клоне:

  ```bash
  grep -rl --include='*.graphql' 'noteVersionHistory' assets/ui
  ```

### Как сделать своё

1. Сначала решите формат markdown. Заметка должна читаться и правиться без вашего приложения: список, таблица, заголовки. Канбан взял формат плагина obsidian-kanban, поэтому та же заметка — доска и в Obsidian.
2. Напишите парсер и сериализатор, которые возвращают формат в точности, и проверьте тестом, что `serialize(parse(text)) === text`. Всё, что приложение не понимает, должно пережить сохранение без изменений.
3. Напишите шаблон как выше, со своим элементом для монтирования и своим бандлом.
4. Прочитайте данные со страницы, нарисуйте их и сохраняйте через `updateNotes` с `expectedHash`.
5. Показывайте кнопки редактирования, только когда `editable` — `true`. Сервер всё равно проверит.

### Чек-лист

- [ ] У заметки `layout: app`, а шаблон выводит `defaultTemplate.Styles()` и `defaultTemplate.UserSpaceScripts()` в `<head>`, `Header()` перед приложением и `Footer()` после него.
- [ ] У страницы есть заметка-шапка, или в шаблоне есть свой элемент монтирования `$trip2g_user_space`: иначе нет кнопки входа.
- [ ] Приложение обращается к `/_system/graphql` по относительному адресу, методом `POST` с `Content-Type: application/json`, с credentials по умолчанию или `credentials: 'include'`.
- [ ] В странице нет API-ключа или токена, у приложения нет своего экрана входа.
- [ ] Кнопки редактирования видны, только когда так решил `currentUser.IsAdmin()` в шаблоне; посетитель видит заметку только для чтения и кнопку входа в шапке.
- [ ] Каждое сохранение передаёт `expectedHash` и обрабатывает `UpdateNotesHashMismatchPayload` и запись в `errors`.

### Смотрите также

- [[ru/user/kanban|Шаблон канбан-доски]] и его исходники, [github.com/trip2g/kanban_template](https://github.com/trip2g/kanban_template)
- [[ru/user/update_notes|updateNotes]] — API записи целиком
- [[ru/user/graphql|GraphQL API]]
- [[ru/user/jet-functions#json и writeJson|json() и writeJson()]]
- [[ru/user/templates|Шаблоны]]
