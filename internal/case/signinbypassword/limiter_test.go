package signinbypassword

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAttemptLimiter(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewAttemptLimiter(2, time.Minute)

	require.True(t, l.Allow("a", start))
	require.True(t, l.Allow("a", start))
	require.False(t, l.Allow("a", start), "third attempt in the window is refused")
	require.True(t, l.Allow("b", start), "another key has its own count")

	require.True(t, l.Allow("a", start.Add(time.Minute)), "a new window starts from zero")
}
