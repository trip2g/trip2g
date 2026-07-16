# Go testing

Read the repository `CLAUDE.md` before changing tests. Match the package's
existing test style and keep tests close to the code they exercise.

## Running tests

Run the smallest useful scope first, then broaden it when the change touches
shared code:

```sh
go test ./internal/case/materializenotefrontmatters
go test ./internal/case/...
go test ./...
go test -race ./path/to/package
```

Use `-run TestName` while iterating on one test. Run `gofmt` on changed Go
files. If an interface has a `go:generate` directive, regenerate its mock
after changing the interface.

## Case tests and generated mocks

Use the use-case `Env` interface as the mock boundary. Declare the generator
next to the interface:

```go
//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env
```

Run it from the repository root:

```sh
go generate ./internal/case/example
```

Configure only the callbacks needed by the test:

```go
env := &EnvMock{
	LoadFunc: func(context.Context, int64) (Item, error) {
		return item, nil
	},
}
```

Do not add hand-written slices to record calls. `moq` already records every
call and exposes typed accessors such as `LoadCalls()`:

```go
calls := env.LoadCalls()
require.Len(t, calls, 1)
require.Equal(t, int64(42), calls[0].ID)
```

This keeps tests aligned with the interface and avoids a second, incomplete
mock implementation.

## Assertions and errors

Use `github.com/stretchr/testify/require` for assertions that make the rest
of the test invalid when they fail. It calls `t.Fatalf`, so subsequent code
can safely use values established by the assertion:

```go
result, err := Resolve(ctx, env, input)
require.NoError(t, err)
require.Len(t, result, 1)
require.Equal(t, "notes/a.md", result[0].Value)
```

Use `assert` only when the test should continue collecting independent
failures. Prefer `require.Error`, `require.ErrorContains`, and
`require.NoError` over manually checking `err` and calling `t.Fatal`.

For errors returned by use cases, wrap failures at the operation boundary so
the failing method is visible while preserving the original error with `%w`:

```go
if err := env.Save(ctx, params); err != nil {
	return fmt.Errorf("failed to save item: %w", err)
}
```

Tests should assert both that an error is returned and that its context is
useful, for example with `require.ErrorContains`.

## Test structure

Prefer table-driven tests for the same behavior with different inputs. Use
subtests with descriptive names:

```go
for _, tt := range tests {
	t.Run(tt.name, func(t *testing.T) {
		// arrange, act, assert
	})
}
```

Keep each test focused on observable behavior: returned values, errors, and
calls made to the `Env`. Avoid testing implementation details that are not
part of the use-case contract.

Use `github.com/kr/pretty` when a structural diff is more useful than a long
sequence of individual assertions:

```go
if diff := pretty.Diff(want, got); diff != "" {
	t.Fatalf("result mismatch (-want +got):\n%s", diff)
}
```

## Database and generated code

Database-facing behavior should be covered at the appropriate level. Use
generated sqlc methods through the package `Env` in unit tests; do not mock
the database internals behind a second abstraction. After editing sqlc input,
run:

```sh
make sqlc
```

After editing GraphQL schema, regenerate gqlgen output and run the affected
GraphQL tests.
