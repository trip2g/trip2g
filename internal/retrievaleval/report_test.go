package retrievaleval

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildReportAggregates(t *testing.T) {
	queries := []GoldenQuery{
		{Query: "q1", Direction: "ru->ru", ExpectedURLs: []string{"/a"}, Verified: true},
		{Query: "q2", Direction: "en->en", ExpectedURLs: []string{"/z"}, Verified: true},
	}
	retrievedByQuery := [][]string{
		{"/a", "/b"},
		{"/y", "/x"},
	}

	rep := BuildReport("baseline", queries, retrievedByQuery, 10)

	require.InDelta(t, 0.5, rep.Overall.RecallAtK, 1e-9)
	require.InDelta(t, 0.5, rep.Overall.MRR, 1e-9)
	require.Contains(t, rep.ByDirection, "ru->ru")
	require.InDelta(t, 1.0, rep.ByDirection["ru->ru"].RecallAtK, 1e-9)
	require.InDelta(t, 0.0, rep.ByDirection["en->en"].RecallAtK, 1e-9)
}

func TestReportGate(t *testing.T) {
	rep := Report{Overall: Metrics{NDCGAtK: 0.80}}
	require.NoError(t, rep.Gate(0.75))
	require.Error(t, rep.Gate(0.85))
}
