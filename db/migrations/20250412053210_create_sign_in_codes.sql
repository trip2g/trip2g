-- migrate:up

create table sign_in_codes (
  user_id integer not null,
  code integer not null,
  created_at datetime not null default current_timestamp
);

-- migrate:down

drop table sign_in_codes;
