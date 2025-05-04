-- migrate:up

alter table purchases drop column expire_at;
alter table purchases add column email text not null;

-- remove not null from user_id
-- user will register after the successful purchase
alter table purchases drop column user_id;
alter table purchases add column user_id references users(id) on delete restrict;

-- migrate:down

alter table purchases drop column email;
alter table purchases drop column user_id; -- to add the not null constraint
alter table purchases add column user_id references users(id) on delete restrict;
alter table purchases add column expire_at datetime;
