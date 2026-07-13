-- migrate:up
-- The cron formerly named "apply_git_changes" only materializes the DB→git
-- mirror (it never ingested git→DB — that happens synchronously during git
-- push). Rename the registered row so its configured schedule and execution
-- history (cron_job_executions.job_id FK) are preserved instead of orphaned.
-- Guarded + no-op-safe: skips if the new name already exists or the old row is
-- absent (e.g. fresh installs seed the new name directly).
update cron_jobs
set name = 'materialize_git_mirror'
where name = 'apply_git_changes'
  and not exists (select 1 from cron_jobs where name = 'materialize_git_mirror');

-- migrate:down
update cron_jobs
set name = 'apply_git_changes'
where name = 'materialize_git_mirror'
  and not exists (select 1 from cron_jobs where name = 'apply_git_changes');
