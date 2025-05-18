-- migrate:up

pragma foreign_keys = on;

create table note_assets (
  id integer primary key autoincrement,
  absolute_path text not null,
  file_name text not null,
  sha256_hash text not null,
  content_type text not null,
  created_at datetime not null default current_timestamp,
  size integer not null,
  unique (absolute_path, sha256_hash)
);

create table note_version_assets (
  asset_id integer not null references note_assets(id) on delete cascade,
  version_id integer not null references note_versions(id) on delete cascade,
  path text not null, -- path in the note for replacement
  created_at datetime not null default current_timestamp,
  primary key (asset_id, version_id, path)
);

-- migrate:down

drop table note_version_assets;
drop table note_assets;
