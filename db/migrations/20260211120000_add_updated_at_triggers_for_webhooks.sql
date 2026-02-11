-- migrate:up

create trigger if not exists trg_change_webhooks_updated_at
after update on change_webhooks
begin
  update change_webhooks set updated_at = datetime('now') where id = new.id;
end;

create trigger if not exists trg_cron_webhooks_updated_at
after update on cron_webhooks
begin
  update cron_webhooks set updated_at = datetime('now') where id = new.id;
end;

-- migrate:down

drop trigger if exists trg_change_webhooks_updated_at;
drop trigger if exists trg_cron_webhooks_updated_at;
