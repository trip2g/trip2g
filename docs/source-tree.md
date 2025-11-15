# Source Tree Documentation

**Generated:** 2025-11-15
**Project Root:** `/home/alexes/projects2/trip2g`

## Project Structure Overview

```
trip2g/
├── cmd/                    # Entry points and executables
├── internal/               # Private application code
├── assets/ui/              # Frontend ($mol framework)
├── db/                     # Database schema and migrations
├── docs/                   # Documentation
├── e2e/                    # End-to-end tests
├── testdata/               # Test data and fixtures
├── infra/                  # Infrastructure as code
├── scripts/                # Build and utility scripts
└── demo/                   # Demo content (legacy)
```

## Entry Points (`/cmd`)

**Purpose:** Application entry points and command-line tools

```
cmd/
├── server/                 # Main HTTP server
│   ├── main.go            # Server entry point
│   ├── cronjobs.go        # Cron job configuration
│   └── routes.go          # HTTP route setup
├── hosting/               # Hosting utility
├── patreon/               # Patreon integration tool
└── testgoqite/            # Queue testing utility
```

**Key Files:**
- `cmd/server/main.go` - Initializes fasthttp server, GraphQL, database, cron jobs, background workers
- `cmd/server/cronjobs.go` - Registers all cron jobs with their schedules
- `cmd/server/routes.go` - Maps HTTP routes to handlers

## Backend Core (`/internal`)

**Purpose:** Private Go packages (not importable by external projects)

### Application Layer

```
internal/
├── case/                   # Business logic (use cases)
│   ├── admin/             # Admin mutations (41 packages)
│   ├── cronjob/           # Scheduled jobs (2 jobs)
│   └── backjob/           # Background jobs (7 jobs)
├── graph/                  # GraphQL layer
│   ├── schema.graphqls    # GraphQL schema definition
│   ├── resolver.go        # Root resolver
│   └── schema.resolvers.go # Generated resolvers
└── router/                 # HTTP routing
    ├── endpoints_gen.go   # Auto-generated endpoint registry
    └── router.go          # Route handler
```

**Use Case Structure:**
- Each case is a single-responsibility business operation
- Follows pattern: `Resolve(ctx, env, input) (output, error)`
- Admin cases in `case/admin/*`
- Public cases in `case/*`
- Cron jobs in `case/cronjob/*`
- Background jobs in `case/backjob/*`

**Notable Use Cases:**
```
case/
├── admin/
│   ├── banuser/           # Ban user functionality
│   ├── createoffer/       # Create subscription offer
│   ├── createrelease/     # Deploy new release
│   ├── createtgbot/       # Register Telegram bot
│   ├── refreshpatreondata/ # Sync Patreon members
│   └── updatecronjob/     # Manage cron jobs
├── createpaymentlink/     # Generate payment URL
├── processnowpaymentsipn/ # Handle crypto payment webhook
├── handletgupdate/        # Process Telegram updates
├── insertnote/            # Create/update note version
└── canreadnote/           # Check note access permissions
```

### Data Layer

```
internal/
├── db/                     # Database access
│   ├── queries.sql        # SQL queries (sqlc source)
│   ├── queries.sql.go     # Generated type-safe queries
│   ├── models.go          # Database models
│   └── fix_write_queries.sh # Post-generation fixes
└── mdloader/               # Markdown processing
    ├── loader.go          # Parse markdown to NoteView
    ├── wikilink.go        # Obsidian wikilink support
    └── enclave.go         # Protected content sections
```

