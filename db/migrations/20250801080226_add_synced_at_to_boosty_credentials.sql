-- migrate:up
alter table boosty_credentials add column synced_at datetime;

-- migrate:down
alter table boosty_credentials drop column synced_at;

