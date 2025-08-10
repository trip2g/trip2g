-- migrate:up

alter table users add column created_via text not null default 'unknown';

-- migrate:down

alter table users drop column created_via;