**Database Pattern:**
- Uses [sqlc](https://sqlc.dev/) for type-safe SQL
- `queries.sql` contains all SQL queries with `-- name:` annotations
- `queries.sql.go` auto-generated from queries.sql
- WAL mode for concurrency

### Integration Layers

```
internal/
├── boosty/                 # Boosty API client
│   └── client.go
├── boostyjobs/             # Boosty sync jobs
├── gitapi/                 # Git protocol server
├── hotauthtoken/           # JWT token management
├── layoutloader/           # Hugo layout integration
├── miniostorage/           # S3-compatible storage
├── noteloader/             # Note version management
└── telegram/               # Telegram bot framework
    ├── bot.go
    ├── handlers/
    └── publish/
```

### Infrastructure

```
internal/
├── logger/                 # Structured logging (zerolog)
├── jobs/                   # Background job queue
├── cronjobs/               # Cron job system
├── fasthttp/               # HTTP utilities
├── appreq/                 # Request context
├── appresp/                # Response helpers
├── auditlogger/            # Admin action logging
└── appconfig/              # Configuration management
```

### Utilities

```
internal/
├── model/                  # Shared domain models
├── apperrors/              # Error handling
├── acmecache/              # ACME cert storage
├── aipipeline/             # AI processing pipeline
├── enclavefix/             # Markdown enclave processing
├── image/                  # Image processing
├── markdownv2/             # Telegram MarkdownV2 formatter
└── tgusers/                # Telegram user management
```

## Frontend (`/assets/ui`)

**Purpose:** $mol framework components and UI logic

```
assets/ui/
├── admin/                  # Admin panel (103 components)
│   ├── user/              # User management
│   ├── subgraph/          # Access control
│   ├── offer/             # Subscription offers
│   ├── tgbot/             # Telegram bots
│   ├── telegrampublishnote/ # Publishing queue
│   ├── patreoncredentials/ # Patreon integration
│   ├── boostycredentials/ # Boosty integration
│   ├── cronjob/           # Cron job management
│   └── ... (see ui-components.md for full list)
├── user/                   # User-facing components (13 components)
│   ├── paywall/           # Subscription paywall
│   ├── search/            # Content search
│   └── space/             # User dashboard
├── reader/                 # Markdown reader
├── auth/                   # Authentication
├── graphql/                # GraphQL client setup
├── settings/               # Application settings
├── state/                  # Global state
└── theme/                  # Theme management
```

**Component Pattern:**
```
componentname/
├── componentname.view.tree      # Structure definition
├── componentname.view.ts        # TypeScript behavior
├── componentname.view.css.ts    # Component styles
└── -view.tree/                  # Generated types
    └── componentname.view.tree.d.ts
```

## Database (`/db`)

**Purpose:** Schema management and migrations

```
db/
├── migrations/             # dbmate migrations (80+ files)
│   ├── 20250402131258_create_note_tables.sql
│   ├── 20250724085424_create_patreon_credentials_table.sql
│   ├── 20251021134341_create_telegram_publish_notes.sql
│   └── ...
└── schema.sql             # Current database schema
```

**Migration Pattern:**
- Uses [dbmate](https://github.com/amacneil/dbmate)
- Format: `YYYYMMDDHHMMSS_description.sql`
- Each file has `-- migrate:up` and `-- migrate:down` sections
- Run with `make db-up` / `make db-down`

## Documentation (`/docs`)

**Purpose:** Project documentation and AI context

```
docs/
├── instructions.md         # Development patterns
├── frontend.md             # Frontend guide
├── mol.md                  # $mol framework docs
├── telegram.md             # Telegram integration
├── obsidian_links.md       # Wikilink handling
├── queues.md               # Job queue documentation
├── sqlite.md               # Database patterns
├── TESTING.md              # Testing guide
├── aicontext.md            # AI assistant context
├── sprint-artifacts/       # BMad workflow artifacts
├── api-contracts.md        # GraphQL API docs (generated)
├── data-models.md          # Database schema docs (generated)
├── ui-components.md        # UI component catalog (generated)
└── source-tree.md          # This file (generated)
```

## Testing (`/e2e`)

**Purpose:** End-to-end tests with Playwright

```
e2e/
├── README.md               # Test documentation
├── playwright.config.ts    # Playwright configuration
├── fixtures/               # Test fixtures
├── pages/                  # Page objects
└── tests/                  # Test specs
    ├── admin/             # Admin panel tests
    └── user/              # User flow tests
```

**Run tests:**
```bash
npm run test:e2e          # Headless
npm run test:e2e:ui       # UI mode
npm run test:e2e:headed   # Headed mode
```

## Test Data (`/testdata`)

**Purpose:** Test fixtures and sample content

```
testdata/
└── vault/                  # Current test vault content
    ├── index.md
    ├── private.md
    └── ... (sample markdown files)
```

**Note:** `demo/` and `demo2/` are legacy demo content (not actively used)

## Infrastructure (`/infra`)

**Purpose:** Deployment and infrastructure configuration

```
infra/
├── ansible/               # Ansible playbooks
├── terraform/             # Terraform configs (if any)
└── docker/                # Docker configurations
```

## Scripts (`/scripts`)

**Purpose:** Build, development, and utility scripts

```
scripts/
├── test-e2e.sh           # E2E test runner
├── waitfor               # Wait for service startup
└── ... (various utilities)
```

## Configuration Files (Root)

```
/
├── .air.toml              # Hot reload config (Air)
├── Caddyfile              # Reverse proxy config
├── Dockerfile             # Container build
├── docker-compose.yml     # Local development setup
├── go.mod / go.sum        # Go dependencies
├── package.json           # Node.js dependencies
├── tsconfig.json          # TypeScript config
├── Makefile               # Build commands
├── CLAUDE.md              # AI development instructions
└── README.md              # Project overview
```

## Key Directories by Function

### Content Processing
```
internal/mdloader/         # Markdown → NoteView
internal/noteloader/       # Note version management
internal/layoutloader/     # Hugo layout integration
assets/ui/reader/          # Frontend markdown renderer
```

### Access Control
```
internal/case/canreadnote/ # Permission checks
db/migrations/*subgraph*   # Schema for access groups
internal/case/admin/*subgraph* # Admin management
assets/ui/admin/subgraph/  # Admin UI
```

### Payment Processing
```
internal/case/createpaymentlink/        # Generate payment URLs
internal/case/processnowpaymentsipn/    # Crypto payments
internal/case/processpatreonwebhook/    # Patreon webhooks
internal/boosty/                        # Boosty integration
```

### Telegram Integration
```
internal/telegram/                      # Bot framework
internal/case/handletgupdate/          # Update handler
internal/case/admin/*tgbot*/           # Bot management
internal/case/cronjob/removeexpiredtgchatmembers/ # Membership cleanup
assets/ui/admin/tgbot/                 # Admin UI
```

### Background Processing
```
internal/jobs/                         # Job queue (goqite)
internal/cronjobs/                     # Cron system
internal/case/backjob/                 # Background job definitions
internal/case/cronjob/                 # Cron job definitions
```

## Generated Files (Don't Edit)

These files are auto-generated and should not be edited manually:

```
internal/db/queries.sql.go             # Generated by sqlc
internal/graph/generated.go            # Generated by gqlgen
internal/graph/model/                  # Generated GraphQL models
internal/router/endpoints_gen.go       # Generated route registry
assets/ui/**/-view.tree/*.d.ts         # Generated $mol types
```

## Development Workflow

1. **Database changes:**
   - Create migration: `make db-new name=description`
   - Add queries to `internal/db/queries.sql`
   - Run: `make db-up && make sqlc`

2. **GraphQL changes:**
   - Update `internal/graph/schema.graphqls`
   - Run: `make gqlgen`
   - Implement resolver in `internal/graph/schema.resolvers.go`
   - Add business logic in `internal/case/*/resolve.go`

3. **Frontend changes:**
   - Create `.view.tree` file for structure
   - Add `.view.ts` for behavior
   - Run: `npm run graphqlgen` for GraphQL types

4. **HTTP endpoints:**
   - Create case in `internal/case/*/endpoint.go`
   - Run: `go generate ./internal/router/...`

5. **Run development server:**
   - Backend: `make air`
   - Frontend: Hot reload via browser

## File Counts

- Go files: ~800+
- UI components: 131
- Database tables: 50+
- GraphQL operations: 100+
- Migrations: 80+
- E2E tests: 20+

## Lines of Code (Estimated)

- Backend (Go): ~50,000 LOC
- Frontend (TypeScript): ~15,000 LOC
- SQL: ~5,000 LOC
- Tests: ~3,000 LOC
- **Total: ~73,000 LOC**
