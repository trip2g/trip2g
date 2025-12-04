-- migrate:up

alter table telegram_accounts add column app_config text not null default '{}';

-- migrate:down

alter table telegram_accounts drop column app_config;
