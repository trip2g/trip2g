### Наброски по пользователям:

```sql
create table users (
  id integer primary key,
  email text not null unique,
  password_hash text not null,
  created_at datetime default current_timestamp
);

create table note_views (
  id integer primary key,
  user_id integer not null,
  path_id integer not null,
  version integer not null,
  created_at datetime default current_timestamp,
  foreign key (user_id) references users(id) on delete restrict,
  foreign key (path_id, version) references note_versions(path_id, version) on delete restrict
);
```
