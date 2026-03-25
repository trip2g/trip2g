-- migrate:up

create table config_int_values (
  change_id integer primary key references config_changes(id) on delete cascade,
  value integer not null
);

-- migrate:down

drop table config_int_values;
