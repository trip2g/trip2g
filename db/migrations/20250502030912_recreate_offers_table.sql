-- migrate:up

drop table offers;

create table offers (
  id integer primary key autoincrement,
  public_id text not null unique,
  created_at datetime not null default current_timestamp,
  lifetime text, -- e.g. "+600 days", null means no expiration
  price_usd numeric,
  starts_at datetime,
  ends_at datetime
);

create table offer_subgraphs (
  offer_id integer not null references offers(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  primary key (offer_id, subgraph_id)
);

-- migrate:down

drop table offer_subgraphs;
