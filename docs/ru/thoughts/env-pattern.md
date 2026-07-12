---
title: Env-паттерн: один IO-позвоночник, переносимая бизнес-логика
free: true
---

Этот архитектурный подход я использую в Go уже 7+ лет. Устоявшегося названия у него нет — это естественная эволюция идиоматического Go применительно к реальному монолиту.

## Идея

Каждый use case живёт в своём пакете под `internal/case/`. Каждый объявляет минимальный `Env`-интерфейс — только те методы, которые ему действительно нужны:

```go
// internal/case/hidenotes/resolve.go
type Env interface {
    HideNotePath(ctx context.Context, params db.HideNotePathParams) error
    LatestNoteViews() *model.NoteViews
    PrepareLatestNotes(ctx context.Context, partial bool) (*model.NoteViews, error)
    Logger() logger.Logger
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
    // чистая бизнес-логика
}
```

В `cmd/server/` есть один центральный `app`-struct, который держит всё: подключения к БД, клиенты хранилища, Telegram-сессии, кэши. Он неявно удовлетворяет всем `Env`-интерфейсам сразу — Go duck typing делает это автоматически.

```go
// cmd/server/case_methods.go
func (a *app) HideNotes(ctx context.Context, input model.HideNotesInput) (model.HideNotesOrErrorPayload, error) {
    return hidenotes.Resolve(ctx, a, input)
}
```

`app` передаёт себя как `env`. Use case видит только свой узкий срез.

## Когда кейс вызывает другой кейс

Use case никогда не импортирует другой use case напрямую. Вместо этого `app` оборачивает вызов как метод:

```go
func (a *app) HandleNoteWebhooks(ctx context.Context, changes []NoteChange) error {
    return handlenotewebhooks.Resolve(ctx, a, changes)
}
```

Если `hidenotes` нужно запустить вебхуки после скрытия, он просто добавляет это в свой `Env`:

```go
type Env interface {
    // ...
    HandleNoteWebhooks(ctx context.Context, changes []NoteChange) error
}
```

Метод у `app` уже есть. Никакой дополнительной проводки не нужно.

## Компилятор — это твой интеграционный тест

Если у `app` нет метода, который требует какой-то `Env` — проект не соберётся. Весь wiring проверяется статически. Никакого DI-контейнера в рантайме, никакой рефлексии. Компилятор и есть контейнер.

## Сервисы встраиваются так же

Клиенты внешних сервисов (Telegram, MinIO, Patreon, git API) следуют тому же паттерну. Каждый объявляет свой `Env`:

```go
// internal/gitapi/api.go
type Env interface {
    Logger() logger.Logger
    PutPrivateObject(ctx context.Context, ...) error
    GitTokenByValueSha256(ctx context.Context, ...) (db.GitToken, error)
    PushNotes(ctx context.Context, ...) (model.PushNotesOrErrorPayload, error)
}
```

Struct сервиса принимает `env Env` при создании и встраивается в `app`. Тот же механизм, те же гарантии.

## Тестирование простое

Каждый `Env` маленький. `moq` генерирует мок одной строкой:

```go
//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env
```

Мокаешь только те 3-4 метода, которые use case реально вызывает. Никакого God-мока всего приложения. Тесты путешествуют вместе с пакетом use case.

## Кейсы переносимы

Пакет use case самодостаточен: бизнес-логика + `Env`-контракт + тесты с моками. Перенести в другой проект — значит выполнить контракт. Если новый `app` компилируется с ним — всё работает. Тесты идут в комплекте.

## Какие паттерны это объединяет

Это не изобретение с нуля — это комбинация известных идей, применённых естественно в Go:

- **Interface Segregation** (SOLID-I) — каждый кейс видит только свой срез мира
- **Dependency Inversion** (SOLID-D) — кейсы зависят от абстракций, а не от `*app`
- **Hexagonal Architecture** — каждый `Env` это port, `app` это adapter
- **Implicit DI** — никакого фреймворка, никакой рефлексии; Go-интерфейсы и есть контейнер

`app`-struct — это IO-позвоночник. Всё, что касается внешнего мира — база данных, сеть, файловая система, сторонние API — живёт там. Бизнес-логика парит над ним, прикреплённая только через узкие интерфейсные контракты.

## Сравнение с похожими подходами

### benbjohnson/wtf — интерфейсы определяет провайдер

