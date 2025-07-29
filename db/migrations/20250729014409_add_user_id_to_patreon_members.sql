-- migrate:up

alter table patreon_members add column user_id integer references users(id) on delete restrict;

-- migrate:down

alter table patreon_members drop column user_id;
