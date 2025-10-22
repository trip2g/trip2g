-- migrate:up

alter table config_versions
add column timezone text not null default 'UTC';

-- migrate:down

alter table config_versions
drop column timezone;
