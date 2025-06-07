-- migrate:up

create table not_found_paths (
  id integer primary key autoincrement,
  path text not null unique,
  total_hits integer not null default 1,
  last_hit_at datetime not null default current_timestamp
);

create table not_found_ip_hits (
  ip text primary key,
  total_hits integer not null default 1,
  last_hit_at datetime not null default current_timestamp
);

create table not_found_ignored_patterns (
  id integer primary key autoincrement,
  pattern text not null unique,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);

-- migrate:down

drop table not_found_ip_hits;
drop table not_found_ignored_patterns;
drop table not_found_paths;
