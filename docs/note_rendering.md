# Рендеринг заметок

## Обзор

Процесс рендеринга заметки — это цепочка обработчиков от HTTP-запроса до HTML-ответа. Система использует два набора заметок (latest/live), поддерживает paywall, кастомные layout'ы и версионность.

## HTTP Request Pipeline

### 1. Точка входа: fasthttp server

`cmd/server/main.go:handler()` (строка 1846) — основной обработчик всех HTTP-запросов.

```
HTTP Request → fasthttp handler → middleware chain → GraphQL | Redirect | Router
```

### 2. Middleware chain

Middlewares обрабатываются последовательно (строка 1857-1860). Если middleware вернёт `true` — запрос считается обработанным:

| Порядок | Middleware | Что делает |
|---------|-----------|------------|
| 1 | `handleRobotsTxt` | `/robots.txt` |
| 2 | `handleRSSFeed` | `*.rss.xml` |
| 3 | `handleCors` | CORS headers |
| 4 | `handleDebugAPI` | Debug endpoints |
| 5 | `gitAPI.HandleRequest` | Git operations |
| 6 | `handleAdminAssets` | Admin JS auth check |
| 7 | assets handler | `/assets/*` |
| 8 | `handlePurchaseTokens` | Purchase token processing |
| 9 | `signinbytgauthtoken` | Telegram auth |
| 10 | `TgBots.ProcessWebhookRequest` | Telegram webhooks |

### 3. GraphQL / Redirect / Router

Если middleware не обработал запрос:
- GraphQL handler (`/graphql`) — строка 1886
- Redirect manager — строка 1890
- Router с зарегистрированными endpoints — строка 1897

Если роутер не обработал → падает в catch-all endpoint `rendernotepage`.

## Endpoint: rendernotepage

Точка входа: `internal/case/rendernotepage/endpoint.go:Handle()`

### Фаза 1: Извлечение параметров

```go
request := Request{
    Path:     string(req.Req.URI().Path()),
    Version:  string(req.Req.QueryArgs().Peek("version")),
    Referrer: string(req.Req.Request.Header.Peek("Referer")),
    UserToken: token,
}
```

### Фаза 2: Business logic — Resolve()

`internal/case/rendernotepage/resolve.go:Resolve()`

**Определение версии:**
```go
// Админы по умолчанию видят latest, пользователи — live
isAdmin := request.UserToken.IsAdmin()
isLatest := request.Version == "latest" || (isAdmin && request.Version == "")

if isLatest {
    notes = env.LatestNoteViews()
} else {
    notes = env.LiveNoteViews()
}
```

**Поиск заметки:**
```go
note := notes.GetByPath(path)  // Прямой lookup в map по пути
```

**Access control:**
```go
if !note.Free && request.UserToken == nil {
    return &PaywallError{Message: "Need auth"}
}

hasAccess, err := env.CanReadNote(ctx, note)
if !hasAccess {
    return &PaywallError{Message: "Need subscription"}
}
```

**Response:**
```go
response := Response{
    Title: formatTitle(note.Title, env.SiteTitleTemplate()),
    Note:  note,
    Notes: notes,
    ...
}
```

### Фаза 3: Rendering

`endpoint.go:Handle()` — выбор режима рендера:

**1. Redirect note** (строка 63):
```go
if resp.Note.Redirect != nil {
    ctx.Response.Header.Set("Location", *resp.Note.Redirect)
    ctx.SetStatusCode(http.StatusFound)
}
```

**2. Onboarding** (строка 69) — если нет заметок:
```go
if resp.OnboardingMode {
    return renderlayout.Handle(req, layoutParams, func() {
        WriteOnboarding(ctx, resp)
    })
}
```

**3. Paywall** (строка 79):
```go
var paywallErr *PaywallError
if errors.As(err, &paywallErr) {
    return renderlayout.Handle(req, layoutParams, func() {
        WritePayWall(ctx, resp, paywallErr)
    })
}
```

**4. Turbo response** (строка 97) — только HTML контент без layout:
```go
if turbo := len(ctx.Request.Header.Peek("X-Turbo")) > 0; turbo {
    WriteTurboNote(ctx, resp)
}
```

**5. Custom layout** (строка 109) — Jet template engine:
```go
if layout := resp.Note.Layout; layout != "" {
    layout := env.Layouts().Map["/"+layoutName]

    vars := jet.VarMap{
        "note":  templateviews.NewNote(resp.Note),
        "nvs":   templateviews.NewNVS(resp.Notes, resp.DefaultVersion),
        "title": resp.Title,
    }

    layout.View.Execute(ctx, vars, resp)
}
```

