-- migrate:up

alter table telegram_publish_account_chats add column access_hash text;
alter table telegram_publish_account_instant_chats add column access_hash text;

-- migrate:down

alter table telegram_publish_account_chats drop column access_hash;
alter table telegram_publish_account_instant_chats drop column access_hash;
