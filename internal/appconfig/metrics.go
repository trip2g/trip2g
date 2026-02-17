package appconfig

import (
	"flag"
	"time"
)

// MetricsConfig holds Prometheus metrics configuration.
type MetricsConfig struct {
	UpdateInterval time.Duration
}

// DefaultMetricsConfig returns default metrics configuration.
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		UpdateInterval: 30 * time.Second,
	}
}

// defineMetricsFlags registers command-line flags for metrics configuration.
func (c *Config) defineMetricsFlags() {
	defaults := DefaultMetricsConfig()

	flag.DurationVar(
		&c.Metrics.UpdateInterval,
		"metric-update-interval",
		defaults.UpdateInterval,
		"Interval for updating Prometheus metrics",
	)
}
