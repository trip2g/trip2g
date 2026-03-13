# Development & Operations Guide

**Generated:** 2025-11-15
**Project:** trip2g

## Quick Start

### Prerequisites
- Go 1.24+
- Node.js 20+
- SQLite 3.x
- Make
- Docker (optional)

### Initial Setup

```bash
# Clone repository
cd /path/to/trip2g

# Install Go dependencies
go mod download

# Install Node.js dependencies
npm install

# Setup database
make db-up

# Generate code
make sqlc
make gqlgen
```

### Run Development Server

**Option 1: Hot Reload (Recommended)**
```bash
make air
```
This starts the server with automatic reloading on file changes.

**Option 2: Manual Build**
```bash
make build
./tmp/server
```

**Frontend Development:**
The frontend is served directly by the backend server. Changes to `.view.tree` and `.view.ts` files are reflected on browser refresh.

## Build Commands

### Make Targets

**Testing:**
```bash
make test                 # Run all Go tests
```

**Building:**
```bash
make build               # Build server binary
make build-amd64         # Build for Linux AMD64
make build-docker        # Build Docker image
```

**Code Generation:**
```bash
make sqlc                # Generate SQL code from queries.sql
make gqlgen              # Generate GraphQL resolvers
make graphqlgen          # Generate frontend GraphQL types
```

**Database:**
```bash
make db-new name=<desc>  # Create new migration
make db-up               # Apply pending migrations
make db-down             # Rollback last migration
```

**Quality:**
```bash
make lint                # Run golangci-lint
```

**Deployment:**
```bash
make build_and_deploy    # Build and deploy via Ansible
make deploy              # Deploy only (no build)
```

### Air Configuration

**File:** `.air.toml`

**Build Command:**
```bash
go build -o ./tmp/main -race -tags=dev ./cmd/server
```

**Watched Extensions:**
- `.go` - Go source files
- `.qtpl` - QuickTemplate files
- `.tpl`, `.tmpl`, `.html` - Template files

**Excluded:**
- Test files (`*_test.go`)
- Generated files (`easyjson`)
- Dependencies (`node_modules`, `vendor`)
- Build artifacts (`tmp`, `dist`)

**Hot Reload Delay:** 500ms

## Development Workflow

### 1. Adding Database Tables/Columns

```bash
# Create migration
make db-new name=add_user_preferences

# Edit migration file
# db/migrations/YYYYMMDDHHMMSS_add_user_preferences.sql

# Apply migration
make db-up

# Add SQL queries to internal/db/queries.sql
# -- name: GetUserPreferences :one
# select * from user_preferences where user_id = ?

# Generate Go code
make sqlc
```

**Generated Files:**
- `internal/db/queries.sql.go` - Type-safe query methods
- `internal/db/models.go` - Table struct definitions

### 2. Adding GraphQL API

```bash
# Edit schema
# internal/graph/schema.graphqls

# Generate resolvers
make gqlgen

# Implement business logic
# Create: internal/case/mynewfeature/resolve.go

# Wire up resolver
# Edit: internal/graph/schema.resolvers.go

# Generate frontend types
npm run graphqlgen
```

**Pattern:**
```go
// internal/case/mynewfeature/resolve.go
package mynewfeature

type Env interface {
    GetSomething(ctx context.Context, id int64) (*db.Something, error)
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
    // Business logic here
    return &SuccessPayload{}, nil
}
```

### 3. Adding UI Components

```bash
# Create component directory
mkdir -p assets/ui/mycomponent

# Create view.tree file
# assets/ui/mycomponent/mycomponent.view.tree

# Optional: Add TypeScript behavior
# assets/ui/mycomponent/mycomponent.view.ts

# Optional: Add styles
# assets/ui/mycomponent/mycomponent.view.css.ts

# Component auto-discovered on next page load
```

### 4. Adding HTTP Endpoints

```bash
# Create endpoint handler
# internal/case/myendpoint/endpoint.go

# Regenerate router
go generate ./internal/router/...

# Implement Env interface in cmd/server/main.go if needed
```

**Pattern:**
```go
// internal/case/myendpoint/endpoint.go
package myendpoint

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
    // Handle request
    return response, nil
}

func (*Endpoint) Path() string {
    return "/api/my/path"
}

func (*Endpoint) Method() string {
    return http.MethodPost
}
```

### 5. Adding Cron Jobs

```bash
# Create cron job directory
mkdir -p internal/case/cronjob/mynewjob

# Create resolve.go
# internal/case/cronjob/mynewjob/resolve.go

# Create job.go with schedule
# internal/case/cronjob/mynewjob/job.go

# Register in cmd/server/cronjobs.go
```

