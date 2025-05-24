-- migrate:up

create table acme_certs (
  key text primary key,
  value blob,
  updated_at datetime default current_timestamp
);

-- migrate:down

drop table acme_certs;
