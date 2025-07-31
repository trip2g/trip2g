-- migrate:up
alter table boosty_credentials add column expires_at datetime;

-- migrate:down
alter table boosty_credentials drop column expires_at;

