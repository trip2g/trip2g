-- migrate:up

alter table telegram_publish_sent_messages add column post_type text not null default 'text';

-- migrate:down

alter table telegram_publish_sent_messages drop column post_type;
