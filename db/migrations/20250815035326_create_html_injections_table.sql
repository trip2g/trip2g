-- migrate:up

create table html_injections (
  id integer primary key autoincrement,
  created_at datetime not null default current_timestamp,
  active_from datetime,
  active_to datetime,
  description text not null,
  position integer not null default 0,
  placement text not null, -- head / body_end
  content text not null
);

-- migrate:down

drop table html_injections;
