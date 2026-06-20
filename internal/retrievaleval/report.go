package retrievaleval

import "fmt"

type Metrics struct {
	Count     int     `json:"count"`
	RecallAtK float64 `json:"recall_at_k"`
	NDCGAtK   float64 `json:"ndcg_at_k"`
	MRR       float64 `json:"mrr"`
}

type QueryResult struct {
	Query     string   `json:"query"`
	Direction string   `json:"direction"`
	Expected  []string `json:"expected_urls"`
	Retrieved []string `json:"retrieved_urls"`
	RecallAtK float64  `json:"recall_at_k"`
	NDCGAtK   float64  `json:"ndcg_at_k"`
	RR        float64  `json:"rr"`
}

type Report struct {
	Label       string             `json:"label"`
	K           int                `json:"k"`
	Overall     Metrics            `json:"overall"`
	ByDirection map[string]Metrics `json:"by_direction"`
	Queries     []QueryResult      `json:"queries"`
}

// BuildReport scores each query and aggregates overall + per-direction.
// retrievedByQuery[i] is the ranked URL list returned for queries[i].
func BuildReport(label string, queries []GoldenQuery, retrievedByQuery [][]string, k int) Report {
	rep := Report{Label: label, K: k, ByDirection: map[string]Metrics{}}
	dirSum := map[string]*Metrics{}
	var sumR, sumN, sumRR float64

	for i, q := range queries {
		retrieved := retrievedByQuery[i]
		r := RecallAtK(retrieved, q.ExpectedURLs, k)
		n := NDCGAtK(retrieved, q.ExpectedURLs, k)
		rr := ReciprocalRank(retrieved, q.ExpectedURLs)

		rep.Queries = append(rep.Queries, QueryResult{
			Query: q.Query, Direction: q.Direction, Expected: q.ExpectedURLs,
			Retrieved: retrieved, RecallAtK: r, NDCGAtK: n, RR: rr,
		})
		sumR += r
		sumN += n
		sumRR += rr

		d := dirSum[q.Direction]
		if d == nil {
			d = &Metrics{}
			dirSum[q.Direction] = d
		}
		d.Count++
		d.RecallAtK += r
		d.NDCGAtK += n
		d.MRR += rr
	}

	cnt := len(queries)
	if cnt > 0 {
		rep.Overall = Metrics{Count: cnt, RecallAtK: sumR / float64(cnt), NDCGAtK: sumN / float64(cnt), MRR: sumRR / float64(cnt)}
	}
	for dir, m := range dirSum {
		rep.ByDirection[dir] = Metrics{
			Count: m.Count, RecallAtK: m.RecallAtK / float64(m.Count),
			NDCGAtK: m.NDCGAtK / float64(m.Count), MRR: m.MRR / float64(m.Count),
		}
	}
	return rep
}

// Gate returns an error if overall nDCG@k is below minNDCG (for CI regression gating).
func (r Report) Gate(minNDCG float64) error {
	if r.Overall.NDCGAtK < minNDCG {
		return fmt.Errorf("nDCG@%d %.4f below threshold %.4f", r.K, r.Overall.NDCGAtK, minNDCG)
	}
	return nil
}
