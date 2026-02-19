# Template Processors

## Концепция

Шаблон может объявить один или несколько HTTP-процессоров. После рендеринга шаблона вывод последовательно отправляется на каждый процессор, и финальный результат сохраняется в кеш.

```jet
{{ response.SetContentType("application/pdf") }}
{{ apply_processor "http://localhost:3500/html-to-pdf" }}

<html>
  <body>{{ note.HTMLString() | unsafe }}</body>
</html>
```

## Зачем

Процессор — это любой HTTP-сервис, который принимает на входе текст (HTML, markdown, JSON) и возвращает другой текст или бинарные данные (PDF, изображение). Примеры:

- HTML → PDF (для экспорта или печати)
- Markdown → изображение (превью поста)
- JSON → построение OG-image через headless Chrome
- Шаблон рендерит данные → внешний AI-сервис генерирует текст

## Как работает

### Только через кеш

Процессоры вызываются **только при прогреве кеша**, не при каждом запросе. Пользователь всегда читает из кеша. Это снимает проблему round-trip latency.

```
Заметка сохранена/опубликована
    → Кеш инвалидируется
    → Шаблон рендерится в bytes.Buffer
    → apply_processor отправляет буфер на HTTP-процессор (POST)
    → Ответ процессора → кеш
    → Пользователи читают кеш
```

### Синтаксис в шаблоне

`{{ apply_processor "url" }}` объявляет процессор — не вызывает HTTP сразу. Вызов происходит после полного рендеринга шаблона.

Порядок применения = порядок объявления:

```jet
{{ apply_processor "http://localhost/step1" }}
{{ apply_processor "http://localhost/step2" }}

...тело шаблона...
```

Шаг 2 получает на вход вывод шага 1.

### HTTP-протокол

```
POST http://localhost:3500/converter
Content-Type: text/html; charset=utf-8   (Content-Type шаблона)

<html>...</html>
```

```
200 OK
Content-Type: application/pdf

%PDF-1.4...
```

Процессор может изменить Content-Type ответа — система обновит его в кеше.

### Безопасность

URL процессора задаётся в шаблоне — потенциально SSRF-вектор. Митигации:

- Allowlist хостов в конфиге (например только `localhost` и `127.0.0.1`)
- Таймаут на запрос
- Процессоры — только для шаблонов из репозитория (не user-generated контент)

## Реализация

### Шаг 1: Буферизованный рендер

Сейчас шаблон пишет напрямую в `*fasthttp.RequestCtx`. Нужно рендерить в `bytes.Buffer` (как уже делает `renderlayoutpreview`):

```go
var buf bytes.Buffer
layout.View.Execute(&buf, vars, resp)
```

### Шаг 2: Сбор процессоров из шаблона

`apply_processor` — Jet-функция через `ResponseWriter`, накапливает URL в slice:

```go
type ResponseWriter struct {
    Ctx        *fasthttp.RequestCtx
    processors []string
}

func (rw *ResponseWriter) ApplyProcessor(url string) string {
    rw.processors = append(rw.processors, url)
    return ""
}
```

### Шаг 3: Применение процессоров

```go
body := buf.Bytes()
ct := string(ctx.Response.Header.ContentType())

for _, procURL := range rw.processors {
    result, newCT, err := callProcessor(procURL, body, ct)
    if err != nil {
        // log error, skip or abort
        break
    }
    body = result
    if newCT != "" {
        ct = newCT
    }
}

ctx.SetContentType(ct)
ctx.SetBody(body)
```

### Шаг 4: Кеш

При наличии кеша: финальный `body` сохраняется в кеш, процессоры вызываются только при инвалидации.

## Зависимости

Эта фича зависит от:
- [template_content_type.md](template_content_type.md) — `ResponseWriter` и буферизованный рендер
- Кеш рендеринга (будущая фича)

## Статус

- [ ] Не реализовано (зависит от template_content_type и кеша)
