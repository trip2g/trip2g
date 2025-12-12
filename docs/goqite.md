# goqite Job Queue

This project uses [goqite](https://github.com/maragudk/goqite) for background job processing.

## Important: Separate Long-Running and Short-Running Jobs

**CRITICAL**: Long-running jobs (minutes to hours) MUST be placed in a separate queue from short-running jobs.

### Why?

While a job is running, goqite periodically calls `Extend()` to prevent the message from being re-processed by another worker. The extend interval is `Extend - Extend/5` (default: 4 seconds for 5s Extend).

For a job running 1 hour with default settings:
- `Extend()` is called every 4 seconds
- That's 900 UPDATE queries to the database
- Creates GC pressure from object allocations

### Solution

Create separate queues with appropriate settings:

```go
// Short-running jobs (seconds to minutes)
shortQueue := createQueue(ctx, "short_jobs", jobs.NewRunnerOpts{
    Limit:        5,
    PollInterval: time.Second * 5,
    // Default Extend: 5s (calls Extend every 4s)
})

// Long-running jobs (minutes to hours)
longQueue := createQueue(ctx, "long_jobs", jobs.NewRunnerOpts{
    Limit:        1,
    PollInterval: time.Second * 30,
    Extend:       time.Minute * 1,  // Calls Extend every 48s instead of 4s
})
```

### Current Queues

| Queue | PollInterval | Extend | Use Case |
|-------|-------------|--------|----------|
| `global_jobs` | 3s | 5s (default) | General background tasks |
| `tg_api_jobs` | 5s | 5s (default) | Telegram API calls (rate limited) |
| `tg_task_jobs` | 5s | 5s (default) | Telegram processing tasks |
| `tg_long_jobs` | 30s | 60s | Channel imports, long operations |

### PollInterval vs Extend

- **PollInterval**: How often the queue checks for new jobs (affects pickup latency when queue is empty)
- **Extend**: How long before a running job's message can be re-received (affects DB write frequency during job execution)

For long-running jobs:
- Higher `PollInterval` is acceptable (30s delay for starting an hour-long job is fine)
- Higher `Extend` reduces database writes during execution
