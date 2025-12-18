-- migrate:up
create table note_uncommitted_paths (
    note_path_id integer primary key references note_paths(id) on delete cascade
);

-- migrate:down
drop table note_uncommitted_paths;
