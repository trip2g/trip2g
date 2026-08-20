-- migrate:up

alter table change_webhook_deliveries add column parent_kind text;
alter table change_webhook_deliveries add column parent_id integer;
alter table change_webhook_deliveries add column trace text;
alter table change_webhook_deliveries add column depth_reached integer not null default 0;

alter table cron_webhook_deliveries add column parent_kind text;
alter table cron_webhook_deliveries add column parent_id integer;
alter table cron_webhook_deliveries add column trace text;
alter table cron_webhook_deliveries add column depth_reached integer not null default 0;

create index idx_change_webhook_deliveries_trace on change_webhook_deliveries(trace, created_at);
create index idx_cron_webhook_deliveries_trace on cron_webhook_deliveries(trace, created_at);

-- migrate:down

drop index if exists idx_cron_webhook_deliveries_trace;
drop index if exists idx_change_webhook_deliveries_trace;

alter table cron_webhook_deliveries drop column depth_reached;
alter table cron_webhook_deliveries drop column trace;
alter table cron_webhook_deliveries drop column parent_id;
alter table cron_webhook_deliveries drop column parent_kind;

alter table change_webhook_deliveries drop column depth_reached;
alter table change_webhook_deliveries drop column trace;
alter table change_webhook_deliveries drop column parent_id;
alter table change_webhook_deliveries drop column parent_kind;
