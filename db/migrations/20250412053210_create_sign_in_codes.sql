-- migrate:up

create table sign_in_codes (
  user_id integer not null,
  code text not null,
  created_at datetime not null default current_timestamp
);

create index idx_sign_in_codes_user_id on sign_in_codes(user_id);

-- migrate:down

drop table sign_in_codes;
