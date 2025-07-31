-- migrate:up
alter table boosty_members add column user_id integer references users(id) on delete restrict;

-- migrate:down
alter table boosty_members drop column user_id;

