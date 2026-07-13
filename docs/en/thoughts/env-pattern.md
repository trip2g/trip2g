---
title: "The Env Pattern: One IO Spine, Portable Business Logic"
free: true
lang_redirect: "[[ru/thoughts/env-pattern]]"
---

I have been using this architectural approach in Go for 7+ years. It has no established name; it is a natural evolution of idiomatic Go applied to a real monolith.

The first version of this article described the pattern as I remembered it. In the summer of 2026 I ran an audit over the trip2g codebase and found that the code had drifted from the text in a few places. Some of that drift we fixed in five refactoring PRs; some I kept on purpose. This version describes what actually exists, scars included.

## The idea

Every use case lives in its own package under `internal/case/`. Each one declares a minimal `Env` interface: only the methods it actually needs.

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

In `cmd/server/` there is one central `app` struct that holds everything: database connections, storage clients, Telegram sessions, caches. It implicitly satisfies every `Env` interface at once; Go duck typing does that automatically.

```go
// A GraphQL resolver is a thin adapter
func (r *queryResolver) HideNotes(ctx context.Context, input model.HideNotesInput) (model.HideNotesOrErrorPayload, error) {
    return hidenotes.Resolve(ctx, r.env(ctx), input)
}
```

`app` passes itself as `env`. The use case sees only its narrow slice.

## When one case calls another

The first version of this article said: "a use case never imports another use case directly." The audit showed the code had quietly stopped keeping that promise. Not by design, but by accumulated entropy. In five places, cases reached for a foreign contract through a runtime cast of the env carried in the request context:

```go
// how it was: a runtime cast to someone else's Env in the middle of the logic
webhookEnv, ok := req.Env.(handlenotewebhooks.Env)
if !ok {
    env.Logger().Error("failed to cast env")
    return // webhooks silently skipped
}
```

The compiler cannot catch that. Forget a method and you learn about it from the logs, if you are lucky.

Cross-case dependencies are now expressed in two ways, and the compiler checks both.

The first is an opaque port: the case declares a method in its own `Env`, and `app` wraps the other case's `Resolve`:

```go
// internal/case/hidenotes: Env declares the port
type Env interface {
    // ...
    HandleNoteWebhooks(ctx context.Context, changes []handlenotewebhooks.NoteChange, depth int) error
}

// cmd/server: app implements it in one line
func (a *app) HandleNoteWebhooks(ctx context.Context, changes []handlenotewebhooks.NoteChange, depth int) error {
    return handlenotewebhooks.Resolve(ctx, a, changes, depth)
}
```

The second is contract embedding, for when a case needs a whole foreign contract:

```go
// internal/case/rendernotepage: page rendering needs the entire layout contract
type Env interface {
    renderlayout.Env // Go 1.14+: duplicate methods in embedded interfaces are allowed
    // ... its own methods
}
```

A port is for orchestration ("do X, don't ask me how"), embedding is for shared capabilities ("I need everything layout can do"). There is no third way anymore.

## The compiler is your integration test

If `app` lacks a method that some `Env` requires, the project does not build. All wiring is checked statically. No runtime DI container, no reflection. The compiler is the container.

One nuance I only understood after the audit: the proof must be a side effect of the wiring, not a separate discipline. We had a file with two dozen lines of `var _ pkg.Env = app`, a manual ritual that drifts from reality over time. We deleted it and replaced it with constructors:

```go
// how it was: the job receives env at runtime and casts
func (j *Job) Execute(ctx context.Context, env any) (any, error) {
    return Resolve(ctx, env.(Env)) //nolint // "guaranteed"
}

// how it is: the constructor captures a typed env
func New(env Env) *Job { return &Job{env: env} }
func (j *Job) Execute(ctx context.Context) (any, error) {
    return Resolve(ctx, j.env)
}
```

The `pkg.New(app)` call at the registration site is the compile-time proof. The job type itself is unexported, so there is no way to construct one around `New`: the proof has nowhere to go stale.

The same principle killed a fake generic. Background job registration used to look like `Register[T, P]`, where `T` was the Env type. It looks type safe; inside there was `env.(T)` with a panic. A runtime check dressed up as a type parameter. We kept the one honest parameter, `P` (it does real work: unmarshaling the JSON payload into the right type), and the case captures env in a closure:

