# Content-Type из шаблона

## Концепция

Jet-шаблон может задавать Content-Type ответа через переменную `response`. Это позволяет отдавать из шаблона не только HTML, но и RSS, JSON, XML, CSV и любой другой текстовый формат.

```jet
{{ response.SetContentType("application/rss+xml") }}
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  ...
</rss>
```

## Зачем

Сейчас Content-Type всегда `text/html` — задаётся хардкодом в `rendernotepage/endpoint.go`. Кастомные layouts уже позволяют писать произвольный вывод (рендер идёт напрямую в `ctx`, HTML-обёртка не применяется), но Content-Type поменять нельзя.

Примеры использования:
- Кастомный RSS: пользователь сам определяет какие заметки попадают в фид и как они форматируются
- JSON API endpoint из заметки
- XML-экспорт
- CSV-дамп через шаблон

**Существующий RSS** (`handleRSSFeed` middleware) остаётся — он автоматически создаёт `.rss.xml` для любой заметки по её ссылкам. Шаблонный подход — дополнение для случаев когда нужен кастомный фид.

## Как будет работать

### Template variable `response`

В `renderLayout()` в переменные шаблона добавляется `response`:

```go
vars["response"] = reflect.ValueOf(&templateviews.ResponseWriter{Ctx: ctx})
```

### Структура ResponseWriter

```go
// internal/templateviews/response_writer.go

type ResponseWriter struct {
    Ctx *fasthttp.RequestCtx
}

func (rw *ResponseWriter) SetContentType(ct string) string {
    rw.Ctx.SetContentType(ct)
    return ""
}
```

Возвращает пустую строку чтобы вызов не выводил ничего в шаблоне.

### Использование в шаблоне

```jet
{{ response.SetContentType("application/json") }}
{
  "title": "{{ note.Title() }}",
  "url": "{{ note.Permalink() }}"
}
```

```jet
{{ response.SetContentType("application/rss+xml") }}
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
  <title>{{ note.Title() }}</title>
  <link>{{ nvs.ResolveURL(note) }}</link>
  {{ range i, item := nvs.ByGlob("blog/*.md").SortBy("CreatedAt").Desc().Limit(20).All() }}
  <item>
    <title>{{ item.Title() }}</title>
    <link>{{ item.Permalink() }}</link>
    <pubDate>{{ item.CreatedAt().Format("Mon, 02 Jan 2006 15:04:05 -0700") }}</pubDate>
  </item>
  {{ end }}
</channel>
</rss>
```

### Escape-хелперы

Для безопасной вставки в JSON/XML зарегистрировать глобальные функции в `layoutloader/loader.go`:

```go
views.AddGlobalFunc("json_escape", func(a jet.Arguments) reflect.Value {
    var s string
    a.ParseInto(&s)
    b, _ := json.Marshal(s)
    return reflect.ValueOf(string(b[1 : len(b)-1])) // без кавычек
})

views.AddGlobalFunc("xml_escape", func(a jet.Arguments) reflect.Value {
    var s string
    a.ParseInto(&s)
    // html.EscapeString работает для XML тоже
    return reflect.ValueOf(html.EscapeString(s))
})
```

Использование:

```jet
<title>{{ note.Title() | xml_escape }}</title>
```

## Что нужно изменить

| Файл | Изменение |
|------|-----------|
| `internal/templateviews/response_writer.go` | Новый файл: `ResponseWriter` struct |
| `internal/case/rendernotepage/endpoint.go` | Передавать `vars["response"]` в `renderLayout()` |
| `internal/layoutloader/loader.go` | Зарегистрировать `json_escape`, `xml_escape` |

Rendering pipeline уже правильный: когда layout используется, вывод идёт напрямую в `ctx` без HTML-обёртки (`processed == true` → `return nil, nil`). Достаточно добавить `response` переменную.

## Статус

- [ ] Не реализовано
