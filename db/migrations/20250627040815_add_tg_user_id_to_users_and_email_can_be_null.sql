-- migrate:up
-- Add tg_user_id column to users table and make email nullable
-- Note: No FK constraint since tg_user_profiles.chat_id is not unique

create table users_new (
    id integer primary key,
    email text unique, -- Made nullable but still unique
    created_at datetime not null default current_timestamp,
    last_signin_code_sent_at datetime,
    note_view_count integer default 0,
    tg_user_id integer unique -- Also unique - one account per Telegram user
    -- Note: No FK constraint because tg_user_profiles.chat_id is not unique
);

insert into users_new (id, email, created_at, last_signin_code_sent_at, note_view_count)
select id, email, created_at, last_signin_code_sent_at, note_view_count from users;

drop table users;
alter table users_new rename to users;

-- migrate:down
-- Restore original table structure with email not null unique and remove tg_user_id

create table users_new (
    id integer primary key,
    email text not null unique,
    created_at datetime not null default current_timestamp,
    last_signin_code_sent_at datetime,
    note_view_count integer default 0
);

insert into users_new (id, email, created_at, last_signin_code_sent_at, note_view_count)
select id, email, created_at, last_signin_code_sent_at, note_view_count from users
where email is not null;

drop table users;
alter table users_new rename to users;
