-- migrate:up

-- What an agent wants an operator to see about a run: a list of {ts, level, msg,
-- data} entries. Only the first three mean anything here — data is the agent's
-- own bag, which trip2g stores and hands back unread, because an agent's tool
-- vocabulary is its own and may name tools this platform never heard of.
alter table change_webhook_deliveries add column logs text;
alter table cron_webhook_deliveries add column logs text;

-- migrate:down

alter table cron_webhook_deliveries drop column logs;
alter table change_webhook_deliveries drop column logs;
