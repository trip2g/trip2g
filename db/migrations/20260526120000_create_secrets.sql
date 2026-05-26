-- migrate:up
CREATE TABLE secrets (
  id          integer primary key autoincrement,
  key         text    not null unique,
  value_crypt blob    not null,
  created_at  datetime not null default (datetime('now')),
  created_by  integer not null references admins(user_id)
);

-- migrate:down
DROP TABLE IF EXISTS secrets;
