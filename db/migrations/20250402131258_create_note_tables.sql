-- migrate:up

create table note_paths (
  id integer primary key,
  path text not null unique on conflict ignore,
  path_hash text not null unique on conflict fail,
  created_at datetime default current_timestamp,
  version_count integer not null default 0
);

create table note_versions (
  path_id integer not null,
  version integer not null,
  content text not null,
  content_hash text not null,
  created_at datetime default current_timestamp,
  primary key (path_id, version),
  unique (path_id, content_hash) on conflict fail,
  foreign key (path_id) references note_paths(id) on delete restrict
);

create table users (
  id integer primary key,
  email text not null unique,
  password_hash text not null,
  created_at datetime default current_timestamp
);

create table note_views (
  id integer primary key,
  user_id integer not null,
  path_id integer not null,
  version integer not null,
  created_at datetime default current_timestamp,
  foreign key (user_id) references users(id) on delete restrict,
  foreign key (path_id, version) references note_versions(path_id, version) on delete restrict
);

-- migrate:down

drop table note_versions;
drop table note_paths;
