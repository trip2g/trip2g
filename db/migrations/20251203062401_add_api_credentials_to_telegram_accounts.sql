-- migrate:up

alter table telegram_accounts add column api_id integer not null default 0;
alter table telegram_accounts add column api_hash text not null default '';

-- migrate:down

alter table telegram_accounts drop column api_id;
alter table telegram_accounts drop column api_hash;
