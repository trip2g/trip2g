-- migrate:up

-- Deliveries report what a run cost as an open {unit: amount} object, the unit
-- living in the key: {"tokens": 5186} from an LLM agent, {"usd": 0.004} from
-- something that bills money. tokens_used and steps were the same thing with one
-- executor's vocabulary baked into the schema.
alter table change_webhook_deliveries add column costs text;
alter table cron_webhook_deliveries add column costs text;

alter table change_webhook_deliveries drop column tokens_used;
alter table change_webhook_deliveries drop column steps;
alter table cron_webhook_deliveries drop column tokens_used;
alter table cron_webhook_deliveries drop column steps;

-- migrate:down

alter table change_webhook_deliveries add column tokens_used integer;
alter table change_webhook_deliveries add column steps integer;
alter table cron_webhook_deliveries add column tokens_used integer;
alter table cron_webhook_deliveries add column steps integer;

alter table cron_webhook_deliveries drop column costs;
alter table change_webhook_deliveries drop column costs;
