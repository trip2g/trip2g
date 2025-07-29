-- migrate:up

create table boosty_credentials (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  deleted_at datetime,
  deleted_by integer references admins(user_id) on delete restrict,
  auth_data text not null, -- json from the site cookie
  device_id text not null, -- client_id from the site cookie
  blog_name text not null -- the user page name
);

-- migrate:down

drop table boosty_credentials;
