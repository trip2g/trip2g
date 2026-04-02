-- migrate:up
alter table subgraphs add column require_signin boolean not null default false;

-- migrate:down
-- SQLite does not support DROP COLUMN before 3.35.0, so we leave it.
