-- migrate:up

alter table note_versions add column created_by_delivery_kind text;
alter table note_versions add column created_by_delivery_id integer;

create index idx_note_versions_delivery on note_versions(created_by_delivery_kind, created_by_delivery_id);

-- migrate:down

drop index if exists idx_note_versions_delivery;

alter table note_versions drop column created_by_delivery_id;
alter table note_versions drop column created_by_delivery_kind;
