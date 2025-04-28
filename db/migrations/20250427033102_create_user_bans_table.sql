-- migrate:up

create table user_bans (
  user_id integer primary key references users(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  banned_by integer references admins(id) on delete restrict,
  reason text not null
);

-- migrate:down

drop table user_bans;
