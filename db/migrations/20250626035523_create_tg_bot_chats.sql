-- migrate:up

create table tg_bot_chats (
  id int primary key,
  chat_type string not null, -- channel, group, supergroup
  chat_title string not null,
  added_at datetime not null default current_timestamp,
  removed_at datetime null
);

-- migrate:down

drop table tg_bot_chats;
