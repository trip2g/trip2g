package webhookutil

import "time"

// BasePayload contains fields common to all webhook payloads.
type BasePayload struct {
	Version   int   `json:"version"`
	ID        int64 `json:"id"`
	Timestamp int64 `json:"timestamp"`
	Attempt   int   `json:"attempt"`
}

// NewBasePayload creates a BasePayload with current time.
func NewBasePayload(deliveryID int64, attempt int) BasePayload {
	return BasePayload{
		Version:   1,
		ID:        deliveryID,
		Timestamp: time.Now().Unix(),
		Attempt:   attempt,
	}
}
