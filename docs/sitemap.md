# Sitemap.xml

## Концепция

Sitemap.xml автоматически генерируется из всех опубликованных заметок и отдаётся по URL `/sitemap.xml`. Это помогает поисковым системам индексировать контент сайта.

## Правила включения страниц

В sitemap попадают только:
- **Опубликованные заметки** — с флагом `free: true` в frontmatter
- **Пользовательские страницы** — исключаются системные страницы с префиксом `/_*` (например `/_404`, `/_about`)

Заметки с `free: false` (за paywall) **не включаются** в sitemap.

## Метаданные

Для каждой страницы в sitemap.xml указывается:
- **URL** — полный путь к заметке (например `https://example.com/blog/my-post`)
- **lastmod** — дата последнего изменения, берётся из:
  - Frontmatter поля `created_at` или `created_on` (если указано)
  - Иначе из `CreatedAt` заметки

## Реализация

### Пакет `internal/sitemap/`

Отвечает за генерацию XML в формате [sitemap protocol](https://www.sitemaps.org/protocol.html):

```go
package sitemap

func Generate(notes []*model.NoteView, baseURL string) ([]byte, error)
```

На вход принимает список заметок и базовый URL сайта. Возвращает готовый XML.

### Генерация при загрузке

Sitemap генерируется один раз при старте приложения в `internal/noteloader/loader.go`:

```go
func Load(ctx context.Context, q Queryer) (*model.NoteViews, error) {
    // ... загрузка заметок ...

    // Генерируем sitemap из всех заметок
    sitemapXML, err := sitemap.Generate(allNotes, baseURL)
    if err != nil {
        return nil, fmt.Errorf("generate sitemap: %w", err)
    }

    return &model.NoteViews{
        Notes:   allNotes,
        Sitemap: sitemapXML,
    }, nil
}
```

Результат сохраняется в поле `NoteViews.Sitemap []byte`.

### Отдача по HTTP

В `cmd/server/main.go` middleware `handleSitemap` обрабатывает запросы к `/sitemap.xml`:

```go
func handleSitemap(views *model.NoteViews) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !strings.HasSuffix(r.URL.Path, "/sitemap.xml") {
            return // не наш запрос
        }

        w.Header().Set("Content-Type", "application/xml; charset=utf-8")
        w.Write(views.Sitemap)
    }
}
```

При каждом обновлении заметок (перезагрузка конфигурации) sitemap автоматически пересоздаётся.

## Формат

Sitemap генерируется в стандартном формате [sitemaps.org](https://www.sitemaps.org/):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <url>
        <loc>https://example.com/blog/my-post</loc>
        <lastmod>2024-01-15T10:30:00Z</lastmod>
    </url>
    <url>
        <loc>https://example.com/about</loc>
        <lastmod>2024-01-10T15:00:00Z</lastmod>
    </url>
</urlset>
```

## URL

Sitemap доступен по адресу: **`/sitemap.xml`**

Пример: `https://yoursite.com/sitemap.xml`