**Cron Expression Format:** 6 fields (seconds, minutes, hours, day, month, weekday)

Examples:
- `0 0 * * * *` - Every hour
- `0 0 0 * * *` - Daily at midnight
- `0 30 2 * * *` - Daily at 2:30 AM

### 6. Adding Background Jobs

```bash
# Create job directory
mkdir -p internal/case/backjob/mynewjob

# Implement job interface
# internal/case/backjob/mynewjob/job.go

# Enqueue from anywhere
jobs.Enqueue(ctx, &mynewjob.Job{Params: ...})
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/case/packagename -v

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

**Test Pattern:**
```go
// internal/case/myfeature/resolve_test.go
package myfeature_test

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg myfeature_test . Env

func TestResolve(t *testing.T) {
    tests := []struct {
        name    string
        input   myfeature.Input
        want    myfeature.Payload
        wantErr bool
    }{
        {
            name: "success case",
            input: myfeature.Input{Field: "value"},
            want: &myfeature.SuccessPayload{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            env := &EnvMock{
                GetSomethingFunc: func(ctx context.Context, id int64) (*db.Something, error) {
                    return &db.Something{}, nil
                },
            }

            got, err := myfeature.Resolve(context.Background(), env, tt.input)
            // Assertions
        })
    }
}
```

### End-to-End Tests

```bash
# Run E2E tests
npm run test:e2e          # Headless mode
npm run test:e2e:ui       # Interactive UI
npm run test:e2e:headed   # Headed browser

# Run specific test file
npx playwright test e2e/tests/admin/user.spec.ts
```

**Test Structure:**
```typescript
// e2e/tests/myfeature.spec.ts
import { test, expect } from '@playwright/test';

test('should do something', async ({ page }) => {
    await page.goto('/');
    await page.click('[data-testid="button"]');
    await expect(page.locator('.result')).toBeVisible();
});
```

## Database Management

### Migration Workflow

**Create Migration:**
```bash
make db-new name=add_user_settings
```

**Migration File Format:**
```sql
-- migrate:up
create table user_settings (
  id integer primary key autoincrement,
  user_id integer not null references users(id) on delete cascade,
  key text not null,
  value text not null,
  created_at datetime not null default current_timestamp
);

-- migrate:down
drop table user_settings;
```

**Apply Migrations:**
```bash
make db-up                    # Apply pending
make db-down                  # Rollback one
```

**Database Location:**
- Development: `data12.sqlite3` (or latest numbered file)
- WAL mode enabled for concurrency

### SQL Queries (sqlc)

**Add Query:**
```sql
-- internal/db/queries.sql

-- name: GetUserSettings :many
select * from user_settings where user_id = ?;

