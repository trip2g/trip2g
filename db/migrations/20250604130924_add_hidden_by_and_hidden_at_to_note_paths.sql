-- migrate:up

alter table note_paths
add column hidden_by integer references admins(user_id) on delete restrict;

alter table note_paths
add column hidden_at datetime;

-- migrate:down

alter table note_paths
drop column hidden_by;

alter table note_paths
drop column hidden_at;
