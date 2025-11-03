-- migrate:up

-- Drop existing regular indexes
drop index if exists idx_telegram_publish_sent_messages_chat_id;
drop index if exists idx_telegram_publish_sent_messages_note_path_id;

-- Create partial unique index: uniqueness only for non-instant messages (instant = 0)
-- This allows multiple instant messages for the same chat_id + note_path_id combination,
-- but ensures only one scheduled message exists per chat_id + note_path_id
create unique index idx_telegram_publish_sent_messages_unique_scheduled
on telegram_publish_sent_messages(chat_id, note_path_id)
where instant = 0;

-- Recreate regular indexes for performance
create index idx_telegram_publish_sent_messages_chat_id on telegram_publish_sent_messages(chat_id);
create index idx_telegram_publish_sent_messages_note_path_id on telegram_publish_sent_messages(note_path_id);

-- migrate:down

-- Drop partial unique index
drop index if exists idx_telegram_publish_sent_messages_unique_scheduled;

-- Recreate regular indexes (if they were removed)
create index if not exists idx_telegram_publish_sent_messages_chat_id on telegram_publish_sent_messages(chat_id);
create index if not exists idx_telegram_publish_sent_messages_note_path_id on telegram_publish_sent_messages(note_path_id);
