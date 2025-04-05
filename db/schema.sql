CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
CREATE TABLE note_paths (
  id integer primary key,
  value text not null unique on conflict ignore,
  value_hash text not null unique on conflict fail,
  latest_content_hash text not null,
  created_at datetime default current_timestamp,
  version_count integer not null default 0
);
CREATE TABLE note_versions (
  path_id integer not null,
  version integer not null,
  content text not null,
  created_at datetime default current_timestamp,
  primary key (path_id, version),
  foreign key (path_id) references note_paths(id) on delete restrict
);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20250402131258');
