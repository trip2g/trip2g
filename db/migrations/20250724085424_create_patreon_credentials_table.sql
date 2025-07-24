-- migrate:up

create table patreon_credentials (
    id serial primary key,
    created_at datetime not null default current_timestamp,
    created_by integer not null references admins(id) on delete restrict,
    deleted_at datetime,
    deleted_by integer references admins(id) on delete restrict,
    creator_access_token text not null
);

-- migrate:down

drop table patreon_credentials;
