-- migrate:up

create table telegram_publish_instant_chats (
  chat_id integer not null references tg_bot_chats(id) on delete cascade,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);

-- migrate:down

drop table telegram_publish_instant_chats;
