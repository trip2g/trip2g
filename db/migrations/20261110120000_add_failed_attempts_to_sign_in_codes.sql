-- migrate:up
alter table sign_in_codes add column failed_attempts integer not null default 0;

-- migrate:down
alter table sign_in_codes drop column failed_attempts;
