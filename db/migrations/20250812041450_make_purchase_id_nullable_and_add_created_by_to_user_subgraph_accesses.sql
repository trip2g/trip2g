-- migrate:up

-- Create new table with updated schema
create table user_subgraph_accesses_new (
  id integer primary key autoincrement,
  user_id integer not null references users(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  expires_at datetime,
  revoke_id int references revokes(id) on delete restrict,
  purchase_id text references purchases(id) on delete restrict,
  created_by integer references admins(user_id) on delete restrict
);

-- Copy data from old table to new table
insert into user_subgraph_accesses_new (
  id, user_id, subgraph_id, created_at, expires_at, revoke_id, purchase_id
)
select 
  id, user_id, subgraph_id, created_at, expires_at, revoke_id, purchase_id
from user_subgraph_accesses;

-- Drop old table
drop table user_subgraph_accesses;

-- Rename new table to original name
alter table user_subgraph_accesses_new rename to user_subgraph_accesses;

-- migrate:down

-- Recreate original table structure
create table user_subgraph_accesses_new (
  id integer primary key autoincrement,
  user_id integer not null references users(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  expires_at datetime,
  revoke_id int references revokes(id) on delete restrict,
  purchase_id text not null references purchases(id) on delete restrict
);

-- Copy data back (only records with non-null purchase_id)
insert into user_subgraph_accesses_new (
  id, user_id, subgraph_id, created_at, expires_at, revoke_id, purchase_id
)
select 
  id, user_id, subgraph_id, created_at, expires_at, revoke_id, purchase_id
from user_subgraph_accesses
where purchase_id is not null;

-- Drop modified table
drop table user_subgraph_accesses;

-- Rename back to original name
alter table user_subgraph_accesses_new rename to user_subgraph_accesses;

