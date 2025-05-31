-- migrate:up

-- Add new columns (nullable first)
alter table user_note_views
add column referer_version_id integer references note_versions(id) on delete cascade;

alter table user_note_views
add column version_id integer references note_versions(id) on delete cascade;

-- Update version_id to the latest version for each path_id
update user_note_views
set version_id = (
    select nv.id 
    from note_versions nv 
    where nv.path_id = user_note_views.path_id 
    order by nv.version desc 
    limit 1
);

-- Now make version_id NOT NULL since all rows have been updated
-- SQLite doesn't support ALTER COLUMN, so we need to recreate the table
create table user_note_views_new (
  user_id int not null references users(id) on delete cascade,
  version_id integer not null references note_versions(id) on delete cascade,
  referer_version_id integer references note_versions(id) on delete cascade,
  created_at datetime not null default current_timestamp
);

-- Copy data from old table
insert into user_note_views_new (user_id, version_id, referer_version_id, created_at)
select user_id, version_id, referer_version_id, created_at
from user_note_views;

-- Drop old table and rename new one
drop table user_note_views;
alter table user_note_views_new rename to user_note_views;

-- migrate:down

-- Add back path_id column
alter table user_note_views
add column path_id int references note_paths(id) on delete cascade;

-- Restore path_id from version_id
update user_note_views
set path_id = (
    select nv.path_id 
    from note_versions nv 
    where nv.id = user_note_views.version_id
);

-- Drop the new columns
alter table user_note_views drop column referer_version_id;
alter table user_note_views drop column version_id;
