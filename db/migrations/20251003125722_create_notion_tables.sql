-- migrate:up

create table notion_integrations (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  enabled boolean not null default true,
  secret_token text not null,
  verification_token text,
  base_path text not null default '/'
);

-- migrate:down

drop table notion_integrations;
