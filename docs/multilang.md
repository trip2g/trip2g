# Мультиязычность (`lang` / `lang_redirect`)

Фича позволяет публиковать контент на нескольких языках с автоматическим редиректом посетителей, SEO-тегами `hreflang` и переключателем языков в шаблонах.

---

## Концепция

Мультиязычный раздел строится из двух типов страниц:

- **Хаб** — страница-диспетчер. Объявляет `lang_redirect` со списком языковых версий. Посетитель попадает на хаб и сразу перенаправляется на нужный язык.
- **Языковая версия** — обычная заметка с полем `lang: xx`. Содержит контент на конкретном языке.

```
index.md (хаб)
├── en/index.md   → lang: en
└── ru/index.md   → lang: ru
```

---

## Frontmatter API

### Хаб

```yaml
lang_redirect:
  - "[[en/index]]"
  - "[[ru/index]]"
```

Принимает строку или массив строк. Значения — ссылки в формате вики-ссылок (`[[путь]]`). Двойные скобки опциональны — можно писать просто `en/index`.

Одна ссылка — укороченная форма:

```yaml
lang_redirect: "[[en/index]]"
```

### Языковая версия

```yaml
lang: en
```

Значение приводится к нижнему регистру и обрезается от пробелов. Используйте стандартные BCP 47 теги: `en`, `ru`, `de`, `zh`, `fr`.

---

## Как это работает

### Загрузка (`mdloader`)

При загрузке vault происходит в такой последовательности:

1. Все заметки разбираются, `lang` и `lang_redirect` парсятся из frontmatter в `NoteView`.
2. Строится индекс basename'ов для разрешения вики-ссылок.
3. Извлекаются in-links (`extractInLinks`).
4. `resolveLangRedirects()` — для каждого хаба находит целевые заметки и создаёт `LangGroup`.
5. Генерируются HTML-рендеры страниц.

`resolveLangRedirects()` запускается после `extractInLinks()` и до `generatePageHTMLs()`. Это важно: разрешение языковых ссылок требует готового индекса заметок.

### Разрешение вики-ссылок

`resolveWikilinkTarget()` использует ту же логику, что и контентные ссылки. Порядок разрешения:

1. **Явный относительный путь** (`./`, `../`) — путь относительно текущего файла.
2. **Простой basename** — ищет заметку с таким именем файла.
3. **Путь с `/`** — обходит дерево директорий вверх.

### Запрос страницы

```
Посетитель открывает хаб
│
├── resp.Note.Redirect != nil → статический redirect (приоритет)
│
├── resp.Note.LangRedirects != nil && ?nolang не задан
│   ├── Читаем cookie "lang"
│   ├── Если нет cookie → парсим заголовок Accept-Language
│   ├── Ищем LangRedirect с подходящим языком
│   ├── Нашли && цель != текущая страница → 302 + ставим cookie "lang"
│   └── Не нашли → рендерим хаб как есть
│
└── Рендер страницы
    ├── Если у заметки есть поле lang → ставим cookie "lang"
    └── Если есть LangGroup → инжектируем hreflang теги в <head>
```

### Определение языка

Реализовано в `internal/langdetect/langdetect.go`.

```go
// Возвращает предпочтительный язык пользователя.
// Приоритет: cookie > Accept-Language > ""
DetectPreferred(cookieValue, acceptLanguage string) string
```

```go
// Парсит заголовок Accept-Language с учётом quality values.
// Пример: "en-US,en;q=0.9,ru;q=0.8" → "en"
// Возвращает первичный тег (en-US → en), игнорирует "*".
ParseAcceptLanguage(header string) string
```

Cookie `lang` ставится:
- При редиректе с хаба на языковую версию.
- При рендере любой страницы с полем `lang`.

Параметры cookie: срок 1 год, `SameSite=Lax`.

---

## Структуры данных

```go
// internal/model/note.go

// LangRedirect — одна разрешённая языковая альтернатива.
type LangRedirect struct {
    Lang string     // код языка из поля lang целевой заметки (например, "en", "ru")
    Note *NoteView  // разрешённая целевая заметка
    URL  string     // permalink целевой заметки
}

// LangGroup — связь хаба со всеми языковыми версиями.
// Один объект разделяется хабом и всеми целевыми заметками.
type LangGroup struct {
    Hub      *NoteView      // страница, объявившая lang_redirect
    Versions []LangRedirect // все разрешённые языковые версии
}

type NoteView struct {
    // ...

    // Код языка из frontmatter (например, "en"). Пустая строка = не задан.
    Lang string

    // Промежуточные вики-ссылки из lang_redirect до разрешения.
    LangRedirectTargets []string

    // Разрешённые языковые редиректы (только у хаба).
    LangRedirects []LangRedirect

    // Shared-объект: хаб + все версии. Есть у хаба и у каждой целевой заметки.
    LangGroup *LangGroup

    // Языковые альтернативы: lang → *NoteView. Не включает саму заметку.
    // Есть у хаба и у каждой целевой заметки.
    LangAlternatives map[string]*NoteView
}
```

---

## Layout: hreflang и html lang

При рендере страницы в `renderlayout.Params` передаются:

```go
type Params struct {
    // ...
    HrefLangs []HrefLang // теги <link rel="alternate" hreflang="xx" href="...">
    HTMLLang  string     // значение атрибута lang на <html>; по умолчанию "ru"
}

type HrefLang struct {
    Lang string // код языка или "x-default"
    Href string // полный URL (scheme + host + path)
}
```

