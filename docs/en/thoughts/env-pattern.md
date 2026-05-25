---
title: "The Env Pattern: One IO Spine, Portable Business Logic"
free: true
lang_redirect: "[[ru/thoughts/env-pattern]]"
---

This is an architecture pattern I've been using in Go for 7+ years. It doesn't have a single established name — it's a natural evolution of idiomatic Go applied to a real monolith.

## The idea

Every use case lives in its own package under `internal/case/`. Each one declares a minimal `Env` interface — only the methods it actually needs:

```go
// internal/case/hidenotes/resolve.go
type Env interface {
    HideNotePath(ctx context.Context, params db.HideNotePathParams) error
    LatestNoteViews() *model.NoteViews
    PrepareLatestNotes(ctx context.Context, partial bool) (*model.NoteViews, error)
    Logger() logger.Logger
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
    // pure business logic
}
```

There's one central `app` struct in `cmd/server/` that holds everything: database connections, storage clients, Telegram sessions, caches. It implicitly satisfies every `Env` interface across all use cases — Go's duck typing handles this automatically.

```go
// cmd/server/case_methods.go
func (a *app) HideNotes(ctx context.Context, input model.HideNotesInput) (model.HideNotesOrErrorPayload, error) {
    return hidenotes.Resolve(ctx, a, input)
}
```

`app` passes itself as `env`. The use case only sees its narrow slice.

## When a case needs another case

A use case never imports another use case directly. Instead, `app` wraps it as a method:

```go
func (a *app) HandleNoteWebhooks(ctx context.Context, changes []NoteChange) error {
    return handlenotewebhooks.Resolve(ctx, a, changes)
}
```

If `hidenotes` needs to trigger webhooks after a hide, it declares that in its own `Env`:

```go
type Env interface {
    // ...
    HandleNoteWebhooks(ctx context.Context, changes []NoteChange) error
}
```

`app` already has that method. No new wiring needed.

## The compiler is your integration test

If `app` is missing a method that any `Env` requires — the project won't compile. All wiring is verified statically. You don't need a runtime DI container or reflection. The compiler is the container.

## Services embed the same way

External service clients (Telegram, MinIO, Patreon, git API) follow the same pattern. Each declares its own `Env`:

```go
// internal/gitapi/api.go
type Env interface {
    Logger() logger.Logger
    PutPrivateObject(ctx context.Context, ...) error
    GitTokenByValueSha256(ctx context.Context, ...) (db.GitToken, error)
    PushNotes(ctx context.Context, ...) (model.PushNotesOrErrorPayload, error)
}
```

The service struct takes `env Env` at construction time and gets embedded in `app`. Same mechanism, same guarantees.

## Testing is straightforward

Each `Env` is small. `moq` generates a mock for it in one line:

```go
//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env
```

You mock only the 3-4 methods the use case actually calls. No God-mock of the entire application. Tests travel with the use case package.

## Cases are portable

A use case package is self-contained: business logic + its `Env` contract + tests with mocks. Moving it to another project means satisfying the contract. If the new `app` compiles against it — it works. The tests come along for free.

## What patterns does this combine

This isn't a novel invention — it's a combination of well-known ideas applied naturally in Go:

- **Interface Segregation** (SOLID-I) — each case sees only its slice of the world
- **Dependency Inversion** (SOLID-D) — cases depend on abstractions, not on `*app`
- **Hexagonal Architecture** — each `Env` is a port; `app` is the adapter
- **Implicit DI** — no framework, no reflection; Go interfaces are the container

The `app` struct is the IO spine. Everything that touches the outside world — database, network, filesystem, third-party APIs — lives there. Business logic floats above it, attached only through narrow interface contracts.

## How it compares to similar approaches

### benbjohnson/wtf — provider-defined interfaces

