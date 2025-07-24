-- migrate:up

create table patreon_credentials (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  deleted_at datetime,
  deleted_by integer references admins(user_id) on delete restrict,
  creator_access_token text not null
);

-- migrate:down

drop table patreon_credentials;
