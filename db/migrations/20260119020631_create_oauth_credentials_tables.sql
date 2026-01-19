-- migrate:up
create table google_oauth_credentials (
    id integer primary key,
    name text not null,
    client_id text not null,
    client_secret_encrypted blob not null,
    active boolean not null default false,
    created_at datetime not null default (datetime('now')),
    created_by integer not null references users(id)
);

create table github_oauth_credentials (
    id integer primary key,
    name text not null,
    client_id text not null,
    client_secret_encrypted blob not null,
    active boolean not null default false,
    created_at datetime not null default (datetime('now')),
    created_by integer not null references users(id)
);

-- migrate:down
drop table github_oauth_credentials;
drop table google_oauth_credentials;
