package cleanupapikeylogs

import "time"

// Config holds configuration for the API key logs cleanup job.
type Config struct {
	Retention time.Duration
}

// DefaultConfig returns a Config with a 90-day retention period.
func DefaultConfig() Config {
	return Config{
		Retention: 90 * 24 * time.Hour,
	}
}
