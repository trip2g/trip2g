# SQLite Configuration and Best Practices

This document describes the SQLite configuration used in this project, best practices, and maintenance procedures.

## Current Configuration

### Database Setup (`internal/db/setup.go`)

Our SQLite database is configured for optimal performance and safety with the following settings:

#### Connection Parameters

```go
// URL parameters (applied at connection initialization)
_journal=WAL           // Hint for initial journal mode
_timeout=20000         // 20 second timeout
_busy_timeout=20000    // 20 second busy timeout

// Note: These URL parameters provide hints during connection initialization.
// PRAGMA statements in enablePragmas() ensure settings are applied regardless
// of database state.
```

#### Connection Pool Settings

```go
// Write connection (main database)
SetMaxOpenConns(1)      // CRITICAL: Only 1 writer for SQLite WAL mode
SetMaxIdleConns(1)
SetConnMaxLifetime(0)   // Connections never expire
SetConnMaxIdleTime(0)

// Read connection (optional read-only connection)
SetMaxOpenConns(25)     // Multiple readers allowed in WAL mode
SetMaxIdleConns(25)
```

**Why only 1 writer?**
SQLite with WAL mode allows multiple concurrent readers but only **one writer at a time**. Having more than one write connection can cause `SQLITE_BUSY` errors.

#### PRAGMA Settings

Applied at startup via `enablePragmas()`:

```sql
PRAGMA foreign_keys = ON;           -- Enable foreign key constraints
PRAGMA synchronous = NORMAL;        -- Balance between speed and safety
PRAGMA strict = ON;                 -- Strict aggregate functions
PRAGMA temp_store = MEMORY;         -- Store temporary tables in RAM
PRAGMA mmap_size = 268435456;       -- 256MB memory-mapped I/O
PRAGMA cache_size = -64000;         -- 64MB page cache
PRAGMA wal_autocheckpoint = 1000;   -- Checkpoint every 1000 pages
```

## PRAGMA Settings Explained

### `foreign_keys = ON`
**CRITICAL**: Enables foreign key constraint enforcement. Without this, foreign keys are ignored.

```sql
-- With foreign_keys = ON:
DELETE FROM users WHERE id = 1;  -- Error if purchases reference this user

-- Without it:
DELETE FROM users WHERE id = 1;  -- Succeeds, orphans purchases!
```

### `synchronous = NORMAL`
Controls how often SQLite syncs to disk:
- `FULL` (default): Very safe, slower
- `NORMAL`: Good balance - safe in WAL mode
- `OFF`: Fast but risks corruption on power loss

With WAL mode, `NORMAL` is safe and recommended.

### `strict = ON`
Strict mode for aggregate functions:
```sql
-- With strict = ON:
SELECT sum(text_column) FROM table;  -- Error

-- Without it:
SELECT sum(text_column) FROM table;  -- Returns 0 (silent failure)
```

### `temp_store = MEMORY`
Stores temporary tables and indices in RAM instead of disk. Faster but uses more memory.

### `mmap_size = 268435456`
Memory-mapped I/O (256MB). SQLite maps database pages directly to memory, reducing system calls. Improves read performance significantly.

### `cache_size = -64000`
Page cache size in KB (negative = KB, positive = pages). 64MB cache improves performance for frequently accessed data.

### `wal_autocheckpoint = 1000`
Automatically checkpoint WAL file after 1000 pages. Prevents WAL file from growing too large.

## WAL Mode (Write-Ahead Logging)

### What is WAL?

Instead of writing directly to the database file, SQLite writes changes to a separate WAL file:

```
database.db       -- Main database file
database.db-wal   -- Write-Ahead Log (pending changes)
database.db-shm   -- Shared memory file (index for WAL)
```

### Benefits

1. **Multiple concurrent readers** - Readers don't block each other
2. **Readers don't block writer** - Reads from checkpoint, writes to WAL
3. **Better performance** - Sequential writes to WAL are faster
4. **Crash recovery** - WAL file can be replayed on startup

