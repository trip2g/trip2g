package retrievaleval

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecallAtK(t *testing.T) {
	retrieved := []string{"A", "B", "C", "D"}
	require.InDelta(t, 0.0, RecallAtK(retrieved, []string{"C"}, 2), 1e-9)
	require.InDelta(t, 1.0, RecallAtK(retrieved, []string{"C"}, 3), 1e-9)
	require.InDelta(t, 0.5, RecallAtK(retrieved, []string{"C", "X"}, 4), 1e-9)
}

func TestReciprocalRank(t *testing.T) {
	retrieved := []string{"A", "B", "C"}
	require.InDelta(t, 1.0/3.0, ReciprocalRank(retrieved, []string{"C"}), 1e-9)
	require.InDelta(t, 1.0, ReciprocalRank(retrieved, []string{"A"}), 1e-9)
	require.InDelta(t, 0.0, ReciprocalRank(retrieved, []string{"Z"}), 1e-9)
}

func TestNDCGAtK(t *testing.T) {
	// C at index 2 (position 3): DCG = 1/log2(4) = 0.5; IDCG (1 relevant) = 1/log2(2) = 1.0
	require.InDelta(t, 0.5, NDCGAtK([]string{"A", "B", "C", "D"}, []string{"C"}, 4), 1e-9)
	// perfect order, 2 relevant
	require.InDelta(t, 1.0, NDCGAtK([]string{"A", "B"}, []string{"A", "B"}, 2), 1e-9)
	require.InDelta(t, 0.0, NDCGAtK([]string{"A"}, []string{"Z"}, 1), 1e-9)
}
