-- migrate:up
-- Step 1: Create new table with correct structure
create table tg_bot_chats_new (
  id integer primary key autoincrement,
  telegram_id integer not null unique,
  chat_type text not null, -- channel, group, supergroup
  chat_title text not null,
  added_at datetime not null default current_timestamp,
  removed_at datetime null,
  can_invite boolean not null default false
);

-- Step 2: Copy data from old table, using the old id as telegram_id
insert into tg_bot_chats_new (telegram_id, chat_type, chat_title, added_at, removed_at, can_invite)
select id, chat_type, chat_title, added_at, removed_at, can_invite
from tg_bot_chats;

-- Step 3: Create temporary tables for related data with new chat_id references
create table tg_chat_subgraph_accesses_new (
  id integer primary key autoincrement,
  chat_id integer not null,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp
);

create table tg_bot_chat_subgraph_invites_new (
  chat_id integer not null,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  primary key(chat_id, subgraph_id)
);

create table tg_chat_members_new (
  user_id integer not null, -- tg id
  chat_id integer not null,
  created_at datetime not null default current_timestamp,
  primary key (user_id, chat_id)
);

-- Note: tg_user_states uses telegram chat_id directly, not referencing tg_bot_chats
-- So we don't need to update it in this migration

-- Step 4: Copy data from related tables, mapping old chat_id to new id
insert into tg_chat_subgraph_accesses_new (id, chat_id, subgraph_id, created_at)
select tcsa.id, tcbn.id, tcsa.subgraph_id, tcsa.created_at
from tg_chat_subgraph_accesses tcsa
join tg_bot_chats_new tcbn on tcbn.telegram_id = tcsa.chat_id;

insert into tg_bot_chat_subgraph_invites_new (chat_id, subgraph_id, created_at, created_by)
select tcbn.id, tbcsi.subgraph_id, tbcsi.created_at, tbcsi.created_by
from tg_bot_chat_subgraph_invites tbcsi
join tg_bot_chats_new tcbn on tcbn.telegram_id = tbcsi.chat_id;

insert into tg_chat_members_new (user_id, chat_id, created_at)
select tcm.user_id, tcbn.id, tcm.created_at
from tg_chat_members tcm
join tg_bot_chats_new tcbn on tcbn.telegram_id = tcm.chat_id;

-- Step 5: Drop old tables and constraints
drop table tg_chat_subgraph_accesses;
drop table tg_bot_chat_subgraph_invites;
drop table tg_chat_members;
drop table tg_bot_chats;

-- Step 6: Rename new tables to original names
alter table tg_bot_chats_new rename to tg_bot_chats;
alter table tg_chat_subgraph_accesses_new rename to tg_chat_subgraph_accesses;
alter table tg_bot_chat_subgraph_invites_new rename to tg_bot_chat_subgraph_invites;
alter table tg_chat_members_new rename to tg_chat_members;

-- Step 7: Create indexes for performance
create index idx_tg_bot_chats_telegram_id on tg_bot_chats(telegram_id);
create index idx_tg_chat_subgraph_accesses_chat_id on tg_chat_subgraph_accesses(chat_id);
create index idx_tg_bot_chat_subgraph_invites_chat_id on tg_bot_chat_subgraph_invites(chat_id);
create index idx_tg_chat_members_chat_id on tg_chat_members(chat_id);

-- migrate:down

-- This is a complex migration that changes primary keys and foreign key relationships
-- A proper rollback would require:
-- 1. Saving the current autoincrement IDs to telegram_id mapping
-- 2. Recreating tables with old structure
-- 3. Restoring data with original IDs
-- Due to the complexity and potential data loss, we'll make this migration irreversible
select raise(abort, 'This migration cannot be rolled back due to primary key changes');
