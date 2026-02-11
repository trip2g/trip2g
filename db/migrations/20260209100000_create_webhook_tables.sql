-- migrate:up

create table change_webhooks (
  id integer primary key autoincrement,
  url text not null,
  include_patterns text not null,
  exclude_patterns text not null default '[]',
  instruction text not null default '',
  secret text not null,
  max_depth integer not null default 1,
  pass_api_key boolean not null default false,
  include_content boolean not null default true,
  timeout_seconds integer not null default 60,
  max_retries integer not null default 0,
  on_create boolean not null default true,
  on_update boolean not null default true,
  on_remove boolean not null default true,
  read_patterns text not null default '["*"]',
  write_patterns text not null default '[]',
  enabled boolean not null default true,
  description text not null default '',
  created_at datetime not null default (datetime('now')),
  created_by integer not null references admins(user_id) on delete restrict,
  updated_at datetime not null default (datetime('now')),
  disabled_at datetime,
  disabled_by integer references admins(user_id) on delete restrict
);

create table change_webhook_deliveries (
  id integer primary key autoincrement,
  webhook_id integer not null references change_webhooks(id) on delete cascade,
  status text not null default 'pending',
  response_status integer,
  attempt integer not null default 1,
  duration_ms integer,
  created_at datetime not null default (datetime('now')),
  completed_at datetime
);

create index idx_change_webhook_deliveries_webhook_created
  on change_webhook_deliveries(webhook_id, created_at);

create table cron_webhooks (
  id integer primary key autoincrement,
  url text not null,
  cron_schedule text not null,
  instruction text not null default '',
  secret text not null,
  pass_api_key boolean not null default false,
  timeout_seconds integer not null default 60,
  max_depth integer not null default 1,
  max_retries integer not null default 0,
  next_run_at datetime,
  read_patterns text not null default '["*"]',
  write_patterns text not null default '[]',
  enabled boolean not null default true,
  description text not null default '',
  created_at datetime not null default (datetime('now')),
  created_by integer not null references admins(user_id) on delete restrict,
  updated_at datetime not null default (datetime('now')),
  disabled_at datetime,
  disabled_by integer references admins(user_id) on delete restrict
);

create table cron_webhook_deliveries (
  id integer primary key autoincrement,
  cron_webhook_id integer not null references cron_webhooks(id) on delete cascade,
  status text not null default 'pending',
  response_status integer,
  attempt integer not null default 1,
  duration_ms integer,
  created_at datetime not null default (datetime('now')),
  completed_at datetime
);

create index idx_cron_webhook_deliveries_webhook_created
  on cron_webhook_deliveries(cron_webhook_id, created_at);

create table webhook_delivery_logs (
  id integer primary key autoincrement,
  delivery_id integer not null,
  kind text not null,
  request_body text,
  response_body text,
  error_message text,
  created_at datetime not null default (datetime('now'))
);

create index idx_wdl_delivery on webhook_delivery_logs(kind, delivery_id);
create index idx_wdl_created on webhook_delivery_logs(created_at);

alter table api_keys add column skip_webhooks boolean not null default false;

-- migrate:down

drop index if exists idx_wdl_created;
drop index if exists idx_wdl_delivery;
drop table if exists webhook_delivery_logs;

drop index if exists idx_cron_webhook_deliveries_webhook_created;
drop table if exists cron_webhook_deliveries;
drop table if exists cron_webhooks;

drop index if exists idx_change_webhook_deliveries_webhook_created;
drop table if exists change_webhook_deliveries;
drop table if exists change_webhooks;
