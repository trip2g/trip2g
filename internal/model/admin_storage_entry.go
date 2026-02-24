package model

// AdminStorageEntry holds raw byte counts for storage monitoring.
// The GraphQL fields limit(format:) and current(format:) are resolved
// via resolvers that apply the requested unit conversion.
type AdminStorageEntry struct {
	LimitBytes   int64
	CurrentBytes int64
}
