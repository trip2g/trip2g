# UI Components Documentation

**Generated:** 2025-11-15
**Framework:** $mol
**Total Components:** 131
**Location:** `/assets/ui/`

## Overview

The trip2g UI is built with the $mol framework, using `.view.tree` component definitions. The component structure follows a domain-driven organization with two main areas: **admin** (103 components) and **user** (13 components).

## Component Organization Patterns

### CRUD Pattern
Most admin entities follow this structure:
```
entity/
  ├── catalog/          # List view
  ├── show/             # Detail view
  ├── create/           # Create form
  ├── update/           # Edit form (optional)
  └── button/           # Action buttons (delete, disable, etc.)
```

### $mol Naming Convention
- `catalog` - Collection/list views
- `show` - Detail/display views
- `select` - Selection/picker components
- `button` - Action components

## User-Facing Components (13)

### Authentication & Entry
- `auth/auth` - Authentication flow
- `user/enter/userenter` - User sign-in page
- `user/user` - Main user dashboard
- `user/-/web` - Web layout wrapper

### Content & Reading
- `reader/reader` - Markdown content reader
- `user/search/search` - Content search interface
- `user/favoritenote/favoritenote` - Favorite notes feature

### Subscriptions & Payments
- `user/paywall/paywall` - Paywall component
- `user/paywall/offers/offers` - Available offers display
- `user/paywall/activepurchases/activepurchases` - Active subscriptions
- `user/paywall/conversationprompt/conversationprompt` - Conversion prompts
- `user/paywall/conversationprompt/email/email` - Email capture
- `user/space/space` - User personal space
- `user/space/subscriptions/subscriptions` - Subscription management

## Admin Components (103)

### Dashboard & Overview
- `admin/admin` - Main admin panel
- `admin/dashboard/dashboard` - Admin dashboard
- `admin/catalog/catalog` - Admin entity catalog
- `admin/healthchecks/healthchecks` - System health monitoring

### User Management (9 components)
```
admin/user/
  ├── catalog/            # User list
  ├── show/               # User details
  ├── create/             # Create user
  ├── update/             # Update user
  ├── bans/userbans       # Banned users list
  ├── banuser/banuser     # Ban user form
  ├── button/unban/       # Unban action
  ├── subgraphaccesses/   # User access list
  └── subgraphaccess/     # Access detail
```

### Admin Management (2 components)
```
admin/admin/
  ├── catalog/            # Admin list
  └── show/               # Admin details
```

### Subgraph Management (4 components)
```
admin/subgraph/
  ├── catalog/            # Subgraph list
  ├── show/subgraph       # Subgraph details
  └── select/             # Subgraph selector
      ├── select
      └── list/list
```

### Content Management

**Note Views (6 components)**
```
admin/noteview/
  ├── catalog/noteviews   # Note list
  ├── show/noteview       # Note details
  ├── select/select       # Note picker
  ├── warnings/warnings   # Note warnings display
  └── graph/graph         # Graph visualization
```

**Note Assets (2 components)**
```
admin/noteasset/
  ├── catalog/            # Asset list
  └── show/               # Asset details
```

**Releases (4 components)**
```
admin/release/
  ├── catalog/            # Release list
  ├── show/               # Release details
  ├── create/             # Create release
  └── button/makelive/    # Make release live
```

### Offers & Purchases (6 components)
```
admin/offer/
  ├── catalog/            # Offer list
  ├── show/               # Offer details
  ├── create/             # Create offer
  └── update/             # Update offer

admin/purchase/
  ├── catalog/            # Purchase list
  └── show/               # Purchase details
```

### Telegram Bot Management (15 components)
```
admin/tgbot/
  ├── catalog/            # Bot list
  ├── show/               # Bot details
  │   ├── chats/chats             # Bot chats
  │   │   └── subgraphs/subgraphs # Chat access
  │   ├── invitechats/            # Invite chats
  │   │   └── subgraphs/subgraphs # Invite access
  │   └── publishtags/            # Publishing tags
  │       ├── publishtags
  │       ├── instanttags/instanttags
  │       └── tags/tags
  ├── create/             # Create bot
  └── update/             # Update bot
```

### Telegram Publishing (6 components)
```
admin/telegrampublishnote/
  ├── catalog/            # Publish queue
  ├── show/               # Publish note details
  ├── status/status       # Publish status
  ├── message/message     # Message preview
  ├── button/reset/       # Reset publish
  └── button/send/        # Send now
```

### Patreon Integration (7 components)
```
admin/patreoncredentials/
  ├── catalog/            # Credentials list
  ├── show/               # Credential details
  │   └── subgraphs/subgraphs # Tier mappings
  ├── create/             # Add credentials
  ├── button/delete/      # Delete
  ├── button/restore/     # Restore
  └── button/refresh/     # Sync data
```

### Boosty Integration (7 components)
```
admin/boostycredentials/
  ├── catalog/            # Credentials list
  ├── show/               # Credential details
  │   └── subgraphs/subgraphs # Tier mappings
  ├── create/             # Add credentials
  ├── button/delete/      # Delete
  ├── button/restore/     # Restore
  └── button/refresh/     # Sync data
```

