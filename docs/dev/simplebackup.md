# Simple Backup System

## Overview

The Simple Backup system provides a lightweight alternative to Litestream for free-tier users. It reduces memory overhead by using hourly backups instead of real-time replication, accepting a 1-hour Recovery Point Objective (RPO).

**Key Features:**
- Hourly automatic backups to S3-compatible storage (MinIO, AWS S3, etc.)
- Automatic database restoration on startup
- Graceful shutdown backup for instance migration
- Retains 3 most recent backups
- Pseudo-stateless SQLite deployment

**Trade-offs:**
- **Memory**: Low (no replication overhead)
- **RPO**: Up to 1 hour (vs. near-zero with Litestream)
- **Startup time**: 10-60 seconds for restore
- **Use case**: Free tier users with 1-50MB databases

## Architecture

```
internal/simplebackup/
├── backup.go          # VACUUM INTO + gzip + upload + cleanup
├── restore.go         # Download + integrity check + atomic placement
└── manager.go         # Lifecycle orchestration

internal/case/cronjob/simplebackup/
├── resolve.go         # Hourly backup business logic
└── job.go             # Cron job registration (every hour at :00)
```

### Storage Structure

**S3 Path:** `{bucket}/{prefix}/db-backup-{unix-timestamp}.db.gz`

**Example:**
```
s3://mybucket/backups/db-backup-1732435200.db.gz
s3://mybucket/backups/db-backup-1732438800.db.gz
s3://mybucket/backups/db-backup-1732442400.db.gz
```

**Retention:** 3 most recent backups (older backups auto-deleted after each upload)

## How It Works

### Backup Process

1. **Check for concurrent backup**
   - Uses mutex to prevent overlapping backups
   - Returns error immediately if backup already in progress

2. **Create transactionally-safe backup**
   ```sql
   VACUUM INTO '/tmp/backup-{timestamp}.db'
   ```
   - Captures consistent snapshot from single transaction
   - Safer than `.backup` under high write load

3. **Compress**
   ```bash
   gzip /tmp/backup-{timestamp}.db
   ```
   - SQLite B-tree databases compress ~60-80%
   - Reduces storage costs and upload time

4. **Upload to S3 storage**
   ```
   PUT {bucket}/{prefix}/db-backup-{unix-timestamp}.db.gz
   ```
   - Uses existing `miniostorage` client (supports any S3-compatible storage)
   - Filename includes unix timestamp for ordering

5. **Cleanup old backups**
   - Lists all `db-backup-*.db.gz` files
   - Sorts by timestamp (parsed from filename)
   - Deletes all except 3 most recent

6. **Cleanup temp files**
   - Removes local temp files whether upload succeeded or failed

### Restore Process

**Triggered on startup if local database file does not exist**

1. **Check if local database exists**
   - If `config.DatabaseFile` exists → skip restore
   - If missing → proceed with restore

2. **List available backups from S3 storage**
   ```
   LIST {bucket}/{prefix}/db-backup-*.db.gz
   ```
   - If no backups found → Skip restore, start with empty database
   - If S3 storage unavailable → FATAL error, prevent startup

3. **Select most recent backup**
   - Parse unix timestamp from each filename
   - Sort descending, select latest

4. **Download and decompress**
   ```
   GET db-backup-{timestamp}.db.gz
   gunzip → /tmp/restore-{timestamp}.db
   ```

5. **Integrity check**
   ```sql
   PRAGMA integrity_check
   ```
   - Fatal error if check fails (corrupted backup)
   - Does NOT fallback to older backups (fail immediately)

6. **Atomic placement**
   ```bash
   mv /tmp/restore-{timestamp}.db {config.DatabaseFile}
   ```
   - Atomic rename ensures no partial DB file

### Graceful Shutdown Backup

**Triggered on SIGTERM/SIGINT when `--simple-backup` flag is set**

1. Receives shutdown signal
2. Calls `simplebackup.Backup()` with 30-second timeout
3. Creates new backup with current timestamp
4. Uploads to MinIO (ensures instance on new node has latest state)
5. Cleanup deletes old backups (keeps 3 most recent)
6. If backup fails: logs error but continues shutdown (non-blocking)

## Configuration

### CLI Flag

```bash
./server --simple-backup
```

**When enabled:**
- Hourly cronjob is registered
- Startup restore logic activates
- Graceful shutdown backup triggers

**When disabled:**
- No backup operations occur
- Use Litestream or other backup solution instead

### MinIO Configuration

Uses existing `internal/miniostorage` configuration:

