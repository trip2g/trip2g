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
Посетитель открывает страницу
│
├── resp.Note.Redirect != nil → статический redirect (приоритет)
│
├── ?setlang=xxx задан
│   ├── Ставим cookie trip2g_lang=xxx (1 год, SameSite=Lax)
│   ├── note.Lang != xxx && note.LangAlternatives[xxx] != nil → 302 на альтернативу
│   └── Иначе → 302 на note.Permalink (убирает query params)
│
├── resp.Note.LangRedirects != nil && note.Lang == "" (хаб без своего языка)
│   ├── Читаем cookie trip2g_lang, затем Accept-Language
│   ├── Ищем LangRedirect с подходящим языком
│   ├── Нашли && цель != текущая страница → 302
│   └── Не нашли → рендерим хаб как есть
│
└── Рендер страницы
    ├── UILang = DetectPreferred(cookie trip2g_lang, Accept-Language)
    ├── HTMLLang = note.Lang (для <html lang="">)
    └── Если есть LangGroup → hreflang теги в <head>
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

Cookie `trip2g_lang` ставится:
- При обработке `?setlang=xxx` (явное переключение языка пользователем).

Параметры cookie: срок 1 год, `SameSite=Lax`, `Path=/`.

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

## ?setlang — смена языка интерфейса

`?setlang=xxx` — единственный способ явно установить предпочтительный язык пользователя.

**Поведение:**
1. Ставит cookie `trip2g_lang=xxx`
2. Если текущая заметка имеет `lang` и он отличается от `xxx`, и есть альтернатива для `xxx` → 302 на альтернативу
3. Иначе → 302 на текущий permalink (убирает `?setlang` из URL)

**Реализация:** `handleSetLang()` в `internal/case/rendernotepage/endpoint.go`

### Два типа переключателей

| Тип | Механизм | Меняет cookie | Назначение |
|-----|----------|---------------|------------|
| **Язык интерфейса** (JS виджет) | `?setlang=xxx` | ✓ | Глобальное переключение языка сайта |
| **Язык контента** (`lang-switcher`) | Прямая ссылка на альтернативу | ✗ | Прочитать конкретную статью на другом языке |

Разделение важно: пользователь может кликнуть на контентный переключатель из интереса, не желая менять язык всего сайта.

### Хаб как языковая версия

Хаб может одновременно быть и языковой версией — для этого задаётся и `lang`, и `lang_redirect`:

```yaml
---
lang: en
lang_redirect: "[[ru/_index]]"
---
```

В таком случае хаб получает `hreflang` своего языка + `x-default`, и редирект на него не происходит при совпадении языка.

---

## Layout: hreflang и html lang

При рендере страницы в `defaulttemplate.Ctx` передаются два разных языковых поля:

```go
type Ctx struct {
    HTMLLang string // note.Lang → атрибут <html lang="">; язык контента заметки
    UILang   string // DetectPreferred(cookie trip2g_lang, Accept-Language) → язык интерфейса
}
```

`UILang` используется в JS через `window.__trip2g_settings.ui_lang`.

В `renderlayout.Params` (для Jet-шаблонов):

```go
type Params struct {
    HrefLangs []HrefLang // теги <link rel="alternate" hreflang="xx" href="...">
    HTMLLang  string     // значение атрибута lang на <html>
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

## JS API

### `window.__trip2g_settings`

Поля, доступные на каждой странице:

```typescript
window.__trip2g_settings = {
  ui_lang: string,   // предпочтительный язык интерфейса (cookie → Accept-Language → "")
  note_lang: string, // язык текущей заметки (note.Lang, только если note != nil)
}
```

### `$trip2g_settings`

```typescript
// Язык интерфейса пользователя
$trip2g_settings.ui_lang(): string

// Язык текущей заметки
$trip2g_settings.note_lang(): string

// Переключить язык интерфейса.
// Если ui_lang уже равен lang — ничего не делает.
// Иначе: добавляет ?setlang=lang к текущему URL и навигирует.
$trip2g_settings.set_lang(lang: string): string
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
| `internal/case/rendernotepage/endpoint.go` | `handleSetLang()` (cookie + redirect), `redirectToRightLang()`, `buildHrefLangs()`; заполняет `Ctx.UILang` и `Ctx.HTMLLang` |
| `internal/defaulttemplate/template.go` | `Ctx.UILang` (язык интерфейса), `Ctx.HTMLLang` (язык заметки для `<html lang>`) |
| `internal/case/renderlayout/render.go` | `Params.HrefLangs`, `Params.HTMLLang`, `HrefLang` struct |
| `internal/templateviews/note.go` | `Lang()`, `HasLangAlternatives()`, `LangAlternative()`, `LangAlternativesList()` |
| `assets/ui/settings/settings.ts` | `$trip2g_settings.ui_lang()`, `note_lang()`, `set_lang()` |
