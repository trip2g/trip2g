package fedinstr

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func resultWithText(text string) model.FederationResult {
	return model.FederationResult{Content: []model.FederationContent{{Type: "text", Text: text}}}
}

func TestCacheStoreAndHit(t *testing.T) {
	c := New()
	_, ok := c.CachedFederatedInstructions("a/b")
	require.False(t, ok)

	c.StoreFederatedInstructions("a/b", resultWithText("guidance"))
	got, ok := c.CachedFederatedInstructions("a/b")
	require.True(t, ok)
	require.Equal(t, "guidance", got.Content[0].Text)
}

func TestCacheTTLExpiry(t *testing.T) {
	c := New()
	now := time.Now()
	c.now = func() time.Time { return now }

	c.StoreFederatedInstructions("a", resultWithText("x"))
	_, ok := c.CachedFederatedInstructions("a")
	require.True(t, ok)

	now = now.Add(DefaultTTL + time.Second)
	_, ok = c.CachedFederatedInstructions("a")
	require.False(t, ok, "entry past TTL must miss")
}

func TestCacheEvictsOldestWhenFull(t *testing.T) {
	c := New()
	base := time.Now()
	tick := 0
	c.now = func() time.Time { tick++; return base.Add(time.Duration(tick) * time.Millisecond) }
	c.maxEntries = 3

	c.StoreFederatedInstructions("k0", resultWithText("0")) // oldest
	c.StoreFederatedInstructions("k1", resultWithText("1"))
	c.StoreFederatedInstructions("k2", resultWithText("2"))
	c.StoreFederatedInstructions("k3", resultWithText("3")) // evicts k0

	_, ok := c.CachedFederatedInstructions("k0")
	require.False(t, ok, "oldest entry must be evicted")
	for _, k := range []string{"k1", "k2", "k3"} {
		_, survived := c.CachedFederatedInstructions(k)
		require.True(t, survived, "recent entry %s must survive", k)
	}
	require.Len(t, c.entries, 3)
}

func TestCacheConcurrent(t *testing.T) {
	c := New()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("kb/%d", i%8)
			c.StoreFederatedInstructions(key, resultWithText(key))
			c.CachedFederatedInstructions(key)
		}(i)
	}
	wg.Wait()
}
