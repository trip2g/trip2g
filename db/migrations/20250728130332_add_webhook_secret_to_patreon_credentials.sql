-- migrate:up

alter table patreon_credentials add column webhook_secret text;

-- migrate:down

alter table patreon_credentials drop column webhook_secret;
