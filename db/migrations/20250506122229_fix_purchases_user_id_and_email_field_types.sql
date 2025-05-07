-- migrate:up

alter table purchases drop column email;
alter table purchases drop column user_id;
alter table purchases add column email text;
alter table purchases add column user_id integer references users(id) on delete restrict;

-- migrate:down

-- just skip
