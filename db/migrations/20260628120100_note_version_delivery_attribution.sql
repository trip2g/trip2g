-- migrate:up

create table note_version_delivery_attribution (
  note_version_id integer primary key references note_versions(id) on delete cascade,
  delivery_kind text not null,
  delivery_id integer not null
);

create index idx_nvda_delivery on note_version_delivery_attribution(delivery_kind, delivery_id);

-- migrate:down

drop table if exists note_version_delivery_attribution;
