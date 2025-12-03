-- migrate:up

create table telegram_publish_account_chats (
  account_id integer not null references telegram_accounts(id) on delete cascade,
  telegram_chat_id integer not null,
  tag_id integer not null references telegram_publish_tags(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  primary key (account_id, telegram_chat_id, tag_id)
);

-- migrate:down

drop table telegram_publish_account_chats;
