-- migrate:up

create table redirects (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  pattern text not null,
  ignore_case boolean not null default true,
  is_regex boolean not null default false,
  target text not null
);

-- migrate:down

drop table redirects;
