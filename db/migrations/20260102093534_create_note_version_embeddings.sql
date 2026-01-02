-- migrate:up
create table note_version_embeddings (
    version_id integer primary key references note_versions(id) on delete cascade,
    embedding blob not null,
    model_id integer not null,
    content_hash blob not null,
    tokens integer not null,
    created_at datetime not null default (datetime('now'))
);

create index idx_note_version_embeddings_model_id on note_version_embeddings(model_id);

-- migrate:down
drop table note_version_embeddings;