Правила формирования hreflang:
- Хаб без `lang` → получает только тег `hreflang="x-default"`.
- Хаб с `lang` → получает тег для своего языка и `x-default`.
- Каждая языковая версия → получает тег для своего языка + теги всех сиблингов + `x-default` (хаб).

В HTML шаблоне:

```html
<html lang="{{ params.HTMLLang }}">
<head>
  {% for _, hl := range params.HrefLangs %}
    <link rel="alternate" hreflang="{{ hl.Lang }}" href="{{ hl.Href }}">
  {% endfor %}
</head>
```

---

## templateviews API

Используется в кастомных Jet-шаблонах через переменную `note`.

```go
// Код языка страницы (например, "en"). Пустая строка если не задан.
note.Lang() string

// Есть ли языковые альтернативы (хаб или языковая версия в группе).
note.HasLangAlternatives() bool

// Возвращает заметку-альтернативу для указанного языка. nil если нет.
note.LangAlternative("en") *Note

// Все альтернативы в виде среза, отсортированного по коду языка.
note.LangAlternativesList() []*Note
```

### Пример: переключатель языков

```jet
{# Переключатель языков в кастомном layout #}
{% if note.HasLangAlternatives() %}
  <nav class="lang-switcher">
    {% for _, alt := range note.LangAlternativesList() %}
      <a href="{{ alt.Permalink() }}">{{ alt.Lang() }}</a>
    {% endfor %}
  </nav>
{% endif %}
```

`LangAlternativesList()` не включает текущую страницу — только сиблинги. Для полного переключателя (включая текущий язык) добавьте текущую заметку вручную:

```jet
{% if note.HasLangAlternatives() %}
  <nav class="lang-switcher">
    <span>{{ note.Lang() }}</span>  {# текущий язык #}
    {% for _, alt := range note.LangAlternativesList() %}
      <a href="{{ alt.Permalink() }}">{{ alt.Lang() }}</a>
    {% endfor %}
  </nav>
{% endif %}
```

---

## Edge cases

### `?nolang`

Любое значение параметра `?nolang` подавляет редирект и установку cookie для данного запроса. Полезно для авторов, ботов и SEO-инструментов.

```
https://example.com/docs?nolang
```

### Хаб без поля `lang`

Получает только `hreflang="x-default"`. Языковые альтернативы доступны через `LangAlternatives`.

### Хаб с полем `lang`

Получает и тег своего языка, и `x-default`.

### Самоссылка

Если разрешённая цель `lang_redirect` совпадает с самой страницей (`lr.Note == resp.Note`) — редирект не происходит. Страница рендерится как есть.

### Цель без поля `lang`

Пропускается с предупреждением на хабе:

```
lang_redirect target en/index has no lang field
```

### Цель не найдена

Пропускается с предупреждением, не является фатальной ошибкой:

```
lang_redirect target not found: en/index
```

### Дублирующиеся коды языков

Первый побеждает, остальные пропускаются с предупреждением:

```
lang_redirect duplicate language: en
```

### Два хаба, одна цель

Если заметка уже принадлежит `LangGroup` (её `LangGroup != nil`), второй хаб не перезаписывает группу. Второй хаб получает предупреждение:

```
lang_redirect target ru/index already belongs to another lang group, skipping
```

### Циклические ссылки

A → B, B → A — каждый разрешается независимо, без бесконечных циклов. Каждая заметка будет одновременно хабом и частью группы другого хаба (если проходит проверку `LangGroup != nil`).

### Кастомные домены

`hreflang` строится как `publicURL + Permalink`. Для заметок на кастомном домене это может быть некорректным — URL будет указывать на главный домен. Workaround: не используйте `lang_redirect` совместно с кастомными доменами, если URL должны отличаться.

---

## Типичная структура vault

```
vault/
├── index.md           ← хаб (lang_redirect: [[en/index]], [[ru/index]])
├── en/
│   └── index.md       ← lang: en
└── ru/
    └── index.md       ← lang: ru
```

`vault/index.md`:

```yaml
---
lang_redirect:
  - "[[en/index]]"
  - "[[ru/index]]"
---
```

`vault/en/index.md`:

```yaml
---
lang: en
---

# Welcome
```

`vault/ru/index.md`:

```yaml
---
lang: ru
---

# Добро пожаловать
```

---

## Ключевые файлы

| Файл | Что делает |
|------|------------|
| `internal/model/note.go` | `LangRedirect`, `LangGroup`, поля `Lang`/`LangRedirects`/`LangGroup`/`LangAlternatives` в `NoteView`; парсинг `extractLang()`, `extractLangRedirectTargets()` |
| `internal/mdloader/loader.go` | `resolveLangRedirects()`, `buildLangGroup()`, `resolveWikilinkTarget()` |
| `internal/langdetect/langdetect.go` | `DetectPreferred()`, `ParseAcceptLanguage()` |
| `internal/case/rendernotepage/endpoint.go` | Проверка редиректа, `setLangCookie()`, `buildHrefLangs()` |
| `internal/case/renderlayout/render.go` | `Params.HrefLangs`, `Params.HTMLLang`, `HrefLang` struct |
| `internal/templateviews/note.go` | `Lang()`, `HasLangAlternatives()`, `LangAlternative()`, `LangAlternativesList()` |
