-- migrate:up

create table telegram_publish_sent_messages (
  note_path_id integer not null references note_paths(id) on delete restrict,
  chat_id integer not null references tg_bot_chats(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  message_id integer not null,
  primary key (note_path_id, chat_id)
);

-- migrate:down

drop table telegram_publish_sent_messages;
