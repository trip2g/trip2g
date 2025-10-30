-- migrate:up

alter table config_versions
  add column robots_txt text not null default 'open';

-- migrate:down

alter table config_versions
  drop column robots_txt;
