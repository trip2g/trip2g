-- migrate:up

create table audit_logs (
  id integer primary key autoincrement,
  created_at timestamp not null default current_timestamp,
  level int not null default 0,
  message text not null,
  params text not null
);

create index idx_audit_logs_created_at on audit_logs (created_at);

-- migrate:down

drop table if exists audit_logs;
