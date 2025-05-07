-- migrate:up

create table purchases_new (
  id text primary key,
  created_at datetime not null default current_timestamp,
  payment_provider text not null,
  payment_data json not null,
  status text not null default 'pending',
  offer_id integer not null references offers(id) on delete restrict,
  user_id integer references users(id) on delete restrict,
  email text not null
);

insert into purchases_new (id, created_at, payment_provider, payment_data, status, offer_id, user_id, email)
select id, created_at, payment_provider, payment_data, status, offer_id, user_id, email from purchases;

drop table purchases;
alter table purchases_new rename to purchases;

-- migrate:down

create table purchases_new (
  id text primary key,
  created_at datetime not null default current_timestamp,
  payment_provider text not null,
  payment_data json not null,
  status text not null default 'pending',
  offer_id integer not null references offers(id) on delete restrict,
  user_id integer references users(id) on delete restrict,
  email text
);

insert into purchases_new (id, created_at, payment_provider, payment_data, status, offer_id, user_id, email)
select id, created_at, payment_provider, payment_data, status, offer_id, user_id, email from purchases;

drop table purchases;
alter table purchases_new rename to purchases;
