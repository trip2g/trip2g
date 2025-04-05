-- migrate:up

create table note_paths (
  id integer primary key,
  value text not null unique on conflict ignore,
  value_hash text not null unique on conflict fail,
  latest_content_hash text not null,
  created_at datetime default current_timestamp,
  version_count integer not null default 0
);

create table note_versions (
  path_id integer not null,
  version integer not null,
  content text not null,
  created_at datetime default current_timestamp,
  primary key (path_id, version),
  foreign key (path_id) references note_paths(id) on delete restrict
);

-- migrate:down

drop table note_versions;
drop table note_paths;