-- name: UpsertUserSetting :exec
insert into user_settings (user_id, key, value)
values (?, ?, ?)
on conflict (user_id, key) do update set value = excluded.value;
```

**Generate Code:**
```bash
make sqlc
```

**Usage:**
```go
settings, err := queries.GetUserSettings(ctx, userID)
```

## Code Generation

### Auto-Generated Files

**DO NOT EDIT:**
- `internal/db/queries.sql.go` - From `make sqlc`
- `internal/graph/generated.go` - From `make gqlgen`
- `internal/graph/model/*.go` - From `make gqlgen`
- `internal/router/endpoints_gen.go` - From `go generate ./internal/router/...`
- `assets/ui/**/-view.tree/*.d.ts` - From $mol framework

**Generation Commands:**
```bash
# After schema.graphqls changes
make gqlgen

# After queries.sql changes
make sqlc

# After adding HTTP endpoints
go generate ./internal/router/...

# After GraphQL schema changes (frontend)
npm run graphqlgen

# Generate all
make sqlc && make gqlgen && npm run graphqlgen
```

## Environment Configuration

### Environment Variables

```bash
# Database
DATABASE_URL=data.sqlite3

# Server
PORT=8080
HOST=0.0.0.0

# External Services
TELEGRAM_BOT_TOKEN=<token>
PATREON_CLIENT_ID=<id>
PATREON_CLIENT_SECRET=<secret>
BOOSTY_DEVICE_ID=<id>
NOWPAYMENTS_API_KEY=<key>
MINIO_ENDPOINT=<endpoint>
MINIO_ACCESS_KEY=<key>
MINIO_SECRET_KEY=<secret>

# Email
RESEND_API_KEY=<key>

# Development
DEV_MODE=true
```

**Load from .env:**
```bash
# Create .env file
cp .env.example .env

# Edit values
nano .env
```

### Configuration Files

**Application Config:**
- See `/internal/appconfig/` for configuration management
- Config stored in database (`config_versions` table)
- Editable via admin panel

## Deployment

### Docker Deployment

**Build Image:**
```bash
make build-docker
```

**Dockerfile Stages:**
1. Builder stage (Go 1.24)
   - Download dependencies
   - Build static binary
2. Runtime stage (Alpine)
   - Copy binary
   - Install git and ca-certificates
   - Run as non-root

**Run Container:**
```bash
docker run -p 8080:8080 \
  -v $(pwd)/data.sqlite3:/app/data.sqlite3 \
  -e DATABASE_URL=/app/data.sqlite3 \
  trip2g
```

### Ansible Deployment

**Configuration:** `/infra/`

**Deploy:**
```bash
make deploy                  # Deploy only
make build_and_deploy        # Build then deploy
```

**Process:**
1. Build for AMD64
2. Upload binary via Ansible
3. Restart service
4. Verify health

### Manual Deployment

```bash
# Build for target
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -o trip2g \
  -ldflags="-s -w" \
  ./cmd/server

# Copy to server
scp trip2g server:/opt/trip2g/

# Restart service
ssh server 'systemctl restart trip2g'
```

## Monitoring & Logging

### Structured Logging

**Framework:** zerolog

**Log Levels:**
- Debug
- Info
- Warning
- Error

**Example:**
```go
log.Info().
    Str("user_id", userID).
    Msg("User signed in")
```

### Health Checks

**Endpoint:** `/health` (if implemented)

**Admin Panel:**
- Health checks visible in admin dashboard
- See `admin/healthchecks/`

### Audit Logs

**Location:** `audit_logs` table

**Query via Admin:**
```graphql
query AuditLogs {
  admin {
    auditLogs(filter: { limit: 100 }) {
      nodes {
        id
        level
        message
        params
        createdAt
      }
    }
  }
}
```

### Cron Job Monitoring

**Via Admin Panel:**
- View execution history
- Check error messages
- Manual trigger for testing

**Via Database:**
```sql
select * from cron_job_executions
where status = 3  -- failed
order by started_at desc
limit 10;
```

## Troubleshooting

### Common Issues

**Database Locked:**
```bash
# Check for active connections
lsof data.sqlite3

# Restart server
make air
```

**Port Already in Use:**
```bash
# Find process
lsof -i :8080

# Kill process
kill -9 <PID>
```

**Code Generation Errors:**
```bash
# Clean generated files
rm -rf internal/graph/generated.go
rm -rf internal/db/queries.sql.go

# Regenerate
make gqlgen sqlc
```

**Module Issues:**
```bash
# Clean module cache
go clean -modcache

# Re-download
go mod download
go mod tidy
```

### Debug Mode

**Enable Race Detector:**
```bash
go build -race ./cmd/server
```

**Verbose Logging:**
```bash
export LOG_LEVEL=debug
make air
```

**Profile Performance:**
```bash
go build -o server ./cmd/server
./server -cpuprofile=cpu.prof
```

## Scripts

**Test E2E:** `/scripts/test-e2e.sh`
```bash
./scripts/test-e2e.sh          # Run tests
./scripts/test-e2e.sh --ui     # Interactive mode
./scripts/test-e2e.sh --headed # Show browser
```

**Wait for Service:** `/scripts/waitfor`
```bash
./scripts/waitfor localhost:8080
```

**Push Notes (Dev Tool):** `/scripts/push_notes.py`
```bash
python scripts/push_notes.py --vault /path/to/vault
```

**Upload Asset:** `/scripts/upload_asset`
```bash
./scripts/upload_asset <note_id> <file_path>
```

## Performance Tips

1. **Use WAL Mode:** Already enabled in SQLite
2. **Connection Pooling:** Configured in database setup
3. **Background Jobs:** Use queues for heavy operations
4. **Caching:** Consider adding Redis for frequent queries
5. **Index Optimization:** Check query plans with `EXPLAIN QUERY PLAN`

## Security Practices

1. **Never commit secrets:** Use environment variables
2. **Validate input:** Use ozzo-validation
3. **SQL injection:** Using sqlc (parameterized queries)
4. **XSS protection:** Markdown sanitization in renderer
5. **CSRF:** Implement tokens for state-changing operations
6. **Rate limiting:** Email codes, API endpoints

## Development Tools

**Recommended:**
- **IDE:** VSCode with Go extension
- **Database:** DB Browser for SQLite
- **API Testing:** GraphQL Playground (built-in at `/playground`)
- **HTTP Testing:** cURL, HTTPie, Postman
- **Git Client:** Command line, GitKraken, SourceTree

**VSCode Extensions:**
- Go (golang.go)
- GraphQL (GraphQL.vscode-graphql)
- SQLite (alexcvzz.vscode-sqlite)
- Playwright Test (ms-playwright.playwright)
