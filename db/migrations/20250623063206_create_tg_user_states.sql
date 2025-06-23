-- migrate:up

create table tg_user_states (
  chat_id int not null primary key,
  bot_id int not null references tg_bots(id) on delete restrict,
  user_id int references users(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  updated_at datetime not null default current_timestamp,
  state text not null
);

create table tg_user_profiles (
  chat_id int not null,
  created_at datetime not null default current_timestamp,
  first_name text,
  last_name text,
  username text
);

create index tg_user_profiles_chat_id_idx on tg_user_profiles(chat_id);
create unique index tg_user_profiles_first_last_username_idx on tg_user_profiles(first_name, last_name, username);

-- migrate:down

drop table tg_user_profiles;
drop table tg_user_states;
