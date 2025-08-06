-- migrate:up

-- SQLite doesn't support dropping columns directly, so we need to recreate the table
-- Create new table without content_type
create table note_assets_new (
  id integer primary key autoincrement,
  absolute_path text not null,
  file_name text not null,
  sha256_hash text not null unique,
  created_at datetime not null default current_timestamp,
  size integer not null default 0
);

-- Copy data from old table
insert into note_assets_new (id, absolute_path, file_name, sha256_hash, created_at, size)
select id, absolute_path, file_name, sha256_hash, created_at, size
from note_assets;

-- Drop old table
drop table note_assets;

-- Rename new table
alter table note_assets_new rename to note_assets;

-- Recreate index
create index idx_note_assets_absolute_path_sha256_hash on note_assets (absolute_path, sha256_hash);

-- migrate:down

-- Recreate the table with content_type column
create table note_assets_new (
  id integer primary key autoincrement,
  absolute_path text not null,
  file_name text not null,
  sha256_hash text not null unique,
  content_type text not null default 'application/octet-stream',
  created_at datetime not null default current_timestamp,
  size integer not null default 0
);

-- Copy data back with default content_type
insert into note_assets_new (id, absolute_path, file_name, sha256_hash, content_type, created_at, size)
select id, absolute_path, file_name, sha256_hash, 'application/octet-stream', created_at, size
from note_assets;

-- Drop the table without content_type
drop table note_assets;

-- Rename back
alter table note_assets_new rename to note_assets;

-- Recreate index
create index idx_note_assets_absolute_path_sha256_hash on note_assets (absolute_path, sha256_hash);
