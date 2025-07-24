-- migrate:up

create table patreon_campaigns (
  id integer primary key autoincrement,
  credentials_id integer not null references patreon_credentials(id) on delete cascade,
  campaign_id text not null
);

-- migrate:down

drop table patreon_campaigns;
