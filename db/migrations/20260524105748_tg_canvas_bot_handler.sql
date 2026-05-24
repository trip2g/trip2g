-- migrate:up

alter table tg_bots add column default_canvas text not null default '';

create table tg_user_canvas_states (
  bot_id                 int  not null references tg_bots(id) on delete cascade,
  business_connection_id text not null default '',
  user_id                int  not null,
  canvas_path            text not null,
  current_node           text not null,
  stack                  text not null default '[]',
  last_media             text not null default '',
  message_id             int  not null default 0,
  updated_at             datetime not null default current_timestamp,
  primary key (bot_id, business_connection_id, user_id)
);

create index tg_user_canvas_states_by_path
  on tg_user_canvas_states(bot_id, canvas_path);

-- migrate:down

drop index tg_user_canvas_states_by_path;
drop table tg_user_canvas_states;
-- SQLite has no DROP COLUMN; default_canvas stays in place.
