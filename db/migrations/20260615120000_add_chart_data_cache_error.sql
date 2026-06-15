-- migrate:up
ALTER TABLE chart_data_cache ADD COLUMN last_error    text    not null default '';
ALTER TABLE chart_data_cache ADD COLUMN last_error_at integer not null default 0;

-- migrate:down
ALTER TABLE chart_data_cache DROP COLUMN last_error_at;
ALTER TABLE chart_data_cache DROP COLUMN last_error;
