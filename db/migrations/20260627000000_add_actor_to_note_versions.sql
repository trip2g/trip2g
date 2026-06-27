-- migrate:up

alter table note_versions
add column created_by_user_id integer references users(id) on delete set null;

alter table note_versions
add column created_by_api_key_id integer references api_keys(id) on delete set null;

-- migrate:down

alter table note_versions
drop column created_by_user_id;

alter table note_versions
drop column created_by_api_key_id;
