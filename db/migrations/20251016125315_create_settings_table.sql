-- migrate:up

create table config_versions (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  show_draft_versions boolean not null default false,
  default_layout text not null default ''
);

-- migrate:down

drop table config_versions;
