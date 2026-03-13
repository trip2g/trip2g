---
title: trip2g Documentation Index
---

> **При добавлении новой документации** — добавь ссылку в этот файл.

[[changelog]] — история изменений

## Overview

trip2g is a publishing platform that transforms Obsidian markdown vaults into websites with paid subscription sections and Telegram channel integration. The platform enables knowledge workers to monetize their content through multiple payment providers while maintaining their Obsidian-based workflow.

## Quick Start

**New Developers:**
1. Read [[development-operations|Development & Operations Guide]] for setup
2. Review [[source-tree|Source Tree]] to understand project structure
3. Check [[architecture|Architecture]] for system design overview

**Existing Team:**
- [[api-contracts|API Contracts]] - GraphQL API reference
- [[data-models|Data Models]] - Database schema
- [[ui-components|UI Components]] - Frontend components

## Generated Documentation

### Core Reference Documentation

**[[api-contracts|API Contracts]]**
- Complete GraphQL API documentation
- 100+ queries and mutations
- Public and Admin API sections
- Request/response examples
- Authentication patterns

**[[data-models|Data Models]]**
- Full database schema (50+ tables)
- Entity relationships
- Access control patterns
- Payment integration models
- Telegram integration schema

**[[ui-components|UI Components]]**
- 131 $mol framework components
- Component organization patterns
- Admin panel catalog (103 components)
- User-facing components (13 components)
- Shared utilities (15 components)

### System Documentation

**[[architecture|Architecture]]**
- High-level system design
- Component interactions
- Data flow diagrams
- Integration patterns
- Scalability considerations
- Technology decisions

**[[source-tree|Source Tree]]**
- Complete directory structure
- Component descriptions
- File organization patterns
- Generated vs. source files
- Development workflow

**[[development-operations|Development & Operations]]**
- Quick start guide
- Build commands
- Testing procedures
- Database migrations
- Code generation
- Deployment processes
- Troubleshooting

## Existing Documentation

### Development Guides

**[[instructions|Instructions]]**
- Common development patterns
- Code style guidelines
- Best practices

**[[frontend|Frontend Guide]]**
- $mol framework usage
- Component patterns
- CRUD interfaces

**[[mol|Mol Framework]]**
- $mol view properties
- view.tree syntax
- Component patterns

**[[TESTING|Testing Guide]]**
- Testing strategies
- E2E test patterns
- Unit test examples

### Feature Documentation

**[[telegram|Telegram Integration]]**
- Bot setup
- Publishing system
- Chat management

**[[obsidian_links|Obsidian Links]]**
- Wikilink processing
- Cross-referencing
- Link resolution

**[[queues|Job Queues]]**
- Background job system
- Queue management
- Job patterns

**[[sqlite|SQLite Usage]]**
- Database patterns
- Performance tips
- Migration strategies

### AI Context

**[[aicontext|AI Context]]**
- AI assistant instructions
- Project-specific context

## Project Statistics

### Codebase Size
- **Backend (Go):** ~50,000 LOC
- **Frontend (TypeScript):** ~15,000 LOC
- **SQL:** ~5,000 LOC
- **Tests:** ~3,000 LOC
- **Total:** ~73,000 LOC

### Component Count
- **Go Packages:** 800+
- **UI Components:** 131
- **Database Tables:** 50+
- **GraphQL Operations:** 100+
- **Database Migrations:** 80+
- **E2E Tests:** 20+

### Technology Stack Summary

| Category | Technology | Purpose |
|----------|-----------|---------|
| **Backend Language** | Go 1.24 | Primary backend |
| **Web Framework** | fasthttp | HTTP server |
| **API** | gqlgen | GraphQL server |
| **Database** | SQLite | Data persistence |
| **SQL Toolkit** | sqlc | Type-safe queries |
| **Frontend Framework** | $mol | UI components |
| **Frontend Language** | TypeScript 5.7 | Type-safe JavaScript |
| **Markdown Parser** | goldmark | Content processing |
| **Job Queues** | goqite, backlite | Background processing |
| **Cron** | robfig/cron | Scheduled tasks |
| **Telegram** | telegram-bot-api | Bot integration |
| **Payments** | NowPayments, Patreon, Boosty | Monetization |
| **Email** | Resend | Transactional email |
| **Storage** | MinIO | Object storage |
| **Search** | Bleve | Full-text search |
| **Logging** | zerolog | Structured logging |
| **Testing** | Playwright | E2E testing |

## Architecture at a Glance

### System Type
**Monolithic Full-Stack Web Application**

### Deployment Model
- Single Go binary
- Embedded SQLite database
- External MinIO for assets
- Reverse proxy (Caddy/Nginx)

### Core Features
1. **Content Management**
   - Markdown-based publishing
   - Versioning system
   - Wikilink support
   - Asset management

2. **Access Control**
   - Subgraph-based permissions
   - Subscription management
   - Time-based expiration

3. **Payment Integration**
   - NowPayments (crypto)
   - Patreon sync
   - Boosty sync
   - Offer management

4. **Telegram Integration**
   - Multi-bot support
   - Scheduled publishing
   - Chat access control
   - Member management

5. **Administration**
   - Full CRUD interfaces
   - User management
   - Content moderation
   - System monitoring

## Development Workflow Quick Reference

### Database Changes
```bash
make db-new name=description    # Create migration
# Edit db/migrations/*.sql
make db-up                      # Apply migration
# Add queries to internal/db/queries.sql
make sqlc                       # Generate Go code
```

### GraphQL Changes
```bash
# Edit internal/graph/schema.graphqls
make gqlgen                     # Generate resolvers
# Implement in internal/case/*/resolve.go
npm run graphqlgen              # Generate frontend types
```

