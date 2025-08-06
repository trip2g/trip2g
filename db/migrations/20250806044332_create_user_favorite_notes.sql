-- migrate:up

create table user_favorite_notes (
  user_id integer not null references users(id) on delete cascade,
  note_version_id integer not null references note_versions(id) on delete restrict,
  created_at datetime not null default current_timestamp,

  primary key (user_id, note_version_id)
);

-- migrate:down

drop table user_favorite_notes;
