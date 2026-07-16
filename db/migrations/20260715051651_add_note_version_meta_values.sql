-- migrate:up

select sqlite_compileoption_used('ENABLE_JSON1');

create table note_version_frontmatters (
  version_id integer not null primary key references note_versions (id) on delete cascade,
  data string not null check (json_valid(data))
);

create table note_version_frontmatter_key_values (
  value string not null primary key,
  created_by_version_id integer not null references note_versions (id) on delete restrict,
  hidden_at datetime -- live and latest notes don't contain this key anymore
);

create table note_version_frontmatter_keys (
  note_version_id integer not null references note_versions (id) on delete cascade,
  key_id string not null references note_version_frontmatter_key_values (value) on delete cascade,
  unique (note_version_id, key_id)
);

create index note_version_frontmatter_keys_key_id_idx
  on note_version_frontmatter_keys (key_id);

-- A typed EAV schema for every frontmatter key would be overengineered here.
-- Keep the canonical frontmatter JSON in a separate table: note_versions stays
-- narrow for hot queries, while callers fetch the full metadata only when they
-- need it. Add targeted projections or indexes later for frequent predicates
-- such as fleet_id.
-- it's not for hot path logic.

-- create table note_version_meta_keys (
--   id integer primary key autoincrement,
--   key string not null unique,
--   -- join note_versions to get created_atc
--   created_by_version_id integer not null references note_versions (id) on delete restrict,
--   -- live and latest notes don't contain this key anymore
--   hidden_at datetime
-- );
--
-- create table note_version_meta_string_values (
--   id integer primary key autoincrement,
--   version_id integer not null references note_versions (id) on delete cascade,
--   key_id integer not null references note_version_meta_keys (id) on delete restrict,
--   value string not null,
--   unique (version_id, key_id)
-- );
--
-- -- number json type: ints & floats
-- create table note_version_meta_number_values (
--   id integer primary key autoincrement,
--   version_id integer not null references note_versions (id) on delete cascade,
--   key_id integer not null references note_version_meta_keys (id) on delete restrict,
--   value decimal not null,
--   unique (version_id, key_id)
-- );
--
-- create table note_version_meta_bool_values (
--   id integer primary key autoincrement,
--   version_id integer not null references note_versions (id) on delete cascade,
--   key_id integer not null references note_version_meta_keys (id) on delete restrict,
--   value boolean not null,
--   unique (version_id, key_id)
-- );
--
-- -- other json types
-- create table note_version_meta_json_values (
--   id integer primary key autoincrement,
--   version_id integer not null references note_versions (id) on delete cascade,
--   key_id integer not null references note_version_meta_keys (id) on delete restrict,
--   value string not null,
--   unique (version_id, key_id)
-- );

-- migrate:down

-- drop table note_version_meta_number_values;
-- drop table note_version_meta_bool_values;
-- drop table note_version_meta_json_values;
-- drop table note_version_meta_string_values;
-- drop table note_version_meta_keys;

drop table note_version_frontmatter_keys;
drop table note_version_frontmatter_key_values;
drop table note_version_frontmatters;