### UI Components
```bash
# Create assets/ui/component/component.view.tree
# Create assets/ui/component/component.view.ts (optional)
# Auto-discovered on reload
```

### Running Tests
```bash
go test ./...                   # Unit tests
npm run test:e2e                # E2E tests
make lint                       # Linting
```

### Running Dev Server
```bash
make air                        # Hot reload
```

## Common Tasks

### Adding a New Feature
1. **Plan** - Define GraphQL schema and database changes
2. **Database** - Create migration, add queries
3. **Backend** - Implement use case in `internal/case/`
4. **API** - Wire up GraphQL resolver
5. **Frontend** - Create UI components
6. **Test** - Write unit and E2E tests

### Debugging
- **Logs:** Check application logs (zerolog output)
- **Database:** Query SQLite directly
- **API:** Use GraphQL Playground
- **Frontend:** Browser dev tools

### Deployment
```bash
make build-amd64                # Build for production
make deploy                     # Deploy via Ansible
```

## Domain Areas

### Content Domain
- `note_paths`, `note_versions`, `note_assets`
- Markdown processing pipeline
- Wikilink resolution
- Asset management

### User Domain
- `users`, `admins`, `user_bans`
- Authentication (email codes, JWT)
- API key management
- User preferences

### Access Control Domain
- `subgraphs`, `user_subgraph_accesses`
- Permission checking
- Expiration management
- Revocation tracking

### Payment Domain
- `offers`, `purchases`, `offer_subgraphs`
- NowPayments integration
- Provider webhooks
- Purchase tracking

### Telegram Domain
- `tg_bots`, `tg_bot_chats`, `tg_chat_members`
- Bot management
- Message publishing
- Chat access control

### Platform Domain
- `releases`, `cron_jobs`, `audit_logs`
- Version management
- System jobs
- Admin actions

## Integration Points

### External Services
- **NowPayments** - Crypto payment processing
- **Patreon** - Creator memberships
- **Boosty** - Russian creator platform
- **Telegram** - Bot API
- **Resend** - Email delivery
- **MinIO** - Object storage (S3-compatible)

### Webhook Endpoints
- `/api/nowpayments/ipn` - Payment notifications
- `/api/patreon/webhook` - Member updates
- `/api/telegram/webhook/{bot_id}` - Bot updates
- `/api/notion/webhook` - Notion integration (experimental)

### API Endpoints
- GraphQL - `/graphql`
- GraphQL Playground - `/playground`
- Git Protocol - `/git/*`
- Health Check - `/health` (if implemented)

## Security Notes

### Authentication Methods
1. **Email + Code** - User sign-in
2. **JWT Tokens** - Session management
3. **API Keys** - Obsidian plugin, programmatic access
4. **Admin Tokens** - Admin operations

### Authorization Layers
1. **Role-based** - Guest, User, Admin
2. **Resource-based** - Subgraph access
3. **Time-based** - Expiration dates
4. **Revocation** - Manual or automatic

### Sensitive Data
- User emails (users.email)
- Payment data (purchases.payment_data)
- API tokens (api_keys.value, tg_bots.token)
- Credentials (patreon_credentials.*, boosty_credentials.*)

## Performance Characteristics

### Optimizations
- SQLite WAL mode (concurrent reads)
- Database indexes on foreign keys
- Connection pooling
- Background job queues
- Asset CDN (via MinIO)

### Bottlenecks
- Single SQLite database
- In-process job queues
- Markdown processing (CPU-bound)
- Large vault syncing

### Scaling Options
- Migrate to PostgreSQL
- External job queue (Redis/RabbitMQ)
- Add caching layer
- Horizontal scaling with load balancer

## Next Steps for New Developers

1. **Setup Environment**
   - Follow [[development-operations|Development & Operations Guide]]
   - Get database running
   - Start development server

2. **Explore Codebase**
   - Read [[source-tree|Source Tree]]
   - Browse key directories
   - Run existing tests

3. **Understand Architecture**
   - Review [[architecture|Architecture]]
   - Study data flows
   - Explore integration patterns

4. **Make First Change**
   - Pick a small task
   - Follow development workflow
   - Submit for review

5. **Deep Dive**
   - Choose a domain area
   - Read relevant documentation
   - Trace code execution

## Getting Help

### Documentation
- Check this index first
- Search existing docs
- Review inline code comments

### Code Examples
- Browse `internal/case/` for patterns
- Check test files for usage examples
- Review existing components

### Resources
- Go documentation: https://go.dev/doc/
- GraphQL: https://graphql.org/
- $mol framework: https://github.com/hyoo-ru/mam_mol
- sqlc: https://sqlc.dev/
- gqlgen: https://gqlgen.com/

## Contributing

### Code Style
- Follow existing patterns
- Use gofmt for Go code
- Write tests for new features
- Document complex logic

### Testing
- Unit tests for business logic
- Integration tests for database operations
- E2E tests for critical user flows

### Documentation
- Update relevant docs when changing features
- Add comments for non-obvious code
- Keep README and CLAUDE.md current

## Maintenance

### Regular Tasks
- Review cron job executions
- Monitor error logs
- Check disk space (SQLite grows)
- Update dependencies
- Review security advisories

### Backup Strategy
- SQLite database (copy file)
- MinIO buckets (S3 sync)
- Configuration files
- Credentials (encrypted)

### Monitoring
- Cron job status (admin panel)
- Background queue depth
- Error rate in logs
- Payment webhook health

---

**Documentation Generated:** 2025-11-15
**Generator:** BMad document-project workflow
**Next Update:** As needed when architecture changes
