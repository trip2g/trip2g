-- migrate:up

create table patreon_members (
  id integer primary key autoincrement,
  patreon_id text not null, -- uuid
  campaign_id integer not null references patreon_campaigns(id) on delete cascade,
  current_tier_id integer references patreon_tiers(id) on delete set null,
  status text not null,
  email text not null
);

-- migrate:down

drop table patreon_members;
