-- migrate:up

create table telegram_publish_sent_account_messages (
  note_path_id integer not null references note_paths(id) on delete restrict,
  account_id integer not null references telegram_accounts(id) on delete restrict,
  telegram_chat_id integer not null,
  created_at datetime not null default current_timestamp,
  message_id integer not null,
  instant integer not null default 0 check (instant in (0, 1)),
  content_hash text not null default '',
  content text not null default '',
  post_type text not null default 'text'
);

create unique index idx_telegram_publish_sent_account_messages_unique
  on telegram_publish_sent_account_messages(note_path_id, account_id, telegram_chat_id)
  where instant = 0;

create index idx_telegram_publish_sent_account_messages_account_id
  on telegram_publish_sent_account_messages(account_id);

create index idx_telegram_publish_sent_account_messages_note_path_id
  on telegram_publish_sent_account_messages(note_path_id);

-- migrate:down

drop table telegram_publish_sent_account_messages;
