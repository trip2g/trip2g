package model

import "time"

// PrivateObject represents metadata about an object in S3-compatible storage.
// Key is relative to the configured storage prefix.
// Example: If storage prefix is "backups/" and S3 key is "backups/db-backup-123.db.gz",
// then PrivateObject.Key will be "db-backup-123.db.gz"
type PrivateObject struct {
	Key          string    // Relative key (prefix stripped)
	Size         int64     // Size in bytes
	LastModified time.Time // Last modification timestamp
	ETag         string    // Entity tag for integrity verification
}