```go
func New(env UpdateTelegramMessageEnv) *UpdateTelegramMessageJob {
    return &UpdateTelegramMessageJob{
        enqueue: jobs.Register(env, QueueID, JobID, Priority,
            func(ctx context.Context, params model.TelegramUpdatePostParams) error {
                return Resolve(ctx, env, params) // env is checked by the compiler
            }),
    }
}
```

The rule is simple: a generic earns its place when the alternative is copying code (JSON unmarshaling in 17 jobs). A generic does not earn its place when it performs type safety that does not exist.

## Transactions: why Begin() does not return Env

The classic question about this pattern: how do you do transactions? The naive option, a `Begin() Env` method on the interface, does not compile:

```go
type Env interface {
    Begin() Env
}

func (e *RealEnv) Begin() *RealEnv { ... }
// cannot use e (type *RealEnv) as type Env:
//   wrong type for Begin method: have Begin() *RealEnv, want Begin() Env
```

Go has no covariant return types: a method returning a concrete type does not satisfy an interface that expects the interface. I first hit this myself years ago.

The answer the code arrived at on its own: transactions belong to the `app` layer, not to the cases. `WithTransaction` passes the closure a concrete `*app`, a copy with transaction-bound queries:

```go
func (a *app) WithTransaction(ctx context.Context, fn func(context.Context, *app) (bool, error)) error {
    tx, _ := a.writeConn.BeginTx(ctx, nil)
    newEnv := *a               // a copy of app
    newEnv.queries = txQueries // with transactional queries
    commit, err := fn(txCtx, &newEnv)
    // commit / rollback
}
```

The cases know nothing about transactions. The covariance problem is not solved; it is sidestepped. Inside `app` you do not need the interface, you can work with the concrete type.

## Services plug in the same way

External service clients (Telegram, MinIO, Patreon, git API) follow the same pattern. Each declares its own `Env`, takes it at construction, and embeds into `app`. Same mechanism, same guarantees.

## Testing is easy

Every `Env` is small. `moq` generates a mock in one line:

```go
//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env
```

You mock only the 3-4 methods the use case actually calls. No God mock of the whole application. Tests travel with the use case package.

## The edge of the system: one fat interface is the price, and it is honest

If every case declares a narrow Env, something has to gather them all. For us that is `graph.Env`, the GraphQL layer's interface. It has hundreds of methods, and I used to mock interfaces like that myself: "a fat interface with 50 methods that everyone uses partially."

The difference is in how it is built. It is not a handwritten god interface; it is a sum of contracts:

```go
// internal/graph/resolver.go
type Env interface {
    hidenotes.Env
    pushnotes.Env
    sitesearch.Env
    // ... ~120 case Envs, one line per case
    // + graph's own methods: the Selects that serve queries
}
```

The pattern's price at the edge of the system is one line per use case. In return, the interface becomes a table of contents: here is everything GraphQL serves. It cannot drift away from the cases, because it consists of them. A fat interface at the boundary is not a retreat from Interface Segregation; it is its aggregate. Segregated contracts have to meet somewhere, and one explicit place beats being smeared across the codebase.

The second consequence of this edge: the app-graph import cycle breaks without tricks. The graph package cannot import `cmd/server`, so it declares the contract on its side and `app` fulfills it. The same consumer-defined interface, just a big one.

## Scars

The 2026 audit found the following across 61 cases.

29 runtime casts of `req.Env.(...)`. Of those, 24 turned out to be legitimate: an HTTP endpoint takes env from the request context and casts it to its own Env at the entry point. That is a loud fail-fast, and it is backed by the compiler from the other side: `router.New(app)` requires `router.Env`, which transitively embeds every endpoint's Env. That left 5 real holes: casts to foreign contracts in the middle of the logic, where a miss meant silently skipped webhooks. Those we cleaned out with ports and embedding (see above).

Business logic in resolvers. Five GraphQL resolvers had grown to 34-73 lines each, including access control enforcement: token read-scope checks living in the transport layer, where case tests cannot reach them. We moved that into cases; the resolvers went back to 3-13 lines.

