-- migrate:up

create table patreon_campaigns (
  id integer primary key autoincrement,
  credentials_id integer not null references patreon_credentials(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  missed_at datetime,
  campaign_id text not null,
  attributes text not null,
  unique(credentials_id, campaign_id)
);

-- migrate:down

drop table patreon_campaigns;
