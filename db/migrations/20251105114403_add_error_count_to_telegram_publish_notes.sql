-- migrate:up

alter table telegram_publish_notes add column error_count integer not null default 0;

-- migrate:down

alter table telegram_publish_notes drop column error_count;
