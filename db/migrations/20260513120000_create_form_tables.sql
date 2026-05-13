-- migrate:up
create table form_submits (
    id              integer primary key,
    note_version_id integer not null references note_versions(id),
    form_id         text not null default '',
    user_id         integer references users(id),
    ip              text not null default '',
    status          text not null default 'visible',
    created_at      datetime not null default current_timestamp
);

create index form_submits_note_version_id on form_submits(note_version_id);

create table form_string_values (
    submit_id  integer not null references form_submits(id) on delete cascade,
    field_name text not null,
    value      text not null,
    primary key (submit_id, field_name)
);

create table form_int_values (
    submit_id  integer not null references form_submits(id) on delete cascade,
    field_name text not null,
    value      integer not null,
    primary key (submit_id, field_name)
);

create table form_bool_values (
    submit_id  integer not null references form_submits(id) on delete cascade,
    field_name text not null,
    value      integer not null,
    primary key (submit_id, field_name)
);

-- migrate:down
drop table if exists form_bool_values;
drop table if exists form_int_values;
drop table if exists form_string_values;
drop index if exists form_submits_note_version_id;
drop table if exists form_submits;
