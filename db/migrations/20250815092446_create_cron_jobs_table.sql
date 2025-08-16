-- migrate:up

create table cron_jobs (
  id integer primary key autoincrement,
  name text not null unique,
  enabled boolean not null default true,
  expression text not null,
  last_exec_at datetime
);

create table cron_job_executions (
  id integer primary key autoincrement,
  job_id int not null references cron_jobs(id) on delete cascade,
  started_at datetime not null default current_timestamp,
  finished_at datetime,
  status int not null default 0,
  report_data text,
  error_message text
);

-- migrate:down

drop table cron_job_executions;
drop table cron_jobs;
