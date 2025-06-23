-- migrate:up

create table tg_user_states (
  chat_id int not null,
  bot_id int not null references tg_bots(id) on delete restrict,
  user_id int references users(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  updated_at datetime not null default current_timestamp,
  update_count int not null default 0,
  value text not null default 'pending',
  data text not null,
  primary key (chat_id, bot_id)
);

create table tg_user_profiles (
  sha256_hash text primary key,
  chat_id int not null,
  bot_id int not null references tg_bots(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  first_name text,
  last_name text,
  username text
);

create index tg_user_profiles_chat_id_idx on tg_user_profiles(chat_id);

-- migrate:down

drop table tg_user_profiles;
drop table tg_user_states;
