-- migrate:up

pragma foreign_keys = on;

create table releases (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  title text not null default '',
  home_note_version_id integer references note_versions(id) on delete restrict,
  is_live boolean not null default false
);

create index idx_releases_is_live on releases(is_live);

create table release_note_versions (
  release_id integer not null references releases(id) on delete cascade,
  note_version_id integer not null references note_versions(id) on delete cascade,
  primary key (release_id, note_version_id)
);

-- migrate:down

drop table release_note_versions;
drop table releases;
