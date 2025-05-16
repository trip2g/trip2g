-- migrate:up

create table note_versions_new (
  id integer primary key autoincrement,
  path_id integer not null,
  version integer not null,
  content text not null,
  created_at datetime not null default current_timestamp,
  unique(path_id, version),
  foreign key (path_id) references note_paths(id) on delete restrict
);

insert into note_versions_new (path_id, version, content, created_at)
select path_id, version, content, created_at from note_versions;

drop table note_versions;

alter table note_versions_new rename to note_versions;

-- migrate:down

create table note_versions_new (
  id integer primary key autoincrement,
  path_id integer not null,
  version integer not null,
  content text not null,
  created_at datetime not null default current_timestamp,
  unique(path_id, version),
  foreign key (path_id) references note_paths(id) on delete restrict
);

insert into note_versions_new (path_id, version, content, created_at)
select path_id, version, content, created_at FROM note_versions;

drop table note_versions;

alter table note_versions_new RENAME TO note_versions;
