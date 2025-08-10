-- migrate:up

create table tg_attach_codes (
  user_id integer not null references users(id) on delete cascade,
  bot_id integer not null references tg_bots(id) on delete restrict,
  code text not null unique,
  created_at datetime not null default current_timestamp
);

-- migrate:down

drop table if exists tg_attach_codes;
