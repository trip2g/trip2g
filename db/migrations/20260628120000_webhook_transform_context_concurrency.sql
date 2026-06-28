-- migrate:up

alter table change_webhooks add column transform_jsonnet text not null default '';
alter table change_webhooks add column attach_notes text not null default '[]';
alter table change_webhooks add column concurrency_mode text not null default 'allow_overlap' check (concurrency_mode in ('allow_overlap','skip','queue_one'));

alter table cron_webhooks add column transform_jsonnet text not null default '';
alter table cron_webhooks add column attach_notes text not null default '[]';
alter table cron_webhooks add column concurrency_mode text not null default 'allow_overlap' check (concurrency_mode in ('allow_overlap','skip','queue_one'));

alter table change_webhook_deliveries add column started_at datetime;
alter table change_webhook_deliveries add column heartbeat_at datetime;
alter table change_webhook_deliveries add column tokens_used integer;
alter table change_webhook_deliveries add column steps integer;

alter table cron_webhook_deliveries add column started_at datetime;
alter table cron_webhook_deliveries add column heartbeat_at datetime;
alter table cron_webhook_deliveries add column tokens_used integer;
alter table cron_webhook_deliveries add column steps integer;

create index idx_change_webhook_deliveries_inflight on change_webhook_deliveries(webhook_id, status);
create index idx_cron_webhook_deliveries_inflight on cron_webhook_deliveries(cron_webhook_id, status);

-- migrate:down

drop index if exists idx_cron_webhook_deliveries_inflight;
drop index if exists idx_change_webhook_deliveries_inflight;

alter table cron_webhook_deliveries drop column steps;
alter table cron_webhook_deliveries drop column tokens_used;
alter table cron_webhook_deliveries drop column heartbeat_at;
alter table cron_webhook_deliveries drop column started_at;

alter table change_webhook_deliveries drop column steps;
alter table change_webhook_deliveries drop column tokens_used;
alter table change_webhook_deliveries drop column heartbeat_at;
alter table change_webhook_deliveries drop column started_at;

alter table cron_webhooks drop column concurrency_mode;
alter table cron_webhooks drop column attach_notes;
alter table cron_webhooks drop column transform_jsonnet;

alter table change_webhooks drop column concurrency_mode;
alter table change_webhooks drop column attach_notes;
alter table change_webhooks drop column transform_jsonnet;
