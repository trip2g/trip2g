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

### Telegram API Queue (`BackgroundTelegramAPICallQueue`)

**Queue Name**: `tg_api_jobs`
**Concurrency**: 1 worker (to avoid rate limits)
**Poll Interval**: 1 second

Used exclusively for direct Telegram API calls. Limited to 1 concurrent worker to respect Telegram rate limits.

**Jobs:**
- `send_message` - Send messages via Telegram API (includes rate limit retry logic)
- `update_message` - Update existing messages via Telegram API (includes rate limit retry logic)

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
