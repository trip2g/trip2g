-- migrate:up

create table tg_chat_subgraph_accesses (
  id integer primary key autoincrement,
  chat_id integer not null references tg_bots_chats(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp
);

-- migrate:down

drop table if exists tg_chat_subgraph_accesses;
