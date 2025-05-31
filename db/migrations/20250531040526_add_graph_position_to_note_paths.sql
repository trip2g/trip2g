-- migrate:up

alter table note_paths add column graph_position_x real;
alter table note_paths add column graph_position_y real;

-- migrate:down

alter table note_paths drop column graph_position_x;
alter table note_paths drop column graph_position_y;
