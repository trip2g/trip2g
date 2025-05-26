-- migrate:up

create table api_keys (
  id integer primary key autoincrement,
  value text not null unique,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete cascade,
  disabled_at datetime,
  disabled_by integer references admins(user_id) on delete restrict,
  description not null default '' -- the form field always has a value
);

create table api_key_log_actions (
  id integer primary key autoincrement,
  name text not null unique
);

create table api_key_log_ips (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  value text not null unique
);

create table api_key_logs (
  api_key_id integer not null references api_keys(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  action_id integer not null references api_key_log_actions(id) on delete restrict,
  ip_id integer not null references api_key_log_ips(id) on delete restrict
);

-- migrate:down

drop table api_key_logs;
drop table api_key_log_ips;
drop table api_key_log_actions;
drop table api_keys;
