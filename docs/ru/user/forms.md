---
title: Формы в заметках
free: true
lang_redirect: "[[en/user/forms]]"
---

Опишите форму в frontmatter заметки — trip2g встроит её на страницу как `<script id="form-spec">`, примет сабмиты через GraphQL API и сохранит их рядом с заметкой. Подходит для сбора заявок, домашних заданий, контактных форм — всего, что принимает структурированный ввод привязанный к странице.

Заметка остаётся обычной — открывается по permalink, попадает в поиск, экспортируется. Форма — её свойство.

### Минимальный пример

```yaml
---
title: Скажи привет
form:
  fields:
    - name: email
      type: email
      required: true
    - name: message
      type: text
      required: true
      max_length: 2000
---
```

И всё. При рендере страница получит:

```html
<script id="form-spec" type="application/json">
  { "note_version_id": 1234, "forms": { "": { "fields": [...] } } }
</script>
```

Дефолтный шаблон автоматически рисует форму в конце страницы. Кастомный layout читает тот же JSON и собирает свой вёрстку — см. [Кастомный layout](#kastomnyy_layout) ниже.

### Типы полей

| `type` | Хранится как | Валидаторы |
|---|---|---|
| `text` | строка | `required`, `min_length`, `max_length`, `enum: ["a","b"]` |
| `email` | строка | `required` (формат проверяется на сервере) |
| `int` | число | `required`, `min`, `max`, `enum: [1,2,3]` |
| `bool` | булево | `required`, `enum: [true]` (для чекбокса согласия) |
| `file` | — | пока не реализовано — сервер вернёт `file_upload_not_supported` |

Каждое поле требует `name` и `type`. `required` по умолчанию `false`. `enum` принимает значения того же типа, что и поле — выбирается ровно одно из перечисленных.

### Кто может отправлять — `can_submit`

```yaml
form:
  can_submit: admin    # guest (по умолчанию) | admin | paid_user
```

| Значение | Поведение |
|---|---|
| отсутствует / `guest` | Любой, включая анонимных посетителей |
| `admin` | Только авторизованные админы |
| `paid_user` | Принимается в spec, но **пока не энфорсится** — сервер вернёт `not_implemented` при сабмите |

Если посетитель может видеть заметку, но не может сабмитить — форма всё равно рендерится. На отправку сервер возвращает `FormSubmitDeniedPayload`:

```json
{ "__typename": "FormSubmitDeniedPayload", "reason": "admin_required" }
```

Дефолтный layout показывает подсказку «войдите как админ». Кастомный layout может ветвиться по `reason` и предлагать свой UX.

### Редирект после успеха — `success_url`

```yaml
form:
  success_url: /thanks
```

После успешного сабмита layout уводит браузер на `success_url`. Относительные URL остаются в том же домене, абсолютные могут вести куда угодно (например, на внешнюю страницу с подтверждением).

### Защита от спама — `turnstile`

Cloudflare Turnstile **включён по умолчанию** на каждой форме. Чтобы отключить — поставьте `turnstile: false`:

```yaml
form:
  turnstile: false   # принимать сабмиты без капчи
```

Как это работает:

- Если токена нет или он не прошёл — сервер возвращает `TurnstileRequiredPayload { siteKey }`.
- Стандартный layout читает `siteKey`, рисует виджет Turnstile и пересабмитит с заполненным `turnstileToken`.
- Локально (без `turnstile-secret-key`) проверка отключена — любой сабмит проходит. В проде ключ задан и капча гейтит каждый сабмит.

Сочетайте с `can_submit: admin` для чувствительных форм; на публичных формах капча — единственная защита от анонимного спама.

### Несколько форм на одной заметке — `forms:`

Если на заметке нужно больше одной формы, используйте `forms:` (map с именованными ключами) вместо одиночного `form:`:

```yaml
forms:
  contact:
    fields:
      - name: email
        type: email
  survey:
    can_submit: admin
    fields:
      - name: rating
        type: int
        min: 1
        max: 5
```

Клиент обращается к нужной форме по ключу — передаёт `formId: "contact"` или `formId: "survey"` в `submitForm`. Одиночный `form:` эквивалентен `forms` с пустым ключом.

### Переиспользование спеки — `form_ref:`

Чтобы не дублировать одинаковые поля на множестве заметок, опишите spec один раз и ссылайтесь на него:

```yaml
# В общей заметке, например templates/comment_form.md
form:
  fields:
    - name: text
      type: text
      required: true
      max_length: 4000
```

```yaml
# В любой другой
form_ref: "[[templates/comment_form]]"
```

Или по пути файла:

```yaml
form_ref: templates/comment_form.md
```

Сабмиты по-прежнему сохраняются у ссылающейся заметки, а сама spec живёт в одном месте. Комбинируется с [[ru/user/frontmatter_patches|frontmatter-патчами]] — можно прикрепить форму ко всей папке, не трогая отдельные файлы:

```jsonnet
{ form_ref: "[[templates/comment_form]]" }
```

### Кастомный layout

В кастомном Jet-layout у `note` есть метод `FormSpecJSON()` — встройте его в `<script>` и читайте из JS:

```jet
{{ block content() }}
  <article>{{ note.HTMLString() | unsafe }}</article>

  <form id="my-form"></form>
  <div id="my-status"></div>

  <script id="form-spec" type="application/json">{{ note.FormSpecJSON() | unsafe }}</script>
  <script>
    const spec = JSON.parse(document.getElementById('form-spec').textContent);
    const def = spec.forms[''];
    // соберите поля по def.fields, потом сабмитьте через fetch('/_system/graphql', {...})
  </script>
{{ end }}
```

Готовый пример лежит в `docs/_layouts/forms/example.html` — с рендером полей, сообщениями об ошибках и редиректом по `success_url`. Можно взять за основу.

### Сабмит через GraphQL

Мутация — часть публичной схемы, её может вызвать любой HTTP-клиент.

```graphql
mutation Submit($input: SubmitFormInput!) {
  submitForm(input: $input) {
    __typename
    ... on SubmitFormPayload          { submitId }
    ... on FormSubmitDeniedPayload    { reason }
    ... on ErrorPayload               { message byFields { name value } }
  }
}
```

Переменные для одиночной формы:

```json
{
  "input": {
    "noteVersionId": 1234,
    "formId": "",
    "fields": [
      { "name": "email",   "stringValue": "alice@example.com" },
      { "name": "rating",  "intValue":    5 },
      { "name": "agree",   "boolValue":   true }
    ]
  }
}
```

`noteVersionId` берётся из поля `note_version_id` в `<script id="form-spec">`. `formId` равен `""` для одиночного `form:` блока; для `forms:` map — имя ключа.

| `__typename` ответа | Что значит |
|---|---|
| `SubmitFormPayload` | Принято; `submitId` — id записи |
| `FormSubmitDeniedPayload` | `reason` = `admin_required` / `paid_required` / `not_implemented` |
| `ErrorPayload` | Валидация не прошла; `message` — общее сообщение, `byFields[]` — детали по полям |

После принятого сабмита автоматически ставится фоновая задача на email админам.

### Чтение сабмитов

#### Админ-панель

В админ-панели у каждой заметки есть раздел «Формы»: список сабмитов с датой, IP, статусом и значениями полей. Оттуда же можно пометить запись как обработанную.

#### Через API-ключ

Для программного получения сабмитов используйте админский personal token (`Authorization: Bearer t2g_…`). Токен должен принадлежать администратору — иначе запрос вернёт `unauthorized`. Все запросы отправляются на `/_system/graphql` с заголовками `Content-Type: application/json` и `Authorization: Bearer $TRIP2G_TOKEN`.

**Найти заметки с сабмитами**

```graphql
query { admin { formNotes { notePathId path title submitCount } } }
```

Используйте этот запрос первым — он даст `notePathId` для дальнейшей фильтрации.

**Получить список сабмитов**

```graphql
query AdminFormSubmits($filter: AdminFormSubmitsFilterInput) {
  admin {
    formSubmits(filter: $filter) {
      totalCount
      nodes {
        id
        noteVersionId
        formId
        ip
        status
        createdAt
        processedAt
        comment
        fields {
          __typename
          ... on AdminFormStringValue { name value }
          ... on AdminFormIntValue    { name iv: value }
          ... on AdminFormBoolValue   { name bv: value }
        }
      }
    }
  }
}
```

Поля фильтра — все необязательны:

| Поле | Тип | Описание |
|---|---|---|
| `notePathId` | Int64 | Сабмиты конкретной заметки |
| `formId` | String | Имя формы (`""` для одиночного `form:`) |
| `status` | String | `"pending"` или `"processed"` |
| `processed` | Boolean | `false` — только необработанные |
| `createdAt.gte` / `.lte` | DateTime | Диапазон дат |
| `limit` | Int | Размер страницы |
| `offset` | Int | Смещение для пагинации |

Пример — необработанные сабмиты конкретной заметки за май 2026:

```json
{
  "filter": {
    "notePathId": 123,
    "formId": "contact",
    "processed": false,
    "createdAt": { "gte": "2026-05-01T00:00:00Z", "lte": "2026-05-31T23:59:59Z" },
    "limit": 50,
    "offset": 0
  }
}
```

**Пометить сабмит как обработанный**

```graphql
mutation MarkProcessed($input: MarkFormSubmitProcessedInput!) {
  admin {
    markFormSubmitProcessed(input: $input) {
      __typename
      ... on MarkFormSubmitProcessedPayload {
        submit { id processedAt comment }
      }
      ... on ErrorPayload {
        message
        fields { name message }
      }
    }
  }
}
```

Переменные:

```json
{ "input": { "id": 17, "comment": "ответили по email" } }
```

Поле `comment` необязательно — удобно оставить заметку о том, как обработан сабмит.

### Массово: форма на всю папку

Свяжите формы с [[ru/user/frontmatter_patches|frontmatter-патчами]], чтобы раскатать форму комментариев (или контактную) на весь раздел, не редактируя каждую заметку:

```jsonnet
// blog/**.md
{ form_ref: "[[templates/comment_form]]" }
```

Все заметки в `blog/` получат общую форму. Добавьте `exclude: ["blog/drafts/*"]` чтобы пропустить черновики.

### Что ещё не реализовано

- **`type: file`** — загрузка файлов возвращает `file_upload_not_supported`. Пока используйте отдельный шаг с загрузкой в object storage.
- **`can_submit: paid_user`** — поле признаётся, но сервер возвращает `FormSubmitDeniedPayload { reason: "not_implemented" }`. Пока используйте `admin`.
- **Капчи, кроме Cloudflare Turnstile** — hCaptcha и Yandex SmartCaptcha в роадмапе (Turnstile подключён и включён по умолчанию, см. «Защита от спама» выше).

### Связанные доки

- [[ru/user/frontmatter_patches|Frontmatter-патчи]] — массово навесить `form:` или `form_ref:` на папку
- [[ru/user/change_webhooks|Вебхуки при изменении]] — узнать о новом сабмите вне админки
- [[ru/user/selfhosted|Self-hosted]] — собственная инсталляция, где живёт GraphQL-эндпоинт и API-ключи
