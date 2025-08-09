-- migrate:up
alter table tg_bot_chats add column can_invite boolean not null default false;

-- migrate:down
alter table tg_bot_chats drop column can_invite;
