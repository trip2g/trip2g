-- migrate:up
create table federation_secrets (
  id integer primary key autoincrement,
  kid text not null,
  secret_crypt blob not null,
  kb_url text,
  description text,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  revoked_at datetime
);

create index idx_federation_secrets_kid
  on federation_secrets(kid);

create index idx_federation_secrets_kb_url
  on federation_secrets(kb_url) where kb_url is not null;

create table federation_secret_subgraphs (
  kid text not null,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  primary key (kid, subgraph_id)
);

-- migrate:down
drop table federation_secret_subgraphs;

drop index idx_federation_secrets_kb_url;
drop index idx_federation_secrets_kid;

drop table federation_secrets;
