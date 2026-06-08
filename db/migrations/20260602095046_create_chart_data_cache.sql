-- migrate:up
CREATE TABLE chart_data_cache (
  version_id integer not null,
  chart_hash text    not null,
  data_json  text    not null,
  fetched_at integer not null,
  primary key (version_id, chart_hash)
);

-- migrate:down
DROP TABLE IF EXISTS chart_data_cache;
