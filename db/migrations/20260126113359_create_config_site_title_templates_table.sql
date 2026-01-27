-- migrate:up

create table config_site_title_templates (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  value text not null
);

-- migrate:down

drop table config_site_title_templates;