**Environment Variables:**
```bash
MINIO_ENDPOINT=s3.amazonaws.com
MINIO_ACCESS_KEY=your-access-key
MINIO_SECRET_KEY=your-secret-key
MINIO_BUCKET=mybucket
MINIO_PREFIX=backups
MINIO_USE_SSL=true
```

**Backup location:** `{MINIO_BUCKET}/{MINIO_PREFIX}/db-backup-{timestamp}.db.gz`

### Database Path

Uses `config.DatabaseFile` from `internal/appconfig`

Typical value: `/var/lib/app/database.db`

## Usage

### Starting with Simple Backup

```bash
# Start application with simple backup enabled
./server --simple-backup

# Logs on startup (if database missing):
[SIMPLE-BACKUP] Local database not found at /var/lib/app/database.db
[SIMPLE-BACKUP] Listing backups from MinIO: s3://mybucket/backups/
[SIMPLE-BACKUP] Found 3 backups, selecting latest: db-backup-1732442400.db.gz
[SIMPLE-BACKUP] Downloading 2.1MB from MinIO...
[SIMPLE-BACKUP] Decompressed to 8.2MB
[SIMPLE-BACKUP] Running integrity check... ok
[SIMPLE-BACKUP] Restore completed in 2.4s
[SERVER] Starting application...
```

### Hourly Backup

```bash
# Automatically runs every hour at :00
[SIMPLE-BACKUP] Starting hourly backup...
[SIMPLE-BACKUP] VACUUM INTO completed: 8.2MB
[SIMPLE-BACKUP] Compressed to 2.1MB (74% reduction)
[SIMPLE-BACKUP] Uploaded to s3://mybucket/backups/db-backup-1732446000.db.gz
[SIMPLE-BACKUP] Cleanup: keeping 3 most recent backups, deleted 1 old backup
[SIMPLE-BACKUP] Backup completed in 1.2s
```

### Graceful Shutdown

```bash
# SIGTERM received
[SERVER] Received shutdown signal
[SIMPLE-BACKUP] Starting shutdown backup...
[SIMPLE-BACKUP] Backup completed in 0.8s
[SERVER] Shutdown complete
```

## Error Handling

### Fatal Errors (Prevent Startup)

**S3-compatible storage unavailable:**
```
[SIMPLE-BACKUP] FATAL: Cannot connect to S3 storage at s3://mybucket
[SIMPLE-BACKUP] FATAL: Verify MINIO_ENDPOINT, MINIO_ACCESS_KEY, and network connectivity
Exit code: 1
```

**Corrupted backup (when attempting restore):**
```
[SIMPLE-BACKUP] FATAL: Integrity check failed on db-backup-1732442400.db.gz
[SIMPLE-BACKUP] FATAL: Backup file is corrupted. Manual intervention required.
[SIMPLE-BACKUP] ERROR: Database corruption detected
Exit code: 1
```

### Non-Fatal Errors

**Concurrent backup attempt:**
```
[SIMPLE-BACKUP] ERROR: Backup already in progress, skipping this attempt
[CRONJOB] simplebackup: execution failed: backup in progress
```

**Shutdown backup failure:**
```
[SIMPLE-BACKUP] ERROR: Shutdown backup failed: S3 storage connection timeout
[SIMPLE-BACKUP] WARNING: Continuing shutdown without backup
[SERVER] Shutdown complete
```

**No backups found on first startup (when S3 storage is accessible):**
```
[SIMPLE-BACKUP] Local database not found at /var/lib/app/database.db
[SIMPLE-BACKUP] No backups found in S3 storage at s3://mybucket/backups/
[SIMPLE-BACKUP] Starting with fresh database, first backup will run in next hour
[SERVER] Starting application...
```

### Error Recovery

**Corrupted backup:**
1. Administrator must investigate MinIO storage
2. Check S3 versioning if enabled
3. Restore from older backup manually:
   ```bash
   # Download second-newest backup
   aws s3 cp s3://mybucket/backups/db-backup-1732438800.db.gz /tmp/
   gunzip /tmp/db-backup-1732438800.db.gz
   sqlite3 /tmp/db-backup-1732438800.db 'PRAGMA integrity_check'
   mv /tmp/db-backup-1732438800.db /var/lib/app/database.db
   ```

**First deployment without existing backups:**
- Application starts normally with empty database
- First hourly backup creates initial snapshot
- No manual intervention required

## Testing

### Unit Tests

**Location:** `internal/simplebackup/backup_test.go`

**Run:**
```bash
go test ./internal/simplebackup/...
```

