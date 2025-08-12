-- migrate:up

create table tg_bot_chat_subgraph_accesses (
  chat_id integer not null references tg_bot_chats(id) on delete cascade,
  user_id integer not null references users(id) on delete restrict,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  joined_at datetime,

  primary key (chat_id, user_id, subgraph_id)
);

-- migrate:down

drop table tg_bot_chat_subgraph_accesses;
