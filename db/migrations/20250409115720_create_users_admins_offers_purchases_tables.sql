-- migrate:up

create table users (
  id text primary key,
  email text not null unique,
  created_at datetime not null default current_timestamp,
  last_signin_code_sent_at datetime
);

create table admins (
  user_id text primary key references users(id) on delete cascade,
  granted_at datetime not null default current_timestamp,
  granted_by text references admins(user_id)
);

create table offers (
  id text primary key,
  created_at datetime not null default current_timestamp,
  names text not null,-- e.g. "course-a|course-b"
  lifetime text not null, -- e.g. "+600 days"
  price_usd numeric,
  price_rub numeric,
  price_btc numeric,
  starts_at datetime,
  ends_at datetime
);

create table purchases (
  id text primary key,
  user_id text not null references users(id) on delete cascade,
  offer_id text not null references offers(id) on delete restrict,
  expire_at datetime, -- e.g. now() + offers.lifetime
  created_at datetime not null default current_timestamp,
  payment_provider text not null,
  payment_data json not null
);

-- migrate:down

drop table if exists purchases;
drop table if exists offers;
drop table if exists admins;
drop table if exists users;
