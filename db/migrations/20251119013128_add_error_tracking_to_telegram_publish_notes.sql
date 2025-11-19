-- migrate:up

alter table telegram_publish_notes add column last_error text;

-- migrate:down

alter table telegram_publish_notes drop column last_error;