**Test cases:**
- `TestPerformBackup_Success` — Creates real SQLite DB, verifies backup uploads valid gzipped SQLite
- `TestPerformBackup_ConcurrentBlocked` — Second concurrent backup returns "already in progress" error
- `TestPerformBackup_NilDB` — Returns error when DB is nil
- `TestPerformBackup_UploadFails` — Upload failure returns error, temp file cleaned up
- `TestPerformBackup_RetentionDeletesOldest` — Verifies oldest backups deleted when over retention limit
- `TestPerformBackup_RetentionNoDeleteWhenUnderLimit` — No deletion when under retention count
- `TestRestoreOnStartup_SkipsWhenDBExists` — No storage calls when local DB already exists
- `TestRestoreOnStartup_NoBackups` — Returns nil (fresh start) when no backups in storage
- `TestRestoreOnStartup_Success` — Restores DB from gzipped backup, verifies data integrity
- `TestRestoreOnStartup_IntegrityCheckFails` — Returns error when downloaded backup fails integrity check

### E2E Backup/Restore Test

**Location:** `scripts/test-backup.sh`

**Run after `scripts/test-e2e.sh`** (requires running containers):
```bash
# First run e2e suite (populates data, keeps containers running)
./scripts/test-e2e.sh

# Then run backup/restore test
./scripts/test-backup.sh
```

**Test flow:**
1. Verifies app container is running (from test-e2e.sh)
2. Stops app gracefully (`docker compose stop` → SIGTERM → shutdown backup → MinIO)
3. Deletes local database file
4. Starts app again (startup restore → downloads from MinIO)
5. Waits for health check
6. Runs Playwright smoke/vault tests to verify all data is intact

### Manual Testing

**Test backup:**
```bash
# Enable simple backup
./server --simple-backup &

# Wait for hourly backup or trigger shutdown
kill -SIGTERM $PID

# Verify backup exists in MinIO
aws s3 ls s3://mybucket/backups/
```

**Test restore:**
```bash
# Remove local database
rm /var/lib/app/database.db

# Start with simple backup (should restore)
./server --simple-backup

# Verify application starts successfully
curl http://localhost:8080/health
```

## Monitoring

### Recommended Metrics

**Backup operations:**
- `simple_backup_duration_seconds` (histogram) - Track backup performance
- `simple_backup_size_bytes` (gauge) - Monitor database growth
- `simple_backup_success_total` (counter) - Count successful backups
- `simple_backup_failure_total` (counter) - Alert on failures

**Restore operations:**
- `simple_restore_duration_seconds` (histogram) - Track restore performance
- `simple_restore_success_total` (counter) - Count successful restores
- `simple_restore_failure_total` (counter) - Alert on failures

### Health Check Endpoint

**Endpoint:** `GET /health/backup-status`

**Response:**
```json
{
  "enabled": true,
  "last_backup": "2024-11-24T12:00:00Z",
  "last_backup_size_bytes": 2185728,
  "next_backup": "2024-11-24T13:00:00Z",
  "backup_count": 3,
  "restored_on_startup": false
}
```

### Dead Man's Snitch (Optional)

Add monitoring to detect backup failures:

```go
// After successful backup
http.Get("https://nosnch.in/xxxxxxxxxx")
```

If backups stop, Dead Man's Snitch alerts you.

## Nomad Deployment Considerations

### Behavior During Restart

| Scenario | Behavior | Notes |
|----------|----------|-------|
| **Restart on same allocation** (local DB exists) | Restore skipped | `os.Stat` check returns early — existing data preserved |
| **Fresh allocation** (no local DB) | Downloads from MinIO | `RestoreOnStartup` fetches latest backup |
| **First ever deployment** (no backups in MinIO) | Starts with empty DB | Logged as warning, not an error |

### Shutdown Backup Timeout

The graceful shutdown backup uses a **30-second context timeout** (`cmd/server/main.go`).

**Risk:** For large databases (>100MB) on slow networks, 30 seconds may not be sufficient.

**Recommendation:** Set Nomad `kill_timeout` to at least **45 seconds** to give the backup time to complete before Nomad sends SIGKILL:

```hcl
task "app" {
  kill_timeout = "60s"
  kill_signal  = "SIGTERM"
}
```

The default Nomad `kill_timeout` is 5 seconds — too short for backup completion.

### SIGKILL Behavior

If Nomad sends SIGKILL after `kill_timeout`, any in-flight backup is **silently lost**. This is acceptable because:
- Hourly cron backups provide a 1-hour safety net
- The shutdown backup is best-effort, not guaranteed
- The worst case is losing up to ~1 hour of data, not the entire database

