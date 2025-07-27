-- migrate:up

create table patreon_tier_subgraphs (
  tier_id integer not null references patreon_tiers(id) on delete cascade,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,

  primary key (tier_id, subgraph_id)
);

-- migrate:down

drop table if exists patreon_tier_subgraphs;
