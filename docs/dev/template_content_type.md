# Content-Type для заметок

**TL;DR:** Frontmatter-поле `content_type` задаёт MIME-тип ответа. Это единственный и декларативный способ — никакого `response.SetContentType()` в шаблонах нет.

## Механизм

Заметка объявляет `content_type:` во frontmatter:

```yaml
---
slug: /robots.txt
layout: robots
content_type: text/plain; charset=utf-8
---
```

Render-path в `rendernotepage/endpoint.go` проверяет это поле и, если оно задано:

1. Устанавливает `Content-Type` из frontmatter.
2. Если у заметки есть `layout` — рендерит через Jet-шаблон (тело формирует шаблон, например `robots.html` с `{{ publicURL }}/sitemap.xml`).
3. Если `layout` не задан — отдаёт тело заметки как есть (frontmatter вырезается через `mdchunk.StripFrontmatter`).
4. Оба варианта обходят page cache (он хранит только `text/html`).

## Примеры

### robots.txt (`content_type` + `layout`)

`robots.md`:

```yaml
---
slug: /robots.txt
layout: robots
content_type: text/plain; charset=utf-8
free: true
search: false
---
```

`_layouts/robots.html`:

```
User-agent: *
Disallow:

Sitemap: {{ publicURL }}/sitemap.xml
```

Результат: `/robots.txt` отдаётся как `text/plain`, тело — вывод шаблона (абсолютная Sitemap-ссылка через `publicURL`).

### llms.txt (`content_type`, без layout)

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

Результат: `/llms.txt` отдаётся как `text/plain`, тело — текст заметки без frontmatter.

## Почему не `response.SetContentType()` в шаблоне

Ранее рассматривался вариант с переменной `response` в Jet-шаблоне:

```jet
{{ response.SetContentType("text/plain") }}...
```

Отказались: Content-Type — это атрибут ресурса, а не поведение шаблона. Декларировать его в frontmatter чище — один источник правды, никакого импедансного несоответствия между шаблоном и его MIME-типом.

## Что реализовано

| Файл | Роль |
|------|------|
| `rendernotepage/endpoint.go` | `handleContentTypeNote()` — проверяет frontmatter, ставит Content-Type, рендерит через layout или отдаёт raw |
| `rendernotepage/pagecache.go` | Пропускает кэширование для не-`text/html` ответов |
| `docs/_layouts/robots.html` | Шаблон для robots.txt (использует `publicURL`) |
| `onboarding-vault/_layouts/robots.html` | То же, в онбординг-хранилище |
| `docs/robots.md` | Заметка `/robots.txt` для trip2g.com |
| `onboarding-vault/robots.md` | Заметка `/robots.txt` для новых сайтов по умолчанию |
| `docs/llms.md` | Заметка `/llms.txt` (plain, без layout) |

Escape-хелперы `json_escape`/`xml_escape` пока не добавлены (не требовались); добавить при появлении JSON/XML-лейаутов.
