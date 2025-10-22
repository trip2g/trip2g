-- migrate:up

create table telegram_publish_tags (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  hidden boolean not null default false,
  label text not null unique
);

create table telegram_publish_chats (
  chat_id integer not null references tg_bot_chats(id) on delete cascade,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);

create table telegram_publish_notes (
  note_path_id integer not null primary key references note_paths(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  publish_at datetime not null,
  published_version_id integer references note_versions(id) on delete restrict,
  published_at datetime
);

create table telegram_publish_note_tags (
  note_path_id integer not null references telegram_publish_notes(note_path_id) on delete cascade,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  primary key (note_path_id, tag_id)
);

-- migrate:down

drop table telegram_publish_note_tags;
drop table telegram_publish_chats;
drop table telegram_publish_notes;
drop table telegram_publish_tags;
