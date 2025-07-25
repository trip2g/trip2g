-- migrate:up

create table patreon_tiers (
  id integer primary key autoincrement,
  campaign_id integer not null references patreon_campaigns(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  missed_at datetime,
  tier_id text not null,
  title text not null,
  amount_cents integer not null,
  attributes text not null,
  unique(campaign_id, tier_id)
);

-- migrate:down

drop table patreon_tiers;