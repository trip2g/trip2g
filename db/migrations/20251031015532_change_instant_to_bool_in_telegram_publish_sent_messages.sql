-- migrate:up

-- SQLite doesn't support ALTER COLUMN, so we need to recreate the table
create table telegram_publish_sent_messages_new (
  note_path_id integer not null references note_paths(id) on delete restrict,
  chat_id integer not null references tg_bot_chats(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  message_id integer not null,
  instant integer not null default 0 check (instant in (0, 1)),
  content_hash text not null default '',
  content text not null default ''
);

-- Copy data from old table (convert int to bool by keeping the same values)
insert into telegram_publish_sent_messages_new (note_path_id, chat_id, created_at, message_id, instant, content_hash, content)
select note_path_id, chat_id, created_at, message_id, instant, content_hash, content
from telegram_publish_sent_messages;

-- Drop old table
drop table telegram_publish_sent_messages;

-- Rename new table to original name
alter table telegram_publish_sent_messages_new rename to telegram_publish_sent_messages;

-- Recreate indexes
create index idx_telegram_publish_sent_messages_chat_id on telegram_publish_sent_messages(chat_id);
create index idx_telegram_publish_sent_messages_note_path_id on telegram_publish_sent_messages(note_path_id);

-- migrate:down

-- Revert to integer without check constraint
create table telegram_publish_sent_messages_new (
  note_path_id integer not null references note_paths(id) on delete restrict,
  chat_id integer not null references tg_bot_chats(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  message_id integer not null,
  instant integer not null default 0,
  content_hash text not null default '',
  content text not null default ''
);

-- Copy data back
insert into telegram_publish_sent_messages_new (note_path_id, chat_id, created_at, message_id, instant, content_hash, content)
select note_path_id, chat_id, created_at, message_id, instant, content_hash, content
from telegram_publish_sent_messages;

-- Drop current table
drop table telegram_publish_sent_messages;

-- Rename to original name
alter table telegram_publish_sent_messages_new rename to telegram_publish_sent_messages;

-- Recreate indexes
create index idx_telegram_publish_sent_messages_chat_id on telegram_publish_sent_messages(chat_id);
create index idx_telegram_publish_sent_messages_note_path_id on telegram_publish_sent_messages(note_path_id);
