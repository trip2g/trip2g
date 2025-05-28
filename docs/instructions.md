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

### Overview

This project follows comprehensive testing practices with table-driven tests, mock generation, and specific testing libraries. All tests should be thorough, maintainable, and follow established patterns.

### Required Testing Libraries

```go
import (
	"github.com/kr/pretty"                    // For detailed diff output
	"github.com/matryer/moq"                  // For mock generation  
	"github.com/stretchr/testify/require"     // For assertions
)
```

### Error Handling Pattern in Tests

Follow the project's two-line error handling standard in tests:

```go
// CORRECT: Two-line pattern
result, err := someFunction()
require.NoError(t, err)

// INCORRECT: Single-line pattern
require.NoError(t, someFunction())
```

### Mock Generation with Moq

Generate mocks for interfaces using `moq`:

```bash
# Generate mocks for testing
//go:generate moq -out mocks_test.go . Env
```

Example mock generation comment:
```go
//go:generate moq -out mocks_test.go . Env

type Env interface {
	UserByEmail(ctx context.Context, email string) (db.User, error)
	SendSignInCode(ctx context.Context, params bqtask.SendSignInCodeParams) error
}
```

### Table-Driven Test Pattern

Use table-driven tests for comprehensive coverage:

```go
func TestResolve(t *testing.T) {
	tests := []struct {
		name           string
		input          model.SignInByEmailInput
		mockSetup      func(*EnvMock)
		expectedResult func() model.SignInOrErrorPayload
		expectedError  string
	}{
		{
			name: "success",
			input: model.SignInByEmailInput{
				Email: "test@example.com",
				Code:  "123456",
			},
			mockSetup: func(env *EnvMock) {
				env.UserByEmailFunc = func(ctx context.Context, email string) (db.User, error) {
					return db.User{ID: 1, Email: email}, nil
				}
				env.SendSignInCodeFunc = func(ctx context.Context, params bqtask.SendSignInCodeParams) error {
					return nil
				}
			},
			expectedResult: func() model.SignInOrErrorPayload {
				return &model.SignInPayload{
					User: &db.User{ID: 1, Email: "test@example.com"},
				}
			},
		},
		{
			name: "validation_error_missing_email",
			input: model.SignInByEmailInput{
				Code: "123456",
			},
			mockSetup: func(env *EnvMock) {},
			expectedResult: func() model.SignInOrErrorPayload {
				return &model.ErrorPayload{Message: "Email: cannot be blank."}
			},
		},
		{
			name: "user_not_found",
			input: model.SignInByEmailInput{
				Email: "notfound@example.com",
				Code:  "123456",
			},
			mockSetup: func(env *EnvMock) {
				env.UserByEmailFunc = func(ctx context.Context, email string) (db.User, error) {
					return db.User{}, db.ErrNoRows
				}
			},
			expectedResult: func() model.SignInOrErrorPayload {
				return &model.ErrorPayload{Message: "Invalid email or code"}
			},
		},
		{
			name: "database_error",
			input: model.SignInByEmailInput{
				Email: "test@example.com", 
				Code:  "123456",
			},
			mockSetup: func(env *EnvMock) {
				env.UserByEmailFunc = func(ctx context.Context, email string) (db.User, error) {
					return db.User{}, errors.New("database connection failed")
				}
			},
			expectedError: "failed to get user: database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.mockSetup(env)

			result, err := signinbyemail.Resolve(context.Background(), env, tt.input)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			
			expected := tt.expectedResult()
			if diff := pretty.Diff(expected, result); len(diff) > 0 {
				t.Errorf("Unexpected result (-expected +actual):\n%s", strings.Join(diff, "\n"))
			}
		})
	}
}
```

### Testing Cache Behavior

For packages with caching (like userbans), test cache behavior explicitly:

