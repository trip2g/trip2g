-- migrate:up

alter table subgraphs
    add column show_unsubgraph_notes_for_paid_users boolean default true;

-- migrate:down

alter table subgraphs
    drop column show_unsubgraph_notes_for_paid_users;
