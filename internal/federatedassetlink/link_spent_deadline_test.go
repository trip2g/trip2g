package federatedassetlink_test

import (
	"sync"
	"testing"
	"time"

	"trip2g/internal/federatedassetlink"

	"github.com/stretchr/testify/require"
)

func TestSpentRefusesAnIdWhoseDeadlineHasPassedEvenWhenItWasNeverSeen(t *testing.T) {
	var spent federatedassetlink.Spent

	require.False(t, spent.SpendFederatedAssetLink("late", time.Now().Add(-time.Second)))
}

func TestSpentDelayedReplayAfterTheDeadlineIsNotReadmitted(t *testing.T) {
	var spent federatedassetlink.Spent
	deadline := time.Now().Add(40 * time.Millisecond)

	require.True(t, spent.SpendFederatedAssetLink("id", deadline))

	time.Sleep(60 * time.Millisecond)

	require.False(t, spent.SpendFederatedAssetLink("id", deadline), "the replay verified before the deadline but spends after it")
}

func TestSpentConcurrentReplaysAroundTheDeadlineAdmitAtMostOnce(t *testing.T) {
	var spent federatedassetlink.Spent
	deadline := time.Now().Add(30 * time.Millisecond)

	var admitted int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 40 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(time.Now().Nanosecond()%60) * time.Millisecond)
			if spent.SpendFederatedAssetLink("id", deadline) {
				mu.Lock()
				admitted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	require.LessOrEqual(t, admitted, 1)
}
