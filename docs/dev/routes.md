# Маршруты (`route`/`routes`)

## Концепция

Поля `route`/`routes` во frontmatter позволяют заметке быть доступной по произвольным URL — на главном домене или на кастомных доменах — **не меняя её permalink** в `nv.Map`.

Ключевое разграничение:

| Поле | Меняет permalink в `nv.Map` | Попадает в `RouteMap` | Назначение |
|------|-----------------------------|-----------------------|------------|
| `slug` | **Да** | Нет | Обратная совместимость, кастомный URL заметки |
| `route`/`routes` | **Нет** | **Да** | Алиасы на главном домене, кастомные домены |

`slug` и `route` полностью независимы: можно использовать оба на одной заметке.

---

## Frontmatter

```yaml
# Алиас на главном домене (без хоста)
route: /about

# Корень кастомного домена
route: mysite.com/

# Кастомный домен, конкретный путь
route: mysite.com/hello

# Несколько маршрутов (mix главный + кастомный)
routes:
  - /my-alias
  - mysite.com/
  - other.com/landing
```

### Правила парсинга

| Значение | host | path | Поведение |
|----------|------|------|-----------|
| `/about` | `""` | `/about` | Алиас на главном домене |
| `/` | `""` | `/` | Алиас на корне главного домена |
| `foo.com` | `foo.com` | `""` | Кастомный домен, path = permalink заметки |
| `foo.com/` | `foo.com` | `/` | Кастомный домен, явный корень |
| `foo.com/hello` | `foo.com` | `/hello` | Кастомный домен, конкретный путь |
| `localhost:8081/p` | `localhost:8081` | `/p` | Dev-режим с портом |

**Нормализация домена:** lowercase + удаление `www.`. Реализация: `model.NormalizeDomain()`.

Пример: `www.FOO.COM/about` → host=`foo.com`, path=`/about`.

---

## Разрешение запроса

`resolveNote()` в `internal/case/rendernotepage/resolve.go`:

```
запрос пришёл на host H, path P
│
├── H == главный домен (или пустой)?
│   ├── RouteMap[""][P] → нашли → отдаём
│   └── nv.Map[P]       → нашли → отдаём
│
└── H == кастомный домен
    ├── RouteMap[H][P]  → нашли → отдаём
    └── nv.Map[P]       → fallthrough по permalink
```

**Важно:** Алиасы главного домена (`route: /x`) **не доступны** на кастомном домене по тому же пути. `RouteMap[""]["/x"]` не используется при запросе с `H != главный_домен`.

**Fallthrough на кастомном домене:** если маршрут не найден в `RouteMap[H]`, сервер всё равно ищет заметку в `nv.Map` по permalink. Это означает, что посетители кастомного домена могут получить доступ к любой публичной заметке по её permalink — не только к явно прописанным в `route`.

---

## Приоритет при коллизиях

- `route: /` на главном домене **перекрывает** `_index.md` в `RouteMap`, но `nv.Map["/"]` по-прежнему указывает на `_index.md` (роут не трогает `nv.Map`)
- Два маршрута с одинаковым host+path → побеждает последний зарегистрированный (порядок загрузки заметок)
- `slug` + `route` на одной заметке — независимы, не конфликтуют

---

## OG теги

На кастомном домене `og:url` формируется из маршрута, а не из permalink.

Логика (`buildOGTags` → `ogURLForNote` в `endpoint.go`):
1. Если запрос пришёл с кастомного домена — ищем в `note.Routes` маршрут с `Host == requestHost`
2. Предпочтение: точное совпадение host+path; иначе первый маршрут с нужным host
3. Схема берётся из `env.PublicURL()` (http/https)

---

## Sitemap для кастомных доменов

При перезагрузке заметок (`noteloader/loader.go`) генерируются:
- `nvs.Sitemap` — основной sitemap для главного домена (все `free: true` заметки по permalink)
- `nvs.DomainSitemaps[domain]` — отдельный sitemap для каждого кастомного домена

Sitemap кастомного домена содержит только заметки с явными маршрутами на этот домен. URL в sitemap строятся как `<domainURL><route.Path>`.

Запрос `/sitemap.xml` с заголовком `Host: foo.com` отдаёт `DomainSitemaps["foo.com"]`, если он существует.

---

## Frontmatter patches

`route`/`routes` извлекаются **после** применения всех патчей (в `ExtractMetaData()` → `ExtractRoutes()`). Это значит, что патч может **добавить** маршрут к заметке:

```
# Патч для multidomain/landing.md:
{ "route": "mydomain.com/" }
```

Подробнее: [docs/frontmatter_patches.md → раздел «Маршруты и патчи»](frontmatter_patches.md).

---

## Ограничения

**TLS.** Кастомные домены требуют внешней TLS-терминации (reverse proxy, Cloudflare и т.п.). Список ACME-доменов в конфиге статичен и не синхронизируется с `RouteMap` автоматически.

**Cookies / авторизация.** Браузерная политика CORS/cookies: пользователь, авторизованный на главном домене, будет гостем на кастомном. Для контента под paywall используйте `free: true`.

**TrustedDomains.** Кастомные домены из `RouteMap` не добавляются автоматически в `TrustedDomains()` (используется при валидации redirect URL после Telegram-авторизации).

---

## Структуры данных

```go
// internal/model/note_routes.go

type ParsedRoute struct {
    Host string // "" = главный домен; "foo.com" = кастомный
    Path string // "" = использовать Permalink заметки; "/x" = явный путь
}

func ParseRoute(value string) ParsedRoute
func NormalizeDomain(d string) string
func ExtractHost(rawURL string) string
```

```go
// internal/model/note.go

type NoteView struct {
    // ...
    Routes []ParsedRoute // из frontmatter route/routes; не влияет на Permalink
}

type NoteViews struct {
    Map     map[string]*NoteView            // keyed by Permalink
    PathMap map[string]*NoteView            // keyed by file path
    RouteMap map[string]map[string]*NoteView // host → path → note

    Sitemap        []byte            // основной sitemap.xml
    DomainSitemaps map[string][]byte // host → sitemap XML

    // ...
}

func (nv *NoteViews) GetByRoute(host, path string) *NoteView
func (nv *NoteViews) RegisterNoteRoutes(note *NoteView)
func (nv *NoteViews) CustomDomains() []string
```

---

## Ключевые файлы

| Файл | Что делает |
|------|------------|
| `internal/model/note_routes.go` | `ParsedRoute`, `ParseRoute`, `NormalizeDomain`, `ExtractHost`, `ExtractRoutes` |
| `internal/model/note.go` | `Routes` в `NoteView`; `RouteMap`, `DomainSitemaps` в `NoteViews`; `RegisterNoteRoutes`, `GetByRoute`, `CustomDomains` |
| `internal/mdloader/loader.go` | Вызов `ExtractRoutes()` после патчей (строка ~607) |
| `internal/noteloader/loader.go` | Генерация `DomainSitemaps` после основного sitemap |
| `internal/case/rendernotepage/resolve.go` | `resolveNote()` — domain-aware lookup |
| `internal/case/rendernotepage/endpoint.go` | `buildOGTags()` / `ogURLForNote()` / `findRouteForHost()` |
| `internal/sitemap/sitemap.go` | `Generate()` + `GenerateForDomain()` |
| `cmd/server/main.go` | `handleSitemap()` — роутинг по Host заголовку |
