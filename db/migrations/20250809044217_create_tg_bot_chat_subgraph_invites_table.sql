-- migrate:up

create table tg_bot_chat_subgraph_invites (
  chat_id integer not null references tg_bot_chats(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,

  primary key(chat_id, subgraph_id)
);

-- migrate:down

drop table tg_bot_chat_subgraph_invites;
