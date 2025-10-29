-- migrate:up

-- Create new table without primary key
create table telegram_publish_sent_messages_new (
  note_path_id integer not null references note_paths(id) on delete restrict,
  chat_id integer not null references tg_bot_chats(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  message_id integer not null
);

-- Copy data from old table
insert into telegram_publish_sent_messages_new (note_path_id, chat_id, created_at, message_id)
select note_path_id, chat_id, created_at, message_id
from telegram_publish_sent_messages;

-- Drop old table
drop table telegram_publish_sent_messages;

-- Rename new table to original name
alter table telegram_publish_sent_messages_new rename to telegram_publish_sent_messages;

-- Create indexes for performance
create index idx_telegram_publish_sent_messages_chat_id on telegram_publish_sent_messages(chat_id);
create index idx_telegram_publish_sent_messages_note_path_id on telegram_publish_sent_messages(note_path_id);

-- migrate:down

-- Recreate original table with primary key
create table telegram_publish_sent_messages_new (
  note_path_id integer not null references note_paths(id) on delete restrict,
  chat_id integer not null references tg_bot_chats(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  message_id integer not null,
  primary key (note_path_id, chat_id)
);

-- Copy data (keeping only the latest entry per note_path_id, chat_id)
insert into telegram_publish_sent_messages_new (note_path_id, chat_id, created_at, message_id)
select note_path_id, chat_id, max(created_at) as created_at, message_id
from telegram_publish_sent_messages
group by note_path_id, chat_id;

-- Drop current table
drop table telegram_publish_sent_messages;

-- Rename to original name
alter table telegram_publish_sent_messages_new rename to telegram_publish_sent_messages;
