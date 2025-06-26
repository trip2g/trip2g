-- migrate:up

create table tg_chat_members (
  user_id integer not null, -- tg id
  chat_id integer not null,
  created_at datetime not null default current_timestamp,
  primary key (user_id, chat_id)
);

-- migrate:down

drop table if exists tg_chat_members;
