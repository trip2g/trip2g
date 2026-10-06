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
| `UpdateNotesPatchNotFoundPayload` | Изменение `patch` не нашло свой текст `find` или нашло его больше одного раза |
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

### Готовые GraphQL-запросы

Эти запросы можно копировать в приложение. Каждый уходит на `/_system/graphql` как `POST` с JSON-телом `{ query, variables }`: операция — это `query`, JSON-блок под ней — `variables`. Эта функция отправляет запрос и возвращает `data`:

```js
async function gql(query, variables) {
  const res = await fetch('/_system/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  const body = await res.json()
  if (body.errors) throw new Error(body.errors[0].message)
  return body.data
}
```

Большинство результатов — union-типы: запросите `__typename` и по фрагменту `... on` на каждый тип, потом ветвитесь по `__typename`. Отказ, который приложение должно показать, приходит типом результата, например `ErrorPayload`. Вызов без доступа получает запись в `errors`, и `gql` превращает её в исключение.

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

Транспорт — server-sent events, не WebSocket: `POST` на `/_system/graphql` с `Accept: text/event-stream`. `EventSource` не умеет `POST`, поэтому читайте поток ответа:

```js
async function watch(query, variables, onData) {
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
      const data = frame.match(/^data: (.*)$/m)
      if (frame.startsWith('event: next') && data) {
        onData(JSON.parse(data[1]).data.noteChanges)
      }
    }
  }
}
```

Сервер шлёт `event: next` с результатом, `event: complete` в конце и комментарий `: ping` каждые 30 секунд. Поток может оборваться, например при перезапуске сервера, поэтому после паузы вызовите `watch` снова. Канбан делает это с задержкой в своём [src/api.ts](https://github.com/trip2g/kanban_template/blob/main/src/api.ts).

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

### Где искать остальное

Запросы выше проверены тестом по схеме. Для всего остального:

- **Схема.** В [internal/graph/schema.graphqls](https://github.com/trip2g/trip2g/blob/main/internal/graph/schema.graphqls) перечислены все запросы, мутации и подписки с типами входа и результата. Войдя как админ, откройте `/_system/graphql` в браузере, чтобы изучать её в GraphiQL.
- **Клиент синхронизации Obsidian.** Его файл операций, [src/operations.graphql](https://github.com/trip2g/obsidian-sync/blob/master/src/operations.graphql) в [github.com/trip2g/obsidian-sync](https://github.com/trip2g/obsidian-sync), содержит запросы с API-ключом, которые он выполняет: чтение заметок и их файлов, `pushNotes`, `hideNotes`, `uploadNoteAsset`, `commitNotes`. В репозитории trip2g клиент лежит сабмодулем `obsidian-sync`; файлы сабмодуля GitHub внутри trip2g не показывает, поэтому вот [тот же файл на коммите, который закреплён в trip2g](https://github.com/trip2g/obsidian-sync/blob/05b385ae943a5f6b5f162c6b3b2ed014d51ee66c/src/operations.graphql).
- **Админка.** Каждый экран хранит свои операции в файлах `.graphql` рядом с кодом, в [assets/ui/](https://github.com/trip2g/trip2g/tree/main/assets/ui): `admin/` — админка, `editor/` — редактор заметок, `user/` — сторона читателя. Нужный файл ищите по полю, которое он вызывает, [поиском GitHub по этим файлам](https://github.com/search?q=repo%3Atrip2g%2Ftrip2g+path%3Aassets%2Fui+extension%3Agraphql&type=code) (допишите имя поля в запрос) или в клоне:

  ```bash
  grep -rl --include='*.graphql' 'noteVersionHistory' assets/ui
  ```

  Всё внутри `admin { … }` требует вошедшего админа: cookie сессии или личный токен, не API-ключ. Это подходит приложению, которое открывают только админы, например внутреннему дашборду или редактору.

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
