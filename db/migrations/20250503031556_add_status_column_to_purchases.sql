-- migrate:up

alter table purchases add column status text not null default 'pending';

create index purchases_status_idx on purchases (status);

-- migrate:down

alter table purchases drop column status;
