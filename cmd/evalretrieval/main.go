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
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"trip2g/internal/retrievaleval"
)

func main() {
	gateErr, runErr := run()
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "error:", runErr)
		os.Exit(2)
	}
	if gateErr != nil {
		fmt.Fprintln(os.Stderr, "GATE FAILED:", gateErr)
		os.Exit(1)
	}
}

// run executes the evaluation.
// Returns (gateFailure, fatalError): gateFailure is non-nil when the metric
// threshold was not met (exit 1); fatalError is non-nil for unexpected errors (exit 2).
func run() (error, error) {
	golden := flag.String("golden", "testdata/eval/golden_set.json", "path to golden set JSON")
	endpoint := flag.String(
		"endpoint", "http://localhost:21081/_system/graphql",
		"GraphQL endpoint (/_system/graphql; /graphql is deprecated)",
	)
	bearer := flag.String("bearer", "", "admin session JWT (required to see the latest index)")
	cookieName := flag.String("cookie-name", "", "if set, send the token as this cookie instead of Authorization: Bearer (reliable admin path; Bearer session JWTs are misclassified as scoped tokens and yield deny-all in search)")
	label := flag.String("label", "run", "label for this run")
	k := flag.Int("k", 10, "k for recall@k / ndcg@k")
	out := flag.String("out", "", "write JSON artifact to this path (optional)")
	failUnder := flag.Float64("fail-under-ndcg", 0, "exit nonzero if overall nDCG@k below this")
	flag.Parse()

	gs, err := retrievaleval.LoadGoldenSet(*golden)
	if err != nil {
		return nil, err
	}
	queries := gs.Verified()
	if len(queries) == 0 {
		return nil, errors.New("golden set has no verified queries")
	}

	client := retrievaleval.NewSearchClient(*endpoint, *bearer)
	if *cookieName != "" {
		client = client.WithCookieAuth(*cookieName)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	retrieved := make([][]string, len(queries))
	for i, q := range queries {
		var res *retrievaleval.SearchResponse
		res, err = client.Search(ctx, q.Query)
		if err != nil {
			return nil, fmt.Errorf("query %q: %w", q.Query, err)
		}
		retrieved[i] = res.URLs()
	}

	rep := retrievaleval.BuildReport(*label, queries, retrieved, *k)

	// Use Fprintf(os.Stdout) to avoid the forbidigo fmt.Printf restriction.
	_, _ = fmt.Fprintf(os.Stdout, "=== %s (n=%d, k=%d) ===\n", rep.Label, rep.Overall.Count, rep.K)
	_, _ = fmt.Fprintf(os.Stdout,
		"Recall@%d=%.4f  nDCG@%d=%.4f  MRR=%.4f\n",
		rep.K, rep.Overall.RecallAtK, rep.K, rep.Overall.NDCGAtK, rep.Overall.MRR,
	)
	for dir, m := range rep.ByDirection {
		_, _ = fmt.Fprintf(
			os.Stdout, "  %-8s n=%d Recall@%d=%.4f nDCG@%d=%.4f MRR=%.4f\n",
			dir, m.Count, rep.K, m.RecallAtK, rep.K, m.NDCGAtK, m.MRR,
		)
	}

	if *out != "" {
		data, _ := json.MarshalIndent(rep, "", "  ")
		if err = os.WriteFile(*out, data, 0o600); err != nil {
			return nil, err
		}
		_, _ = fmt.Fprintf(os.Stdout, "artifact: %s\n", *out)
	}

	return rep.Gate(*failUnder), nil
}