### Checkpointing

Periodically, WAL changes are merged back into the main database file. This is called "checkpointing".

**Manual checkpoint:**
```sql
PRAGMA wal_checkpoint(TRUNCATE);  -- Full checkpoint, truncate WAL
```

In this project, checkpointing happens:
- **Automatically**: Every 1000 pages (`wal_autocheckpoint`)
- **Weekly**: During VACUUM cronjob (Sunday 3 AM)

## Maintenance

### VACUUM Cronjob

**Schedule**: Every Sunday at 3:00 AM (`internal/case/cronjob/vacuumdatabase/`)

**What it does:**
```go
func VacuumDB(ctx context.Context) error {
    // 1. Checkpoint WAL file
    PRAGMA wal_checkpoint(TRUNCATE)

    // 2. Reclaim unused space (defragment)
    VACUUM

    // 3. Update query optimizer statistics
    ANALYZE
}
```

**Why weekly?**
- `VACUUM` reclaims space from deleted rows
- `ANALYZE` updates statistics for the query planner
- Both operations lock the database, so run during low-traffic hours

### Manual Maintenance

If needed, you can trigger manually via GraphQL admin API or:

```bash
sqlite3 database.db "PRAGMA wal_checkpoint(TRUNCATE);"
sqlite3 database.db "VACUUM;"
sqlite3 database.db "ANALYZE;"
```

## Best Practices

### 1. Always Use Transactions for Writes

```go
// Bad: Multiple statements without transaction
db.Exec("INSERT INTO users ...")
db.Exec("INSERT INTO purchases ...")

// Good: Single transaction
tx, _ := db.Begin()
tx.Exec("INSERT INTO users ...")
tx.Exec("INSERT INTO purchases ...")
tx.Commit()
```

Benefits:
- **Atomicity**: All-or-nothing
- **Performance**: 100x faster for bulk inserts
- **Consistency**: Readers see consistent state

### 2. Use Prepared Statements

```go
// Bad: String concatenation (SQL injection risk)
db.Exec("SELECT * FROM users WHERE email = '" + email + "'")

// Good: Prepared statement (via sqlc)
queries.GetUserByEmail(ctx, email)
```

### 3. Add Indexes for Frequent Queries

```sql
-- Before adding index:
SELECT * FROM purchases WHERE user_id = ?;  -- Table scan (slow)

-- Add index:
CREATE INDEX idx_purchases_user_id ON purchases(user_id);

-- After:
SELECT * FROM purchases WHERE user_id = ?;  -- Index seek (fast)
```

**Where to add indexes:**
- Foreign key columns (for JOINs)
- WHERE clause columns
- ORDER BY columns
- Composite indexes for multi-column queries

**Example composite index:**
```sql
-- Query:
SELECT * FROM user_subgraph_accesses
WHERE user_id = ? AND subgraph_id = ?;

-- Index:
CREATE INDEX idx_user_subgraph_accesses_user_subgraph
ON user_subgraph_accesses(user_id, subgraph_id);
```

### 4. Use Partial Indexes for Filtered Queries

```sql
-- Instead of:
CREATE INDEX idx_api_keys_disabled_at ON api_keys(disabled_at);

-- Better (only active keys):
CREATE INDEX idx_api_keys_active
ON api_keys(id)
WHERE disabled_at IS NULL;
```

Smaller index, faster queries for common case.

### 5. Use CHECK Constraints for Validation

```sql
CREATE TABLE users (
  email TEXT NOT NULL UNIQUE,
  CHECK (email LIKE '%@%')  -- Database-level validation
);

CREATE TABLE purchases (
  status TEXT NOT NULL DEFAULT 'pending',
  CHECK (status IN ('pending', 'completed', 'failed'))
);
```

### 6. Prefer NOT NULL Where Possible

```sql
-- Bad:
created_at DATETIME  -- Can be NULL

-- Good:
created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
```

