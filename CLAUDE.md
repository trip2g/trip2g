Read instructions of common patterns in docs/instructions.md

## Database Schema

### tg_bot_chat_subgraph_accesses

Tracks when users request and actually join Telegram chats:
- Insert record when user clicks "Join Chat" button (sets `created_at`)
- Update `joined_at` when user actually joins the chat (`new_chat_members` event)
- Cron job uses this to remove users from chats when their subgraph access expires

## Cron Jobs

The application uses a robust cron job system for scheduled tasks. Cron jobs are implemented in the `internal/case/cronjob/` directory and managed by the `internal/cronjobs` package.

### Creating a New Cron Job

1. **Create the case directory**:
   ```bash
   mkdir -p internal/case/cronjob/yourcronjobname
   ```

2. **Create `resolve.go`** with the business logic:
   ```go
   package yourcronjobname

   import (
       "context"
       "trip2g/internal/logger"
   )

   type Env interface {
       // Add required database methods
   }

   type Result struct {
       // Add result fields
   }

   func Resolve(ctx context.Context, env Env) (*Result, error) {
       // Implementation here
       return &Result{}, nil
   }
   ```

3. **Create `job.go`** to define the job schedule:
   ```go
   package yourcronjobname

   import "context"

   type Job struct{}

   func (j *Job) Name() string {
       return "your_cron_job_name"
   }

   func (j *Job) Schedule() string {
       // Cron expression (seconds, minutes, hours, day of month, month, day of week)
       return "0 0 0 * * *" // daily at midnight
   }

   func (j *Job) ExecuteAfterStart() bool {
       return false // set to true if job should run immediately on startup
   }

   func (j *Job) Execute(ctx context.Context, env any) (any, error) {
       return Resolve(ctx, env.(Env), Filter{})
   }
   ```

4. **Add to `cmd/server/cronjobs.go`**:
   ```go
   import (
       "trip2g/internal/case/cronjob/yourcronjobname"
   )

   func getCronJobConfigs(app *app) []cronjobs.Job {
       // Compile-time interface checks
       var (
           _ yourcronjobname.Env = app
       )

       return []cronjobs.Job{
           &yourcronjobname.Job{},
       }
   }
   ```

### Existing Cron Jobs

**`remove_expired_tg_chat_members`**:
- **Schedule**: Every hour (`0 0 * * * *`)
- **Purpose**: Remove users from Telegram chats when their subgraph access expires
- **Location**: `internal/case/cronjob/removeexpiredtgchatmembers/`

**`clear_cronjob_execution_history`**:
- **Schedule**: Daily at midnight (`0 0 0 * * *`)
- **Purpose**: Clean up cron job execution history older than 7 days
- **Location**: `internal/case/cronjob/clearcronjobexecutionhistory/`

### Cron Expression Format

The system uses 6-field cron expressions (with seconds):
```
┌───────────── second (0 - 59)
│ ┌───────────── minute (0 - 59)
│ │ ┌───────────── hour (0 - 23)
│ │ │ ┌───────────── day of month (1 - 31)
│ │ │ │ ┌───────────── month (1 - 12)
│ │ │ │ │ ┌───────────── day of week (0 - 6) (Sunday to Saturday)
│ │ │ │ │ │
* * * * * *
```

**Examples**:
- `0 0 * * * *` - Every hour at the top of the hour
- `0 0 0 * * *` - Every day at midnight
- `0 30 2 * * *` - Every day at 2:30 AM
- `0 0 0 * * 0` - Every Sunday at midnight
- `0 0 0 1 * *` - First day of every month at midnight

### Job Management

- Jobs are automatically registered in the database on startup
- Jobs can be enabled/disabled via the GraphQL admin API
- Job schedules can be updated via the GraphQL admin API
- Job execution history is tracked and can be viewed via GraphQL
- Failed jobs are logged with error details

## Golang

Don’t write

```golang
if err := ...; err != nil
```

Always use two lines:

```golang
err = ...
if err != nil {
```

**IMPORTANT**: After making changes to Go code:
1. Format code with: `gofmt -w .` (or for specific files: `gofmt -w file.go`)
2. Run tests for affected packages: `go test ./internal/case/packagename -v`
3. Run all tests to ensure nothing is broken: `go test ./...`
4. Run `make lint` to ensure:
   - Code compiles without errors
   - All linting rules pass
   - Generated code is up to date

## Commit Guidelines

When creating commits, follow these guidelines:

### Message Format
- Use conventional commit format: `type(scope): description`
- Keep first line under 50 characters when possible
- Use present tense: "add feature" not "added feature"
- Use imperative mood: "fix bug" not "fixes bug"

### Common Types
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring without functionality change
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

### Examples
```
feat(ui/admin): add release management catalog
fix(db): handle null values in user queries
refactor(ui): move components to proper namespaces
docs: update API documentation
```

### Commit Process
```bash
git add .
git commit -m "type(scope): brief description"
```

Do not add co-author comments or generated signatures unless specifically requested.

## Technology Stack

