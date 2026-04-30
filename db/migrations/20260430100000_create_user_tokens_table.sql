-- migrate:up
create table user_tokens (
  id text primary key,
  user_id integer not null references users(id) on delete cascade,
  name text not null default '',
  token_hash text not null unique,
  token_prefix text not null,
  scope text not null default 'all',
  created_at datetime not null default current_timestamp,
  expires_at datetime,
  last_used_at datetime,
  revoked_at datetime
);

create index idx_user_tokens_user_id on user_tokens(user_id);
create index idx_user_tokens_token_hash on user_tokens(token_hash);

-- migrate:down
drop index idx_user_tokens_token_hash;
drop index idx_user_tokens_user_id;
drop table user_tokens;
