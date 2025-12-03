-- migrate:up

create table telegram_accounts (
  id integer primary key autoincrement,
  phone text not null unique,
  session_data blob not null,
  display_name text not null default '',
  is_premium integer not null default 0 check (is_premium in (0, 1)),
  enabled integer not null default 1 check (enabled in (0, 1)),
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);

-- migrate:down

drop table telegram_accounts;
