-- migrate:up
create table telegram_chat_usernames (
  telegram_chat_id integer primary key,
  username text not null default '',
  title text not null default '',
  refreshed_at datetime not null default current_timestamp,
  refresh_requested_at datetime,
  last_error text,
  created_at datetime not null default current_timestamp,
  updated_at datetime not null default current_timestamp
);

create index idx_telegram_chat_usernames_refresh
  on telegram_chat_usernames(refresh_requested_at, refreshed_at);

-- migrate:down
drop index idx_telegram_chat_usernames_refresh;

drop table telegram_chat_usernames;
