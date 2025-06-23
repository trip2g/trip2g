-- migrate:up

create table tg_bots (
  token text not null primary key,
  enabled boolean not null default true,
  name text,
  description text not null default '',
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);

-- migrate:down

drop table tg_bots;
