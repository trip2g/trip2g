-- migrate:up

alter table tg_bots add column default_handler text not null default '';

create table tg_user_current_handlers (
  bot_id                 int  not null references tg_bots(id) on delete cascade,
  business_connection_id text not null default '',
  user_id                int  not null,
  value                  text not null default '',
  updated_at             datetime not null default current_timestamp,
  primary key (bot_id, business_connection_id, user_id)
);

create table tg_user_navigation_states (
  bot_id                 int  not null references tg_bots(id) on delete cascade,
  business_connection_id text not null default '',
  user_id                int  not null,
  value                  text not null default '{}',
  updated_at             datetime not null default current_timestamp,
  primary key (bot_id, business_connection_id, user_id)
);

-- migrate:down

drop table tg_user_navigation_states;
drop table tg_user_current_handlers;
-- SQLite has no DROP COLUMN; default_handler stays in place.
