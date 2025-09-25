-- migrate:up

create table git_tokens (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  last_used_at datetime not null default current_timestamp,
  admin_id integer references admins(user_id) on delete restrict,
  value_sha256 text not null unique,
  description text not null default '',
  can_pull boolean default false,
  can_push boolean default true,
  usage_count integer default 0
);

-- migrate:down

drop table git_tokens;
