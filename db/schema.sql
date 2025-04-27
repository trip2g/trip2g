CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
CREATE TABLE note_paths (
  id integer primary key,
  value text not null unique on conflict ignore,
  value_hash text not null unique on conflict fail,
  latest_content_hash text not null,
  created_at datetime no null default current_timestamp,
  version_count integer not null default 0
);
CREATE TABLE note_versions (
  path_id integer not null,
  version integer not null,
  content text not null,
  created_at datetime not null default current_timestamp,
  primary key (path_id, version),
  foreign key (path_id) references note_paths(id) on delete restrict
);
CREATE TABLE users (
  id integer primary key,
  email text not null unique,
  created_at datetime not null default current_timestamp,
  last_signin_code_sent_at datetime
);
CREATE TABLE offers (
  id text primary key,
  created_at datetime not null default current_timestamp,
  names text not null,-- e.g. "course-a|course-b" sorted alphabetically
  lifetime text, -- e.g. "+600 days", null means no expiration
  price_usd numeric,
  price_rub numeric,
  price_btc numeric,
  starts_at datetime,
  ends_at datetime
);
CREATE TABLE purchases (
  id text primary key,
  user_id text not null references users(id) on delete cascade,
  offer_id text not null references offers(id) on delete restrict,
  expire_at datetime, -- e.g. now() + offers.lifetime
  created_at datetime not null default current_timestamp,
  payment_provider text not null,
  payment_data json not null
);
CREATE TABLE sign_in_codes (
  user_id integer not null,
  code integer not null,
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
CREATE INDEX backlite_tasks_wait_until ON backlite_tasks (wait_until) WHERE wait_until IS NOT NULL;
CREATE TABLE subgraphs (
  id integer primary key autoincrement,
  name text not null unique,
  color text,
  created_at datetime not null default current_timestamp
);
CREATE TABLE user_subgraph_accesses (
  id integer primary key autoincrement,
  user_id integer not null references users(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  purchase_id integer references purchases(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  expires_at datetime
, revoke_id int references revokes(id) on delete restrict);
CREATE TABLE revokes (
  id integer primary key autoincrement,
  target_type text not null,
  target_id integer not null,
  created_at datetime not null default current_timestamp,
  by admin_id integer not null references admins(id) on delete restrict,
  reason text
);
CREATE TABLE user_bans (
  user_id integer primary key references users(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  banned_by integer references admin(id) on delete restrict,
  reason text not null
);
CREATE TABLE admins (
  user_id int primary key references users(id) on delete cascade,
  granted_at datetime not null default current_timestamp,
  granted_by text references admins(user_id)
);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20250402131258'),
  ('20250409115720'),
  ('20250412053210'),
  ('20250414025612'),
  ('20250417050444'),
  ('20250419030458'),
  ('20250427033102');
