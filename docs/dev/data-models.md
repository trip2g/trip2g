# Data Models Documentation

**Generated:** 2025-11-15
**Source:** `/db/schema.sql`
**Database:** SQLite with WAL mode

## Overview

The trip2g database schema consists of 50+ tables organized into 11 major domain areas.

## Core Domain Areas

### 1. Content Management

**note_paths** - Unique note file paths
```sql
- id (PK)
- value (UNIQUE) - note path (e.g., "folder/note.md")
- value_hash (UNIQUE) - hash of path for fast lookups
- latest_content_hash - hash of latest content
- version_count - number of versions
- graph_position_x, graph_position_y - graph visualization coordinates
- hidden_by, hidden_at - soft delete tracking
- created_at
```

**note_versions** - Versioned note content
```sql
- id (PK)
- path_id (FK → note_paths.id)
- version - version number
- content - markdown content
- created_at
- UNIQUE(path_id, version)
```

**note_assets** - Uploaded files (images, attachments)
```sql
- id (PK)
- absolute_path - path in vault
- file_name - original filename
- sha256_hash - content hash for deduplication
- size - file size in bytes
- created_at
- UNIQUE(absolute_path, sha256_hash)
```

**note_version_assets** - Links assets to note versions
```sql
- asset_id (FK → note_assets.id)
- version_id (FK → note_versions.id)
- path - markdown reference path
- PK(asset_id, version_id, path)
```

### 2. User Management

**users** - Platform users
```sql
- id (PK)
- email (UNIQUE, nullable) - for email-based accounts
- tg_user_id (UNIQUE, nullable) - for Telegram-linked accounts
- created_at
- created_via - how user registered (unknown/email/telegram)
- last_signin_code_sent_at - rate limiting
- note_view_count - total note views
```

**admins** - Administrator accounts
```sql
- user_id (PK, FK → users.id CASCADE)
- granted_at
- granted_by (FK → admins.user_id) - audit trail
```

**user_bans** - Banned users
```sql
- user_id (PK, FK → users.id CASCADE)
- created_at
- banned_by (FK → admins.user_id)
- reason
```

**sign_in_codes** - Email authentication codes
```sql
- user_id
- code
- created_at
- IDX on user_id
```

### 3. Access Control (Subgraphs)

**subgraphs** - Content access groups/sections
```sql
- id (PK)
- name (UNIQUE) - display name
- color - UI color coding
- hidden - hide from public listings
- show_unsubgraph_notes_for_paid_users - access control flag
- created_at
```

**user_subgraph_accesses** - User access grants
```sql
- id (PK)
- user_id (FK → users.id CASCADE)
- subgraph_id (FK → subgraphs.id RESTRICT)
- created_at
- expires_at (nullable) - expiration timestamp
- revoke_id (FK → revokes.id) - if revoked
- purchase_id (FK → purchases.id) - if from purchase
- created_by (FK → admins.user_id) - if manual grant
```

**revokes** - Access revocation audit log
```sql
- id (PK)
- target_type - type of revoked resource
- target_id - ID of revoked resource
- created_at
- by_id (FK → admins.user_id)
- reason
```

### 4. Payments & Offers

**offers** - Subscription/purchase offers
```sql
- id (PK)
- public_id (UNIQUE) - UUID for URLs
- created_at
- lifetime - duration string (e.g., "+600 days")
- price_usd - price in USD
- starts_at, ends_at - offer validity window
```

**offer_subgraphs** - Maps offers to subgraphs
```sql
- offer_id (FK → offers.id CASCADE)
- subgraph_id (FK → subgraphs.id RESTRICT)
- PK(offer_id, subgraph_id)
```

**purchases** - Completed purchases
```sql
- id (PK) - external payment ID
- created_at
- payment_provider - "nowpayments", "patreon", "boosty"
- payment_data - JSON provider data
- status - payment status
- offer_id (FK → offers.id)
- user_id (FK → users.id SET NULL) - linked user
- email - purchaser email
- price_usd - actual paid amount
```

### 5. Telegram Bot Integration

**tg_bots** - Registered Telegram bots
```sql
- id (PK)
- token (UNIQUE) - bot API token
- name - bot username
- enabled - active/inactive flag
- description - admin notes
- created_at
- created_by (FK → admins.user_id)
```

**tg_bot_chats** - Telegram chats the bot is in
```sql
- id (PK)
- telegram_id (UNIQUE) - Telegram chat ID
- chat_type - "group", "supergroup", "channel"
- chat_title
- added_at
- removed_at (nullable) - when bot was removed
- can_invite - if bot can invite users
- bot_id - bot reference
- IDX on telegram_id
```

