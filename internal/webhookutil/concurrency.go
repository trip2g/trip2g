package webhookutil

import "fmt"

const (
	ConcurrencyAllowOverlap = "allow_overlap"
	ConcurrencySkip         = "skip"
	ConcurrencyQueueOne     = "queue_one"
)

// NormalizeConcurrencyMode returns the default mode for an empty value.
func NormalizeConcurrencyMode(mode string) string {
	if mode == "" {
		return ConcurrencyAllowOverlap
	}
	return mode
}

// ValidateConcurrencyMode checks the DB-backed concurrency enum.
func ValidateConcurrencyMode(mode string) error {
	switch NormalizeConcurrencyMode(mode) {
	case ConcurrencyAllowOverlap, ConcurrencySkip, ConcurrencyQueueOne:
		return nil
	default:
		return fmt.Errorf("must be one of %s, %s, %s", ConcurrencyAllowOverlap, ConcurrencySkip, ConcurrencyQueueOne)
	}
}
