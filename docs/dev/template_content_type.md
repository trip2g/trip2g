# Content-Type для заметок

**TL;DR:** Frontmatter-поле `content_type` задаёт MIME-тип ответа. Это единственный и декларативный способ.

## Механизм

Заметка объявляет `content_type:` во frontmatter:

```yaml
---
slug: /robots.txt
content_type: text/plain; charset=utf-8
---
User-agent: *
Disallow:

Sitemap: https://yourdomain.com/sitemap.xml
```

Render-path в `rendernotepage/endpoint.go` (`handleContentTypeNote`) проверяет это поле и, если оно задано:

1. Устанавливает `Content-Type` из frontmatter.
2. Если у заметки есть `layout` — рендерит через Jet-шаблон (тело формирует шаблон).
3. Если `layout` не задан — отдаёт тело заметки как есть (frontmatter вырезается через `mdchunk.StripFrontmatter`).
4. Оба варианта обходят page cache (он хранит только `text/html`).

## Примеры

### robots.txt (pure content_type note, без layout)

`robots.md`:

```yaml
---
slug: /robots.txt
content_type: text/plain; charset=utf-8
free: true
search: false
---
User-agent: *
Disallow:

Sitemap: https://trip2g.com/sitemap.xml
```

Результат: `/robots.txt` отдаётся как `text/plain`, тело — текст заметки без frontmatter.

### llms.txt (аналогично)

`llms.md`:

```yaml
---
slug: /llms.txt
content_type: text/plain; charset=utf-8
free: true
search: false
---
# trip2g
...
```

### content_type + layout (для динамического тела)

Если тело нужно формировать через шаблон (например, перечислять заметки), задайте и `content_type`, и `layout`. Content-Type берётся из frontmatter, тело — из Jet-шаблона.

## Почему не `response.SetContentType()` в шаблоне

Рассматривался вариант с переменной `response` в Jet-шаблоне:

```jet
{{ response.SetContentType("text/plain") }}...
```

Отказались: Content-Type — это атрибут ресурса, а не поведение шаблона. Декларировать его в frontmatter чище — один источник правды.

## Что реализовано

| Файл | Роль |
|------|------|
| `rendernotepage/endpoint.go` | `handleContentTypeNote()` — проверяет frontmatter, ставит Content-Type, рендерит через layout или отдаёт raw |
| `rendernotepage/pagecache.go` | Пропускает кэширование для не-`text/html` ответов |
| `docs/robots.md` | Заметка `/robots.txt` для trip2g.com (hardcoded absolute Sitemap) |
| `onboarding-vault/robots.md` | Заметка `/robots.txt` по умолчанию для новых сайтов (relative `/sitemap.xml`) |
| `docs/llms.md` | Заметка `/llms.txt` |

Escape-хелперы `json_escape`/`xml_escape` пока не добавлены (не требовались); добавить при появлении JSON/XML-лейаутов.
