-- migrate:up
ALTER TABLE api_keys ADD COLUMN enable_mcp_admin_tools boolean;

-- migrate:down
-- SQLite does not support DROP COLUMN before 3.35.0 – no-op
