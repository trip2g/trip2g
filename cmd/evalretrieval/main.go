// Command evalretrieval runs the retrieval golden set against a running trip2g
// GraphQL endpoint and writes a JSON metrics artifact.
//
// Usage:
//
//	go run ./cmd/evalretrieval \
//	  -golden testdata/eval/golden_set.json \
//	  -endpoint http://localhost:21081/_system/graphql \
//	  -bearer "$TOKEN" -label baseline -k 10 \
//	  -out docs/superpowers/eval-runs/00-baseline.json \
//	  -fail-under-ndcg 0
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"trip2g/internal/retrievaleval"
)

func main() {
	golden := flag.String("golden", "testdata/eval/golden_set.json", "path to golden set JSON")
	endpoint := flag.String("endpoint", "http://localhost:21081/_system/graphql", "GraphQL endpoint (/_system/graphql; /graphql is deprecated)")
	bearer := flag.String("bearer", "", "Authorization bearer token (admin session JWT; required to see the latest index)")
	label := flag.String("label", "run", "label for this run")
	k := flag.Int("k", 10, "k for recall@k / ndcg@k")
	out := flag.String("out", "", "write JSON artifact to this path (optional)")
	failUnder := flag.Float64("fail-under-ndcg", 0, "exit nonzero if overall nDCG@k below this")
	flag.Parse()

	gs, err := retrievaleval.LoadGoldenSet(*golden)
	if err != nil {
		fail(err)
	}
	queries := gs.Verified()
	if len(queries) == 0 {
		fail(fmt.Errorf("golden set has no verified queries"))
	}

	client := retrievaleval.NewSearchClient(*endpoint, *bearer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	retrieved := make([][]string, len(queries))
	for i, q := range queries {
		res, err := client.Search(ctx, q.Query)
		if err != nil {
			fail(fmt.Errorf("query %q: %w", q.Query, err))
		}
		retrieved[i] = res.URLs()
	}

	rep := retrievaleval.BuildReport(*label, queries, retrieved, *k)

	fmt.Printf("=== %s (n=%d, k=%d) ===\n", rep.Label, rep.Overall.Count, rep.K)
	fmt.Printf("Recall@%d=%.4f  nDCG@%d=%.4f  MRR=%.4f\n", rep.K, rep.Overall.RecallAtK, rep.K, rep.Overall.NDCGAtK, rep.Overall.MRR)
	for dir, m := range rep.ByDirection {
		fmt.Printf("  %-8s n=%d Recall@%d=%.4f nDCG@%d=%.4f MRR=%.4f\n", dir, m.Count, rep.K, m.RecallAtK, rep.K, m.NDCGAtK, m.MRR)
	}

	if *out != "" {
		data, _ := json.MarshalIndent(rep, "", "  ")
		if err := os.WriteFile(*out, data, 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("artifact: %s\n", *out)
	}

	if err := rep.Gate(*failUnder); err != nil {
		fmt.Fprintln(os.Stderr, "GATE FAILED:", err)
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(2)
}
