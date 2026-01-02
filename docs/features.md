# Feature Flags System

Feature flags allow enabling/disabling optional functionality with per-feature configuration.

## Configuration

Features are configured via JSON passed through environment variable or command-line argument:

```bash
# Environment variable
export FEATURES='{"vector_search":{"enabled":true,"model":"text-embedding-3-small"}}'

# Command-line argument
./server --features='{"vector_search":{"enabled":true}}'
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    internal/features/                    │
├─────────────────────────────────────────────────────────┤
│  features.go          - Features struct, CheckPanic()   │
│  vector_search.go     - VectorSearchConfig              │
│  (future features...) - Additional feature configs      │
└─────────────────────────────────────────────────────────┘
```

## Features Struct

```go
// internal/features/features.go

type Features struct {
    VectorSearch VectorSearchConfig `json:"vector_search"`
}

// CheckPanic validates all features and panics if:
// - Validation fails (invalid config values)
// - Required dependencies missing (env vars, etc.)
func (f *Features) CheckPanic() {
    // 1. ozzo validation for all features
    // 2. Check required env vars for enabled features
}
```

## Adding a New Feature

1. **Create config struct** in `internal/features/`:

```go
// internal/features/my_feature.go

type MyFeatureConfig struct {
    Enabled bool   `json:"enabled"`
    Option1 string `json:"option1"`
    Option2 int    `json:"option2"`
}

func (c MyFeatureConfig) Validate() error {
    return ozzo.ValidateStruct(&c,
        ozzo.Field(&c.Option1, ozzo.When(c.Enabled, ozzo.Required)),
        ozzo.Field(&c.Option2, ozzo.When(c.Enabled, ozzo.Min(1))),
    )
}
```

2. **Add to Features struct**:

```go
type Features struct {
    VectorSearch VectorSearchConfig `json:"vector_search"`
    MyFeature    MyFeatureConfig    `json:"my_feature"`  // add here
}
```

3. **Add validation in CheckPanic()**:

```go
func (f *Features) CheckPanic() {
    err := ozzo.ValidateStruct(f,
        ozzo.Field(&f.VectorSearch),
        ozzo.Field(&f.MyFeature),  // add here
    )
    if err != nil {
        panic(fmt.Sprintf("features validation failed: %v", err))
    }

    // Check required env vars
    if f.MyFeature.Enabled {
        if os.Getenv("MY_FEATURE_API_KEY") == "" {
            panic("MY_FEATURE_API_KEY required when my_feature.enabled=true")
        }
    }
}
```

4. **Use in application**:

```go
// cmd/server/main.go
cfg.Features.CheckPanic()

// anywhere via env
if env.Features().MyFeature.Enabled {
    // feature-specific logic
}
```

## Usage in Code

Access features through the app's Env interface:

```go
// In use case
type Env interface {
    Features() features.Features
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
    if env.Features().VectorSearch.Enabled {
        // perform vector search
    }
    // fallback to regular search
}
```

## Default Values

When `FEATURES` env var is not set or empty, all features are disabled by default:

```go
func DefaultFeatures() Features {
    return Features{
        VectorSearch: VectorSearchConfig{
            Enabled: false,
            Model:   "text-embedding-3-small",
        },
    }
}
```

## Validation Rules

1. **Struct validation** - ozzo-validation for field constraints
2. **Dependency validation** - CheckPanic() verifies required env vars
3. **Startup enforcement** - Server panics if validation fails (fail-fast)

## Environment Variables per Feature

| Feature | Required Env Vars |
|---------|-------------------|
| `vector_search` | `OPENAI_API_KEY` |
