# How to Write GraphQL Mutations

This document describes the best practices and step-by-step process for implementing GraphQL mutations in this project.

## Project Overview

This is a Go-based web application with the following technology stack:

- **Language**: Go 1.21+
- **Database**: SQLite with WAL mode
- **Query Builder**: [sqlc](https://sqlc.dev/) for type-safe SQL
- **GraphQL**: [gqlgen](https://gqlgen.com/) for GraphQL server
- **HTTP Server**: [fasthttp](https://github.com/valyala/fasthttp) for high performance
- **Validation**: [ozzo-validation](https://github.com/go-ozzo/ozzo-validation) for input validation
- **Authentication**: JWT tokens with custom user token management
- **Database Migrations**: [dbmate](https://github.com/amacneil/dbmate) for schema migrations
- **Task Queue**: [backlite](https://github.com/mikestefanello/backlite) for background jobs
- **File Storage**: MinIO S3-compatible object storage
- **Frontend**: Server-side rendered HTML with TypeScript components
- **CSS**: Tailwind CSS for styling
- **Build Tools**: 
  - `make sqlc` - Generate database code from SQL
  - `make gqlgen` - Generate GraphQL resolvers and types

### Project Structure

```
├── cmd/server/          # Main application entry point
├── internal/
│   ├── case/           # Business logic resolvers (mutations/queries)
│   │   ├── admin/      # Admin-only operations
│   │   └── ...         # Other business cases
│   ├── db/             # Generated database code (sqlc)
│   ├── graph/          # GraphQL schema and resolvers (gqlgen)
│   └── ...             # Other internal packages
├── db/migrations/      # Database schema migrations
├── queries.sql         # SQL queries for sqlc generation
├── assets/             # Frontend assets and UI components
└── docs/              # Documentation
```

## Overview

Mutations in this project follow a structured pattern that ensures consistency, testability, and maintainability. Each mutation is implemented as a resolver function in a dedicated package under `internal/case/`.

## Step-by-Step Process

### 1. Write Database Queries

First, add your SQL queries to `queries.sql`:

```sql
-- name: InsertApiKey :one
insert into api_keys (value, created_by, description)
values (?, ?, ?)
returning *;

-- name: DisableApiKey :exec
update api_keys
  set disabled_by = ?, disabled_at = datetime('now')
 where id = ?;
```

Then regenerate the database code:

```bash
make sqlc
```

### 2. Define GraphQL Schema

Add your mutation types to `internal/graph/schema.graphqls`:

```graphql
input CreateAPIKeyInput {
  description: String!
}

type CreateAPIKeyPayload {
  apiKey: AdminApiKey
  value: String!
}

union CreateAPIKeyOrErrorPayload = CreateAPIKeyPayload | ErrorPayload

extend type AdminMutation {
  createApiKey(input: CreateAPIKeyInput!): CreateAPIKeyOrErrorPayload!
  disableApiKey(input: DeleteAPIKeyInput!): DeleteAPIKeyOrErrorPayload!
}
```

Then regenerate the GraphQL code:

```bash
make gqlgen
```

### 3. Implement the Resolver Package

Create a new package under `internal/case/` (e.g., `internal/case/admin/createapikey/`) with a `resolve.go` file:

```go
package createapikey

import (
	"context"
	"trip2g/internal/appreq"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

// Env interface describes all IO dependencies
// This allows writing tests with mocked implementations
type Env interface {
	GenerateApiKey() string
	InsertApiKey(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error)
}

func Resolve(ctx context.Context, env Env, input model.CreateAPIKeyInput) (model.CreateAPIKeyOrErrorPayload, error) {
	// Extract user token from context
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}

	// Check authorization
	if !token.IsAdmin() {
		return &model.ErrorPayload{Message: "Unauthorized"}, nil
	}

	// Business logic
	apiKey := env.GenerateApiKey()

	params := db.InsertApiKeyParams{
		Value:       apiKey,
		CreatedBy:   int64(token.ID),
		Description: input.Description,
	}

	createdKey, err := env.InsertApiKey(ctx, params)
	if err != nil {
		return nil, err
	}

	// Return success payload
	response := model.CreateAPIKeyPayload{
		APIKey: &createdKey,
		Value:  apiKey,
	}

	return &response, nil
}
```

### 4. Update GraphQL Resolvers

Update `internal/graph/schema.resolvers.go`:

1. Add the import for your new package:
```go
import (
	// ... other imports
	"trip2g/internal/case/admin/createapikey"
)
```

2. Update the mutation resolver:
```go
func (r *adminMutationResolver) CreateAPIKey(ctx context.Context, obj *appmodel.AdminMutation, input model.CreateAPIKeyInput) (model.CreateAPIKeyOrErrorPayload, error) {
	return createapikey.Resolve(ctx, r.env(ctx), input)
}
```

### 5. Add Env Interface to Main Resolver

Update `internal/graph/resolver.go` to include your new Env interface:

1. Add the import:
```go
import (
	// ... other imports
	"trip2g/internal/case/admin/createapikey"
)
```

2. Add to the main Env interface:
```go
type Env interface {
	// ... other interfaces
	createapikey.Env
}
```

## Best Practices

### Authorization
- Always extract and validate user tokens from context using `appreq.FromCtx(ctx)`
- Return `ErrorPayload` for authorization failures rather than throwing errors
- Use `token.IsAdmin()` for admin-only operations

### Error Handling
- Return `ErrorPayload` for business logic errors (validation, authorization)
- Return actual Go errors for unexpected system errors (database failures, etc.)
- Use descriptive error messages in `ErrorPayload`

### Transaction Management
- All mutations are automatically wrapped in database transactions
- Transactions are committed on success and rolled back on errors
- You don't need to manage transactions manually

### Env Interface Pattern
- Define an `Env` interface in each resolver package
- Include only the dependencies that specific resolver needs
- This enables easy testing with mocked implementations
- Keeps resolvers focused and dependencies explicit

### Code Organization
- One package per major mutation or related group of mutations
- Keep resolver logic focused on orchestration, not business rules
- Extract complex business logic into separate functions or packages

### Input Validation
- Use GraphQL schema validation for basic type checking
- Use `github.com/go-ozzo/ozzo-validation` for input validation
- Return `ErrorPayload` with clear validation messages
- Follow the normalize → validate → process pattern

### Naming Conventions
- Package names should be descriptive (e.g., `createapikey`, `updateuser`)
- Resolver function is always named `Resolve`
- Input types follow pattern: `{Operation}{Entity}Input`
- Payload types follow pattern: `{Operation}{Entity}Payload`
- Union types follow pattern: `{Operation}{Entity}OrErrorPayload`

## Testing

The Env interface pattern makes testing straightforward:

```go
type mockEnv struct{}

func (m *mockEnv) GenerateApiKey() string {
	return "test-api-key"
}

func (m *mockEnv) InsertApiKey(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error) {
	return db.ApiKey{ID: 1, Value: params.Value}, nil
}

func TestResolve(t *testing.T) {
	env := &mockEnv{}
	input := model.CreateAPIKeyInput{Description: "test"}
	
	result, err := createapikey.Resolve(ctx, env, input)
	// ... assertions
}
```

## Input Validation with Ozzo Validation

This project uses `github.com/go-ozzo/ozzo-validation` for robust input validation. The validation follows a three-step pattern: normalize, validate, process.

### Validation Pattern

```go
import (
	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	gmodel "trip2g/internal/graph/model"
)

func normalizeRequest(r *gmodel.SignInByEmailInput) {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
}

func validateRequest(r *gmodel.SignInByEmailInput) *gmodel.ErrorPayload {
	return gmodel.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Email, ozzo.Required, is.Email),
		ozzo.Field(&r.Code, ozzo.Required, ozzo.Length(6, 6)),
	))
}

func Resolve(ctx context.Context, env Env, req gmodel.SignInByEmailInput) (gmodel.SignInOrErrorPayload, error) {
	// Step 1: Normalize input
	normalizeRequest(&req)

	// Step 2: Validate input
	errorPayload := validateRequest(&req)
	if errorPayload != nil {
		return errorPayload, nil
	}

	// Step 3: Process business logic
	// ... rest of the resolver logic
}
```

### Helper Functions

The project provides two helper functions for creating error responses:

1. **`gmodel.NewOzzoError(err error)`** - Converts ozzo validation errors to `ErrorPayload`
2. **`gmodel.NewFieldError(field, message string)`** - Creates single field error

### Common Validation Rules

```go
// Required field
ozzo.Field(&input.Name, ozzo.Required)

// Email validation
ozzo.Field(&input.Email, ozzo.Required, is.Email)

// Length validation
ozzo.Field(&input.Code, ozzo.Required, ozzo.Length(6, 6))
ozzo.Field(&input.Description, ozzo.Length(1, 500))

// Custom validation
ozzo.Field(&input.Status, ozzo.Required, ozzo.In("active", "inactive"))

// Conditional validation
ozzo.Field(&input.Password, ozzo.When(input.RequirePassword, ozzo.Required, ozzo.Length(8, 50)))
```

### Business Logic Validation

For validation that requires database access or complex business rules, use manual validation after the initial ozzo validation:

```go
// After ozzo validation passes
if input.UserID != 0 {
	user, err := env.UserByID(ctx, input.UserID)
	if err != nil {
		if db.IsNoFound(err) {
			return gmodel.NewFieldError("userID", "user_not_found"), nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
}
```

## Common Patterns

### User ID Extraction
```go
req, err := appreq.FromCtx(ctx)
if err != nil {
	return nil, err
}

token, err := req.UserToken()
if err != nil {
	return nil, err
}

userID := int64(token.ID)
```

### Admin Authorization
```go
if !token.IsAdmin() {
	return &model.ErrorPayload{Message: "Unauthorized"}, nil
}
```

### Error Response
```go
return &model.ErrorPayload{Message: "Validation failed"}, nil
```

### Success Response
```go
response := model.CreateEntityPayload{
	Entity: &createdEntity,
}
return &response, nil
```