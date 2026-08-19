# App Patterns

Key architectural patterns used throughout the codebase. Read this before adding new features.

## Env Interface (Dependency Declaration)

Each use case declares a minimal `Env` interface with only the methods it needs. The `app` struct in `cmd/server/main.go` implements all of them.

```go
// internal/case/requestemailsignin/resolve.go
type Env interface {
    UserByEmail(ctx context.Context, email string) (db.User, error)
    CreateSignInCode(ctx context.Context, userID int64) (string, error)
    TurnstileSiteKey() string
    VerifyCaptcha(ctx context.Context, token, remoteIP string) error
}
```

Compile-time checks in `cmd/server/cronjobs.go` verify `app` satisfies each interface:
```go
var _ requestemailsignin.Env = app
```

**Rule:** Never import the full app — declare only what you use.

## External Service Client (gitapi pattern)

External services follow: `Config` struct + `New()` constructor + `Client` with methods. The client is embedded in the `app` struct.

Reference: `internal/gitapi/api.go`, `internal/turnstile/verify.go`

```go
// internal/turnstile/verify.go
type Config struct {
    SiteKey   string
    SecretKey string
}

func New(config Config) *Client {
    return &Client{config: config}
}

type Client struct {
    config    Config
    verifyURL string // override for testing
}

func (c *Client) VerifyCaptcha(ctx context.Context, token, remoteIP string) error { ... }
```

Config goes in `internal/appconfig/config.go` as a named field:
```go
Turnstile turnstile.Config
GitAPI    gitapi.Config
```

Client is embedded in `app`:
```go
type app struct {
    *turnstile.Client
    gitAPI *gitapi.API
}
```

Use cases call methods via Env: `env.VerifyCaptcha(ctx, token, ip)`.

**Rules for these packages** (see `internal/chartdata` for a full example):
- Declare a minimal **`Env interface`** the package needs; `app` implements it. Add a compile-time check: `var _ chartdata.Env = (*app)(nil)`.
- Name the type after the **domain**, not `Service`: `chartdata.ChartData`, not `chartdata.Service`.
- **Embed anonymously** in `app` (`*chartdata.ChartData`) so the type's methods promote onto `app`. The renderer/job then take `app` directly — **no proxy methods** like `func (a *app) SaveChartData(...) { return a.x.SaveChartData(...) }`.
- A promoted method must not collide with the embedded field's name. With type `ChartData`, the provider method is `ChartRows` (not `ChartData`, which is the field).
- In the `Env` impl, **reuse existing `app` methods — don't duplicate.** e.g. a reload helper calls `PrepareLatestNotes`/`PrepareLiveNotes` by mode, it does not re-implement the loader.

## IO Belongs on the App Layer

Stateful resources (counters, caches, connections) live on the `app` struct, not as package globals. Use cases access them through Env methods.

```go
// cmd/server/main.go
type app struct {
    signinCounter *requestemailsignin.SignInCounter
}

func (a *app) IncrementAndCheckSigninCounter() bool {
    threshold := a.captchaSigninThreshold()
    return a.signinCounter.IncrementAndCheck(threshold)
}
```

**Rule:** No package-level `var counter = ...`. Put it on `app`, expose via Env.

## Error Handling

Two error channels:
- **Business errors** → return `ErrorPayload, nil` (user sees the message)
- **System errors** → return `nil, error` (user sees "Internal Error")

```go
// Business error (validation, rate limit, captcha)
if count > 3 {
    return &model.ErrorPayload{Message: "too_many_sign_in_codes"}, nil
}

// System error (infrastructure failure)
user, err := env.UserByEmail(ctx, input.Email)
if err != nil {
    return nil, fmt.Errorf("failed to get user by email: %w", err)
}
```

Use `model.NewOzzoError()` and `model.NewFieldError()` for validation errors.

### HTTP endpoints a person opens in a browser

An endpoint whose caller is a human (a sign-in link, a confirmation URL) returns
`appreq.SystemMessageError` instead of writing the failure into the body. The
router (`internal/router/router.go`) catches it via `errors.As`, logs the cause,
and answers with a plain standalone page — `defaulttemplate.WriteSystemMessage`.

```go
return nil, &appreq.SystemMessageError{
    Code: http.StatusUnauthorized,
    Msg:  "hat_expired", // <msg>_title + <msg>_text in langs/*.toml
    Err:  err,           // cause: logged, never shown
}
```

