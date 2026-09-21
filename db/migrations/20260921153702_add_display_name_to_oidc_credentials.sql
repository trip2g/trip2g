-- migrate:up
alter table oidc_credentials add column display_name text not null default '';

-- migrate:down
alter table oidc_credentials drop column display_name;