Retry logic written in four copies, all four with the same bugs: a timeout derived from `context.Background()` (ignores shutdown), a recursive retry with a fresh budget per attempt and no limit (a persistent rate limit from Telegram means unbounded recursion), and `time.Sleep` instead of cancellable waiting. The split into `Resolve` and `Resolve1` existed to get around a lint limit on function length, so the retry wrapper lived apart from the logic it wrapped for years. The pattern is not to blame here, but it did not save us either: copied code copies bugs.

There is one conclusion from the scars, and it is not "the pattern does not work." All three families of problems grew from the same root: places where the check was taken away from the compiler and handed to the runtime. Where the wiring stayed static, nothing rotted in seven years.

## Places we did not press all the way

What stayed as is, deliberately.

Entry casts at the boundary. Env travels into HTTP handlers through the request context (`req.Env`), and every endpoint has a cast at its entrance. Removing it entirely would mean giving up context-carried env, and we need that: the same mechanism swaps env inside transactions. The cast stays, backed by the router's transitive proof.

The subscription pump in a resolver. The GraphQL subscription `NoteChanges` is 48 lines of goroutine and channel plumbing right in the resolver. That is transport machinery; it has no business in a case. Auth and ACL were extracted from it and covered with tests; the pump stayed.

Two cron jobs without their own logic package. `simplebackup` and `refreshtelegramchatusernames` are one-line delegations. Giving them a `resolve.go` + Env + mocks would be ceremony for ceremony's sake.

An implicit contract between cases. `listnotepaths` does not filter search results by scope in its search branch, because `sitesearch` already filters. That is true, and it is verified, but the contract is not recorded in any type. If `sitesearch` ever stops filtering, the compiler will stay silent.

A shared retry budget. The new retry loop splits one timeout across all attempts; the old one issued a fresh budget per attempt. A pathologically large `retry_after` can eat the whole budget. Acceptable: queue redelivery is the backstop.

## Which patterns this combines

This is not an invention from scratch; it is a combination of known ideas applied naturally in Go:

- **Interface Segregation** (SOLID-I): every case sees only its slice of the world
- **Dependency Inversion** (SOLID-D): cases depend on abstractions, not on `*app`
- **Hexagonal Architecture**: every `Env` is a port, `app` is the adapter
- **Implicit DI**: no framework, no reflection; Go interfaces are the container

The `app` struct is the IO spine. Everything that touches the outside world (database, network, file system, third-party APIs) lives there. Business logic floats above it, attached only through narrow interface contracts.

## Comparison with similar approaches

### benbjohnson/wtf: provider-defined interfaces

