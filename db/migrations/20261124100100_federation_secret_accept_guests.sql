-- migrate:up
alter table federation_secrets add column accept_guests boolean not null default false;

-- migrate:down
alter table federation_secrets drop column accept_guests;
