# GraphQL Subscriptions: gqlgen + fasthttp

## Проблема

gqlgen транспорты для подписок (WebSocket, SSE) построены на `net/http`. fasthttp использует другую модель запросов/ответов — API несовместимы на фундаментальном уровне.

## Варианты решения

### 1. Dual Server + Caddy reverse proxy (рекомендуется)

Два сервера в одном процессе: fasthttp для queries/mutations, net/http для subscriptions. Caddy роутит по типу запроса.

**Плюсы:**
- Производительность fasthttp для 99% трафика (queries/mutations)
- Стандартные gqlgen подписки работают без модификаций
- Один endpoint для клиентов
- Caddy уже используется в проекте

**Минусы:**
- Два сервера в одном процессе
- Нужно шарить ExecutableSchema между ними

**Пример:**

```go
// internal/server/subscriptions.go
func StartSubscriptionServer(schema graphql.ExecutableSchema) {
    srv := handler.New(schema)
    srv.AddTransport(transport.Websocket{
        KeepAlivePingInterval: 10 * time.Second,
    })
    http.Handle("/graphql", srv)
    http.ListenAndServe(":8082", nil)
}

// cmd/server/main.go
func main() {
    go StartFasthttpServer()      // :8081
    go StartSubscriptionServer()  // :8082
    select {}
}
```

**Caddyfile:**

```caddyfile
handle /graphql {
    @websocket {
        header Connection *Upgrade*
        header Upgrade websocket
    }
    reverse_proxy @websocket localhost:8082
    reverse_proxy localhost:8081
}
```

### 2. SSE вместо WebSocket

SSE проще WebSocket и лучше подходит для GraphQL подписок (нужен только server→client поток).

**Плюсы:**
- Проще протокол
- Firewall-friendly (обычный HTTP)
- Можно тестировать через curl
- HTTP/2 даёт ~100 конкурентных подписок

**Минусы:**
- Требует SSE-совместимый клиент (`graphql-sse`)
- GraphQL Playground не поддерживает SSE
- HTTP/1.1 — лимит 6 соединений на браузер
- Всё равно требует `net/http` (та же проблема с fasthttp)

**Конфигурация gqlgen:**

```go
srv.AddTransport(transport.SSE{
    KeepAlivePingInterval: 10 * time.Second,
})
```

**Тест через curl:**

```bash
curl -N --request POST --url http://localhost:8082/graphql \
  --data '{"query":"subscription { currentTime { unixTime } }"}' \
  -H "accept: text/event-stream" -H 'content-type: application/json'
```

### 3. Полный переход на net/http

Заменить fasthttp на стандартный `net/http`.

**Плюсы:**
- Всё работает из коробки
- HTTP/2 поддержка
- Больше экосистема и middleware

**Минусы:**
- Потеря производительности fasthttp (30-70% в синтетических тестах)

**Когда выбирать:** если подписки критичны, а выигрыш fasthttp не оправдывает сложности.

### 4. Не рекомендуется

| Вариант | Почему нет |
|---------|-----------|
| **fastgql (arsmn/fastgql)** | Не поддерживается с 2021, нет документации по подпискам |
| **Кастомный адаптер** | Высокая сложность, баги, поддержка |
| **Два endpoint для клиентов** | Плохой DX, сложная конфигурация клиента |

## WebSocket vs SSE

| Аспект | WebSocket | SSE |
|--------|-----------|-----|
| Сложность | Высокая (upgrade, свой протокол) | Низкая (обычный HTTP) |
| Firewall | Могут блокировать upgrade | Без проблем |
| HTTP/2 | Нет | Да |
| Тестирование | Нужен WS клиент | curl работает |
| Для GraphQL | Overkill (bidirectional не нужен) | Идеально (server→client) |
| Поддержка клиентов | Apollo, urql, все | graphql-sse, растёт |
| gqlgen | Встроенный, зрелый | Встроенный |

Индустрия движется к SSE для GraphQL подписок — WebSocket избыточен когда нужен только server→client поток.

## Рекомендация

**Вариант 1 (Dual Server + Caddy)** с **SSE транспортом**:

- fasthttp на :8081 — queries/mutations (основной трафик)
- net/http на :8082 — subscriptions через SSE
- Caddy роутит по заголовку `Accept: text/event-stream`
- Один `/graphql` endpoint для клиентов

Это даёт: производительность fasthttp + простоту SSE + стандартный gqlgen без хаков.

## Решение: SSE через fasthttpadaptor (реализовано)

Оказалось, что `fasthttpadaptor.NewFastHTTPHandler()` реализует `http.Flusher` — gqlgen SSE транспорт работает напрямую через fasthttp без второго сервера.

Три изменения, каждое необходимо:

### 1. SSE транспорт первым в списке

`internal/graph/handler.go` — порядок важен, SSE и POST оба принимают POST-запросы. Если POST идёт первым — он перехватывает SSE запрос.

```go
srv.AddTransport(transport.SSE{})       // первым!
srv.AddTransport(transport.Options{})
srv.AddTransport(transport.GET{})
srv.AddTransport(transport.POST{})
```

### 2. Обход TimeoutHandler

`cmd/server/main.go` — SSE соединение живёт бесконечно, `fasthttp.TimeoutHandler` убивает через 60 секунд.

```go
timeoutHandler := fasthttp.TimeoutHandler(handler, handlerTimeout, "timeout")

s := &fasthttp.Server{
    Handler: func(ctx *fasthttp.RequestCtx) {
        if strings.Contains(string(ctx.Request.Header.Peek("Accept")), "text/event-stream") {
            handler(ctx) // напрямую, без таймаута
            return
        }
        timeoutHandler(ctx)
    },
}
```

### 3. Обход CompressHandler

`cmd/server/main.go` — `fasthttp.CompressHandler` буферизирует весь ответ. SSE требует streaming.

```go
case strings.Contains(string(ctx.Request.Header.Peek("Accept")), "text/event-stream"):
    graphqlHandler(ctx)           // без сжатия
default:
    compressedGraphqlHandler(ctx) // обычные запросы со сжатием
```

### Проверка

```bash
curl -N --request POST --url http://localhost:8081/graphql \
  --data '{"query":"subscription { currentTime }"}' \
  -H "accept: text/event-stream" -H "content-type: application/json"
```

Результат — events приходят по одному каждую секунду:
```
event: next
data: {"data":{"currentTime":"2026-02-08 06:51:20"}}

event: next
data: {"data":{"currentTime":"2026-02-08 06:51:21"}}
```

## Ссылки

- [gqlgen subscriptions](https://gqlgen.com/recipes/subscriptions/)
- [fasthttp/websocket](https://github.com/fasthttp/websocket) — WebSocket для fasthttp (не совместим с gqlgen)
- [graphql-sse](https://the-guild.dev/blog/graphql-over-sse) — SSE протокол для GraphQL
- [WunderGraph: Why SSE over WebSocket](https://wundergraph.com/blog/deprecate_graphql_subscriptions_over_websockets)