[wtf](https://github.com/benbjohnson/wtf) is the canonical Go example by Ben Johnson. The pattern is similar but inverted:

```go
// The ROOT package defines the interface: provider-defined
type DialService interface {
    FindDialByID(ctx context.Context, id int) (*Dial, error)
    FindDials(ctx context.Context, filter DialFilter) ([]*Dial, int, error)
    CreateDial(ctx context.Context, dial *Dial) error
    UpdateDial(ctx context.Context, id int, upd DialUpdate) (*Dial, error)
    DeleteDial(ctx context.Context, id int) error
}

// sqlite/ implements explicitly
var _ wtf.DialService = (*DialService)(nil)
```

The interface is declared by the **provider** (the root package), not the consumer. `DialService` is big: all operations on an entity in one interface. Mocks are handwritten. There is no central `app` passing itself as env.

### swaggest/usecase: a framework with reflection

[swaggest/usecase](https://github.com/swaggest/usecase) uses one universal contract through `interface{}`:

```go
u := usecase.NewIOI(new(myInput), new(myOutput),
    func(ctx context.Context, input, output interface{}) error {
        in := input.(*myInput)   // runtime type assertion
        out := output.(*myOutput)
        out.Value1 = in.Param1 * 2
        return nil
    })
```

No `Env` interfaces; dependencies are captured in closures. Input/output are `interface{}` with reflection. It is a framework for generating API documentation, not an architectural pattern for managing dependencies.

### Comparison

| | wtf | swaggest | trip2g |
|---|---|---|---|
| Where the interface lives | root package (provider) | none | use case (consumer) |
| Interface size | whole service (big) | none | only the needed methods |
| Dependencies | struct fields | closures | `Env` interface |
| Mocks | handwritten | not needed | codegen (`moq`) |
| Entry point | method on struct | `Interact(ctx, in, out)` | `Resolve(ctx, env, input)` |
| `app` as hub | no analog | no analog | `app` passes itself as env |
| Type safety | explicit check | runtime (reflection) | duck typing at call sites + constructors as proofs |

The most distinctive part of the trip2g approach: `app` passes itself, as in `hidenotes.Resolve(ctx, a, input)`. One object is the adapter for every port at once, and the compiler verifies it for 60+ use cases.

## Related work

The pattern has no single canonical name, but it is well described by authoritative Go authors:

- [Peter Bourgon: Go for Industrial Programming](https://peter.bourgon.org/go-for-industrial-programming/) (GopherCon EU 2018): the closest to an authoritative description; interfaces as consumer contracts at the call site rather than declarations in the provider's package.
- [Dave Cheney: SOLID Go Design](https://dave.cheney.net/2016/08/20/solid-go-design): Interface Segregation in Go; small interfaces defined by the consumer. A big `app` satisfying dozens of them is the natural consequence.
- [Go Time #102](https://changelog.com/gotime/102): Bourgon, Ben Johnson, and Mat Ryer discuss exactly this; where interfaces live and how a central struct satisfies them.
- [benbjohnson/wtf](https://github.com/benbjohnson/wtf): the canonical example; an SQLite struct satisfies several domain interfaces declared in other packages.

The closest term in the Go community is **"consumer-defined interfaces"**: the use case owns its contract, not the dependency.

## The same approach works in TypeScript

This is no longer a hypothesis. I built a second backend this way, in TypeScript (Bun + GraphQL). TS structural typing works like Go duck typing, and the pattern transferred verbatim:

```typescript
// src/cases/adminAddTrc20Wallet.ts: a real case, one file
const inputSchema = z.object({
  createdBy: z.string(),
  address: z.string().trim().regex(/^T[a-zA-Z0-9]{33}$/, 'Invalid TRC20 wallet address'),
});

type Env = {
  adminAddTrc20Wallet: (args: { createdBy: string; address: string }) => Promise<Wallet | null>;
  sendNotification: (payload: SendMessage) => Promise<void>;
};

export async function adminAddTrc20Wallet(input: Input, env: Env) {
  const { createdBy, address } = inputSchema.parse(input); // validation at the case boundary
  const row = await env.adminAddTrc20Wallet({ createdBy, address });
  if (!row) throw createWalletErr;
  await env.sendNotification({ type: 'TRC20_WALLET_ADDED', walletId: row.id, actorId: createdBy });
  return { walletId: row.id };
}
```

A case is one file: a zod schema, its own `Env`, its logic, a test next to it. A central `context.ts` implements everything, like `app` in Go. Transactions use the same move as `WithTransaction`: `withTx` assembles the closure's environment with transaction-bound queries via spread.

Three places where TS is actually more convenient. Mocks need no generation: structural typing lets you write one as an object literal in five lines, so there is no `moq` equivalent to run. Input validation lives right in the case (the zod schema is part of the contract), not in the transport layer. And you do not have to import types at all: structural matching is enough for the contract, so if the field names and shapes line up, it fits. In Go the match is exact: `Env` method signatures include concrete types from `db` and `model`, and the case drags those imports along with it.

## The last project where I constructed modules

trip2g is the last project where this construction was assembled by hand and collected the bruises described above along the way. In newer Go projects I do not design module structure; I open trip2g as a reference and repeat it. I did the same in the TypeScript project. In that sense the pattern is finished: I know what it gives, I know the price (the fat aggregate at the edge, the casts at the boundary), and I know where it applies.

If you want to try it, start with one case: a package, an `Env` with three methods, a `Resolve`, a test with a mock. The rest (the `app`, the ports, the edge) will grow as the cases multiply. That is exactly how it happened for me.
