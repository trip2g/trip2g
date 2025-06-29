-- migrate:up

create table wait_list_email_requests (
  email text primary key,
  created_at datetime not null default current_timestamp,
  note_path_id int not null references note_paths(id) on delete restrict,
  ip text
);

create table wait_list_tg_bot_requests (
  bot_id int not null references tg_bots(id) on delete restrict,
  chat_id int not null,
  created_at datetime not null default current_timestamp,
  note_path_id int not null references note_paths(id) on delete restrict,
  primary key (bot_id, chat_id)
);

-- migrate:down

drop table wait_list_email_requests;
drop table wait_list_tg_bot_requests;
