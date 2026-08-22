-- migrate:up

alter table subgraphs add column human_description text not null default '';

-- migrate:down

alter table subgraphs drop column human_description;
