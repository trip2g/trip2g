-- migrate:up

alter table subgraphs add column hidden boolean not null default false;

-- migrate:down

alter table subgraphs drop column hidden;
