-- migrate:up

alter table git_tokens add column disabled_at datetime;
alter table git_tokens add column disabled_by integer references admins(user_id) on delete restrict;

-- migrate:down

alter table git_tokens drop column disabled_by;
alter table git_tokens drop column disabled_at;

