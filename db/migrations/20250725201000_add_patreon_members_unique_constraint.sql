-- migrate:up
create unique index unique_patreon_member on patreon_members(patreon_id, campaign_id);

-- migrate:down
drop index unique_patreon_member;