NULL values complicate queries and can cause bugs.

### 7. Use Unique Constraints for Business Logic

```sql
-- Prevent duplicate subscriptions at database level:
CREATE UNIQUE INDEX idx_user_subgraph_accesses_unique
ON user_subgraph_accesses(user_id, subgraph_id)
WHERE revoke_id IS NULL;
```

## Performance Tuning

### Analyze Query Plans

```sql
EXPLAIN QUERY PLAN
SELECT * FROM users WHERE email = 'test@example.com';

-- Output shows if index is used:
-- SEARCH users USING INDEX idx_users_email (email=?)  ✅ Good
-- SCAN users                                          ❌ Bad (table scan)
```

### Check Table Sizes

```sql
SELECT
    name,
    SUM(pgsize) / 1024 / 1024 AS size_mb
FROM dbstat
GROUP BY name
ORDER BY size_mb DESC
LIMIT 10;
```

### Monitor WAL Size

```sql
-- Check WAL file size:
PRAGMA wal_checkpoint;
-- Returns (busy, log, checkpointed)
-- If log > 1000, consider manual checkpoint
```

### Check Fragmentation

```sql
SELECT freelist_count FROM pragma_freelist_count();
-- If > 1000, run VACUUM to defragment
```

### Find Unused Indexes

```sql
-- After running workload, check index usage:
SELECT * FROM sqlite_stat1;

-- If an index has low/no stats, consider dropping it
```

## Common Pitfalls

### 1. ❌ Multiple Write Connections

```go
// BAD: Multiple writers cause SQLITE_BUSY
db.SetMaxOpenConns(10)  // Don't do this for write connection!

// GOOD: Single writer
db.SetMaxOpenConns(1)
```

### 2. ❌ Forgetting Foreign Keys

```go
// Without PRAGMA foreign_keys = ON:
// Foreign key constraints are ignored!
DELETE FROM users WHERE id = 1;  // Orphans dependent data
```

### 3. ❌ Long Transactions

```go
// BAD: Long transaction blocks other writers
tx, _ := db.Begin()
// ... lots of work ...
time.Sleep(10 * time.Second)  // Other writers blocked!
tx.Commit()

// GOOD: Short transactions
tx, _ := db.Begin()
tx.Exec(...)  // Fast operations only
tx.Commit()
```

### 4. ❌ Not Using Transactions for Bulk Inserts

```go
// BAD: 1000 individual transactions (very slow)
for i := 0; i < 1000; i++ {
    db.Exec("INSERT INTO ...")
}

// GOOD: Single transaction (100x faster)
tx, _ := db.Begin()
for i := 0; i < 1000; i++ {
    tx.Exec("INSERT INTO ...")
}
tx.Commit()
```

### 5. ❌ Using LIKE Without Index

```sql
-- Slow (can't use index):
WHERE email LIKE '%@gmail.com'

-- Fast (can use index):
WHERE email LIKE 'user@%'  -- Prefix search
```

### 6. ❌ SELECT * Instead of Specific Columns

```sql
-- Bad: Reads all columns (slower, more memory)
SELECT * FROM users WHERE id = ?

-- Good: Only needed columns
SELECT id, email FROM users WHERE id = ?
```

## Monitoring Queries

### Database Statistics

```sql
-- Overall database info:
SELECT * FROM pragma_database_list;

-- Page count and size:
SELECT
    page_count * page_size / 1024 / 1024 AS db_size_mb,
    page_count,
    page_size
FROM pragma_page_count(), pragma_page_size();

-- Integrity check:
PRAGMA integrity_check;
-- Should return: ok
```

### Table Statistics

```sql
-- Index usage stats:
SELECT * FROM sqlite_stat1;

-- Table sizes:
SELECT
    tbl AS table_name,
    SUM(payload) / 1024 / 1024 AS data_mb,
    SUM(pgsize) / 1024 / 1024 AS total_mb
FROM dbstat
GROUP BY tbl
ORDER BY total_mb DESC;
```

