-- migrate:up

-- Add new column (nullable first)
alter table purchases
add column price_usd numeric;

-- Copy prices from offers by joining
update purchases
set price_usd = (
    select o.price_usd 
    from offers o 
    where o.id = purchases.offer_id
);

-- SQLite doesn't support ALTER COLUMN, so we need to recreate the table
create table purchases_new (
  id text primary key,
  created_at datetime not null default current_timestamp,
  payment_provider text not null,
  payment_data text not null,
  status text not null,
  offer_id integer not null references offers(id) on delete restrict,
  user_id integer references users(id) on delete set null,
  email text not null,
  price_usd numeric not null
);

-- Copy data from old table
insert into purchases_new (id, created_at, payment_provider, payment_data, status, offer_id, user_id, email, price_usd)
select id, created_at, payment_provider, payment_data, status, offer_id, user_id, email, price_usd
from purchases;

-- Drop old table and rename new one
drop table purchases;
alter table purchases_new rename to purchases;

-- migrate:down

-- SQLite doesn't support dropping columns directly, so recreate table without price_usd
create table purchases_old (
  id text primary key,
  created_at datetime not null default current_timestamp,
  payment_provider text not null,
  payment_data text not null,
  status text not null,
  offer_id integer not null references offers(id) on delete restrict,
  user_id integer references users(id) on delete set null,
  email text not null
);

-- Copy data back without price_usd
insert into purchases_old (id, created_at, payment_provider, payment_data, status, offer_id, user_id, email)
select id, created_at, payment_provider, payment_data, status, offer_id, user_id, email
from purchases;

-- Drop current table and rename old one
drop table purchases;
alter table purchases_old rename to purchases;
