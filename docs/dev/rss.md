# RSS

## Концепция

Любая заметка — это RSS лента. Markdown AST страницы преобразуется в RSS feed.

Каждая ссылка в заметке становится RSS item. Заголовок ссылки → `<title>`, URL → `<link>`. Если ссылка ведёт на внутреннюю заметку — подтягиваются её метаданные (description, created_at).

## URL схема

`/path/to/note.rss.xml` — RSS фид для любой заметки.

## Преобразование Markdown AST → RSS

Парсим AST заметки, извлекаем все ссылки. Порядок items = порядок ссылок в документе.

Каждая ссылка → RSS item:
- `<title>` — текст ссылки
- `<link>` — URL (абсолютный)
- `<description>` — description целевой заметки (если внутренняя ссылка)
- `<pubDate>` — `rss_created_at` или `created_at` целевой заметки

## Frontmatter (опционально)

| Поле | Тип | Описание |
|------|-----|----------|
| `rss_title` | `string` | Заголовок фида (по умолчанию title заметки) |
| `rss_description` | `string` | Описание фида (по умолчанию description заметки) |

## Реализация

### Структура кода

| Компонент | Путь | Описание |
|-----------|------|----------|
| Package | `internal/rssfeed/` | Генерация RSS 2.0 XML |
| Middleware | `cmd/server/main.go:handleRSSFeed()` | Перехват `*.rss.xml` запросов |
| Config | `model.SiteConfig.EnableRSS` | Булев конфиг (default `true`) |
| Frontmatter | `NoteView.RSSTitle`, `RSSDescription` | Переопределение заголовка/описания |

### Middleware: handleRSSFeed

`cmd/server/main.go:1763-1796`

```go
func (a *app) handleRSSFeed(req *appreq.Request) bool {
    // 1. Проверка суффикса .rss.xml
    if !strings.HasSuffix(req.Path, ".rss.xml") {
        return false
    }

    // 2. Проверка конфига enable_rss
    cfg := a.SiteConfig(context.Background())
    if !cfg.EnableRSS {
        return false
    }

    // 3. Извлечение пути заметки
    notePath := strings.TrimSuffix(req.Path, ".rss.xml")

    // 4. Поиск заметки в LiveNoteViews
    notes := a.LiveNoteViews()
    note := notes.GetByPath(notePath)
    if note == nil {
        return false
    }

    // 5. Генерация RSS
    xmlBytes, err := rssfeed.Generate(note, a.PublicURL(), notes)

    // 6. Отдача XML
    req.Req.SetContentType("application/rss+xml; charset=utf-8")
    req.Req.SetBody(xmlBytes)
    return true
}
```

### Generator: rssfeed.Generate()

`internal/rssfeed/rssfeed.go:50`

**Входные данные:**
- `note *model.NoteView` — заметка для конвертации
- `publicURL string` — base URL сайта
- `notes *model.NoteViews` — для резолва internal links

**Алгоритм:**

1. **Извлечение ссылок** (`extractLinks()` строка 96):
   - Обход AST заметки через `ast.Walk()`
   - Поиск `wikilink.Node` и `ast.KindLink`
   - Skip embedded images/videos
   - Для wikilinks: резолв через `note.ResolvedLinks[target]`
   - Для markdown links: используется `l.Destination` как есть

2. **Построение feed**:
   ```go
   feedTitle := note.RSSTitle != "" ? note.RSSTitle : note.Title
   feedDesc := note.RSSDescription != "" ? note.RSSDescription : note.Description
   ```

3. **Генерация items**:
   - Каждая ссылка → `RSSItem`
   - Для internal links: обогащение метаданными (`enrichItem()` строка 191)

4. **Сериализация в XML**

### Обогащение внутренних ссылок

`enrichItem()` (строка 191-204):

```go
func enrichItem(item *RSSItem, notes *model.NoteViews, path string) {
    target := notes.GetByPath(path)
    if target == nil {
        return
    }

    if target.Description != nil {
        item.Description = *target.Description
    }

    if !target.CreatedAt.IsZero() {
        item.PubDate = target.CreatedAt.Format(time.RFC1123Z)
    }
}
```

Метаданные целевой заметки:
- `Description` → `<description>`
- `CreatedAt` → `<pubDate>` (RFC1123Z format)

### AST Walking

Поддерживаются два типа ссылок:

| Тип | AST Node | Пример | Обработка |
|-----|----------|--------|-----------|
| Wikilink | `wikilink.Node` | `[[target\|text]]` | Резолв через `ResolvedLinks` |
| Markdown link | `ast.KindLink` | `[text](url)` | URL используется как есть |

**Skip rules:**
- Embedded media: `![[image.png]]` — пропускается (строка 116)
- Image links: `[img](file.jpg)` — пропускается (строка 146)

### Конфиг: enable_rss

Таблица: `config_bools` (строка `value_id = 'enable_rss'`)

Загружается в `app.SiteConfig()` (`cmd/server/main.go:521-558`):
```go
cfg := model.SiteConfig{
    EnableRSS: true, // default
}

bools, err := a.AllLatestConfigBools(ctx)
for _, b := range bools {
    if b.ValueID == "enable_rss" {
        cfg.EnableRSS = b.Value
    }
}
```

### Frontmatter поля

| Поле | Тип | Default | Использование |
|------|-----|---------|---------------|
| `rss_title` | `string` | `note.Title` | Заголовок RSS канала |
| `rss_description` | `string` | `note.Description` | Описание RSS канала |

Парсятся через `goldmark-meta` в `mdloader`, хранятся в `NoteView.RawMeta`.

### Формат RSS 2.0

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Feed Title</title>
    <link>https://example.com/page</link>
    <description>Feed description</description>
    <item>
      <title>Link text</title>
      <link>https://example.com/target</link>
      <description>Target description</description>
      <pubDate>Mon, 02 Jan 2006 15:04:05 -0700</pubDate>
      <guid>https://example.com/target</guid>
    </item>
  </channel>
</rss>
```

## Sitemap.xml

Отдельная задача — генерация `sitemap.xml` из всех опубликованных страниц.