**tg_chat_members** - Chat membership tracking
```sql
- user_id - Telegram user ID
- chat_id - Telegram chat ID
- created_at
- PK(user_id, chat_id)
- IDX on chat_id
```

**tg_user_profiles** - Telegram user profile cache
```sql
- sha256_hash (PK) - hash(chat_id + user_data)
- chat_id
- bot_id (FK → tg_bots.id)
- created_at
- first_name, last_name, username
- IDX on chat_id
```

**tg_user_states** - Bot conversation states
```sql
- chat_id, bot_id (PK)
- user_id (FK → users.id) - linked platform user
- created_at, updated_at
- update_count
- value - state machine state
- data - JSON state data
```

**tg_attach_codes** - Telegram account linking codes
```sql
- user_id (FK → users.id CASCADE)
- bot_id (FK → tg_bots.id)
- code (UNIQUE) - one-time linking code
- created_at
```

**tg_chat_subgraph_accesses** - Chat-based access control
```sql
- id (PK)
- chat_id
- subgraph_id (FK → subgraphs.id)
- created_at
- IDX on chat_id
```

**tg_bot_chat_subgraph_invites** - Pending invites to chats
```sql
- chat_id, subgraph_id (PK)
- created_at
- created_by (FK → admins.user_id)
- IDX on chat_id
```

**tg_bot_chat_subgraph_accesses** - Track user join requests
```sql
- chat_id (FK → tg_bot_chats.id CASCADE)
- user_id (FK → users.id)
- subgraph_id (FK → subgraphs.id)
- created_at - when user requested to join
- joined_at - when user actually joined
- PK(chat_id, user_id, subgraph_id)
```

### 6. Telegram Publishing

**telegram_publish_tags** - Publishing tags/categories
```sql
- id (PK)
- created_at
- hidden - hide from UI
- label (UNIQUE) - tag name
```

**telegram_publish_notes** - Scheduled posts
```sql
- note_path_id (PK, FK → note_paths.id)
- created_at
- publish_at - scheduled time
- published_version_id (FK → note_versions.id) - version that was published
- published_at - actual publish time
- error_count - retry counter
```

**telegram_publish_note_tags** - Maps notes to tags
```sql
- note_path_id (FK → telegram_publish_notes CASCADE)
- tag_id (FK → telegram_publish_tags CASCADE)
- PK(note_path_id, tag_id)
```

**telegram_publish_chats** - Scheduled publishing chats
```sql
- chat_id (FK → tg_bot_chats.id CASCADE)
- tag_id (FK → telegram_publish_tags CASCADE)
- created_at
- created_by (FK → admins.user_id)
```

**telegram_publish_instant_chats** - Instant publishing chats
```sql
- chat_id (FK → tg_bot_chats.id CASCADE)
- tag_id (FK → telegram_publish_tags CASCADE)
- created_at
- created_by (FK → admins.user_id)
```

**telegram_publish_sent_messages** - Published message tracking
```sql
- note_path_id (FK → note_paths.id)
- chat_id (FK → tg_bot_chats.id)
- created_at
- message_id - Telegram message ID
- instant - bool (0=scheduled, 1=instant)
- content_hash - detect content changes
- content - stored message content
- UNIQUE(chat_id, note_path_id) WHERE instant=0
- IDX on chat_id, note_path_id
```

### 7. Patreon Integration

**patreon_credentials** - Patreon API credentials
```sql
- id (PK)
- created_at, created_by (FK → admins.user_id)
- deleted_at, deleted_by (soft delete)
- creator_access_token - OAuth token
- synced_at - last sync timestamp
- webhook_secret - webhook validation
```

**patreon_campaigns** - Creator campaigns
```sql
- id (PK)
- credentials_id (FK → patreon_credentials CASCADE)
- created_at
- missed_at - if no longer found in API
- campaign_id - Patreon campaign ID
- attributes - JSON from API
- UNIQUE(credentials_id, campaign_id)
```

**patreon_tiers** - Membership tiers
```sql
- id (PK)
- campaign_id (FK → patreon_campaigns CASCADE)
- created_at
- missed_at
- tier_id - Patreon tier ID
- title
- amount_cents
- attributes - JSON from API
- UNIQUE(campaign_id, tier_id)
```

**patreon_tier_subgraphs** - Maps tiers to subgraph access
```sql
- tier_id (FK → patreon_tiers CASCADE)
- subgraph_id (FK → subgraphs RESTRICT)
- created_at
- created_by (FK → admins.user_id)
- PK(tier_id, subgraph_id)
```

**patreon_members** - Patron members
```sql
- id (PK)
- patreon_id - UUID from Patreon
- campaign_id (FK → patreon_campaigns CASCADE)
- current_tier_id (FK → patreon_tiers SET NULL)
- status - active/declined/etc
- email
- user_id (FK → users.id) - linked platform user
- UNIQUE(patreon_id, campaign_id)
- IDX on email
```

