-- migrate:up
alter table telegram_publish_sent_messages
  add column instant integer not null default 0;

-- migrate:down
alter table telegram_publish_sent_messages
  drop column instant;
