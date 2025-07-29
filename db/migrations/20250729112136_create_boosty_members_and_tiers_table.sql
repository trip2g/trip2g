-- migrate:up

create table boosty_tiers (
  id integer primary key autoincrement,
  boosty_id integer not null,
  created_at datetime not null default current_timestamp,
  missed_at datetime,
  name text not null,
  data text not null
);

create table boosty_tier_subgraphs (
  tier_id integer not null references boosty_tiers(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,

  primary key (tier_id, subgraph_id)
);

create table boosty_members (
  id integer primary key autoincrement,
  boosty_id integer not null,
  created_at datetime not null default current_timestamp,
  missed_at datetime,
  email text not null,
  status text not null,
  data text not null
);

-- migrate:down

drop table boosty_tier_subgraphs;
drop table boosty_members;
drop table boosty_tiers;
