-- migrate:up

create table revokes (
  id integer primary key autoincrement,
  target_type text not null,
  target_id integer not null,
  created_at datetime not null default current_timestamp,
  by admin_id integer not null references admins(id) on delete restrict,
  reason text
);

alter table user_subgraph_accesses add column revoke_id int references revokes(id) on delete restrict;

-- migrate:down

alter table user_subgraph_accesses drop column revoke_id;

drop table revokes;
