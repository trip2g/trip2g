-- migrate:up

drop table telegram_custom_emojies;

-- migrate:down

create table telegram_custom_emojies (
  id text not null primary key,
  base64_data text not null,
  created_at datetime not null default current_timestamp
);
