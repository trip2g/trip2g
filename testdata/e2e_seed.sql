PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
INSERT INTO schema_migrations VALUES('20250402131258');
INSERT INTO schema_migrations VALUES('20250409115720');
INSERT INTO schema_migrations VALUES('20250412053210');
INSERT INTO schema_migrations VALUES('20250414025612');
INSERT INTO schema_migrations VALUES('20250417050444');
INSERT INTO schema_migrations VALUES('20250419030458');
INSERT INTO schema_migrations VALUES('20250427033102');
INSERT INTO schema_migrations VALUES('20250430041756');
INSERT INTO schema_migrations VALUES('20250430065941');
INSERT INTO schema_migrations VALUES('20250502030912');
INSERT INTO schema_migrations VALUES('20250503030824');
INSERT INTO schema_migrations VALUES('20250503031556');
INSERT INTO schema_migrations VALUES('20250503032418');
INSERT INTO schema_migrations VALUES('20250504074439');
INSERT INTO schema_migrations VALUES('20250506122229');
INSERT INTO schema_migrations VALUES('20250506122811');
INSERT INTO schema_migrations VALUES('20250507032627');
INSERT INTO schema_migrations VALUES('20250515071315');
INSERT INTO schema_migrations VALUES('20250515071316');
INSERT INTO schema_migrations VALUES('20250524091058');
INSERT INTO schema_migrations VALUES('20250525034319');
INSERT INTO schema_migrations VALUES('20250528125918');
INSERT INTO schema_migrations VALUES('20250531040526');
INSERT INTO schema_migrations VALUES('20250531113101');
INSERT INTO schema_migrations VALUES('20250602143243');
INSERT INTO schema_migrations VALUES('20250604130924');
INSERT INTO schema_migrations VALUES('20250605090619');
INSERT INTO schema_migrations VALUES('20250606084510');
INSERT INTO schema_migrations VALUES('20250623041230');
INSERT INTO schema_migrations VALUES('20250623063206');
INSERT INTO schema_migrations VALUES('20250626035523');
INSERT INTO schema_migrations VALUES('20250626041424');
INSERT INTO schema_migrations VALUES('20250626054021');
INSERT INTO schema_migrations VALUES('20250626100000');
INSERT INTO schema_migrations VALUES('20250626120000');
INSERT INTO schema_migrations VALUES('20250627040815');
INSERT INTO schema_migrations VALUES('20250628111216');
INSERT INTO schema_migrations VALUES('20250724085424');
INSERT INTO schema_migrations VALUES('20250724090433');
INSERT INTO schema_migrations VALUES('20250725034851');
INSERT INTO schema_migrations VALUES('20250725200000');
INSERT INTO schema_migrations VALUES('20250725201000');
INSERT INTO schema_migrations VALUES('20250725202000');
INSERT INTO schema_migrations VALUES('20250727034504');
INSERT INTO schema_migrations VALUES('20250728130332');
INSERT INTO schema_migrations VALUES('20250729014409');
INSERT INTO schema_migrations VALUES('20250729111321');
INSERT INTO schema_migrations VALUES('20250729112136');
INSERT INTO schema_migrations VALUES('20250731060940');
INSERT INTO schema_migrations VALUES('20250731061653');
INSERT INTO schema_migrations VALUES('20250801040147');
INSERT INTO schema_migrations VALUES('20250801080226');
INSERT INTO schema_migrations VALUES('20250804051415');
INSERT INTO schema_migrations VALUES('20250806044332');
INSERT INTO schema_migrations VALUES('20250806153321');
INSERT INTO schema_migrations VALUES('20250807124754');
INSERT INTO schema_migrations VALUES('20250809044217');
INSERT INTO schema_migrations VALUES('20250809093139');
INSERT INTO schema_migrations VALUES('20250810022248');
INSERT INTO schema_migrations VALUES('20250810023112');
INSERT INTO schema_migrations VALUES('20250812041450');
INSERT INTO schema_migrations VALUES('20250812095819');
INSERT INTO schema_migrations VALUES('20250813034629');
INSERT INTO schema_migrations VALUES('20250813052619');
INSERT INTO schema_migrations VALUES('20250815035326');
INSERT INTO schema_migrations VALUES('20250815092446');
INSERT INTO schema_migrations VALUES('20250816081838');
INSERT INTO schema_migrations VALUES('20250918140112');
INSERT INTO schema_migrations VALUES('20250925035301');
INSERT INTO schema_migrations VALUES('20250927035933');
INSERT INTO schema_migrations VALUES('20251001113550');
INSERT INTO schema_migrations VALUES('20251003125722');
INSERT INTO schema_migrations VALUES('20251016125315');
INSERT INTO schema_migrations VALUES('20251021134341');
INSERT INTO schema_migrations VALUES('20251022032711');
INSERT INTO schema_migrations VALUES('20251024123641');
INSERT INTO schema_migrations VALUES('20251025034145');
INSERT INTO schema_migrations VALUES('20251029103550');
INSERT INTO schema_migrations VALUES('20251029150445');
INSERT INTO schema_migrations VALUES('20251030012221');
INSERT INTO schema_migrations VALUES('20251030151751');
INSERT INTO schema_migrations VALUES('20251030152642');
INSERT INTO schema_migrations VALUES('20251031015532');
INSERT INTO schema_migrations VALUES('20251103094933');
INSERT INTO schema_migrations VALUES('20251105114403');
INSERT INTO schema_migrations VALUES('20251118053250');
INSERT INTO schema_migrations VALUES('20251119013128');
INSERT INTO schema_migrations VALUES('20251201041923');
INSERT INTO schema_migrations VALUES('20251203034400');
INSERT INTO schema_migrations VALUES('20251203061607');
INSERT INTO schema_migrations VALUES('20251203061630');
INSERT INTO schema_migrations VALUES('20251203061640');
INSERT INTO schema_migrations VALUES('20251203061651');
INSERT INTO schema_migrations VALUES('20251203062401');
INSERT INTO schema_migrations VALUES('20251204121052');
INSERT INTO schema_migrations VALUES('20251210042103');
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
INSERT INTO admins VALUES(1,'2025-12-18 07:34:48',NULL);
CREATE TABLE sign_in_codes (
  user_id integer not null,
  code text not null,
  created_at datetime not null default current_timestamp
);
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
CREATE TABLE subgraphs (
  id integer primary key autoincrement,
  name text not null unique,
  color text,
  created_at datetime not null default current_timestamp
, hidden boolean not null default false, show_unsubgraph_notes_for_paid_users boolean default true);
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
  created_at datetime not null default current_timestamp,
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
);
INSERT INTO api_keys VALUES(1,'d4b8f5ef87a1dd7b57ea27d9f4abfab0ad127c910153c8594ecb98b9e394cd87','2025-12-18 07:41:46',1,NULL,NULL,'');
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
, name text not null default '');
INSERT INTO tg_bots VALUES(1,'PLACEHOLDER',1,'test','2025-12-18 07:37:53',1,'trip2g_e2e_test_bot');
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
INSERT INTO tg_user_states VALUES(-1003576908503,1,NULL,'2025-12-18 07:39:36','2025-12-18 07:39:43',2,'pending','{"quiz_states":{}}');
INSERT INTO tg_user_states VALUES(-1003591599765,1,NULL,'2025-12-18 07:39:51','2025-12-18 07:39:58',2,'pending','{"quiz_states":{}}');
CREATE TABLE tg_user_profiles (
  sha256_hash text primary key,
  chat_id int not null,
  bot_id int not null references tg_bots(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  first_name text,
  last_name text,
  username text
);
CREATE TABLE IF NOT EXISTS "users" (
    id integer primary key,
    email text unique, -- nullable but unique for linked accounts
    created_at datetime not null default current_timestamp,
    last_signin_code_sent_at datetime,
    note_view_count integer default 0,
    tg_user_id integer unique -- Also unique - one account per Telegram user
    -- Note: No FK constraint because tg_user_profiles.chat_id is not unique
, created_via text not null default 'unknown');
INSERT INTO users VALUES(1,'hello@example.com','2025-12-18 07:34:48',NULL,0,NULL,'bootstrap');
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
INSERT INTO tg_bot_chats VALUES(1,-1003576908503,'channel','Trip2G Test Bot Instant','2025-12-18 07:39:43',NULL,1,1);
INSERT INTO tg_bot_chats VALUES(2,-1003591599765,'channel','Trip2G Test Bot','2025-12-18 07:39:58',NULL,1,1);
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
INSERT INTO cron_jobs VALUES(1,'apply_git_changes',1,'0 0 0 * * *',NULL);
INSERT INTO cron_jobs VALUES(2,'remove_expired_tg_chat_members',1,'0 0 * * * *',NULL);
INSERT INTO cron_jobs VALUES(3,'clear_cronjob_execution_history',1,'0 0 0 * * *',NULL);
INSERT INTO cron_jobs VALUES(4,'send_scheduled_telegram_publishposts',1,'0 * * * * *','2025-12-18 07:42:00');
INSERT INTO cron_jobs VALUES(5,'update_telegram_publish_posts',1,'0 0 0 * * *',NULL);
INSERT INTO cron_jobs VALUES(6,'refresh_telegram_accounts',1,'0 0 3 * * *',NULL);
INSERT INTO cron_jobs VALUES(7,'vacuum_database',1,'0 0 3 * * 0',NULL);
CREATE TABLE cron_job_executions (
  id integer primary key autoincrement,
  job_id int not null references cron_jobs(id) on delete cascade,
  started_at datetime not null default current_timestamp,
  finished_at datetime,
  status int not null default 0,
  report_data text,
  error_message text
);
INSERT INTO cron_job_executions VALUES(1,4,'2025-12-18 07:35:00','2025-12-18 07:35:00',2,'{"bot_posts":null,"account_posts":null}',NULL);
INSERT INTO cron_job_executions VALUES(2,4,'2025-12-18 07:36:00','2025-12-18 07:36:00',2,'{"bot_posts":null,"account_posts":null}',NULL);
INSERT INTO cron_job_executions VALUES(3,4,'2025-12-18 07:37:00','2025-12-18 07:37:00',2,'{"bot_posts":null,"account_posts":null}',NULL);
INSERT INTO cron_job_executions VALUES(4,4,'2025-12-18 07:38:00','2025-12-18 07:38:00',2,'{"bot_posts":null,"account_posts":null}',NULL);
INSERT INTO cron_job_executions VALUES(5,4,'2025-12-18 07:39:00','2025-12-18 07:39:00',2,'{"bot_posts":null,"account_posts":null}',NULL);
INSERT INTO cron_job_executions VALUES(6,4,'2025-12-18 07:40:00','2025-12-18 07:40:00',2,'{"bot_posts":null,"account_posts":null}',NULL);
INSERT INTO cron_job_executions VALUES(7,4,'2025-12-18 07:42:00','2025-12-18 07:42:00',2,'{"bot_posts":null,"account_posts":null}',NULL);
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
CREATE TABLE config_versions (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  show_draft_versions boolean not null default false,
  default_layout text not null default ''
, timezone text not null default 'UTC', robots_txt text not null default 'open');
INSERT INTO config_versions VALUES(1,'2025-12-18 07:52:41',1,1,'','UTC','open');
CREATE TABLE telegram_publish_tags (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  hidden boolean not null default false,
  label text not null unique
);
INSERT INTO telegram_publish_tags VALUES(1,'2025-12-18 07:38:13',0,'test_channel');
INSERT INTO telegram_publish_tags VALUES(2,'2025-12-18 07:38:13',0,'test_premium_channel');
CREATE TABLE telegram_publish_chats (
  chat_id integer not null references tg_bot_chats(id) on delete cascade,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);
INSERT INTO telegram_publish_chats VALUES(2,1,'2025-12-18 07:40:14',1);
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
INSERT INTO telegram_publish_instant_chats VALUES(1,1,'2025-12-18 07:40:16',1);
CREATE TABLE IF NOT EXISTS "telegram_publish_sent_messages" (
  note_path_id integer not null references note_paths(id) on delete restrict,
  chat_id integer not null references tg_bot_chats(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  message_id integer not null,
  instant integer not null default 0 check (instant in (0, 1)),
  content_hash text not null default '',
  content text not null default ''
, post_type text not null default 'text');
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
INSERT INTO telegram_accounts VALUES(1,'PLACEHOLDER','PLACEHOLDER','PLACEHOLDER',1,1,'2025-12-18 07:36:46',1,35561633,'28d5910127c3a4ac45ec4e0d884fe593','{"about_length_limit_default":70,"about_length_limit_premium":140,"authorization_autoconfirm_period":604800,"autologin_domains":["instantview.telegram.org","translations.telegram.org","contest.dev","contest.com","bugs.telegram.org","suggestions.telegram.org","themes.telegram.org","promote.telegram.org","ads.telegram.org"],"boosts_channel_level_max":100,"boosts_per_sent_gift":3,"bot_preview_medias_max":12,"bot_verification_description_length_limit":70,"business_chat_links_limit":100,"business_promo_order":["business_location","business_hours","quick_replies","greeting_message","away_message","business_links","business_intro","business_bots","emoji_status","folder_tags","stories"],"caption_length_limit_default":1024,"caption_length_limit_premium":4096,"channel_autotranslation_level_min":3,"channel_bg_icon_level_min":4,"channel_color_level_min":5,"channel_custom_wallpaper_level_min":10,"channel_emoji_status_level_min":8,"channel_profile_bg_icon_level_min":7,"channel_restrict_sponsored_level_min":50,"channel_revenue_withdrawal_enabled":true,"channel_wallpaper_level_min":9,"channels_limit_default":500,"channels_limit_premium":1000,"channels_public_limit_default":10,"channels_public_limit_premium":20,"chat_read_mark_expire_period":604800,"chat_read_mark_size_threshold":100,"chatlist_invites_limit_default":3,"chatlist_invites_limit_premium":100,"chatlist_update_period":300,"chatlists_joined_limit_default":2,"chatlists_joined_limit_premium":20,"conference_call_size_limit":200,"default_emoji_statuses_stickerset_id":"773947703670341676","dialog_filters_chats_limit_default":100,"dialog_filters_chats_limit_premium":200,"dialog_filters_enabled":true,"dialog_filters_limit_default":10,"dialog_filters_limit_premium":30,"dialog_filters_tooltip":true,"dialogs_folder_pinned_limit_default":100,"dialogs_folder_pinned_limit_premium":200,"dialogs_pinned_limit_default":5,"dialogs_pinned_limit_premium":10,"emojies_animated_zoom":0.625,"emojies_send_dice":["🎲","🎯","🏀","⚽","⚽️","🎰","🎳"],"emojies_send_dice_success":{"⚽":{"frame_start":110,"value":5},"⚽️":{"frame_start":110,"value":5},"🎯":{"frame_start":62,"value":6},"🎰":{"frame_start":110,"value":64},"🎳":{"frame_start":110,"value":6},"🏀":{"frame_start":110,"value":5}},"emojies_sounds":{"⚰":{"access_hash":"-1498869544183595185","file_reference_base64":"","id":"4956223179606458540"},"🍑":{"access_hash":"-7431729439735063448","file_reference_base64":"","id":"4963180910661861548"},"🎃":{"access_hash":"-2107001400913062971","file_reference_base64":"","id":"4956223179606458539"},"🎄":{"access_hash":"-4142643820629256996","file_reference_base64":"","id":"5094064004578410733"},"🎊":{"access_hash":"8518192996098758509","file_reference_base64":"","id":"5094064004578410732"},"🦾":{"access_hash":"-8934384022571962340","file_reference_base64":"","id":"5094064004578410734"},"🧟":{"access_hash":"-8929417974289765626","file_reference_base64":"","id":"4960929110848176332"},"🧟‍♀":{"access_hash":"9161696144162881753","file_reference_base64":"","id":"4960929110848176333"},"🧟‍♂":{"access_hash":"3986395821757915468","file_reference_base64":"","id":"4960929110848176331"}},"forum_upgrade_participants_min":0,"fragment_prefixes":["888"],"getfile_experimental_params":false,"gif_search_branding":"tenor","gif_search_emojies":["👍","😘","😍","😡","🥳","😂","😮","🙄","😎","👎"],"giveaway_add_peers_max":10,"giveaway_boosts_per_premium":4,"giveaway_countries_max":10,"giveaway_gifts_purchase_available":true,"giveaway_period_max":2678400,"group_custom_wallpaper_level_min":10,"group_emoji_status_level_min":8,"group_emoji_stickers_level_min":4,"group_profile_bg_icon_level_min":5,"group_transcribe_level_min":6,"group_wallpaper_level_min":9,"groupcall_video_participants_max":1000,"hidden_members_group_size_min":100,"intro_description_length_limit":70,"intro_title_length_limit":32,"ios_disable_parallel_channel_reset":1,"large_queue_max_active_operations_count":2,"message_animated_emoji_max":100,"new_noncontact_peers_require_premium_without_ownpremium":true,"pm_read_date_expire_period":604800,"poll_answers_max":12,"premium_bot_username":"PremiumBot","premium_gift_attach_menu_icon":true,"premium_gift_text_field_icon":false,"premium_playmarket_direct_currency_list":["RUB","BYN"],"premium_promo_order":["stories","more_upload","double_limits","business","last_seen","voice_to_text","faster_download","translations","animated_emoji","emoji_status","saved_tags","peer_colors","wallpapers","profile_badge","message_privacy","advanced_chat_management","no_ads","app_icons","infinite_reactions","animated_userpics","premium_stickers","effects","todo"],"premium_purchase_blocked":false,"qr_login_camera":true,"qr_login_code":"primary","quick_replies_limit":100,"quick_reply_messages_limit":20,"quote_length_max":1024,"reactions_in_chat_max":100,"reactions_uniq_max":11,"reactions_user_max_default":1,"reactions_user_max_premium":3,"recommended_channels_limit_default":10,"recommended_channels_limit_premium":100,"ringtone_duration_max":5,"ringtone_saved_count_max":100,"ringtone_size_max":307200,"round_video_encoding":{"audio_bitrate":64,"diameter":384,"max_size":12582912,"video_bitrate":1000},"saved_dialogs_pinned_limit_default":5,"saved_dialogs_pinned_limit_premium":100,"saved_gifs_limit_default":200,"saved_gifs_limit_premium":400,"small_queue_max_active_operations_count":5,"stargifts_blocked":false,"stargifts_collection_gifts_limit":500,"stargifts_collections_limit":10,"stargifts_convert_period_max":604800,"stargifts_message_length_max":128,"stargifts_pinned_to_top_limit":6,"starref_connect_allowed":true,"starref_max_commission_permille":800,"starref_min_commission_permille":1,"starref_program_allowed":true,"starref_start_param_prefixes":["_tgr_"],"stars_gifts_enabled":true,"stars_groupcall_message_amount_max":10000,"stars_groupcall_message_limits":[{"color1":"5B6676","color2":"7B899D","color_bg":"252C36","emoji_max":20,"pin_period":3600,"stars":10000,"text_length_max":400},{"color1":"E14741","color2":"E96139","color_bg":"8B0503","emoji_max":10,"pin_period":1800,"stars":2000,"text_length_max":280},{"color1":"ED771E","color2":"ED771E","color_bg":"9B3100","emoji_max":7,"pin_period":900,"stars":500,"text_length_max":200},{"color1":"E29A09","color2":"E29A09","color_bg":"9A3E00","emoji_max":4,"pin_period":600,"stars":250,"text_length_max":150},{"color1":"40A920","color2":"40A920","color_bg":"176200","emoji_max":3,"pin_period":300,"stars":100,"text_length_max":110},{"color1":"46A3EB","color2":"46A3EB","color_bg":"00508E","emoji_max":2,"pin_period":120,"stars":50,"text_length_max":80},{"color1":"955CDB","color2":"955CDB","color_bg":"49079B","emoji_max":1,"pin_period":60,"stars":10,"text_length_max":60},{"color1":"955CDB","color2":"955CDB","color_bg":"49079B","emoji_max":0,"pin_period":0,"stars":0,"text_length_max":30}],"stars_paid_message_amount_max":10000,"stars_paid_message_commission_permille":850,"stars_paid_messages_available":true,"stars_paid_messages_channel_amount_default":10,"stars_paid_post_amount_max":10000,"stars_paid_reaction_amount_max":10000,"stars_purchase_blocked":false,"stars_rating_learnmore_url":"https://telegram.org/faq#q-what-does-profile-rating-mean","stars_revenue_withdrawal_max":25000000,"stars_revenue_withdrawal_min":1000,"stars_stargift_resale_amount_max":100000,"stars_stargift_resale_amount_min":125,"stars_stargift_resale_commission_permille":800,"stars_subscription_amount_max":10000,"stars_suggested_post_age_min":86400,"stars_suggested_post_amount_max":100000,"stars_suggested_post_amount_min":5,"stars_suggested_post_commission_permille":850,"stars_suggested_post_future_max":2678400,"stars_suggested_post_future_min":300,"stars_usd_sell_rate_x1000":1410,"stars_usd_withdraw_rate_x1000":1300,"stickers_emoji_cache_time":86400,"stickers_emoji_suggest_only_api":false,"stickers_faved_limit_default":5,"stickers_faved_limit_premium":10,"stickers_normal_by_emoji_per_premium_num":3,"stickers_premium_by_emoji_num":0,"stories_album_stories_limit":1000,"stories_albums_limit":100,"stories_all_hidden":false,"stories_area_url_max":3,"stories_changelog_user_id":777000,"stories_entities":"premium","stories_export_nopublic_link":true,"stories_pinned_to_top_count_max":3,"stories_posting":"premium","stories_sent_monthly_limit_default":10,"stories_sent_monthly_limit_premium":3000,"stories_sent_weekly_limit_default":3,"stories_sent_weekly_limit_premium":700,"stories_stealth_cooldown_period":10800,"stories_stealth_future_period":1500,"stories_stealth_past_period":300,"stories_suggested_reactions_limit_default":1,"stories_suggested_reactions_limit_premium":5,"stories_venue_search_username":"foursquare","story_caption_length_limit_default":200,"story_caption_length_limit_premium":2048,"story_expire_period":86400,"story_expiring_limit_default":1,"story_expiring_limit_premium":100,"story_viewers_expire_period":86400,"story_weather_preload":true,"telegram_antispam_group_size_min":200,"telegram_antispam_user_id":"5434988373","test":1,"todo_item_length_max":200,"todo_items_max":30,"todo_title_length_max":255,"ton_blockchain_explorer_url":"https://tonviewer.com/","ton_proxy_address":"magic.org","ton_stargift_resale_amount_max":100000000000000,"ton_stargift_resale_amount_min":1000000000,"ton_stargift_resale_commission_permille":900,"ton_suggested_post_amount_max":10000000000000,"ton_suggested_post_amount_min":10000000,"ton_suggested_post_commission_permille":850,"ton_topup_url":"https://fragment.com/ads/topup","ton_usd_rate":1.4698460181031379,"topics_pinned_limit":5,"transcribe_audio_trial_duration_max":300,"transcribe_audio_trial_weekly_number":0,"translations_auto_enabled":"enabled","translations_manual_enabled":"enabled","upload_max_fileparts_default":4000,"upload_max_fileparts_premium":8000,"upload_premium_speedup_download":10,"upload_premium_speedup_notify_period":3600,"upload_premium_speedup_upload":10,"url_auth_domains":["web.telegram.org","web.t.me","k.t.me","z.t.me","a.t.me"],"weather_search_username":"StoryWeatherBot","web_app_allowed_protocols":["http","https"],"whitelisted_bots":[429000,93372553,417160374,2847000],"whitelisted_domains":["telegram.dog","telegram.me","telegram.org","t.me","telesco.pe","fragment.com","translations.telegram.org"],"youtube_pip":"inapp"}');
CREATE TABLE telegram_publish_account_chats (
  account_id integer not null references telegram_accounts(id) on delete cascade,
  telegram_chat_id integer not null,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  primary key (account_id, telegram_chat_id, tag_id)
);
INSERT INTO telegram_publish_account_chats VALUES(1,3611189458,1,'2025-12-18 07:39:10',1);
INSERT INTO telegram_publish_account_chats VALUES(1,3611189458,2,'2025-12-18 07:39:10',1);
CREATE TABLE telegram_publish_account_instant_chats (
  account_id integer not null references telegram_accounts(id) on delete cascade,
  telegram_chat_id integer not null,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  primary key (account_id, telegram_chat_id, tag_id)
);
INSERT INTO telegram_publish_account_instant_chats VALUES(1,3513155321,1,'2025-12-18 07:38:50',1);
INSERT INTO telegram_publish_account_instant_chats VALUES(1,3513155321,2,'2025-12-18 07:38:50',1);
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
DELETE FROM sqlite_sequence;
INSERT INTO sqlite_sequence VALUES('note_versions',0);
INSERT INTO sqlite_sequence VALUES('tg_chat_subgraph_accesses',0);
INSERT INTO sqlite_sequence VALUES('user_subgraph_accesses',0);
INSERT INTO sqlite_sequence VALUES('tg_bot_chats',2);
INSERT INTO sqlite_sequence VALUES('note_assets',0);
INSERT INTO sqlite_sequence VALUES('cron_jobs',7);
INSERT INTO sqlite_sequence VALUES('cron_job_executions',7);
INSERT INTO sqlite_sequence VALUES('telegram_accounts',1);
INSERT INTO sqlite_sequence VALUES('tg_bots',1);
INSERT INTO sqlite_sequence VALUES('telegram_publish_tags',2);
INSERT INTO sqlite_sequence VALUES('api_keys',1);
CREATE INDEX idx_sign_in_codes_user_id on sign_in_codes(user_id);
CREATE INDEX backlite_tasks_wait_until ON backlite_tasks (wait_until) WHERE wait_until IS NOT NULL;
CREATE INDEX idx_releases_is_live on releases(is_live);
CREATE INDEX tg_user_profiles_chat_id_idx on tg_user_profiles(chat_id);
CREATE UNIQUE INDEX unique_patreon_member on patreon_members(patreon_id, campaign_id);
CREATE INDEX idx_boosty_members_email on boosty_members(email);
CREATE INDEX idx_patreon_members_email on patreon_members(email);
CREATE INDEX idx_tg_chat_subgraph_accesses_chat_id on tg_chat_subgraph_accesses(chat_id);
CREATE INDEX idx_tg_bot_chat_subgraph_invites_chat_id on tg_bot_chat_subgraph_invites(chat_id);
CREATE INDEX idx_tg_chat_members_chat_id on tg_chat_members(chat_id);
CREATE INDEX idx_audit_logs_created_at on audit_logs (created_at);
CREATE INDEX idx_tg_bot_chats_telegram_id on tg_bot_chats(telegram_id);
CREATE TRIGGER goqite_updated_timestamp after update on goqite begin
  update goqite set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
end;
CREATE INDEX goqite_queue_priority_created_idx on goqite (queue, priority desc, created);
CREATE UNIQUE INDEX idx_telegram_publish_sent_messages_unique_scheduled
on telegram_publish_sent_messages(chat_id, note_path_id)
where instant = 0;
CREATE INDEX idx_telegram_publish_sent_messages_chat_id on telegram_publish_sent_messages(chat_id);
CREATE INDEX idx_telegram_publish_sent_messages_note_path_id on telegram_publish_sent_messages(note_path_id);
CREATE UNIQUE INDEX idx_telegram_publish_sent_account_messages_unique
  on telegram_publish_sent_account_messages(note_path_id, account_id, telegram_chat_id)
  where instant = 0;
CREATE INDEX idx_telegram_publish_sent_account_messages_account_id
  on telegram_publish_sent_account_messages(account_id);
CREATE INDEX idx_telegram_publish_sent_account_messages_note_path_id
  on telegram_publish_sent_account_messages(note_path_id);
CREATE INDEX idx_note_paths_hidden_by on note_paths(hidden_by);
COMMIT;
