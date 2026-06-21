-- migrate:up
create table oidc_credentials (
    id integer primary key,
    name text not null,
    issuer text not null,
    client_id text not null,
    client_secret_encrypted blob not null,
    scopes text not null default 'openid email profile',
    auto_provision boolean not null default false,
    allowed_email_domain text not null default '',
    required_group text not null default '',
    active boolean not null default false,
    created_at datetime not null default (datetime('now')),
    created_by integer not null references users(id)
);

-- migrate:down
drop table oidc_credentials;
