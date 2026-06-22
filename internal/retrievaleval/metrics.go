package retrievaleval

import "math"

// relevance is binary. retrieved is ranked best-first; relevant is the unordered
// set of acceptable result URLs.

func relevantSet(relevant []string) map[string]bool {
	m := make(map[string]bool, len(relevant))
	for _, r := range relevant {
		m[r] = true
	}
	return m
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RecallAtK returns |relevant ∩ top-k| / |relevant|.
func RecallAtK(retrieved, relevant []string, k int) float64 {
	if len(relevant) == 0 {
		return 0
	}
	rel := relevantSet(relevant)
	found := map[string]bool{}
	for _, u := range retrieved[:minInt(k, len(retrieved))] {
		if rel[u] {
			found[u] = true
		}
	}
	return float64(len(found)) / float64(len(relevant))
}

// ReciprocalRank returns 1/rank of the first relevant result, or 0 if none.
func ReciprocalRank(retrieved, relevant []string) float64 {
	rel := relevantSet(relevant)
	for i, u := range retrieved {
		if rel[u] {
			return 1.0 / float64(i+1)
		}
	}
	return 0
}

// NDCGAtK returns the binary-relevance nDCG@k.
func NDCGAtK(retrieved, relevant []string, k int) float64 {
	rel := relevantSet(relevant)
	var dcg float64
	for i, u := range retrieved[:minInt(k, len(retrieved))] {
		if rel[u] {
			dcg += 1.0 / math.Log2(float64(i+2))
		}
	}
	var idcg float64
	for i := range minInt(k, len(relevant)) {
		idcg += 1.0 / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}
