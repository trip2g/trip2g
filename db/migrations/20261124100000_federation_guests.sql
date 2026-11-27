-- migrate:up
create table federation_guests (
  id integer primary key autoincrement,
  kid text not null,
  reader_hash text not null,
  user_id integer not null references users(id) on delete cascade,
  created_at datetime not null default current_timestamp,
  unique (kid, reader_hash)
);

-- migrate:down
drop table federation_guests;