### 8. Boosty Integration

**boosty_credentials** - Boosty API credentials
```sql
- id (PK)
- created_at, created_by (FK → admins.user_id)
- deleted_at, deleted_by (soft delete)
- auth_data - JSON cookie data
- device_id - client_id from cookie
- blog_name - creator page name
- expires_at - credential expiration
- synced_at - last sync timestamp
```

**boosty_tiers** - Subscription tiers
```sql
- id (PK)
- credentials_id (FK → boosty_credentials)
- boosty_id - Boosty tier ID
- created_at
- missed_at
- name
- data - JSON from API
- UNIQUE(credentials_id, boosty_id)
```

**boosty_tier_subgraphs** - Maps tiers to subgraph access
```sql
- tier_id (FK → boosty_tiers CASCADE)
- subgraph_id (FK → subgraphs RESTRICT)
- created_at
- created_by (FK → admins.user_id)
- PK(tier_id, subgraph_id)
```

**boosty_members** - Subscriber members
```sql
- id (PK)
- credentials_id (FK → boosty_credentials)
- boosty_id - Boosty member ID
- created_at
- missed_at
- email
- status
- data - JSON from API
- current_tier_id (FK → boosty_tiers)
- user_id (FK → users.id) - linked platform user
- UNIQUE(credentials_id, boosty_id)
- IDX on email
```

### 9. System Management

**releases** - Content releases/deployments
```sql
- id (PK)
- created_at
- created_by (FK → admins.user_id)
- title
- home_note_version_id (FK → note_versions.id) - homepage
- is_live - currently active release
- IDX on is_live
```

**release_note_versions** - Notes included in release
```sql
- release_id (FK → releases CASCADE)
- note_version_id (FK → note_versions CASCADE)
- PK(release_id, note_version_id)
```

**api_keys** - API keys for programmatic access
```sql
- id (PK)
- value (UNIQUE) - API key string
- created_at
- created_by (FK → admins.user_id CASCADE)
- disabled_at, disabled_by (soft delete)
- description
```

**api_key_logs** - API key usage tracking
```sql
- api_key_id (FK → api_keys CASCADE)
- created_at
- action_id (FK → api_key_log_actions)
- ip_id (FK → api_key_log_ips)
```

**api_key_log_actions** - Normalized action names
```sql
- id (PK)
- name (UNIQUE) - action name
```

**api_key_log_ips** - Normalized IP addresses
```sql
- id (PK)
- created_at
- value (UNIQUE) - IP address
```

**git_tokens** - Git protocol authentication
```sql
- id (PK)
- created_at
- last_used_at
- admin_id (FK → admins.user_id)
- value_sha256 (UNIQUE) - hashed token
- description
- can_pull, can_push - permissions
- usage_count
- disabled_at, disabled_by (soft delete)
```

**redirects** - URL redirect rules
```sql
- id (PK)
- created_at
- created_by (FK → admins.user_id)
- pattern - match pattern
- ignore_case, is_regex - pattern flags
- target - redirect destination
```

**not_found_paths** - 404 tracking
```sql
- id (PK)
- path (UNIQUE)
- total_hits
- last_hit_at
```

**not_found_ip_hits** - 404 IP tracking
```sql
- ip (PK)
- total_hits
- last_hit_at
```

**not_found_ignored_patterns** - 404 ignore rules
```sql
- id (PK)
- pattern (UNIQUE)
- created_at
- created_by (FK → admins.user_id)
```

**html_injections** - Custom HTML insertions
```sql
- id (PK)
- created_at
- active_from, active_to - time window
- description
- position - sort order
- placement - "head" or "body_end"
- content - HTML to inject
```

**config_versions** - Site configuration snapshots
```sql
- id (PK)
- created_at
- created_by (FK → admins.user_id)
- show_draft_versions - visibility flag
- default_layout - layout template name
- timezone - site timezone
- robots_txt - "open" or "restricted"
```

**cron_jobs** - Scheduled job definitions
```sql
- id (PK)
- name (UNIQUE) - job identifier
- enabled - active/inactive
- expression - cron expression (6-field with seconds)
- last_exec_at
```

**cron_job_executions** - Job execution history
```sql
- id (PK)
- job_id (FK → cron_jobs CASCADE)
- started_at
- finished_at
- status - 0=pending, 1=running, 2=completed, 3=failed
- report_data - JSON execution report
- error_message
```

**audit_logs** - Administrative action log
```sql
- id (PK)
- created_at
- level - 0=debug, 1=info, 2=warning, 3=error
- message
- params - JSON event data
- IDX on created_at
```

