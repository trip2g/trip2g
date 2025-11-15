# System Architecture Documentation

**Generated:** 2025-11-15
**Project:** trip2g
**Type:** Full-Stack Web Application (Monolith)

## Architecture Overview

trip2g is a publishing platform built as a **monolithic full-stack application** that transforms Obsidian markdown vaults into websites with subscription-based access control and Telegram channel integration.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
├──────────────────────┬──────────────────────────────────────┤
│   Web Browser        │        Obsidian Plugin              │
│   ($mol Components)  │    (API Key Auth)                   │
└──────────────────────┴──────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     HTTP/GraphQL Layer                       │
│  ┌───────────┐  ┌────────────┐  ┌────────────────────────┐ │
│  │ fasthttp  │  │  GraphQL   │  │   Webhooks             │ │
│  │ Router    │─▶│   API      │  │   - NowPayments        │ │
│  │           │  │  (gqlgen)  │  │   - Patreon            │ │
│  └───────────┘  └────────────┘  │   - Telegram           │ │
│                                  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Business Logic Layer                    │
│  ┌────────────────────────────────────────────────────────┐ │
│  │           Use Cases (internal/case/*)                   │ │
│  │  - GraphQL Resolvers (admin & public mutations)        │ │
│  │  - HTTP Endpoints (webhooks, git protocol)             │ │
│  │  - Background Jobs (payment processing, sync)          │ │
│  │  - Cron Jobs (cleanup, member removal)                 │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                  ┌───────────┴───────────┐
                  ▼                       ▼
┌──────────────────────────┐   ┌──────────────────────────┐
│    Data Access Layer     │   │   Integration Layer      │
│  ┌────────────────────┐  │   │  ┌───────────────────┐  │
│  │ SQLite (WAL mode) │  │   │  │  Telegram Bots    │  │
│  │  - sqlc queries   │  │   │  │  - Bot API        │  │
│  │  - Migrations     │  │   │  │  - Publishing     │  │
│  └────────────────────┘  │   │  └───────────────────┘  │
│  ┌────────────────────┐  │   │  ┌───────────────────┐  │
│  │  Job Queues       │  │   │  │  Payment Providers│  │
│  │  - goqite         │  │   │  │  - NowPayments    │  │
│  │  - backlite       │  │   │  │  - Patreon        │  │
│  └────────────────────┘  │   │  │  - Boosty         │  │
│                          │   │  └───────────────────┘  │
│                          │   │  ┌───────────────────┐  │
│                          │   │  │  Object Storage   │  │
│                          │   │  │  - MinIO (S3)     │  │
│                          │   │  └───────────────────┘  │
└──────────────────────────┘   └──────────────────────────┘
```

## Core Components

### 1. HTTP/GraphQL Layer

**Technology:** fasthttp + gqlgen

**Responsibilities:**
- HTTP request routing
- GraphQL query/mutation handling
- WebSocket support (if used)
- Authentication middleware
- Request context management

**Key Files:**
- `cmd/server/main.go` - Server initialization
- `internal/router/router.go` - Route registration
- `internal/graph/schema.graphqls` - GraphQL schema
- `internal/graph/schema.resolvers.go` - Resolver implementations

**Authentication Flow:**
```
User Request → Router
           ↓
    Extract Auth Header (Bearer token or X-Api-Key)
           ↓
    Validate Token (hotauthtoken package)
           ↓
    Set User Context
           ↓
    Pass to Resolver/Handler
```

### 2. Business Logic Layer (Use Cases)

**Pattern:** Single-responsibility use cases

**Structure:**
```
internal/case/
├── {usecasename}/
│   ├── resolve.go         # Business logic
│   ├── resolve_test.go    # Unit tests
│   └── endpoint.go        # HTTP endpoint (if needed)
```

**Interface Pattern:**
```go
type Env interface {
    // Only methods THIS use case needs
    GetUser(ctx context.Context, id int64) (*db.User, error)
    UpdateUser(ctx context.Context, params db.UpdateUserParams) error
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
    // 1. Validate input
    // 2. Check permissions
    // 3. Execute business logic
    // 4. Return result or error
}
```

**Benefits:**
- Testable (mock Env interface)
- Isolated (no global state)
- Explicit dependencies
- Single responsibility

### 3. Data Access Layer

**Technology:** SQLite + sqlc

**Why SQLite:**
- Embedded (no separate database server)
- ACID compliance
- WAL mode for concurrent reads/writes
- Sufficient for current scale
- Simple deployment

**sqlc Pattern:**
```sql
-- internal/db/queries.sql

-- name: GetUser :one
select * from users where id = ?;

-- name: CreateUser :one
insert into users (email, created_at)
values (?, datetime('now'))
returning *;
```

**Generated Code:**
```go
// internal/db/queries.sql.go (auto-generated)
func (q *Queries) GetUser(ctx context.Context, id int64) (User, error)
func (q *Queries) CreateUser(ctx context.Context, email string) (User, error)
```

**Benefits:**
- Type-safe SQL
- Compile-time query validation
- No ORM overhead
- Direct SQL control

### 4. Content Processing Pipeline

**Markdown → HTML Flow:**

```
Obsidian Vault (.md files)
         │
         ▼
   API Key Upload (pushNotes mutation)
         │
         ▼
   Parse Markdown (goldmark)
         │
    ┌────┴────┐
    ▼         ▼
Frontmatter  Content
(meta tags)  (with wikilinks)
         │
         ▼
   Process Wikilinks
   - Convert [[Note]] → /note
   - Resolve paths
   - Track backlinks
         │
         ▼
   Process Enclaves
   - Extract protected sections
   - Map to subgraphs
         │
         ▼
   Generate HTML
   - Goldmark rendering
   - Syntax highlighting
   - TOC generation
         │
         ▼
   Store NoteVersion
   - content (markdown)
   - html (rendered)
   - metadata
         │
         ▼
   Extract Assets
   - Images
   - Attachments
   - Upload to MinIO
         │
         ▼
   Ready for Publishing
```

**Key Components:**
- `internal/mdloader/` - Markdown parsing
- `internal/noteloader/` - Version management
- `internal/layoutloader/` - Hugo layout integration
- `goldmark` + extensions - Markdown → HTML

### 5. Access Control System

**Subgraph-Based Access:**

```
User
  ├─ user_subgraph_accesses
  │    ├─ subgraph_id
  │    ├─ expires_at
  │    └─ purchase_id / created_by
  │
  └─ Can Access Notes with matching subgraph

Note Version
  └─ Belongs to Subgraphs (via frontmatter)
       └─ free: boolean (override)
```

**Access Check Flow:**
```
1. User requests note
2. Check if note is free → Allow
3. Check user's active subgraph accesses
4. Match note's subgraphs with user's access
5. Allow if ANY match, deny otherwise
```

**Grant Sources:**
- Direct purchase (via offer)
- Manual admin grant
- Patreon tier sync
- Boosty tier sync
- Telegram chat membership

### 6. Payment Integration Architecture

**Multi-Provider Strategy:**

```
                    ┌─────────────┐
                    │   Offer     │
                    │  $X for     │
                    │  Subgraphs  │
                    └──────┬──────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
    ┌─────────┐    ┌──────────┐    ┌──────────┐
    │ Crypto  │    │ Patreon  │    │  Boosty  │
    │NowPay't │    │  Tiers   │    │  Tiers   │
    └────┬────┘    └────┬─────┘    └────┬─────┘
         │              │                │
         ▼              ▼                ▼
    Purchase      Member Sync       Member Sync
    Webhook       Webhook/Cron      Cron Job
         │              │                │
         └──────────────┼────────────────┘
                        ▼
             Create user_subgraph_access
                  (with expiration)
```

**Providers:**

1. **NowPayments (Crypto)**
   - IPN webhook
   - Purchase record created
   - Access granted immediately

2. **Patreon**
   - OAuth2 credentials
   - Webhook for real-time updates
   - Cron job for periodic sync
   - Tier → Subgraph mapping

3. **Boosty**
   - Cookie-based API
   - Cron job for sync (no webhooks)
   - Tier → Subgraph mapping

**Sync Process:**
```
1. Fetch members from provider API
2. Match email to platform users
3. Determine tier → subgraph access
4. Create/update user_subgraph_access
5. Set expiration based on tier status
6. Mark old accesses as revoked
```

### 7. Telegram Integration Architecture

**Multi-Bot Support:**

```
┌──────────────────────────────────────────────┐
│          Telegram Bot Framework               │
├───────────┬─────────────┬────────────────────┤
│  Bot 1    │   Bot 2     │    Bot N           │
│  (Chat A) │  (Chat B)   │   (Chat C)         │
└─────┬─────┴──────┬──────┴──────┬─────────────┘
      │            │             │
      ▼            ▼             ▼
┌─────────────────────────────────────────────┐
│         Update Handler (handletgupdate)     │
│  ┌────────────┐  ┌───────────────────────┐ │
│  │  Commands  │  │   Events               │ │
│  │  /start    │  │   new_chat_members     │ │
│  │  /attach   │  │   left_chat_member     │ │
│  │  /help     │  │   message              │ │
│  └────────────┘  └───────────────────────┘ │
└─────────────────────────────────────────────┘
```

**Publishing Flow:**

```
Admin creates telegram_publish_note
         │
         ├─ publish_at (scheduled time)
         ├─ tags (categories)
         └─ note_path_id
         │
         ▼
  Cron job checks for ready notes
         │
         ▼
  Convert note to Telegram format
  - MarkdownV2 conversion
  - Wikilink → URL
  - Image handling
         │
         ▼
  Send to all chats with matching tags
  - Scheduled tags → scheduled posts
  - Instant tags → immediate posts
         │
         ▼
  Track sent messages
  - telegram_publish_sent_messages
  - content_hash for updates
         │
         ▼
  Mark as published or retry on error
```

**Chat Access Control:**

```
User clicks "Join Chat" on website
         │
         ▼
  Create tg_bot_chat_subgraph_access
  (joined_at = null)
         │
         ▼
  Generate invite link
         │
         ▼
  User joins Telegram chat
         │
         ▼
  Bot receives new_chat_members event
         │
         ▼
  Update joined_at timestamp
         │
         ▼
  Cron job monitors for expiration
         │
         ▼
  Remove user from chat when access expires
```

### 8. Background Job Architecture

**Two Queue Systems:**

**1. goqite (Priority Queue)**
- SQLite-based
- Priority support
- Retry logic
- Used for: Payment processing, email sending

**2. backlite (Task Queue)**
- SQLite-based
- Task scheduling
- Completion tracking
- Used for: Heavy processing, cleanup

**Job Pattern:**
```go
type Job struct {
    Params JobParams
}

func (j *Job) Handle(ctx context.Context, env Env) error {
    // Execute job
    return nil
}

// Enqueue
jobs.Enqueue(ctx, &MyJob{Params: ...})
```

### 9. Cron Job System

**Architecture:**
```
┌──────────────────────────────────────┐
│       Cron Manager (robfig/cron)     │
├──────────────────────────────────────┤
│  Job Registry                        │
│  ├─ remove_expired_tg_chat_members   │
│  │   Schedule: 0 0 * * * *  (hourly) │
│  ├─ clear_cronjob_execution_history  │
│  │   Schedule: 0 0 0 * * *  (daily)  │
│  └─ sync_patreon_members             │
│      Schedule: 0 */30 * * * * (30m)  │
└──────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────┐
│    cron_jobs (database table)        │
│    - name, enabled, expression       │
│    - Can be toggled via admin panel  │
└──────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────┐
│  cron_job_executions (history)       │
│  - job_id, started_at, finished_at   │
│  - status, error_message, report     │
└──────────────────────────────────────┘
```

**Job Lifecycle:**
1. Server starts → Register jobs
2. Cron manager schedules jobs
3. Job executes → Create execution record
4. Update status (pending → running → completed/failed)
5. Store result or error

## Data Flow Patterns

### 1. Content Publishing Flow

```
Obsidian → API Key Upload → Version Storage → Release → Public Access
```

**Steps:**
1. Editor creates/updates markdown in Obsidian
2. Plugin detects changes
3. `pushNotes` mutation uploads content
4. Server creates new note_version
5. Extracts assets, uploads to MinIO
6. Admin creates release
7. Selected versions become live
8. Users can access published content

### 2. Subscription Access Flow

```
User Purchase → Access Grant → Content Unlock
```

**Steps:**
1. User selects offer
2. Payment provider processes payment
3. Webhook/sync creates purchase record
4. System creates user_subgraph_access
5. User can now access matching subgraph content
6. Access expires based on purchase lifetime

### 3. Telegram Publishing Flow

```
Schedule Post → Cron Check → Convert → Send → Track
```

**Steps:**
1. Admin schedules note for publishing
2. Assigns tags (determines target chats)
3. Cron job checks for ready notes
4. Converts markdown to Telegram format
5. Sends to all chats with matching tags
6. Records sent messages for update tracking

## Security Architecture

### Authentication

**JWT Tokens (User Auth):**
- Email-based sign-in codes
- Short-lived tokens
- Stored in hotauthtoken

**API Keys (Programmatic Access):**
- Long-lived keys for Obsidian plugin
- Logged in api_key_logs
- Can be disabled by admin

**Admin Authorization:**
- Checked at start of every admin mutation
- `CurrentAdminUserToken(ctx)` pattern

### Authorization

**Layered Checks:**
1. **Authentication** - Who are you?
2. **Role Check** - Are you admin/user/guest?
3. **Resource Check** - Can you access this note/subgraph?
4. **Action Check** - Can you perform this operation?

**Example Flow:**
```go
// Admin mutation
token, err := env.CurrentAdminUserToken(ctx)
if err != nil {
    return nil, err  // Not authenticated as admin
}

// Resource check
canAccess := checkUserSubgraphAccess(userID, subgraphID)
if !canAccess {
    return ErrorPayload{Message: "Access denied"}
}
```

### Input Validation

**Framework:** ozzo-validation

**Pattern:**
```go
func validateRequest(r *Input) *model.ErrorPayload {
    return model.NewOzzoError(ozzo.ValidateStruct(r,
        ozzo.Field(&r.Email, validation.Required, is.Email),
        ozzo.Field(&r.Amount, validation.Min(0)),
    ))
}
```

**Validation happens:**
- At GraphQL input layer
- In business logic (Resolve functions)
- Before database operations

## Scalability Considerations

### Current Architecture (Monolith)

**Strengths:**
- Simple deployment
- Low latency (no network calls between services)
- Easy to reason about
- Suitable for current scale

**Bottlenecks:**
- Single SQLite database
- In-process job queues
- No horizontal scaling

### Future Migration Path

**If scale demands:**

1. **Database:** SQLite → PostgreSQL
   - sqlc supports PostgreSQL
   - Minimal code changes needed

2. **Job Queues:** goqite → Redis/RabbitMQ
   - Replace in-process queues
   - Separate worker processes

3. **File Storage:** Already using MinIO (S3-compatible)
   - Ready for cloud object storage

4. **Caching:** Add Redis for:
   - Session storage
   - Frequently accessed notes
   - Rendered HTML cache

5. **Microservices (if needed):**
   - Content Service (markdown processing)
   - Auth Service (user management)
   - Payment Service (payment processing)
   - Telegram Service (bot handling)

### Performance Optimizations

**Currently Implemented:**
- SQLite WAL mode (concurrent reads)
- Connection pooling
- Background job processing
- Index optimization

**Potential Additions:**
- CDN for static assets
- Edge caching for public content
- Database query caching
- GraphQL query complexity limits
- Rate limiting on expensive operations

## Deployment Architecture

**Single Server:**
```
┌─────────────────────────────────────┐
│         trip2g Binary               │
│  ┌──────────────────────────────┐   │
│  │  HTTP Server (fasthttp)      │   │
│  ├──────────────────────────────┤   │
│  │  GraphQL API                 │   │
│  ├──────────────────────────────┤   │
│  │  Background Workers          │   │
│  ├──────────────────────────────┤   │
│  │  Cron Jobs                   │   │
│  └──────────────────────────────┘   │
├─────────────────────────────────────┤
│         SQLite Database              │
│         (data.sqlite3)               │
└─────────────────────────────────────┘
```

**With Reverse Proxy:**
```
Internet → Caddy/Nginx → trip2g Binary
                      ↓
                  SQLite DB
                  MinIO Storage
```

## Key Design Patterns

1. **Repository Pattern** - sqlc-generated queries
2. **Use Case Pattern** - Single-responsibility business logic
3. **Dependency Injection** - Env interface pattern
4. **CQRS Light** - Read vs. write separation in queries
5. **Event Sourcing (Partial)** - Note versioning, audit logs
6. **Webhook Pattern** - Payment/Patreon integrations
7. **Queue Pattern** - Background job processing
8. **Strategy Pattern** - Multiple payment providers

## Technology Decisions

### Why Go?
- Performance
- Strong typing
- Excellent stdlib
- Good concurrency support
- Fast compilation
- Single binary deployment

### Why SQLite?
- Zero configuration
- ACID compliance
- Fast for read-heavy workloads
- Embedded (no separate server)
- Simple backups (copy file)

### Why GraphQL?
- Flexible queries
- Type safety
- Self-documenting API
- Efficient data fetching
- Good tooling ecosystem

### Why $mol Framework?
- Reactive by design
- No virtual DOM overhead
- Component-based
- TypeScript support
- Small bundle size

### Why Monolith?
- Faster development
- Simpler deployment
- Sufficient for current scale
- Easy debugging
- Low operational complexity
