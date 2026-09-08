-- migrate:up

alter table subgraphs add column teaser boolean not null default false;

-- migrate:down

alter table subgraphs drop column teaser;
