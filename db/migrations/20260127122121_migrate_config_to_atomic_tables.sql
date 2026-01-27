-- migrate:up

-- Create atomic config tables
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

-- Migrate data from config_versions to atomic tables
insert into config_timezones (created_at, created_by, value)
select created_at, created_by, timezone
  from config_versions
 order by id;

insert into config_default_layouts (created_at, created_by, value)
select created_at, created_by, default_layout
  from config_versions
 order by id;

insert into config_robots_txts (created_at, created_by, value)
select created_at, created_by, robots_txt
  from config_versions
 order by id;

insert into config_show_draft_versions (created_at, created_by, value)
select created_at, created_by, show_draft_versions
  from config_versions
 order by id;

-- migrate:down

drop table config_show_draft_versions;
drop table config_robots_txts;
drop table config_default_layouts;
drop table config_timezones;
