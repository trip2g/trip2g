-- migrate:up

alter table federation_secrets add column prev_secret_crypt blob;
alter table federation_secrets add column rotated_at datetime;

-- migrate:down

alter table federation_secrets drop column rotated_at;
alter table federation_secrets drop column prev_secret_crypt;