The page is deliberately self-contained (own `<html>`, inline CSS, no scripts,
no note chrome), because the requests that need it are the ones that never got
far enough to have any of that. Copy lives in
`internal/defaulttemplate/langs/{en,ru}.toml`; an unknown `Msg` falls back to
`system_error`. Template: `internal/defaulttemplate/system.html`.

## Background Jobs (Cronjobs)

Jobs implement `cronjobs.Job` interface. Each has `job.go` + `resolve.go`.

Reference: `internal/case/cronjob/cleanupwebhookdeliveries/`

```go
// job.go
type Job struct {
    env Env
}
func New(env Env) *Job                    { return &Job{env: env} } // typed env captured at registration
func (j *Job) Name() string              { return "cleanup_webhook_deliveries" }
func (j *Job) Schedule() string           { return "0 0 3 * * *" } // 6-field cron (with seconds)
func (j *Job) ExecuteAfterStart() bool    { return false }
func (j *Job) Execute(ctx context.Context) (any, error) {
    return Resolve(ctx, j.env)
}

// resolve.go
type Env interface {
    CleanupOldChangeWebhookDeliveries(ctx context.Context) error
    Logger() logger.Logger
}

func Resolve(ctx context.Context, env Env) (*Result, error) { ... }
```

Register in `cmd/server/cronjobs.go`:
```go
func getCronJobConfigs(app *app) []cronjobs.Job {
    _ cleanupwebhookdeliveries.Env = app // compile-time check
    return []cronjobs.Job{
        &cleanupwebhookdeliveries.Job{},
    }
}
```

## Config: Startup vs Runtime

**Startup config** (`internal/appconfig/config.go`): env vars + `.env` file, loaded once.
```go
type Config struct {
    ListenAddr string `env:"LISTEN_ADDR"`
    Turnstile  turnstile.Config
    GitAPI     gitapi.Config
}
```

**Runtime config** (`internal/configregistry/registry.go`): DB-backed, changeable via admin panel.
```go
const ConfigCaptchaSigninThreshold = "captcha_signin_threshold"

Registry[ConfigCaptchaSigninThreshold] = ConfigMeta{
    Description: "Max sign-in requests per hour before captcha",
    Type:        ConfigTypeInt,
    Default:     "5",
    Validate:    validateIntRange(1, 10000),
}
```

**Rule:** Use appconfig for secrets/infra. Use configregistry for admin-tunable behavior.

## GraphQL Resolvers

Resolvers are thin — extract context, delegate to use case.

```go
func (r *mutationResolver) RequestEmailSignInCode(
    ctx context.Context, input model.RequestEmailSignInCodeInput,
) (model.RequestEmailSignInCodeOrErrorPayload, error) {
    var remoteIP string
    if req, err := appreq.FromCtx(ctx); err == nil && req.Req != nil {
        remoteIP = req.Req.RemoteIP().String()
    }
    return requestemailsignin.Resolve(ctx, r.env(ctx), input, remoteIP)
}
```

**Rule:** No business logic in resolvers. Extract what the use case needs (IP, user token), pass `r.env(ctx)`, return the result.

## $mol Frontend Widgets

Widgets use `view.tree` (declarative structure) + `view.ts` (GraphQL + logic).

Reference: `assets/ui/user/space/subscriptions/`

**view.tree** — component structure with bindings:
```
$trip2g_user_space_subscriptions $mol_view
    sub /
        <= Page $mol_page
            title @ \Subscriptions
            body /
                <= List $mol_list
                    rows <= rows /
                        <= Row*0 $mol_row
```

**view.ts** — data fetching and transformation:
```typescript
const request = $trip2g_graphql_request(/* GraphQL */ `
    query UserSubscriptions {
        viewer { user { subgraphAccesses { id createdAt subgraph { name } } } }
    }
`)

export class $trip2g_user_space_subscriptions extends $.$trip2g_user_space_subscriptions {
    @$mol_mem
    data(reset?: null) {
        const res = request()
        return $trip2g_graphql_make_map(res.viewer.user.subgraphAccesses)
    }

    override rows() {
        return this.data().map(key => this.Row(key))
    }

    override row_name(id: any): string {
        return this.row(id).subgraph.name
    }
}
```

Key conventions:
- `$trip2g_graphql_request()` wraps GraphQL queries
- `$trip2g_graphql_make_map()` creates indexed collections
- `@$mol_mem` caches reactive data
- `@ \Text` in view.tree marks translatable strings
- Locale files: `*.view.tree.locale=ru.json`
