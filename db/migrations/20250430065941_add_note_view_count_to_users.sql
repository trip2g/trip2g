-- migrate:up

alter table users add column note_view_count integer default 0;

-- migrate:down

alter table users drop column note_view_count;
