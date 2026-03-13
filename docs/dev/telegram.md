# Telegram Post Publishing Architecture

## Purpose

The Telegram post publishing system is organized into layers to properly separate I/O operations from database transactions. This prevents long-running external API calls from blocking database transactions.

## Architecture Layers

### Layer 1: High-level cases (Preparation)
**Location**: `internal/case/`

These cases run within database transactions. They:
- Prepare data
- Validate input
- Enqueue jobs to the queue
- **Do NOT** make external API calls (would block transaction)

### Layer 2: Background jobs (Execution)
**Location**: `internal/case/backjob/`

These jobs run outside of transactions. They:
- Execute external API calls (Telegram)
- Update database after successful API call
- Handle retries and errors

## Call Flow

### Sending New Post

```
internal/case/sendtelegrampublishpost
    ↓ (prepares data, converts note to post)
EnqueueSendTelegramPost(params)
    ↓ (adds job to queue)
    ↓
internal/case/backjob/sendtelegrampost
    ↓ (sends to Telegram API)
    ↓ (inserts to DB after success)
    ↓ (if UpdateLinkedPosts=true)
UpdateTelegramPublishPost(notePathID)
```

### Updating Existing Post

```
internal/case/updatetelegrampublishpost
    ↓ (gets sent messages, converts note, checks hash)
    ↓ (for each changed message)
EnqueueUpdateTelegramPost(params)
    ↓ (adds job to queue)
    ↓
internal/case/backjob/updatetelegrampost
    ↓ (edits message in Telegram API)
    ↓ (updates DB after success)
```

## Why This Separation?

### Problem: Mixing I/O and Transactions

```go
// BAD: Long transaction holding database lock
func BadExample(ctx context.Context) error {
    tx.Begin()

    // Database lock held during slow API call
    SendToTelegramAPI()  // Takes 1-5 seconds

    InsertToDB()
    tx.Commit()
}
```

### Solution: Separate Layers

```go
// GOOD: Quick transaction, I/O happens outside
func HighLevelCase(ctx context.Context) error {
    tx.Begin()
    PrepareData()
    EnqueueJob()  // Fast
    tx.Commit()
}

func BackgroundJob(ctx context.Context) error {
    // No transaction held
    SendToTelegramAPI()  // Takes time, but not blocking DB

    // Quick transaction just for update
    UpdateDB()
}
```

## Benefits

1. **Fast transactions**: Database locks released quickly
2. **Reliable delivery**: Jobs persisted in queue, can retry
3. **Non-blocking**: User requests return immediately
4. **Separation of concerns**: Preparation vs execution
5. **Testability**: Each layer can be tested independently

## Queue System

- **Queue**: `telegram_jobs` (using goqite)
- **Runner**: Single runner with limit=1
  - Avoids SQL_BUSY errors
  - Respects Telegram API rate limits by processing sequentially
- **Jobs**:
  - `send_message` → `backjob/sendtelegrampost`
  - `update_message` → `backjob/updatetelegrampost`

## Telegram API Rate Limits

Telegram Bot API has rate limits:
- **Group messages**: ~20 messages per minute
- **Private messages**: ~30 messages per second
- **Editing messages**: Counts towards sending limits

Our queue system respects these limits by:
1. **Sequential processing**: limit=1 ensures one message at a time
2. **Natural throttling**: Queue processes jobs sequentially, preventing bursts
3. **Retry on rate limit**: If rate limited, job fails and retries later

## Example Scenario

User publishes a note to Telegram:

1. **High-level case** (`sendtelegrampublishpost`):
   - Runs in transaction
   - Converts note to Telegram format
   - Enqueues job
   - Transaction commits (fast)
   - Returns to user immediately

2. **Background job** (`sendtelegrampost`):
   - Picked up from queue
   - Sends to Telegram API (may take seconds)
   - If successful, inserts record to DB
   - If failed, job can retry

## Linked Post Updates

When a note is updated, all posts linking to it can be updated:

```
Note A updated
    ↓
sendtelegrampost (sends A, sets UpdateLinkedPosts=true)
    ↓
UpdateTelegramPublishPost(Note B)  // Note B links to A
    ↓
updatetelegrampublishpost (prepares update for B)
    ↓
EnqueueUpdateTelegramPost(params)
    ↓
updatetelegrampost (updates B in Telegram)
```

This creates a cascade of updates while maintaining proper I/O separation at each level.
