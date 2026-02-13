-- migrate:up
create table note_frontmatter_patches (
  id integer primary key autoincrement,
  include_patterns text not null,
  exclude_patterns text not null default '[]',
  jsonnet text not null,
  priority integer not null default 0,
  description text not null default '',
  enabled boolean not null default true,
  created_at datetime not null default (datetime('now')),
  created_by integer not null references admins(user_id) on delete restrict,
  updated_at datetime not null default (datetime('now'))
);

-- migrate:down
drop table note_frontmatter_patches;
