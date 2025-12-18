# Background Job Queues

The application uses multiple background job queues to process asynchronous tasks. Each queue is configured with specific concurrency limits and priorities.

## Queue Types

### Default Queue (`BackgroundDefaultQueue`)

**Queue Name**: `default`
**Concurrency**: 5 workers
**Poll Interval**: 1 second

Used for general application background tasks.

**Jobs:**
- `extract_notion_pages` - Extract and process Notion pages
- `send_sign_in_code` - Send sign-in code emails to users
- `cronjobs:*` - All cron job executions

### Telegram Task Queue (`BackgroundTelegramJobQueue`)

**Queue Name**: `tg_task_jobs`
**Concurrency**: 1 workers
**Poll Interval**: 1 second

Used for telegram-related background processing tasks that don't directly call the Telegram API.

**Jobs:**
- `send_publish_post` - Process and prepare telegram publish posts
- `update_all_chat_telegram_publish_posts` - Update all telegram posts for a specific chat

### Telegram Bot API Queue (`BackgroundTelegramBotAPIQueue`)

**Queue Name**: `tg_bot_api_jobs`
**Concurrency**: 1 worker (to avoid rate limits)
**Poll Interval**: 2 seconds

Used exclusively for Telegram Bot API calls (via telegram-bot-api library). Limited to 1 concurrent worker to respect Telegram rate limits.

**Jobs:**
- `send_message` - Send messages via Telegram Bot API (includes rate limit retry logic)
- `update_message` - Update existing messages via Telegram Bot API (includes rate limit retry logic)
- `update_telegram_post` - Update telegram publish posts via Bot API

### Telegram Account API Queue (`BackgroundTelegramAccountAPIQueue`)

**Queue Name**: `tg_account_api_jobs`
**Concurrency**: 1 worker (to avoid rate limits)
**Poll Interval**: 2 seconds

Used exclusively for Telegram Account API calls via MTProto (tgtd library). These are user account operations, not bot operations. Limited to 1 concurrent worker to respect Telegram rate limits.

**Jobs:**
- `send_account_message` - Send messages via user account (MTProto)
- `update_account_message` - Update existing messages via user account (MTProto)
- `update_telegram_account_post` - Update telegram publish posts via user account

## Job Structure

All background jobs follow a consistent pattern:

```go
const JobID = "job_identifier"
const QueueID = model.BackgroundQueueID
const Priority = 0  // Higher priority = processed first

type JobStruct struct {
    enqueue jobs.EnqueueFunc
}

func New(env jobs.Env) *JobStruct {
    return &JobStruct{
        enqueue: jobs.Register(env, QueueID, JobID, Priority, Resolve),
    }
}

func (j JobStruct) EnqueueJob(ctx context.Context, params Params) error {
    return j.enqueue(ctx, params)
}
```
