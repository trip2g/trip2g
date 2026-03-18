-- migrate:up
create table note_version_chunks (
    id           integer primary key autoincrement,
    version_id   integer not null references note_versions(id) on delete cascade,
    chunk_index  integer not null,
    content      text    not null,
    embedding    blob,
    model_id     integer,
    content_hash blob,
    tokens       integer,
    created_at   datetime not null default (datetime('now')),
    unique(version_id, chunk_index)
);
create index note_version_chunks_version_id on note_version_chunks(version_id);

-- migrate:down
drop table note_version_chunks;