**6. Standard layout** (строка 120) — quicktemplate:
```go
return renderlayout.Handle(req, layoutParams, func() {
    WriteNote(ctx, resp)  // quicktemplate генерирует HTML
})
```

## NoteViews: загрузка и кеширование

Заметки загружаются в память при старте сервера и хранятся в двух версиях.

### Loaders

`cmd/server/main.go` (строка 360-361):
```go
a.liveNoteLoader = noteloader.New("live", makeLiveNoteLoaderWrapper(a), config)
a.latestNoteLoader = noteloader.New("latest", makeLatestNoteLoaderWrapper(a), config)
```

Загрузка: `a.loadAllNotes(ctx, options)` (строка 720)

### Структура NoteView

Key fields (`internal/model/note.go`):

| Поле | Тип | Описание |
|------|-----|----------|
| `Path` | `string` | URL путь заметки (например `/blog/post`) |
| `PathID` | `int64` | ID пути в БД |
| `VersionID` | `int64` | ID версии заметки |
| `Title` | `string` | Заголовок |
| `Permalink` | `string` | URL для ссылок |
| `Content` | `[]byte` | Исходный markdown |
| `HTML` | `string` | Рендеренный HTML |
| `Description` | `*string` | Описание для meta tags |
| `CreatedAt` | `time.Time` | Дата создания |
| `Free` | `bool` | Доступна без подписки |
| `Redirect` | `*string` | URL для редиректа |
| `RawMeta` | `map[string]interface{}` | Frontmatter |
| `ResolvedLinks` | `map[string]string` | Wikilinks → resolved URLs |
| `SubgraphNames` | `[]string` | Список подграфов |
| `FirstImage` | `*string` | Первое изображение для OG |
| `Layout` | `string` | Кастомный layout |
| `Slug` | `string` | URL slug |
| `RSSTitle` | `string` | Заголовок для RSS |
| `RSSDescription` | `string` | Описание для RSS |

### Парсинг Markdown

`internal/noteloader/` использует `internal/mdloader/`:

**Goldmark extensions:**
- `goldmark-meta` — frontmatter parsing
- `goldmark-wikilink` — `[[wikilinks]]`

**Pipeline:**
```
Markdown → Goldmark Parser → AST → NoteView
                                ↓
                           ResolvedLinks map
```

AST хранится в `NoteView.ast` и используется для:
- Генерации HTML
- Извлечения ссылок для RSS
- Поиска по контенту

**Разрешение ссылок:**
```go
wikilinks → notes.GetByPath(target) → ResolvedLinks[target] = note.Permalink
```

## Title template

`resolve.go:formatTitle()` (строка 303):
```go
func formatTitle(noteTitle, template string) string {
    return fmt.Sprintf(template, noteTitle)
}
```

Template берётся из конфига `site_title_template` (по умолчанию `"%s"`).

Пример:
```
note.Title = "My Post"
template = "%s | My Blog"
→ result = "My Post | My Blog"
```

## Not Found (404)

Если заметка не найдена:
```go
if note := notes.GetByPath(path); note == nil {
    return &response, ErrNotFound
}
```

Обработка в `endpoint.go` (строка 87):
```go
if errors.Is(err, ErrNotFound) {
    ctx.SetStatusCode(http.StatusNotFound)
    return render404.Handle(req)
}
```

## User tracking

Когда пользователь открывает заметку, записывается:
- `user_note_views` — каждое открытие
- `user_note_daily_views` — счётчик за день (max 100)

Запись происходит асинхронно (строка 245):
```go
go func() {
    bgCtx := context.Background()
    env.RecordUserNoteView(bgCtx, userID, note, referrerVersionID)
}()
```

## OG Tags

Open Graph tags генерируются в `endpoint.go` (строка 50):
```go
layoutParams.OGTags = map[string]string{
    "og:url":  env.PublicURL() + resp.Note.Permalink,
    "og:type": "article",
}

if resp.Note.FirstImage != nil {
    layoutParams.OGTags["og:image"] = assetReplace.URL
}
```

## Кеширование

**NoteViews** — хранятся в памяти, перезагружаются при:
- `PrepareLiveNotes()` / `PrepareLatestNotes()` — после сохранения заметок
- Истечении presigned URLs (строка 490-504 в `main.go`)

**Asset URLs** — presigned URLs с TTL, обновляются автоматически перед истечением.
