CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
CREATE TABLE note_paths (
  id integer primary key,
  path text not null unique on conflict ignore,
  path_hash text not null unique on conflict fail,
  created_at datetime default current_timestamp,
  version_count integer not null default 0
);
CREATE TABLE note_versions (
  path_id integer not null,
  version integer not null,
  content text not null,
  content_hash text not null,
  created_at datetime default current_timestamp,
  primary key (path_id, version),
  unique (path_id, content_hash) on conflict fail,
  foreign key (path_id) references note_paths(id) on delete restrict
);
CREATE TABLE users (
  id integer primary key,
  email text not null unique,
  password_hash text not null,
  created_at datetime default current_timestamp
);
CREATE TABLE note_views (
  id integer primary key,
  user_id integer not null,
  path_id integer not null,
  version integer not null,
  created_at datetime default current_timestamp,
  foreign key (user_id) references users(id) on delete restrict,
  foreign key (path_id, version) references note_versions(path_id, version) on delete restrict
);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20250402131258');
