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
CREATE TABLE admins (
  user_id text primary key references users(id) on delete cascade,
  granted_at datetime not null default current_timestamp,
  granted_by text references admins(user_id)
);
CREATE TABLE offers (
  id text primary key,
  created_at datetime not null default current_timestamp,
  names text not null,-- e.g. "course-a|course-b"
  lifetime text not null, -- e.g. "+600 days"
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
CREATE TABLE users (
  id integer primary key,
  email text not null unique,
  created_at datetime not null default current_timestamp,
  last_signin_code_sent_at datetime
);
CREATE TABLE sign_in_codes (
  user_id integer not null,
  code integer not null,
  created_at datetime not null default current_timestamp
);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20250402131258'),
  ('20250409115720'),
  ('20250412053210');
