-- migrate:up

-- New unified config tables
create table config_changes (
  id integer primary key autoincrement,
  value_id text not null,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict
);

create index idx_config_changes_value_id on config_changes(value_id);

create table config_string_values (
  change_id integer primary key references config_changes(id) on delete cascade,
  value text not null
);

create table config_bool_values (
  change_id integer primary key references config_changes(id) on delete cascade,
  value boolean not null
);

-- Migrate data from old tables

-- site_title_template
insert into config_changes (id, value_id, created_at, created_by)
select id, 'site_title_template', created_at, created_by
  from config_site_title_templates;

insert into config_string_values (change_id, value)
select id, value from config_site_title_templates;

-- timezone
insert into config_changes (id, value_id, created_at, created_by)
select id + 10000, 'timezone', created_at, created_by
  from config_timezones;

insert into config_string_values (change_id, value)
select id + 10000, value from config_timezones;

-- default_layout
insert into config_changes (id, value_id, created_at, created_by)
select id + 20000, 'default_layout', created_at, created_by
  from config_default_layouts;

insert into config_string_values (change_id, value)
select id + 20000, value from config_default_layouts;

-- robots_txt
insert into config_changes (id, value_id, created_at, created_by)
select id + 30000, 'robots_txt', created_at, created_by
  from config_robots_txts;

insert into config_string_values (change_id, value)
select id + 30000, value from config_robots_txts;

-- show_draft_versions
insert into config_changes (id, value_id, created_at, created_by)
select id + 40000, 'show_draft_versions', created_at, created_by
  from config_show_draft_versions;

insert into config_bool_values (change_id, value)
select id + 40000, value from config_show_draft_versions;

-- Drop old tables
drop table config_site_title_templates;
drop table config_timezones;
drop table config_default_layouts;
drop table config_robots_txts;
drop table config_show_draft_versions;
drop table config_versions;

-- migrate:down

-- Recreate old tables
create table config_versions (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  show_draft_versions boolean not null default false,
  default_layout text not null default '',
  timezone text not null default 'UTC',
  robots_txt text not null default 'closed'
);

create table config_site_title_templates (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  value text not null
);

create table config_timezones (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  value text not null
);

create table config_default_layouts (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  value text not null
);

create table config_robots_txts (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  value text not null
);

create table config_show_draft_versions (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  value boolean not null
);

-- Migrate data back (simplified - loses some history ordering)
insert into config_site_title_templates (created_at, created_by, value)
select c.created_at, c.created_by, v.value
  from config_changes c
  join config_string_values v on v.change_id = c.id
 where c.value_id = 'site_title_template';

insert into config_timezones (created_at, created_by, value)
select c.created_at, c.created_by, v.value
  from config_changes c
  join config_string_values v on v.change_id = c.id
 where c.value_id = 'timezone';

insert into config_default_layouts (created_at, created_by, value)
select c.created_at, c.created_by, v.value
  from config_changes c
  join config_string_values v on v.change_id = c.id
 where c.value_id = 'default_layout';

insert into config_robots_txts (created_at, created_by, value)
select c.created_at, c.created_by, v.value
  from config_changes c
  join config_string_values v on v.change_id = c.id
 where c.value_id = 'robots_txt';

insert into config_show_draft_versions (created_at, created_by, value)
select c.created_at, c.created_by, v.value
  from config_changes c
  join config_bool_values v on v.change_id = c.id
 where c.value_id = 'show_draft_versions';

-- Drop new tables
drop table config_bool_values;
drop table config_string_values;
drop table config_changes;
