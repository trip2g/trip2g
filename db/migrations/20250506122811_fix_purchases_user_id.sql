-- migrate:up

alter table purchases rename column user_id to old_user_id;
alter table purchases add column user_id integer references users(id) on delete restrict;

update purchases set user_id = old_user_id;

alter table purchases drop column old_user_id;

-- migrate:down

alter table purchases rename column user_id to old_user_id;
alter table purchases add column user_id references users(id) on delete restrict;
update purchases set user_id = old_user_id;
alter table purchases drop column old_user_id;