### SIGTERM During Backup Phases

| Phase | Interrupted Behavior |
|-------|---------------------|
| **During VACUUM INTO** | Temp file left behind; source DB unharmed (VACUUM INTO is atomic). Temp file cleaned by `defer os.Remove()` on next run. |
| **During gzip + upload** | Partial object NOT committed to S3 (MinIO requires full upload). No corrupt backup created. Cron backup provides recovery. |

### Known Limitations

1. **No automatic fallback to older backups** — if the latest backup fails integrity check, the app panics. Manual recovery required (see Troubleshooting).
2. **No retry on restore failure** — transient MinIO errors cause startup failure. The app must be restarted.
3. **No backup progress monitoring** — large backups show no progress logs during upload.

## Comparison: Simple Backup vs. Litestream

| Feature | Simple Backup | Litestream |
|---------|--------------|------------|
| **Memory overhead** | Minimal | ~2x application memory |
| **RPO (data loss)** | Up to 1 hour | Near-zero (seconds) |
| **Backup frequency** | Hourly + shutdown | Continuous replication |
| **Startup time** | 10-60 seconds | Instant |
| **Storage** | 3 backups (~150MB) | Full history |
| **Use case** | Free tier, low-value data | Production, financial data |
| **Cost** | Low (storage only) | Higher (memory + storage) |

## When to Use Simple Backup

**Good for:**
- Free tier users with 1-50MB databases
- Low-traffic personal projects
- Development/staging environments
- Applications where 1-hour data loss is acceptable
- Cost-sensitive deployments

**Not suitable for:**
- Production applications with user-generated content
- Financial transactions or payment processing
- High-value business data
- Compliance requirements for point-in-time recovery
- Applications requiring < 1 minute RPO

## Troubleshooting

### Backup takes too long

**Symptoms:** Backup exceeds 30-second shutdown timeout

**Solutions:**
- Check database size: `ls -lh /var/lib/app/database.db`
- Run `VACUUM` manually to compact database
- Check MinIO upload speed: `aws s3 cp test.gz s3://mybucket/test.gz`
- Increase shutdown timeout in code if needed

### Restore fails on startup

**Symptoms:** Application won't start, restore errors

**Check MinIO connectivity:**
```bash
aws s3 ls s3://mybucket/backups/
```

**Check backup integrity:**
```bash
aws s3 cp s3://mybucket/backups/db-backup-1732442400.db.gz /tmp/
gunzip /tmp/db-backup-1732442400.db.gz
sqlite3 /tmp/db-backup-1732442400.db 'PRAGMA integrity_check'
```

### Backups accumulating (cleanup failing)

**Symptoms:** More than 3 backups in MinIO

**Possible causes:**
- Cleanup code not running
- MinIO delete permissions missing
- Network failures during cleanup

**Manual cleanup:**
```bash
# List all backups
aws s3 ls s3://mybucket/backups/

# Delete old backups manually
aws s3 rm s3://mybucket/backups/db-backup-1732435200.db.gz
```

### Database corruption

**Symptoms:** Integrity check fails repeatedly

**Recovery steps:**
1. Check all 3 backups for integrity:
   ```bash
   for backup in $(aws s3 ls s3://mybucket/backups/ | awk '{print $4}'); do
     echo "Checking $backup"
     aws s3 cp s3://mybucket/backups/$backup /tmp/$backup
     gunzip /tmp/$backup
     sqlite3 /tmp/${backup%.gz} 'PRAGMA integrity_check'
   done
   ```

2. If all backups corrupted, check S3 versioning for older versions

3. If no valid backups exist, start with fresh database (data loss)

## Implementation Checklist

- [x] `internal/simplebackup/backup.go` - Backup logic with cleanup
- [x] `internal/simplebackup/restore.go` - Restore logic with integrity check
- [x] `internal/simplebackup/manager.go` - Lifecycle orchestration
- [x] `internal/case/cronjob/simplebackup/` - Hourly cron job
- [x] `cmd/server/main.go` - CLI flag + startup restore + shutdown backup
- [x] Tests with real SQLite databases (10 unit tests)
- [x] E2E backup/restore test via `scripts/test-backup.sh`
- [x] Logging at INFO level for all operations
- [x] Error handling with clear messages
- [x] Documentation (this file)

## See Also

- [Cron Jobs Documentation](../CLAUDE.md#cron-jobs) - How to create cron jobs
- [MinIO Storage Documentation](../internal/miniostorage/) - MinIO client usage
- [Litestream Documentation](https://litestream.io/) - Alternative for production
