-- migrate:up
create index if not exists idx_note_version_assets_version_id on note_version_assets(version_id);

-- migrate:down
drop index idx_note_version_assets_version_id;
