-- migrate:up
alter table form_submits add column processed_at datetime;
alter table form_submits add column processed_by integer references users(id);
alter table form_submits add column comment text not null default '';

-- migrate:down
alter table form_submits drop column comment;
alter table form_submits drop column processed_by;
alter table form_submits drop column processed_at;
