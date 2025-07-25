-- migrate:up
alter table patreon_credentials add column synced_at datetime;

-- migrate:down
alter table patreon_credentials drop column synced_at;