**Backend:**
- Go 1.21+ with SQLite (WAL mode)
- [sqlc](https://sqlc.dev/) for type-safe SQL queries
- [gqlgen](https://gqlgen.com/) for GraphQL server
- [fasthttp](https://github.com/valyala/fasthttp) for HTTP server
- [ozzo-validation](https://github.com/go-ozzo/ozzo-validation) for input validation
- [dbmate](https://github.com/amacneil/dbmate) for database migrations

**Frontend:**
- [$mol framework](https://github.com/hyoo-ru/mam_mol) with TypeScript

## Admin Authorization

**IMPORTANT**: All admin mutation cases must verify admin authorization at the start:
```go
token, err := env.CurrentAdminUserToken(ctx)
if err != nil {
    return nil, fmt.Errorf("failed to get current user token: %w", err)
}
```

## Development Workflow

### Database Migrations
- **Create new migration**: `make db-new name=create_user_favorite_notes`
- **Apply migrations**: `make db-up`
- Migrations use [dbmate](https://github.com/amacneil/dbmate) format with `-- migrate:up` and `-- migrate:down` sections

### Backend Changes
1. **SQL**: Add queries to `queries.sql` → run `make sqlc`
2. **GraphQL**: Update `internal/graph/schema.graphqls` → run `make gqlgen`
3. **Business Logic**: Implement in `internal/case/.../resolve.go`
4. **Tests**: Write comprehensive tests with table-driven patterns

### Frontend Changes
1. **Components**: Create `.view.tree` files for structure
2. **Behavior**: Add `.view.ts` files for TypeScript behavior
3. **GraphQL**: Use `$trip2g_graphql_request` → run `npm run graphqlgen`
4. **Organization**: Group by entity (e.g., `admin/noteview/select/`)

## Key Patterns

### GraphQL Mutations
- Accept only one `input` argument
- Return `union ${Mutation}OrErrorPayload = ${Mutation}Payload | ErrorPayload`
- Use Env interface pattern for testability

### SQL Style Guide
- **Keywords**: Use lowercase for all SQL keywords (`select`, `from`, `where`, `create table`, etc.)
- **Table/Column names**: Use lowercase with underscores
- **Indentation**: Use consistent indentation for readability
- **Example**:
  ```sql
  -- Good
  create table users (
      id integer primary key,
      email text not null,
      created_at datetime not null default (datetime('now'))
  );
  
  -- Bad
  CREATE TABLE Users (
      ID INTEGER PRIMARY KEY,
      Email TEXT NOT NULL,
      CreatedAt DATETIME NOT NULL DEFAULT (DATETIME('now'))
  );
  ```

## Router and HTTP Endpoints

### Adding HTTP Endpoints

To add a new HTTP endpoint (webhook handler, API endpoint, etc.):

1. **Create case directory**: `mkdir -p internal/case/yourhandlername`

2. **Create resolve.go** with business logic:
   ```go
   package yourhandlername
   
   type Env interface {
       // Define required methods
   }
   
   func Resolve(ctx context.Context, env Env, ...) (ReturnType, error) {
       // Business logic here
   }
   ```

3. **Create endpoint.go** to handle HTTP request/response:
   ```go
   package yourhandlername
   
   import (
       "net/http"
       "trip2g/internal/appreq"
   )
   
   type Endpoint struct{}
   
   func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
       env := req.Env.(Env)
       // Extract parameters, headers, body
       // Call Resolve
       return Resolve(req.Req, env, ...)
   }
   
   func (*Endpoint) Path() string {
       return "/api/your/path"
   }
   
   func (*Endpoint) Method() string {
       return http.MethodPost // or http.MethodGet, etc.
   }
   ```

4. **Generate router**: Run `go generate ./internal/router/...`
   - This scans all `internal/case/*` directories for `Endpoint` types
   - Updates `internal/router/endpoints_gen.go` automatically
   - Updates `RoutesEnv` interface to include your case's `Env` interface

5. **Implement Env methods** in `cmd/server/main.go` if needed

### Example: Webhook Handler
See `internal/case/processnowpaymentsipn` for a complete webhook handler example that:
- Validates signatures
- Parses JSON payload
- Updates database state
- Returns appropriate HTTP status codes

## Adding New Features

### Adding SQL Queries and Database Methods

When you need new database operations:

1. **Add SQL Query to `queries.sql`**:
   ```sql
   -- name: MethodName :one
   select * from table_name where id = ?;
   ```

2. **Generate Go Code**:
   ```bash
   make sqlc
   ```

3. **Check Generated Method** in `internal/db/queries.sql.go`:
   ```go
   func (q *Queries) MethodName(ctx context.Context, id int64) (TableType, error)
   ```

4. **Add to Env Interface** (if needed for GraphQL resolvers):
   - The main `Env` interface automatically includes all `*Queries` methods
   - For case-specific interfaces, add method to the case's `Env` interface

### Adding GraphQL Mutations

1. **Check Schema** in `internal/graph/schema.graphqls`:
   - Mutation may already be defined
   - Input/Output types should follow pattern: `${Mutation}Input`, `${Mutation}Payload`, `${Mutation}OrErrorPayload`

2. **Run GraphQL Generation**:
   ```bash
   make gqlgen
   ```

3. **Implement Business Logic**:
   - Create directory: `internal/case/${mutationname}/` (for user mutations) or `internal/case/admin/${mutationname}/` (for admin mutations)
   - Create `resolve.go` following this pattern:
     ```go
     package ${mutationname}

     import (
         "context"
         "database/sql"
         "fmt"

         ozzo "github.com/go-ozzo/ozzo-validation/v4"
         validation "github.com/go-ozzo/ozzo-validation/v4"
         "github.com/go-ozzo/ozzo-validation/v4/is"

         "trip2g/internal/db"
         "trip2g/internal/graph/model"
     )

     type Env interface {
         // Required database methods
         InsertSomething(ctx context.Context, arg db.InsertSomethingParams) error
         // Other methods as needed
     }

     // Type aliases for cleaner code
     type Input = model.${Mutation}Input
     type Payload = model.${Mutation}OrErrorPayload

     // validateRequest validates input and returns ErrorPayload if invalid
     func validateRequest(r *Input) *model.ErrorPayload {
         return model.NewOzzoError(ozzo.ValidateStruct(r,
             ozzo.Field(&r.Field1, validation.Required),
             ozzo.Field(&r.Email, validation.Required, is.Email),
             // Add all validation rules
         ))
     }

     func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
         // Always validate input first
         errPayload := validateRequest(&input)
         if errPayload != nil {
             return errPayload, nil  // User-visible errors go in ErrorPayload
         }

         // Define params as separate variable for cleaner code
         params := db.InsertSomethingParams{
             Field1: input.Field1,
             Field2: sql.NullString{String: input.Field2, Valid: input.Field2 != ""},
             // Map all fields
         }

         // Execute database operation
         err := env.InsertSomething(ctx, params)
         if err != nil {
             if db.IsUniqueViolation(err) {
                 // Handle unique constraint violations (e.g., ignore duplicates)
                 // Continue to success response
             } else {
                 // System errors are returned as error (will show generic message to user)
                 return nil, fmt.Errorf("failed to insert something: %w", err)
             }
         }

         // Define payload as separate variable
         payload := model.${Mutation}Payload{
             Success: true,
             // Add other return fields
         }

         return &payload, nil
     }
     ```

   **Important patterns:**
   - Use type aliases (`Input`, `Payload`) for cleaner code
   - Create `validateRequest` function that returns `*model.ErrorPayload`
   - User-visible validation errors return `ErrorPayload` with `nil` error
   - System/internal errors return `nil` payload with wrapped error
   - Define params and payload as separate variables before use
   - Return `&payload, nil` for successful responses
   - Handle unique constraint violations with `db.IsUniqueViolation(err)` to ignore duplicates

4. **Define Env Interface** in the case:
   ```go
   type Env interface {
       RequiredMethod1(ctx context.Context, ...) (Type, error)
       RequiredMethod2(ctx context.Context, ...) error
   }
   ```

5. **Add Case Env to Main Interface** in `internal/graph/resolver.go`:
   ```go
   import "trip2g/internal/case/${mutationname}"        // for user mutations
   import "trip2g/internal/case/admin/${mutationname}"  // for admin mutations
   
   type Env interface {
       // ... existing methods ...
       ${mutationname}.Env
   }
   ```

6. **Update GraphQL Resolver** in `internal/graph/schema.resolvers.go`:
   ```go
   // For user mutations (in root Mutation type):
   import "trip2g/internal/case/${mutationname}"
   
   func (r *mutationResolver) ${Mutation}(ctx context.Context, input model.${Mutation}Input) (model.${Mutation}OrErrorPayload, error) {
       return ${mutationname}.Resolve(ctx, r.env(ctx), input)
   }
   
   // For admin mutations (in AdminMutation type):
   import "trip2g/internal/case/admin/${mutationname}"
   
   func (r *adminMutationResolver) ${Mutation}(ctx context.Context, obj *appmodel.AdminMutation, input model.${Mutation}Input) (model.${Mutation}OrErrorPayload, error) {
       return ${mutationname}.Resolve(ctx, r.env(ctx), input)
   }
   ```

7. **Write Tests** following the pattern in `internal/userbans/userbans_test.go`:
   - Create `resolve_test.go` with table-driven tests
   - Use `//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env` for mocking
   - Test success, error, and edge cases
   - **Don't forget**: Run `go generate` if tests contain `//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env` to generate mocks

8. **Add Methods to Main Server** (if needed) in `cmd/server/main.go`:
   - Only if the case requires methods not available in standard `*Queries`

## Frontend Development

For frontend development patterns, admin CRUD interfaces, and UI components, see [Frontend Documentation](docs/frontend.md).

## Mol Framework

For detailed Mol framework documentation including $mol_view properties, view.tree syntax, and component patterns, see [Mol Framework Documentation](docs/mol.md).
