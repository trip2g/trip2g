-- migrate:up
-- Create new table with bot_id column
create table tg_bot_chats_new (
  id integer primary key autoincrement,
  telegram_id integer not null unique,
  chat_type text not null,
  chat_title text not null,
  added_at datetime not null default current_timestamp,
  removed_at datetime null,
  can_invite boolean not null default false,
  bot_id integer not null
);

-- Copy existing data and set bot_id = 1 for all existing records
insert into tg_bot_chats_new (id, telegram_id, chat_type, chat_title, added_at, removed_at, can_invite, bot_id)
select id, telegram_id, chat_type, chat_title, added_at, removed_at, can_invite, 1
from tg_bot_chats;

-- Drop old table and rename new one
drop table tg_bot_chats;
alter table tg_bot_chats_new rename to tg_bot_chats;

-- Recreate index
create index idx_tg_bot_chats_telegram_id on tg_bot_chats(telegram_id);

-- migrate:down
-- Create table without bot_id column
create table tg_bot_chats_old (
  id integer primary key autoincrement,
  telegram_id integer not null unique,
  chat_type text not null,
  chat_title text not null,
  added_at datetime not null default current_timestamp,
  removed_at datetime null,
  can_invite boolean not null default false
);

-- Copy data back without bot_id
insert into tg_bot_chats_old (id, telegram_id, chat_type, chat_title, added_at, removed_at, can_invite)
select id, telegram_id, chat_type, chat_title, added_at, removed_at, can_invite
from tg_bot_chats;

-- Drop new table and rename old one
drop table tg_bot_chats;
alter table tg_bot_chats_old rename to tg_bot_chats;

-- Recreate index
create index idx_tg_bot_chats_telegram_id on tg_bot_chats(telegram_id);

