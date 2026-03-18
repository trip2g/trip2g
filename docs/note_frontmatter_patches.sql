PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE note_frontmatter_patches (
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
INSERT INTO note_frontmatter_patches VALUES(1,'["**/*.md"]','["demo/**/*.md"]','{free:true}',0,'show all',1,'2026-03-10 12:06:26',1,'2026-03-17 04:13:26');
INSERT INTO note_frontmatter_patches VALUES(2,'["ru/user/**/*.md"]','[]','{left_sidebar:"ru/user/_sidebar.md"}',0,'ru docs sidebar',1,'2026-03-12 09:24:10',1,'2026-03-12 09:25:00');
INSERT INTO note_frontmatter_patches VALUES(3,'["ru/**/*.md"]','[]','{lang:"ru"}',0,'ru lang',1,'2026-03-13 10:11:16',1,'2026-03-13 10:11:16');
INSERT INTO note_frontmatter_patches VALUES(4,'["ru/**/*.md"]','[]',replace('{\n  header:"[[ru/_header]]",\n  footer:"[[ru/_footer]]",\n}','\n',char(10)),0,'ru navigation',1,'2026-03-13 13:08:52',1,'2026-03-15 09:54:18');
INSERT INTO note_frontmatter_patches VALUES(5,'["**/*.md"]','["demo/**/*.md"]','{lang:"en"}',-1,'english root',1,'2026-03-14 00:49:46',1,'2026-03-17 10:05:49');
INSERT INTO note_frontmatter_patches VALUES(6,'["ru/thoughts/**/*.md"]','[]','{left_sidebar:"[[ru/thoughts/_sidebar]]"}',0,'ru thoughts sidebar',1,'2026-03-15 03:53:24',1,'2026-03-15 04:07:31');
INSERT INTO note_frontmatter_patches VALUES(7,'["en/thoughts/**/*.md"]','[]','{left_sidebar:"[[en/thoughts/_sidebar]]"}',0,'en thoughts sidebar',1,'2026-03-15 03:54:00',1,'2026-03-15 04:07:18');
INSERT INTO note_frontmatter_patches VALUES(8,'["en/user/**/*.md"]','[]','{left_sidebar:"[[en/user/_sidebar]]"}',0,'en docs sidebar',1,'2026-03-15 12:56:01',1,'2026-03-15 12:56:01');
INSERT INTO note_frontmatter_patches VALUES(9,'["dev/**/*.md"]','[]','{search:false}',0,'hide dev docs from search',1,'2026-03-16 05:59:26',1,'2026-03-16 05:59:26');
COMMIT;
