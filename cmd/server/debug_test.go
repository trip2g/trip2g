package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsDeadJob(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	past := "2026-06-10T11:59:00.000Z"
	future := "2026-06-10T12:01:00.000Z"

	tests := []struct {
		name       string
		received   int64
		maxReceive int
		timeout    string
		want       bool
	}{
		{"never received", 0, 3, past, false},
		{"first attempt failed, will retry", 1, 3, past, false},
		{"retried, attempts left", 2, 3, past, false},
		{"exhausted attempts, timed out — dead", 3, 3, past, true},
		{"final attempt still running", 3, 3, future, false},
		{"exhausted on low-max queue", 2, 2, past, true},
		{"unparseable timeout treated as expired", 3, 3, "garbage", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isDeadJob(tt.received, tt.maxReceive, tt.timeout, now))
		})
	}
}
