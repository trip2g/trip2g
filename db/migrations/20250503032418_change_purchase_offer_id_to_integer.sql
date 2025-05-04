-- migrate:up

alter table purchases drop column offer_id;
alter table purchases add column offer_id integer not null references offers(id) on delete restrict;

-- migrate:down

alter table purchases drop column offer_id;
alter table purchases add column offer_id text;
