```bash
export PATH="./scripts:$PATH:$(go env GOPATH)/bin"
```

## Telegram Errors

```
sendscheduledtelegrampublishposts: failed to SendTelegramPublishPostWithTx error="failed to send telegram message to chat 2: failed to send Telegram message: Too Many Requests: retry after 15" note_path_id=98
```

## Conversations

```sql
-- a cron job will trigger dispensers to quotas when now() - interval > last conversation quota
create table conversation_dispensers (
  id integer primary key auto_increment,
  created_at datetime not null default current_timestamp,
  created_by integer not null references admins(user_id) on delete restrict,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  reset_previous_quotas boolean not null default true,
  interval text not null
);

create table conversation_quotas (
  id integer primary key auto_increment,
  created_at datetime not null default current_timestamp,
  subgraph_id integer not null references subgraphs(id) on delete restrict,
  user_id integer not null references users(id) on delete restrict,
  init_conversation_count integer not null default 0,
  current_conversation_count integer not null default 0, -- can be reset to 0
  -- current_conversation_count - used_conversation_count = available conversations
  used_conversation_count integer not null default 0,
);

create table conversations (
  id integer primary key auto_increment,
  created_at datetime not null default current_timestamp,
  quota_id integer not null references conversation_quotas(id) on delete restrict,
  subject text not null
);

create table conversation_messages (
  id integer primary key auto_increment,
  created_at datetime not null default current_timestamp,
  sender_id integer not null references users(id) on delete restrict,
  body text not null
);
```
