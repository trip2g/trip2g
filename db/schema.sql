CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
CREATE TABLE note_paths (
  id integer primary key,
  value text not null unique on conflict ignore,
  value_hash text not null unique on conflict fail,
  latest_content_hash text not null,
  created_at datetime not null default current_timestamp,
  version_count integer not null default 0
, graph_position_x real, graph_position_y real, hidden_by integer references admins(user_id) on delete restrict, hidden_at datetime);
CREATE TABLE admins (
  user_id int primary key references users(id) on delete cascade,
  granted_at datetime not null default current_timestamp,
  granted_by int references admins(user_id)
);
CREATE TABLE sign_in_codes (
  user_id integer not null,
  code text not null,
  created_at datetime not null default current_timestamp
);
CREATE INDEX idx_sign_in_codes_user_id on sign_in_codes(user_id);
CREATE TABLE backlite_tasks (
    id text PRIMARY KEY,
    created_at integer NOT NULL,
    queue text NOT NULL,
    task blob NOT NULL,
    wait_until integer,
    claimed_at integer,
    last_executed_at integer,
    attempts integer NOT NULL DEFAULT 0
) STRICT;
CREATE TABLE backlite_tasks_completed (
    id text PRIMARY KEY NOT NULL,
    created_at integer NOT NULL,
    queue text NOT NULL,
    last_executed_at integer,
    attempts integer NOT NULL,
    last_duration_micro integer,
    succeeded integer,
    task blob,
    expires_at integer,
    error text
) STRICT;
CREATE INDEX backlite_tasks_wait_until ON backlite_tasks (wait_until) WHERE wait_until IS NOT NULL;
CREATE TABLE subgraphs (
  id integer primary key autoincrement,
  name text not null unique,
  color text,
  created_at datetime not null default current_timestamp
, hidden boolean not null default false, show_unsubgraph_notes_for_paid_users boolean default true, require_signin boolean not null default false);
CREATE TABLE revokes (
  id integer primary key autoincrement,
  target_type text not null,
  target_id integer not null,
  created_at datetime not null default current_timestamp,
  by_id integer not null references admins(user_id) on delete restrict,
  reason text
);
CREATE TABLE user_bans (
  user_id integer primary key references users(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  banned_by integer references admins(user_id) on delete restrict,
  reason text not null
);
CREATE TABLE user_note_daily_view_counts (
  user_id int not null references users(id) on delete cascade,
  path_id int not null references note_paths(id) on delete cascade,
  day date not null default (date()),
  count int not null default 0,
  unique (user_id, path_id)
);
CREATE TABLE offers (
  id integer primary key autoincrement,
  public_id text not null unique,
  created_at datetime not null default current_timestamp,
  lifetime text, -- e.g. "+600 days", null means no expiration
  price_usd numeric,
  starts_at datetime,
  ends_at datetime
);
CREATE TABLE offer_subgraphs (
  offer_id integer not null references offers(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  primary key (offer_id, subgraph_id)
);
CREATE TABLE IF NOT EXISTS "note_versions" (
  id integer primary key autoincrement,
  path_id integer not null,
  version integer not null,
  content text not null,
  created_at datetime not null default current_timestamp, created_by_user_id integer references users(id) on delete set null, created_by_api_key_id integer references api_keys(id) on delete set null, created_by_client text,
  unique(path_id, version),
  foreign key (path_id) references note_paths(id) on delete restrict
);
CREATE TABLE note_version_assets (
  asset_id integer not null references note_assets(id) on delete cascade,
  version_id integer not null references note_versions(id) on delete cascade,
  path text not null, -- path in the note for replacement
  created_at datetime not null default current_timestamp,
  primary key (asset_id, version_id, path)
);
CREATE TABLE acme_certs (
  key text primary key,
  value blob,
  updated_at datetime default current_timestamp
);
CREATE TABLE api_keys (
  id integer primary key autoincrement,
  value text not null unique,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete cascade,
  disabled_at datetime,
  disabled_by integer references admins(user_id) on delete restrict,
  description text not null default '' -- the form field always has a value
, skip_webhooks boolean not null default false, enable_mcp_admin_tools boolean);
CREATE TABLE api_key_log_actions (
  id integer primary key autoincrement,
  name text not null unique
);
CREATE TABLE api_key_log_ips (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  value text not null unique
);
CREATE TABLE api_key_logs (
  api_key_id integer not null references api_keys(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  action_id integer not null references api_key_log_actions(id) on delete restrict,
  ip_id integer not null references api_key_log_ips(id) on delete restrict
);
CREATE TABLE releases (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  title text not null default '',
  home_note_version_id integer references note_versions(id) on delete restrict,
  is_live boolean not null default false
);
CREATE INDEX idx_releases_is_live on releases(is_live);
CREATE TABLE release_note_versions (
  release_id integer not null references releases(id) on delete cascade,
  note_version_id integer not null references note_versions(id) on delete cascade,
  primary key (release_id, note_version_id)
);
CREATE TABLE IF NOT EXISTS "user_note_views" (
  user_id int not null references users(id) on delete cascade,
  version_id integer not null references note_versions(id) on delete cascade,
  referer_version_id integer references note_versions(id) on delete cascade,
  created_at datetime not null default current_timestamp
);
CREATE TABLE IF NOT EXISTS "purchases" (
  id text primary key,
  created_at datetime not null default current_timestamp,
  payment_provider text not null,
  payment_data text not null,
  status text not null,
  offer_id integer not null references offers(id) on delete restrict,
  user_id integer references users(id) on delete set null,
  email text not null,
  price_usd numeric not null
);
CREATE TABLE redirects (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  pattern text not null,
  ignore_case boolean not null default true,
  is_regex boolean not null default false,
  target text not null
);
CREATE TABLE not_found_paths (
  id integer primary key autoincrement,
  path text not null unique,
  total_hits integer not null default 1,
  last_hit_at datetime not null default current_timestamp
);
CREATE TABLE not_found_ip_hits (
  ip text primary key,
  total_hits integer not null default 1,
  last_hit_at datetime not null default current_timestamp
);
CREATE TABLE not_found_ignored_patterns (
  id integer primary key autoincrement,
  pattern text not null unique,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);
CREATE TABLE tg_bots (
  id integer not null primary key autoincrement,
  token text not null unique,
  enabled boolean not null default true,
  description text not null default '',
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
, name text not null default '', default_canvas text not null default '', default_handler text not null default '');
CREATE TABLE tg_user_states (
  chat_id int not null,
  bot_id int not null references tg_bots(id) on delete restrict,
  user_id int references users(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  updated_at datetime not null default current_timestamp,
  update_count int not null default 0,
  value text not null default 'pending',
  data text not null,
  primary key (chat_id, bot_id)
);
CREATE TABLE tg_user_profiles (
  sha256_hash text primary key,
  chat_id int not null,
  bot_id int not null references tg_bots(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  first_name text,
  last_name text,
  username text
);
CREATE INDEX tg_user_profiles_chat_id_idx on tg_user_profiles(chat_id);
CREATE TABLE IF NOT EXISTS "users" (
    id integer primary key,
    email text unique, -- nullable but unique for linked accounts
    created_at datetime not null default current_timestamp,
    last_signin_code_sent_at datetime,
    note_view_count integer default 0,
    tg_user_id integer unique -- Also unique - one account per Telegram user
    -- Note: No FK constraint because tg_user_profiles.chat_id is not unique
, created_via text not null default 'unknown');
CREATE TABLE wait_list_email_requests (
  email text primary key,
  created_at datetime not null default current_timestamp,
  note_path_id int not null references note_paths(id) on delete restrict,
  ip text
);
CREATE TABLE wait_list_tg_bot_requests (
  bot_id int not null references tg_bots(id) on delete restrict,
  chat_id int not null,
  created_at datetime not null default current_timestamp,
  note_path_id int not null references note_paths(id) on delete restrict,
  primary key (bot_id, chat_id)
);
CREATE TABLE patreon_credentials (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  deleted_at datetime,
  deleted_by integer references admins(user_id) on delete restrict,
  creator_access_token text not null
, synced_at datetime, webhook_secret text);
CREATE TABLE patreon_campaigns (
  id integer primary key autoincrement,
  credentials_id integer not null references patreon_credentials(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  missed_at datetime,
  campaign_id text not null,
  attributes text not null,
  unique(credentials_id, campaign_id)
);
CREATE TABLE patreon_members (
  id integer primary key autoincrement,
  patreon_id text not null, -- uuid
  campaign_id integer not null references patreon_campaigns(id) on delete cascade,
  current_tier_id integer references patreon_tiers(id) on delete set null,
  status text not null,
  email text not null
, user_id integer references users(id) on delete restrict);
CREATE TABLE patreon_tiers (
  id integer primary key autoincrement,
  campaign_id integer not null references patreon_campaigns(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  missed_at datetime,
  tier_id text not null,
  title text not null,
  amount_cents integer not null,
  attributes text not null,
  unique(campaign_id, tier_id)
);
CREATE UNIQUE INDEX unique_patreon_member on patreon_members(patreon_id, campaign_id);
CREATE TABLE patreon_tier_subgraphs (
  tier_id integer not null references patreon_tiers(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,

  primary key (tier_id, subgraph_id)
);
CREATE TABLE boosty_credentials (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  deleted_at datetime,
  deleted_by integer references admins(user_id) on delete restrict,
  auth_data text not null, -- json from the site cookie
  device_id text not null, -- client_id from the site cookie
  blog_name text not null -- the user page name
, expires_at datetime, synced_at datetime);
CREATE TABLE boosty_tiers (
  id integer primary key autoincrement,
  credentials_id integer not null references boosty_credentials(id) on delete restrict,
  boosty_id integer not null,
  created_at datetime not null default current_timestamp,
  missed_at datetime,
  name text not null,
  data text not null,

  unique (credentials_id, boosty_id)
);
CREATE TABLE boosty_tier_subgraphs (
  tier_id integer not null references boosty_tiers(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,

  primary key (tier_id, subgraph_id)
);
CREATE TABLE boosty_members (
  id integer primary key autoincrement,
  credentials_id integer not null references boosty_credentials(id) on delete restrict,
  boosty_id integer not null,
  created_at datetime not null default current_timestamp,
  missed_at datetime,
  email text not null,
  status text not null,
  data text not null,
  current_tier_id integer references boosty_tiers(id) on delete restrict, user_id integer references users(id) on delete restrict,

  unique (credentials_id, boosty_id)
);
CREATE INDEX idx_boosty_members_email on boosty_members(email);
CREATE INDEX idx_patreon_members_email on patreon_members(email);
CREATE TABLE user_favorite_notes (
  user_id integer not null references users(id) on delete cascade,
  note_version_id integer not null references note_versions(id) on delete restrict,
  created_at datetime not null default current_timestamp,

  primary key (user_id, note_version_id)
);
CREATE TABLE IF NOT EXISTS "tg_chat_subgraph_accesses" (
  id integer primary key autoincrement,
  chat_id integer not null,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp
);
CREATE TABLE IF NOT EXISTS "tg_bot_chat_subgraph_invites" (
  chat_id integer not null,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  primary key(chat_id, subgraph_id)
);
CREATE TABLE IF NOT EXISTS "tg_chat_members" (
  user_id integer not null, -- tg id
  chat_id integer not null,
  created_at datetime not null default current_timestamp,
  primary key (user_id, chat_id)
);
CREATE INDEX idx_tg_chat_subgraph_accesses_chat_id on tg_chat_subgraph_accesses(chat_id);
CREATE INDEX idx_tg_bot_chat_subgraph_invites_chat_id on tg_bot_chat_subgraph_invites(chat_id);
CREATE INDEX idx_tg_chat_members_chat_id on tg_chat_members(chat_id);
CREATE TABLE tg_attach_codes (
  user_id integer not null references users(id) on delete cascade,
  bot_id integer not null references tg_bots(id) on delete restrict,
  code text not null unique,
  created_at datetime not null default current_timestamp
);
CREATE TABLE IF NOT EXISTS "user_subgraph_accesses" (
  id integer primary key autoincrement,
  user_id integer not null references users(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  expires_at datetime,
  revoke_id int references revokes(id) on delete restrict,
  purchase_id text references purchases(id) on delete restrict,
  created_by integer references admins(user_id) on delete restrict
);
CREATE TABLE tg_bot_chat_subgraph_accesses (
  chat_id integer not null references tg_bot_chats(id) on delete cascade,
  user_id integer not null references users(id) on delete restrict,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  joined_at datetime,

  primary key (chat_id, user_id, subgraph_id)
);
CREATE TABLE audit_logs (
  id integer primary key autoincrement,
  created_at timestamp not null default current_timestamp,
  level int not null default 0,
  message text not null,
  params text not null
);
CREATE INDEX idx_audit_logs_created_at on audit_logs (created_at);
CREATE TABLE IF NOT EXISTS "tg_bot_chats" (
  id integer primary key autoincrement,
  telegram_id integer not null unique,
  chat_type text not null,
  chat_title text not null,
  added_at datetime not null default current_timestamp,
  removed_at datetime null,
  can_invite boolean not null default false,
  bot_id integer not null
);
CREATE INDEX idx_tg_bot_chats_telegram_id on tg_bot_chats(telegram_id);
CREATE TABLE html_injections (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  active_from datetime,
  active_to datetime,
  description text not null,
  position integer not null default 0,
  placement text not null, -- head / body_end
  content text not null
);
CREATE TABLE cron_jobs (
  id integer primary key autoincrement,
  name text not null unique,
  enabled boolean not null default true,
  expression text not null,
  last_exec_at datetime
);
CREATE TABLE cron_job_executions (
  id integer primary key autoincrement,
  job_id int not null references cron_jobs(id) on delete cascade,
  started_at datetime not null default current_timestamp,
  finished_at datetime,
  status int not null default 0,
  report_data text,
  error_message text
);
CREATE TABLE goqite (
  id text primary key default ('m_' || lower(hex(randomblob(16)))),
  created text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  updated text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  queue text not null,
  body blob not null,
  timeout text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  received integer not null default 0,
  priority integer not null default 0
) strict;
CREATE TRIGGER goqite_updated_timestamp after update on goqite begin
  update goqite set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
end;
CREATE INDEX goqite_queue_priority_created_idx on goqite (queue, priority desc, created);
CREATE TABLE git_tokens (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  last_used_at datetime not null default current_timestamp,
  admin_id integer references admins(user_id) on delete restrict,
  value_sha256 text not null unique,
  description text not null default '',
  can_pull boolean default false,
  can_push boolean default true,
  usage_count integer default 0
, disabled_at datetime, disabled_by integer references admins(user_id) on delete restrict);
CREATE TABLE IF NOT EXISTS "note_assets" (
  id integer primary key autoincrement,
  absolute_path text not null,
  file_name text not null,
  sha256_hash text not null,
  created_at datetime not null default current_timestamp,
  size integer not null default 0,
  unique (absolute_path, sha256_hash)
);
CREATE TABLE notion_integrations (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  enabled boolean not null default true,
  secret_token text not null,
  verification_token text,
  base_path text not null default '/'
);
CREATE TABLE telegram_publish_tags (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  hidden boolean not null default false,
  label text not null unique
);
CREATE TABLE telegram_publish_chats (
  chat_id integer not null references tg_bot_chats(id) on delete cascade,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);
CREATE TABLE telegram_publish_notes (
  note_path_id integer not null primary key references note_paths(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  publish_at datetime not null,
  published_version_id integer references note_versions(id) on delete restrict,
  published_at datetime
, error_count integer not null default 0, last_error text);
CREATE TABLE telegram_publish_note_tags (
  note_path_id integer not null references telegram_publish_notes(note_path_id) on delete cascade,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  primary key (note_path_id, tag_id)
);
CREATE TABLE telegram_publish_instant_chats (
  chat_id integer not null references tg_bot_chats(id) on delete cascade,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);
CREATE TABLE IF NOT EXISTS "telegram_publish_sent_messages" (
  note_path_id integer not null references note_paths(id) on delete restrict,
  chat_id integer not null references tg_bot_chats(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  message_id integer not null,
  instant integer not null default 0 check (instant in (0, 1)),
  content_hash text not null default '',
  content text not null default ''
, post_type text not null default 'text');
CREATE UNIQUE INDEX idx_telegram_publish_sent_messages_unique_scheduled
on telegram_publish_sent_messages(chat_id, note_path_id)
where instant = 0;
CREATE INDEX idx_telegram_publish_sent_messages_chat_id on telegram_publish_sent_messages(chat_id);
CREATE INDEX idx_telegram_publish_sent_messages_note_path_id on telegram_publish_sent_messages(note_path_id);
CREATE TABLE telegram_accounts (
  id integer primary key autoincrement,
  phone text not null unique,
  session_data blob not null,
  display_name text not null default '',
  is_premium integer not null default 0 check (is_premium in (0, 1)),
  enabled integer not null default 1 check (enabled in (0, 1)),
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
, api_id integer not null default 0, api_hash text not null default '', app_config text not null default '{}');
CREATE TABLE telegram_publish_account_chats (
  account_id integer not null references telegram_accounts(id) on delete cascade,
  telegram_chat_id integer not null,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict, access_hash text,
  primary key (account_id, telegram_chat_id, tag_id)
);
CREATE TABLE telegram_publish_account_instant_chats (
  account_id integer not null references telegram_accounts(id) on delete cascade,
  telegram_chat_id integer not null,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict, access_hash text,
  primary key (account_id, telegram_chat_id, tag_id)
);
CREATE TABLE telegram_publish_sent_account_messages (
  note_path_id integer not null references note_paths(id) on delete restrict,
  account_id integer not null references telegram_accounts(id) on delete restrict,
  telegram_chat_id integer not null,
  created_at datetime not null default current_timestamp,
  message_id integer not null,
  instant integer not null default 0 check (instant in (0, 1)),
  content_hash text not null default '',
  content text not null default '',
  post_type text not null default 'text'
);
CREATE UNIQUE INDEX idx_telegram_publish_sent_account_messages_unique
  on telegram_publish_sent_account_messages(note_path_id, account_id, telegram_chat_id)
  where instant = 0;
CREATE INDEX idx_telegram_publish_sent_account_messages_account_id
  on telegram_publish_sent_account_messages(account_id);
CREATE INDEX idx_telegram_publish_sent_account_messages_note_path_id
  on telegram_publish_sent_account_messages(note_path_id);
CREATE INDEX idx_note_paths_hidden_by on note_paths(hidden_by);
CREATE TABLE note_uncommitted_paths (
    note_path_id integer primary key references note_paths(id) on delete cascade
);
CREATE INDEX idx_note_version_assets_version_id on note_version_assets(version_id);
CREATE TABLE note_version_embeddings (
    version_id integer primary key references note_versions(id) on delete cascade,
    embedding blob not null,
    model_id integer not null,
    content_hash blob not null,
    tokens integer not null,
    created_at datetime not null default (datetime('now'))
);
CREATE INDEX idx_note_version_embeddings_model_id on note_version_embeddings(model_id);
CREATE TABLE google_oauth_credentials (
    id integer primary key,
    name text not null,
    client_id text not null,
    client_secret_encrypted blob not null,
    active boolean not null default false,
    created_at datetime not null default (datetime('now')),
    created_by integer not null references users(id)
);
CREATE TABLE github_oauth_credentials (
    id integer primary key,
    name text not null,
    client_id text not null,
    client_secret_encrypted blob not null,
    active boolean not null default false,
    created_at datetime not null default (datetime('now')),
    created_by integer not null references users(id)
);
CREATE TABLE config_changes (
  id integer primary key autoincrement,
  value_id text not null,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);
CREATE INDEX idx_config_changes_value_id on config_changes(value_id);
CREATE TABLE config_string_values (
  change_id integer primary key references config_changes(id) on delete cascade,
  value text not null
);
CREATE TABLE config_bool_values (
  change_id integer primary key references config_changes(id) on delete cascade,
  value boolean not null
);
CREATE TABLE change_webhooks (
  id integer primary key autoincrement,
  url text not null,
  include_patterns text not null,
  exclude_patterns text not null default '[]',
  instruction text not null default '',
  secret text not null,
  max_depth integer not null default 1,
  pass_api_key boolean not null default false,
  include_content boolean not null default true,
  timeout_seconds integer not null default 60,
  max_retries integer not null default 0,
  on_create boolean not null default true,
  on_update boolean not null default true,
  on_remove boolean not null default true,
  read_patterns text not null default '["*"]',
  write_patterns text not null default '[]',
  enabled boolean not null default true,
  description text not null default '',
  created_at datetime not null default (datetime('now')),
  created_by integer not null references admins(user_id) on delete restrict,
  updated_at datetime not null default (datetime('now')),
  disabled_at datetime,
  disabled_by integer references admins(user_id) on delete restrict
, transform_jsonnet text not null default '', attach_notes text not null default '[]', concurrency_mode text not null default 'allow_overlap' check (concurrency_mode in ('allow_overlap','skip','queue_one')));
CREATE TABLE change_webhook_deliveries (
  id integer primary key autoincrement,
  webhook_id integer not null references change_webhooks(id) on delete cascade,
  status text not null default 'pending',
  response_status integer,
  attempt integer not null default 1,
  duration_ms integer,
  created_at datetime not null default (datetime('now')),
  completed_at datetime
, started_at datetime, heartbeat_at datetime, parent_kind text, parent_id integer, trace text, depth_reached integer not null default 0, costs text, logs text);
CREATE INDEX idx_change_webhook_deliveries_webhook_created
  on change_webhook_deliveries(webhook_id, created_at);
CREATE TABLE cron_webhooks (
  id integer primary key autoincrement,
  url text not null,
  cron_schedule text not null,
  instruction text not null default '',
  secret text not null,
  pass_api_key boolean not null default false,
  timeout_seconds integer not null default 60,
  max_depth integer not null default 1,
  max_retries integer not null default 0,
  next_run_at datetime,
  read_patterns text not null default '["*"]',
  write_patterns text not null default '[]',
  enabled boolean not null default true,
  description text not null default '',
  created_at datetime not null default (datetime('now')),
  created_by integer not null references admins(user_id) on delete restrict,
  updated_at datetime not null default (datetime('now')),
  disabled_at datetime,
  disabled_by integer references admins(user_id) on delete restrict
, transform_jsonnet text not null default '', attach_notes text not null default '[]', concurrency_mode text not null default 'allow_overlap' check (concurrency_mode in ('allow_overlap','skip','queue_one')));
CREATE TABLE cron_webhook_deliveries (
  id integer primary key autoincrement,
  cron_webhook_id integer not null references cron_webhooks(id) on delete cascade,
  status text not null default 'pending',
  response_status integer,
  attempt integer not null default 1,
  duration_ms integer,
  created_at datetime not null default (datetime('now')),
  completed_at datetime
, started_at datetime, heartbeat_at datetime, parent_kind text, parent_id integer, trace text, depth_reached integer not null default 0, costs text, logs text);
CREATE INDEX idx_cron_webhook_deliveries_webhook_created
  on cron_webhook_deliveries(cron_webhook_id, created_at);
CREATE TABLE webhook_delivery_logs (
  id integer primary key autoincrement,
  delivery_id integer not null,
  kind text not null,
  request_body text,
  response_body text,
  error_message text,
  created_at datetime not null default (datetime('now'))
);
CREATE INDEX idx_wdl_delivery on webhook_delivery_logs(kind, delivery_id);
CREATE INDEX idx_wdl_created on webhook_delivery_logs(created_at);
CREATE TRIGGER trg_change_webhooks_updated_at
after update on change_webhooks
begin
  update change_webhooks set updated_at = datetime('now') where id = new.id;
end;
CREATE TRIGGER trg_cron_webhooks_updated_at
after update on cron_webhooks
begin
  update cron_webhooks set updated_at = datetime('now') where id = new.id;
end;
CREATE TABLE note_frontmatter_patches (
  id integer primary key autoincrement,
  include_patterns text not null,
  exclude_patterns text not null default '[]',
  jsonnet text not null,
  priority integer not null default 0,
  description text not null default '',
  enabled boolean not null default true,
  created_at datetime not null default (datetime('now')),
  created_by integer not null references admins(user_id) on delete restrict,
  updated_at datetime not null default (datetime('now'))
);
CREATE TABLE note_version_chunks (
    id           integer primary key autoincrement,
    version_id   integer not null references note_versions(id) on delete cascade,
    chunk_index  integer not null,
    content      text    not null,
    embedding    blob,
    model_id     integer,
    content_hash blob,
    tokens       integer,
    created_at   datetime not null default (datetime('now')),
    unique(version_id, chunk_index)
);
CREATE INDEX note_version_chunks_version_id on note_version_chunks(version_id);
CREATE TABLE config_int_values (
  change_id integer primary key references config_changes(id) on delete cascade,
  value integer not null
);
CREATE TABLE telegram_chat_usernames (
  telegram_chat_id integer primary key,
  username text not null default '',
  title text not null default '',
  refreshed_at datetime not null default current_timestamp,
  refresh_requested_at datetime,
  last_error text,
  created_at datetime not null default current_timestamp,
  updated_at datetime not null default current_timestamp
);
CREATE INDEX idx_telegram_chat_usernames_refresh
  on telegram_chat_usernames(refresh_requested_at, refreshed_at);
CREATE TABLE federation_secrets (
  id integer primary key autoincrement,
  kid text not null,
  secret_crypt blob not null,
  kb_url text,
  description text,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  revoked_at datetime
);
CREATE INDEX idx_federation_secrets_kid
  on federation_secrets(kid);
CREATE INDEX idx_federation_secrets_kb_url
  on federation_secrets(kb_url) where kb_url is not null;
CREATE TABLE federation_secret_subgraphs (
  kid text not null,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  primary key (kid, subgraph_id)
);
CREATE TABLE user_tokens (
  id text primary key,
  user_id integer not null references users(id) on delete cascade,
  name text not null default '',
  token_hash text not null unique,
  token_prefix text not null,
  scope text not null default 'all',
  created_at datetime not null default current_timestamp,
  expires_at datetime,
  last_used_at datetime,
  revoked_at datetime
);
CREATE INDEX idx_user_tokens_user_id on user_tokens(user_id);
CREATE INDEX idx_user_tokens_token_hash on user_tokens(token_hash);
CREATE TABLE form_submits (
    id              integer primary key,
    note_version_id integer not null references note_versions(id),
    form_id         text not null default '',
    user_id         integer references users(id),
    ip              text not null default '',
    status          text not null default 'visible',
    created_at      datetime not null default current_timestamp
, processed_at datetime, processed_by integer references users(id), comment text not null default '');
CREATE INDEX form_submits_note_version_id on form_submits(note_version_id);
CREATE TABLE form_string_values (
    submit_id  integer not null references form_submits(id) on delete cascade,
    field_name text not null,
    value      text not null,
    primary key (submit_id, field_name)
);
CREATE TABLE form_int_values (
    submit_id  integer not null references form_submits(id) on delete cascade,
    field_name text not null,
    value      integer not null,
    primary key (submit_id, field_name)
);
CREATE TABLE form_bool_values (
    submit_id  integer not null references form_submits(id) on delete cascade,
    field_name text not null,
    value      integer not null,
    primary key (submit_id, field_name)
);
CREATE TABLE tg_user_canvas_states (
  bot_id                 int  not null references tg_bots(id) on delete cascade,
  business_connection_id text not null default '',
  user_id                int  not null,
  canvas_path            text not null,
  current_node           text not null,
  stack                  text not null default '[]',
  last_media             text not null default '',
  message_id             int  not null default 0,
  updated_at             datetime not null default current_timestamp,
  primary key (bot_id, business_connection_id, user_id)
);
CREATE INDEX tg_user_canvas_states_by_path
  on tg_user_canvas_states(bot_id, canvas_path);
CREATE TABLE tg_user_current_handlers (
  bot_id                 int  not null references tg_bots(id) on delete cascade,
  business_connection_id text not null default '',
  user_id                int  not null,
  value                  text not null default '',
  updated_at             datetime not null default current_timestamp,
  primary key (bot_id, business_connection_id, user_id)
);
CREATE TABLE tg_user_navigation_states (
  bot_id                 int  not null references tg_bots(id) on delete cascade,
  business_connection_id text not null default '',
  user_id                int  not null,
  value                  text not null default '{}',
  updated_at             datetime not null default current_timestamp,
  primary key (bot_id, business_connection_id, user_id)
);
CREATE TABLE secrets (
  id          integer primary key autoincrement,
  key         text    not null unique,
  value_crypt blob    not null,
  created_at  datetime not null default (datetime('now')),
  created_by  integer not null references admins(user_id)
);
CREATE TABLE chart_data_cache (
  version_id integer not null,
  chart_hash text    not null,
  data_json  text    not null,
  fetched_at integer not null, last_error    text    not null default '', last_error_at integer not null default 0,
  primary key (version_id, chart_hash)
);
CREATE TABLE oidc_credentials (
    id integer primary key,
    name text not null,
    issuer text not null,
    client_id text not null,
    client_secret_encrypted blob not null,
    scopes text not null default 'openid email profile',
    auto_provision boolean not null default false,
    allowed_email_domain text not null default '',
    required_group text not null default '',
    active boolean not null default false,
    created_at datetime not null default (datetime('now')),
    created_by integer not null references users(id)
);
CREATE INDEX idx_change_webhook_deliveries_inflight on change_webhook_deliveries(webhook_id, status);
CREATE INDEX idx_cron_webhook_deliveries_inflight on cron_webhook_deliveries(cron_webhook_id, status);
CREATE TABLE note_version_delivery_attribution (
  note_version_id integer primary key references note_versions(id) on delete cascade,
  delivery_kind text not null,
  delivery_id integer not null
);
CREATE INDEX idx_nvda_delivery on note_version_delivery_attribution(delivery_kind, delivery_id);
CREATE TABLE note_version_frontmatters (
  version_id integer not null primary key references note_versions (id) on delete cascade,
  data string not null check (json_valid(data))
);
CREATE TABLE note_version_frontmatter_key_values (
  value string not null primary key,
  created_by_version_id integer not null references note_versions (id) on delete restrict,
  hidden_at datetime -- live and latest notes don't contain this key anymore
);
CREATE TABLE note_version_frontmatter_keys (
  note_version_id integer not null references note_versions (id) on delete cascade,
  key_id string not null references note_version_frontmatter_key_values (value) on delete cascade,
  unique (note_version_id, key_id)
);
CREATE INDEX note_version_frontmatter_keys_key_id_idx
  on note_version_frontmatter_keys (key_id);
CREATE INDEX idx_change_webhook_deliveries_trace on change_webhook_deliveries(trace, created_at);
CREATE INDEX idx_cron_webhook_deliveries_trace on cron_webhook_deliveries(trace, created_at);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20250402131258'),
  ('20250409115720'),
  ('20250412053210'),
  ('20250414025612'),
  ('20250417050444'),
  ('20250419030458'),
  ('20250427033102'),
  ('20250430041756'),
  ('20250430065941'),
  ('20250502030912'),
  ('20250503030824'),
  ('20250503031556'),
  ('20250503032418'),
  ('20250504074439'),
  ('20250506122229'),
  ('20250506122811'),
  ('20250507032627'),
  ('20250515071315'),
  ('20250515071316'),
  ('20250524091058'),
  ('20250525034319'),
  ('20250528125918'),
  ('20250531040526'),
  ('20250531113101'),
  ('20250602143243'),
  ('20250604130924'),
  ('20250605090619'),
  ('20250606084510'),
  ('20250623041230'),
  ('20250623063206'),
  ('20250626035523'),
  ('20250626041424'),
  ('20250626054021'),
  ('20250626100000'),
  ('20250626120000'),
  ('20250627040815'),
  ('20250628111216'),
  ('20250724085424'),
  ('20250724090433'),
  ('20250725034851'),
  ('20250725200000'),
  ('20250725201000'),
  ('20250725202000'),
  ('20250727034504'),
  ('20250728130332'),
  ('20250729014409'),
  ('20250729111321'),
  ('20250729112136'),
  ('20250731060940'),
  ('20250731061653'),
  ('20250801040147'),
  ('20250801080226'),
  ('20250804051415'),
  ('20250806044332'),
  ('20250806153321'),
  ('20250807124754'),
  ('20250809044217'),
  ('20250809093139'),
  ('20250810022248'),
  ('20250810023112'),
  ('20250812041450'),
  ('20250812095819'),
  ('20250813034629'),
  ('20250813052619'),
  ('20250815035326'),
  ('20250815092446'),
  ('20250816081838'),
  ('20250918140112'),
  ('20250925035301'),
  ('20250927035933'),
  ('20251001113550'),
  ('20251003125722'),
  ('20251016125315'),
  ('20251021134341'),
  ('20251022032711'),
  ('20251024123641'),
  ('20251025034145'),
  ('20251029103550'),
  ('20251029150445'),
  ('20251030012221'),
  ('20251030151751'),
  ('20251030152642'),
  ('20251031015532'),
  ('20251103094933'),
  ('20251105114403'),
  ('20251118053250'),
  ('20251119013128'),
  ('20251201041923'),
  ('20251203034400'),
  ('20251203061607'),
  ('20251203061630'),
  ('20251203061640'),
  ('20251203061651'),
  ('20251203062401'),
  ('20251204121052'),
  ('20251210042103'),
  ('20251218090744'),
  ('20251219021352'),
  ('20251224090437'),
  ('20260102093534'),
  ('20260119020631'),
  ('20260126113359'),
  ('20260127122121'),
  ('20260128081252'),
  ('20260209100000'),
  ('20260211120000'),
  ('20260213114107'),
  ('20260318100000'),
  ('20260322100000'),
  ('20260330100000'),
  ('20260416132701'),
  ('20260427100000'),
  ('20260430100000'),
  ('20260506120000'),
  ('20260513120000'),
  ('20260518072631'),
  ('20260524105748'),
  ('20260524105749'),
  ('20260526120000'),
  ('20260602095046'),
  ('20260615120000'),
  ('20260621150755'),
  ('20260627000000'),
  ('20260628120000'),
  ('20260628120100'),
  ('20260713100000'),
  ('20260715051651'),
  ('20260820105848'),
  ('20260820125621'),
  ('20260821014419');