[wtf](https://github.com/benbjohnson/wtf) is a canonical Go reference app by Ben Johnson. The pattern looks similar but is inverted:

```go
// ROOT package defines the interface — provider-defined
type DialService interface {
    FindDialByID(ctx context.Context, id int) (*Dial, error)
    FindDials(ctx context.Context, filter DialFilter) ([]*Dial, int, error)
    CreateDial(ctx context.Context, dial *Dial) error
    UpdateDial(ctx context.Context, id int, upd DialUpdate) (*Dial, error)
    DeleteDial(ctx context.Context, id int) error
    // all dial operations in one interface
}

// sqlite/ implements it explicitly
var _ wtf.DialService = (*DialService)(nil) // compile-time assertion

// mock/ — hand-written mocks with function fields
type DialService struct {
    FindDialByIDFn func(ctx context.Context, id int) (*wtf.Dial, error)
    CreateDialFn   func(ctx context.Context, dial *wtf.Dial) error
    // ...
}
```

The interface is declared by the **provider** (root package), not the consumer. `DialService` is large — it contains all operations for the entity. Mocks are hand-written. There's no central app struct passing itself as env.

### swaggest/usecase — framework with reflection

[swaggest/usecase](https://github.com/swaggest/usecase) uses a single universal contract via `interface{}`:

```go
u := usecase.NewIOI(new(myInput), new(myOutput),
    func(ctx context.Context, input, output interface{}) error {
        in := input.(*myInput)   // type assertion at runtime
        out := output.(*myOutput)
        out.Value1 = in.Param1 * 2
        return nil
    })
u.SetTitle("Doubler")
u.SetTags("transformation")
```

No `Env` interfaces at all — dependencies are captured via closures. Input/output are `interface{}` with reflection. It's primarily a framework for API documentation generation, not an architecture pattern for dependency management.

### Comparison

| | wtf | swaggest | trip2g |
|---|---|---|---|
| Interface location | root package (provider) | none | use case (consumer) |
| Interface size | whole service (large) | none | only needed methods |
| Dependencies | struct fields | closures | `Env` interface |
| Mocks | hand-written | not needed | codegen (`moq`) |
| Entry point | method on struct | `Interact(ctx, in, out)` | `Resolve(ctx, env, input)` |
| `app` as hub | no equivalent | no equivalent | `app` passes itself as `env` |
| Type safety | explicit assertion | runtime (reflection) | implicit duck typing |

The most distinctive aspect of the trip2g approach: `app` passes itself — `hidenotes.Resolve(ctx, a, input)`. One object is simultaneously the adapter for all ports, and the compiler verifies this implicitly across 50+ use cases.

## Prior art and references

The pattern doesn't have a single canonical name, but it's well-supported by respected Go authors:

- [Peter Bourgon — Go for Industrial Programming](https://peter.bourgon.org/go-for-industrial-programming/) (GopherCon EU 2018) — the closest authoritative description. Interfaces as consumer contracts at call sites, not declarations in the provider package.
- [Dave Cheney — SOLID Go Design](https://dave.cheney.net/2016/08/20/solid-go-design) — Interface Segregation in Go: small interfaces defined by the consumer. A large `app` struct satisfying dozens of them is the natural consequence.
- [Go Time #102](https://changelog.com/gotime/102) — Bourgon, Ben Johnson, and Mat Ryer discuss exactly this: where interfaces live and how a central struct satisfies them.
- [benbjohnson/wtf](https://github.com/benbjohnson/wtf) — a canonical example app where a concrete SQLite struct satisfies multiple domain interfaces defined elsewhere.

The community term closest to this is **"consumer-defined interfaces"** — the use case owns its contract, not the dependency.

## The same approach works in TypeScript

TypeScript's structural typing is identical to Go's duck typing — a type satisfies an interface if it has the right shape, no explicit declaration needed. The pattern translates directly:

```typescript
// use case defines its own contract
interface Env {
  hideNotePath(params: HideNotePathParams): Promise<void>
  latestNoteViews(): NoteViews
  logger(): Logger
}

export async function resolve(env: Env, input: Input): Promise<Payload> {
  // pure business logic
}
```

TypeScript is actually more convenient for this in one way: you can define the use case's input/output models in the same package, separate from the shared domain types. Each use case is fully self-contained — its own `Env`, its own `Input`, its own `Payload`.

On the server side (Node.js / Bun / Deno) this pattern works exactly as well as in Go. The current trip2g frontend is plain CRUD and doesn't use it — there's no need when a component just calls an API and renders the result. But for complex backend TypeScript services it's the same story: one central object satisfying many small interfaces, compiler as the integration test.
