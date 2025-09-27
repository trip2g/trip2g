-- migrate:up

create table note_assets_new (
  id integer primary key autoincrement,
  absolute_path text not null,
  file_name text not null,
  sha256_hash text not null,
  created_at datetime not null default current_timestamp,
  size integer not null default 0,
  unique (absolute_path, sha256_hash)
);

insert into note_assets_new
select * from note_assets;

drop table note_assets;

alter table note_assets_new rename to note_assets;

-- migrate:down

create table note_assets_new (
  id integer primary key autoincrement,
  absolute_path text not null,
  file_name text not null,
  sha256_hash text not null unique,
  created_at datetime not null default current_timestamp,
  size integer not null default 0
);

insert into note_assets_new
select * from note_assets;

drop table note_assets;

alter table note_assets_new rename to note_assets;
