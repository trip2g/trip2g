-- migrate:up

create table user_note_views (
  user_id int not null references users(id) on delete cascade,
  path_id int not null references note_paths(id) on delete cascade,
  created_at datetime not null default current_timestamp
);

create table user_note_daily_view_counts (
  user_id int not null references users(id) on delete cascade,
  path_id int not null references note_paths(id) on delete cascade,
  day date not null default (date()),
  count int not null default 0,
  unique (user_id, path_id)
);

-- migrate:down

drop table user_note_views;
drop table user_note_daily_view_counts;