[wtf](https://github.com/benbjohnson/wtf) — канонический Go-пример от Ben Johnson. Паттерн похож, но инвертирован:

```go
// ROOT-пакет определяет интерфейс — provider-defined
type DialService interface {
    FindDialByID(ctx context.Context, id int) (*Dial, error)
    FindDials(ctx context.Context, filter DialFilter) ([]*Dial, int, error)
    CreateDial(ctx context.Context, dial *Dial) error
    UpdateDial(ctx context.Context, id int, upd DialUpdate) (*Dial, error)
    DeleteDial(ctx context.Context, id int) error
    // все операции с сущностью в одном интерфейсе
}

// sqlite/ реализует явно
var _ wtf.DialService = (*DialService)(nil) // явная проверка компилятором

// mock/ — ручные моки с полями-функциями
type DialService struct {
    FindDialByIDFn func(ctx context.Context, id int) (*wtf.Dial, error)
    CreateDialFn   func(ctx context.Context, dial *wtf.Dial) error
    // ...
}
```

Интерфейс объявляет **провайдер** (root-пакет), не потребитель. `DialService` большой — все операции с сущностью в одном интерфейсе. Моки написаны вручную. Нет центрального `app`, который передаёт себя как env.

### swaggest/usecase — фреймворк с рефлексией

[swaggest/usecase](https://github.com/swaggest/usecase) использует один универсальный контракт через `interface{}`:

```go
u := usecase.NewIOI(new(myInput), new(myOutput),
    func(ctx context.Context, input, output interface{}) error {
        in := input.(*myInput)   // type assertion в рантайме
        out := output.(*myOutput)
        out.Value1 = in.Param1 * 2
        return nil
    })
u.SetTitle("Doubler")
u.SetTags("transformation")
```

Никаких `Env`-интерфейсов — зависимости захватываются через замыкания. Input/output — `interface{}` с рефлексией. Это фреймворк для генерации API-документации, а не архитектурный паттерн управления зависимостями.

### Сравнение

| | wtf | swaggest | trip2g |
|---|---|---|---|
| Где интерфейс | root-пакет (провайдер) | нет | use case (потребитель) |
| Размер интерфейса | весь сервис (большой) | нет | только нужные методы |
| Зависимости | поля struct | замыкания | `Env`-интерфейс |
| Моки | ручные | не нужны | codegen (`moq`) |
| Точка входа | метод на struct | `Interact(ctx, in, out)` | `Resolve(ctx, env, input)` |
| `app` как hub | нет аналога | нет аналога | `app` передаёт себя как env |
| Type safety | явная проверка | рантайм (рефлексия) | implicit duck typing |

Самое уникальное в trip2g-подходе: `app` передаёт себя — `hidenotes.Resolve(ctx, a, input)`. Один объект одновременно — адаптер для всех портов, и компилятор проверяет это неявно для 50+ use cases.

## Похожее в литературе

У паттерна нет единственного канонического названия, но он хорошо описан авторитетными Go-авторами:

- [Peter Bourgon — Go for Industrial Programming](https://peter.bourgon.org/go-for-industrial-programming/) (GopherCon EU 2018) — ближайшее к авторитетному описанию: интерфейсы как consumer contracts на callsite, а не объявления в пакете провайдера.
- [Dave Cheney — SOLID Go Design](https://dave.cheney.net/2016/08/20/solid-go-design) — Interface Segregation в Go: маленькие интерфейсы, определённые потребителем. Большой `app`, удовлетворяющий десяткам таких интерфейсов, — естественное следствие.
- [Go Time #102](https://changelog.com/gotime/102) — Bourgon, Ben Johnson и Mat Ryer обсуждают именно это: где живут интерфейсы и как центральный struct их удовлетворяет.
- [benbjohnson/wtf](https://github.com/benbjohnson/wtf) — канонический пример: SQLite-struct удовлетворяет нескольким domain-интерфейсам, объявленным в других пакетах.

Ближайший термин в Go-сообществе — **"consumer-defined interfaces"**: use case владеет своим контрактом, а не зависимость.

## Тот же подход работает в TypeScript

Структурная типизация TypeScript идентична duck typing в Go — тип удовлетворяет интерфейсу, если у него нужная форма, без явного объявления. Паттерн переносится напрямую:

```typescript
// use case объявляет свой контракт
interface Env {
  hideNotePath(params: HideNotePathParams): Promise<void>
  latestNoteViews(): NoteViews
  logger(): Logger
}

export async function resolve(env: Env, input: Input): Promise<Payload> {
  // чистая бизнес-логика
}
```

В TypeScript даже удобнее в одном месте: можно описывать модели Input и Payload прямо в пакете use case, отдельно от общих domain-типов. Каждый кейс полностью самодостаточен — свой `Env`, свой `Input`, свой `Payload`.

На серверной стороне (Node.js / Bun / Deno) паттерн работает так же хорошо, как в Go. Текущий фронтенд trip2g — простой CRUD, где это не нужно: компонент вызывает API и рендерит результат. Но для сложных бэкенд-сервисов на TypeScript история та же: один центральный объект удовлетворяет множеству маленьких интерфейсов, компилятор — интеграционный тест.