### API Keys (4 components)
```
admin/apikey/
  ├── catalog/            # API key list
  ├── show/               # Key details
  ├── create/             # Generate key
  └── button/disable/     # Disable key
```

### Git Tokens (4 components)
```
admin/gittoken/
  ├── catalog/            # Token list
  ├── show/               # Token details
  ├── create/             # Generate token
  └── button/disable/     # Disable token
```

### Redirects (5 components)
```
admin/redirect/
  ├── catalog/            # Redirect list
  ├── show/               # Redirect details
  ├── create/             # Create redirect
  └── update/             # Update redirect
```

### 404 Tracking (7 components)
```
admin/notfoundpath/
  ├── catalog/            # 404 paths list
  ├── show/               # Path details
  └── button/reset/       # Reset counter

admin/notfoundpattern/
  ├── catalog/            # Ignore patterns list
  ├── show/               # Pattern details
  ├── create/             # Add pattern
  ├── update/             # Update pattern
  └── button/delete/      # Delete pattern
```

### Cron Jobs (5 components)
```
admin/cronjob/
  ├── catalog/            # Job list
  ├── show/               # Job details
  │   └── executions/executions # Execution history
  ├── update/             # Update job config
  └── button/run/         # Manual trigger
```

### Background Queues (5 components)
```
admin/backgroundqueue/
  ├── catalog/            # Queue list
  ├── show/               # Queue details
  ├── button/start/       # Start queue
  ├── button/stop/        # Stop queue
  └── button/clear/       # Clear queue
```

### HTML Injections (5 components)
```
admin/htmlinjection/
  ├── catalog/            # Injection list
  ├── show/               # Injection details
  ├── create/             # Create injection
  ├── update/             # Update injection
  └── button/delete/      # Delete injection
```

### Config Versions (3 components)
```
admin/configversion/
  ├── catalog/            # Config history
  ├── show/               # Config details
  └── create/             # Create version
```

### Wait Lists (2 components)
```
admin/waitlistemailrequest/catalog/     # Email waitlist
admin/waitlisttgbotrequest/catalog/     # Telegram waitlist
```

### Audit Logs (1 component)
```
admin/auditlog/catalog/                 # System audit log
```

## Shared/Utility Components (15)

### Labeler Components (5)
Helper components for consistent field labeling:
```
admin/labeler/
  ├── description/        # Description field label
  ├── email/              # Email field label
  ├── id/                 # ID field label
  ├── moment/             # Timestamp field label
  └── name/               # Name field label
```

### Time Components
```
time/remining/remining              # Time remaining display
table/cell/time/time                # Time table cell
```

### Content Components
```
obsidian/obsidian                   # Obsidian integration
reader/reader                       # Markdown reader
```

### Theme & Layout
```
theme/theme                         # Theme switcher
user/-/web                          # Web layout wrapper
```

## Component Architecture Patterns

### 1. Catalog Pattern
List views with filtering, sorting, and pagination:
- Displays collection of entities in table format
- Provides search/filter controls
- Links to detail views
- Handles loading states

### 2. Show Pattern
Detail views for individual entities:
- Displays all entity fields
- Shows related entities (tabs/sections)
- Provides action buttons
- Handles edit/delete operations

### 3. Form Pattern
Create/update forms with validation:
- Input fields with $mol binding
- GraphQL mutation integration
- Error handling
- Success redirects

### 4. Button Pattern
Reusable action components:
- Confirmation dialogs
- Loading states
- GraphQL mutation calls
- Success/error feedback

### 5. Select Pattern
Entity selection components:
- Searchable dropdown
- Multi-select support
- Lazy loading
- Current selection display

## GraphQL Integration

Components use the `$trip2g_graphql_request` helper for API calls:
```typescript
// In .view.ts files
const result = await this.$trip2g_graphql_request().query({
  query: AllUsersDocument,
  variables: {}
});
```

Code generation via `npm run graphqlgen` creates TypeScript types from GraphQL schema.

## Component File Structure

Each component typically has:
```
componentname/
  ├── componentname.view.tree      # Component structure (required)
  ├── componentname.view.ts        # TypeScript behavior (optional)
  ├── componentname.view.css.ts    # Styles (optional)
  └── -view.tree/                  # Generated TypeScript definitions
      └── componentname.view.tree.d.ts
```

## State Management

$mol framework uses reactive properties:
- `@` prefix for mutable state
- `*` prefix for computed properties
- Automatic dependency tracking
- No explicit state management library needed

## Routing

URL-based routing via $mol:
```tree
$trip2g_user_web $mol_book2
  plugins /
    <= Theme $trip2g_theme
  Menu_page $trip2g_user
  Paywall_page $trip2g_user_paywall
  Search_page $trip2g_user_search
```

## Testing

E2E tests with Playwright in `/e2e/`:
- Page object pattern
- Covers critical user flows
- Admin functionality tests
- See `/e2e/README.md` for details
