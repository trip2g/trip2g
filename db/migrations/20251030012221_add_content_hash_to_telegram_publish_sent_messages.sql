-- migrate:up
alter table telegram_publish_sent_messages
  add column content_hash text not null default '';

-- migrate:down
alter table telegram_publish_sent_messages
  drop column content_hash;
