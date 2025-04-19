-- migrate:up

create table subgraphs (
  id integer primary key autoincrement,
  name text not null unique,
  color text,
  created_at datetime not null default current_timestamp
);

create table user_subgraph_accesses (
  id integer primary key autoincrement,
  user_id integer not null references users(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  purchase_id integer references purchases(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  expires_at datetime
);

-- migrate:down

drop table user_subgraph_accesses;
