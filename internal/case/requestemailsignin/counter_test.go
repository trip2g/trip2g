package requestemailsignin

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignInCounter_BelowThreshold(t *testing.T) {
	c := &SignInCounter{}
	exceeded := c.IncrementAndCheck(5)
	require.False(t, exceeded)
}

func TestSignInCounter_ExceedsThreshold(t *testing.T) {
	c := &SignInCounter{}
	for range 5 {
		c.IncrementAndCheck(5)
	}
	exceeded := c.IncrementAndCheck(5)
	require.True(t, exceeded)
}

func TestSignInCounter_ConcurrentSafety(t *testing.T) {
	c := &SignInCounter{}
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.IncrementAndCheck(50)
		}()
	}
	wg.Wait()

	// Verify count is exactly 100 (no lost increments).
	c.mu.Lock()
	defer c.mu.Unlock()
	require.Equal(t, int64(100), c.count)
}
