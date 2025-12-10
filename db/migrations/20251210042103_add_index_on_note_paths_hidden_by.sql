-- migrate:up
create index idx_note_paths_hidden_by on note_paths(hidden_by);

-- migrate:down
drop index idx_note_paths_hidden_by;