### Performance Diagnostics

```sql
-- Check if indexes are being used:
EXPLAIN QUERY PLAN <your-query>;

-- Detailed execution plan:
EXPLAIN <your-query>;

-- Current connections (if using shared cache):
PRAGMA wal_checkpoint;
```

## Backup Strategy

### Online Backup (While Server Running)

```bash
# Using SQLite backup API (recommended)
sqlite3 database.db ".backup database-backup.db"

# Or using WAL checkpoint:
sqlite3 database.db "PRAGMA wal_checkpoint(TRUNCATE);"
cp database.db database-backup.db
```

### Backup Files to Include

```bash
# Main database
database.db

# WAL files (if not checkpointed)
database.db-wal
database.db-shm
```

**Note**: With WAL mode, you must either:
1. Checkpoint before copying (`PRAGMA wal_checkpoint(TRUNCATE)`)
2. Copy both `.db`, `.db-wal`, and `.db-shm` files atomically

## Migration Best Practices

### Safe Schema Changes

SQLite doesn't support `ALTER TABLE` for many operations. Use the recreate pattern:

```sql
-- 1. Create new table with desired schema
CREATE TABLE users_new (
  id INTEGER PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  new_column TEXT
) /* NOTE: Don't add STRICT to existing tables */;

-- 2. Copy data
INSERT INTO users_new (id, email, new_column)
SELECT id, email, NULL FROM users;

-- 3. Drop old table
DROP TABLE users;

-- 4. Rename new table
ALTER TABLE users_new RENAME TO users;

-- 5. Recreate indexes
CREATE INDEX idx_users_email ON users(email);
```

### Foreign Key Considerations

When recreating tables with foreign keys:

```sql
-- 1. Disable foreign key checks temporarily
PRAGMA foreign_keys = OFF;

-- 2. Recreate tables (as above)

-- 3. Re-enable foreign key checks
PRAGMA foreign_keys = ON;

-- 4. Verify no violations
PRAGMA foreign_key_check;
```

## Performance Benchmarks

Typical performance characteristics:

- **Reads**: ~100k-500k reads/sec (with proper indexes)
- **Writes**: ~1k-10k writes/sec (single writer limit)
- **Bulk inserts**: ~50k-100k rows/sec (in transaction)
- **Database size**: Handles databases up to 281 TB (theoretical limit)

For this project (note-based application):
- Expected writes: Low-medium (user actions, note updates)
- Expected reads: High (serving note content)
- **Verdict**: SQLite is perfect for this workload

## When to Consider PostgreSQL

SQLite is excellent for this project, but consider PostgreSQL if:

1. **Multiple simultaneous writers** - Need >1 write/sec sustained
2. **Geographic distribution** - Need read replicas in different regions
3. **Very large dataset** - Multi-TB with complex analytics
4. **Network access** - Need remote database access

For single-server applications with moderate write load, SQLite is often faster and simpler than PostgreSQL.

## Resources

- [SQLite Documentation](https://sqlite.org/docs.html)
- [SQLite PRAGMA Statements](https://sqlite.org/pragma.html)
- [WAL Mode](https://sqlite.org/wal.html)
- [Query Planning](https://sqlite.org/queryplanner.html)
- [Performance Tuning](https://sqlite.org/performance.html)

## Summary

Our SQLite configuration is optimized for:
- ✅ Data integrity (foreign keys, constraints)
- ✅ Performance (WAL mode, indexes, caching)
- ✅ Reliability (automated maintenance, backups)
- ✅ Safety (transaction isolation, crash recovery)

Key takeaways:
1. **One writer connection** - Critical for avoiding SQLITE_BUSY
2. **WAL mode** - Enables concurrent reads
3. **Weekly VACUUM** - Keeps database compact
4. **Proper indexes** - Essential for query performance
5. **Foreign keys ON** - Enforces data integrity