### 10. Job Queues

**goqite** - SQLite-based job queue (goqite library)
```sql
- id (PK) - message ID
- created, updated - timestamps
- queue - queue name
- body - job payload
- timeout - visibility timeout
- received - receive count
- priority - priority value
- IDX on (queue, priority DESC, created)
```

**backlite_tasks** - Task queue (backlite library)
```sql
- id (PK)
- created_at
- queue
- task - serialized task
- wait_until - delayed execution
- claimed_at - worker claim timestamp
- last_executed_at
- attempts
- IDX on wait_until WHERE NOT NULL
```

**backlite_tasks_completed** - Completed task archive
```sql
- id (PK)
- created_at, last_executed_at
- queue
- attempts
- last_duration_micro
- succeeded - boolean
- task, error - optional fields
- expires_at - cleanup timestamp
```

### 11. Analytics & Tracking

**user_note_views** - Individual note view events
```sql
- user_id (FK → users.id CASCADE)
- version_id (FK → note_versions.id CASCADE)
- referer_version_id (FK → note_versions.id) - navigation source
- created_at
```

**user_note_daily_view_counts** - Aggregated daily views
```sql
- user_id (FK → users.id CASCADE)
- path_id (FK → note_paths.id CASCADE)
- day - date
- count - view count for that day
- UNIQUE(user_id, path_id)
```

**user_favorite_notes** - User-favorited notes
```sql
- user_id (FK → users.id CASCADE)
- note_version_id (FK → note_versions.id)
- created_at
- PK(user_id, note_version_id)
```

**wait_list_email_requests** - Email waitlist signups
```sql
- email (PK)
- created_at
- note_path_id (FK → note_paths.id) - which page
- ip - requester IP
```

**wait_list_tg_bot_requests** - Telegram waitlist signups
```sql
- bot_id (FK → tg_bots.id)
- chat_id - Telegram chat ID
- created_at
- note_path_id (FK → note_paths.id)
- PK(bot_id, chat_id)
```

### 12. Notion Integration (Experimental)

**notion_integrations** - Notion API integration
```sql
- id (PK)
- created_at
- created_by (FK → admins.user_id)
- enabled
- secret_token
- verification_token
- base_path - content mount point
```

## Key Relationships

### Content Versioning Flow
```
note_paths (1) → (N) note_versions → (N) note_version_assets → (N) note_assets
```

### Access Control Flow
```
users → user_subgraph_accesses → subgraphs
purchases → offer_subgraphs → subgraphs
patreon_members → patreon_tiers → patreon_tier_subgraphs → subgraphs
boosty_members → boosty_tiers → boosty_tier_subgraphs → subgraphs
```

### Telegram Publishing Flow
```
telegram_publish_notes → note_paths → note_versions
telegram_publish_note_tags → telegram_publish_tags → telegram_publish_chats → tg_bot_chats
```

### External Payment Integration Flow
```
Patreon: patreon_credentials → patreon_campaigns → patreon_tiers ← patreon_members
Boosty: boosty_credentials → boosty_tiers ← boosty_members
Both: members.email matched to users.email → user_subgraph_accesses created
```

## Database Patterns

### Soft Deletes
- `patreon_credentials.deleted_at/deleted_by`
- `boosty_credentials.deleted_at/deleted_by`
- `api_keys.disabled_at/disabled_by`
- `git_tokens.disabled_at/disabled_by`

### Audit Trail
- Most admin actions tracked via `admins.granted_by`, `created_by` fields
- `audit_logs` for system events
- `api_key_logs` for API usage

### Normalized Reference Tables
- `api_key_log_actions` - action names
- `api_key_log_ips` - IP addresses

### Content Hashing
- `note_paths.value_hash` - fast path lookups
- `note_paths.latest_content_hash` - change detection
- `note_assets.sha256_hash` - deduplication
- `telegram_publish_sent_messages.content_hash` - update detection

### Time-based Access
- `offers.starts_at/ends_at` - offer validity window
- `user_subgraph_accesses.expires_at` - subscription expiration
- `html_injections.active_from/active_to` - injection scheduling

## Indexes

Notable indexes for query performance:
- `idx_releases_is_live` - fast live release lookup
- `idx_tg_bot_chats_telegram_id` - Telegram ID lookups
- `idx_audit_logs_created_at` - log queries
- `goqite_queue_priority_created_idx` - job queue processing
- `backlite_tasks_wait_until` - delayed job execution
- Multiple foreign key indexes for join performance

## Schema Migration

Managed via **dbmate**:
- Migration files in `/db/migrations/`
- 80+ applied migrations tracked in `schema_migrations`
- Follows `YYYYMMDDHHMMSS_description.sql` naming