```go
func TestUserBanByUserID_Cache(t *testing.T) {
	tests := []struct {
		name         string
		userID       int64
		dbCalls      int
		setupMock    func(*DBMock)
		expectedBan  *db.UserBan
		expectedError string
	}{
		{
			name:    "cache_miss_then_hit",
			userID:  1,
			dbCalls: 1, // Should only call DB once
			setupMock: func(db *DBMock) {
				db.UserBanByUserIDFunc = func(ctx context.Context, userID int64) (db.UserBan, error) {
					return db.UserBan{ID: 1, UserID: userID}, nil
				}
			},
			expectedBan: &db.UserBan{ID: 1, UserID: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &DBMock{}
			tt.setupMock(db)

			ub := &UserBans{db: db}

			// First call - cache miss
			result1, err := ub.UserBanByUserID(context.Background(), tt.userID)
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				return
			}
			require.NoError(t, err)

			// Second call - cache hit
			result2, err := ub.UserBanByUserID(context.Background(), tt.userID)
			require.NoError(t, err)

			// Verify results are identical
			if diff := pretty.Diff(result1, result2); len(diff) > 0 {
				t.Errorf("Cache results differ (-first +second):\n%s", strings.Join(diff, "\n"))
			}

			// Verify DB was called correct number of times
			require.Len(t, db.UserBanByUserIDCalls(), tt.dbCalls)
		})
	}
}
```

### Testing Configuration and Validation

For configuration packages, test validation comprehensively:

```go
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedError string
	}{
		{
			name: "valid_config",
			config: Config{
				PublicURL: "https://example.com",
				Port:      "8080",
			},
		},
		{
			name: "invalid_public_url",
			config: Config{
				PublicURL: "not-a-url",
				Port:      "8080",
			},
			expectedError: "PublicURL: must be a valid URL",
		},
		{
			name: "missing_port",
			config: Config{
				PublicURL: "https://example.com",
			},
			expectedError: "Port: cannot be blank",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
```

### Best Practices

1. **Test Structure**: Use descriptive test names that explain the scenario
2. **Mock Setup**: Keep mock setup functions focused and reusable
3. **Error Testing**: Test both success and error paths thoroughly
4. **Cache Testing**: Verify cache behavior when applicable
5. **Validation Testing**: Test all validation rules and edge cases
6. **Pretty Diff**: Use `pretty.Diff` for detailed comparison output
7. **Require**: Use `require` for assertions that should stop test execution
8. **Context**: Always pass proper context to functions under test

### Test Organization

```go
func TestPackageName(t *testing.T) {
	// Setup common to all tests
	
	t.Run("SubTestGroup", func(t *testing.T) {
		// Grouped related tests
		
		t.Run("specific_scenario", func(t *testing.T) {
			// Individual test case
		})
	})
}
```

### Env Interface Testing Pattern

The Env interface pattern makes testing straightforward with mocks:

```go
//go:generate moq -out mocks_test.go . Env

type Env interface {
	GenerateApiKey() string
	InsertApiKey(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error)
}

func TestResolve(t *testing.T) {
	env := &EnvMock{
		GenerateApiKeyFunc: func() string {
			return "test-api-key"
		},
		InsertApiKeyFunc: func(ctx context.Context, params db.InsertApiKeyParams) (db.ApiKey, error) {
			return db.ApiKey{ID: 1, Value: params.Value}, nil
		},
	}
	
	input := model.CreateAPIKeyInput{Description: "test"}
	
	result, err := createapikey.Resolve(context.Background(), env, input)
	require.NoError(t, err)
	
	// Use pretty.Diff for detailed comparison
	expected := &model.CreateAPIKeyPayload{
		APIKey: &db.ApiKey{ID: 1, Value: "test-api-key"},
		Value:  "test-api-key",
	}
	
	if diff := pretty.Diff(expected, result); len(diff) > 0 {
		t.Errorf("Unexpected result (-expected +actual):\n%s", strings.Join(diff, "\n"))
	}
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