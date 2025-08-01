-- migrate:up
create index idx_boosty_members_email on boosty_members(email);
create index idx_patreon_members_email on patreon_members(email);

-- migrate:down
drop index idx_boosty_members_email;
drop index idx_patreon_members_email;

