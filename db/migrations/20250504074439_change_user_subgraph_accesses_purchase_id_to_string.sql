-- migrate:up

alter table user_subgraph_accesses drop column purchase_id;
alter table user_subgraph_accesses add column purchase_id text not null references purchases(id) on delete restrict;

-- migrate:down

alter table user_subgraph_accesses drop column purchase_id;
alter table user_subgraph_accesses add column purchase_id int not null references purchases(id) on delete restrict;
