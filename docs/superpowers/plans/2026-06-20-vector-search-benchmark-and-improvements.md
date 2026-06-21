# Vector Search Benchmark & Measured Improvements — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a reproducible retrieval-quality benchmark for trip2g's hybrid search, then apply the prioritized fixes one at a time — measuring Recall@k / nDCG@10 / MRR before and after each — to produce a data-backed article on how we improved vector search.

**Architecture:** A pure-Go metrics + golden-set library (`internal/retrievaleval/`) plus a black-box CLI (`cmd/evalretrieval/`) that drives the real GraphQL `search` endpoint over HTTP. The benchmark runs against the in-repo demo vault (`docs/demo`) embedded with **bge-m3** via the existing e2e stack. Each improvement is a self-contained, independently shippable change gated by a before/after eval run whose JSON artifact feeds the final article.

**Tech stack:** Go 1.26, testify/require, bleve (BM25), OpenAI-compatible embeddings (bge-m3, 1024d), SQLite, gqlgen GraphQL, the e2e docker stack (`scripts/test-e2e.sh`, `embedding-server/`), obsidian-sync CLI.

**Decisions locked (interview):** corpus = demo vault; golden set = LLM-generate + hand-verify; harness = CLI + JSON artifact + CI gate; baseline embedding model = bge-m3 (1024d).

---

## Background: what already exists (do NOT rebuild)

Confirmed by code reading — `docs/dev/vector_search.md` is **stale** and understates this:

- Hybrid search is **live**: `internal/case/sitesearch/resolve.go` runs bleve BM25 + brute-force cosine over per-chunk embeddings, fused with Reciprocal Rank Fusion (`rrfK = 60`, resolve.go:38), capped at 20.
- Per-chunk embeddings exist (`internal/mdchunk`, `internal/model/chunk.go`), generated async (`internal/case/backjob/generatenoteversionembedding`) with SHA256 content-hash dedup, plus a daily/startup re-embed cron.
- The GraphQL entry point is `search(input: SearchInput!): SearchConnection!` → `schema.resolvers.go:3051` → `sitesearch.Resolve`. Result nodes carry `url`, `score`, `matchOrigin` (TEXT/VECTOR/HYBRID).
- The MCP path (`internal/case/mcp/resolve.go`) **duplicates** the hybrid logic and has already drifted (`DefaultVectorSearchLimit = 10` vs sitesearch `vectorTopK = 5`).

The known leaks this plan measures and fixes (from the analysis, reconciled with code verification):

| # | Fix | Files | Effort |
|---|-----|-------|--------|
| F1 | Stop truncating the vector list before RRF | sitesearch:118, mcp:26 | S |
| F2 | Per-language bleve analyzer (EN stemmed as RU today) | noteloader/search.go | M |
| F3 | Cross-encoder reranker after RRF | features + sitesearch/mcp | M |
| F4 | Heading-path chunk prefix + token-aware sizing | internal/mdchunk | M |
| F5 | AND→OR fallback + cosine norm precompute + observability | search.go, sitesearch | S |

---

## File Structure

**New — benchmark library (pure, unit-tested in plain `go test`):**
- `internal/retrievaleval/metrics.go` — `RecallAtK`, `NDCGAtK`, `ReciprocalRank` (binary relevance)
- `internal/retrievaleval/goldenset.go` — `GoldenSet`/`GoldenQuery` types + `LoadGoldenSet`
- `internal/retrievaleval/client.go` — `SearchClient` (HTTP → GraphQL `search`), returns ordered URLs
- `internal/retrievaleval/report.go` — per-query + aggregate report, by-direction breakdown, pass/fail gate
- `internal/retrievaleval/*_test.go` — table-driven tests for each of the above

**New — CLI + ops:**
- `cmd/evalretrieval/main.go` — flags → load golden set → run → write JSON artifact → exit code on gate
- `scripts/eval.sh` — bring up nothing new; assumes stack is up, runs the CLI with standard args
- `testdata/eval/golden_set.json` — versioned qrels (committed)
- `docs/superpowers/eval-runs/*.json` — committed run artifacts (the article's evidence)
- `docs/dev/retrieval_eval.md` — how to run the benchmark

**New — reranker (F3):**
- `internal/reranker/client.go` + `client_test.go` — HTTP client for a bge-reranker-v2-m3 sidecar

**Modified:**
- `internal/case/sitesearch/resolve.go` (F1, F3, F5), `internal/case/mcp/resolve.go` (F1, F3)
- `internal/noteloader/search.go` (F2, F5), `internal/features/vector_search.go` (F3)
- `internal/mdchunk/chunk.go` (F4)
- `Makefile` (eval target), `.github/workflows/ci.yml` (opt-in eval job)
- `docs/dev/vector_search.md` (rewrite — closes pending task #2), `docs/dev/search.md` + `docs/dev/frontmatter_patches.md` (lang-per-folder recommendation)

---

# PHASE 0 — Benchmark library + CLI (TDD)

### Task 1: Retrieval metrics

**Files:**
- Create: `internal/retrievaleval/metrics.go`
- Test: `internal/retrievaleval/metrics_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags dev ./internal/retrievaleval/ -run 'TestRecallAtK|TestReciprocalRank|TestNDCGAtK' -v`
Expected: FAIL — `undefined: RecallAtK` (etc.)

- [ ] **Step 3: Write minimal implementation**

```go
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
	for i := 0; i < minInt(k, len(relevant)); i++ {
		idcg += 1.0 / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags dev ./internal/retrievaleval/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/retrievaleval/metrics.go internal/retrievaleval/metrics_test.go
git commit -m "feat(eval): add retrieval metrics (recall@k, ndcg@k, mrr)"
```

---

### Task 2: Golden-set types + loader

**Files:**
- Create: `internal/retrievaleval/goldenset.go`
- Test: `internal/retrievaleval/goldenset_test.go`

- [ ] **Step 1: Write the failing test**

```go
package retrievaleval

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadGoldenSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "golden.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
      "queries": [
        {"query":"как ищут экзопланеты","lang":"ru","direction":"ru->ru","expected_urls":["/search_astronomy"],"verified":true},
        {"query":"unverified one","lang":"en","direction":"en->en","expected_urls":["/x"],"verified":false}
      ]
    }`), 0o644))

	gs, err := LoadGoldenSet(path)
	require.NoError(t, err)
	require.Len(t, gs.Queries, 2)

	// Verified() filters out unverified entries.
	require.Len(t, gs.Verified(), 1)
	require.Equal(t, "как ищут экзопланеты", gs.Verified()[0].Query)
}

func TestGoldenSetValidateRejectsEmptyExpected(t *testing.T) {
	gs := GoldenSet{Queries: []GoldenQuery{{Query: "q", Verified: true}}}
	require.Error(t, gs.Validate())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags dev ./internal/retrievaleval/ -run TestLoadGoldenSet -v`
Expected: FAIL — `undefined: LoadGoldenSet`

- [ ] **Step 3: Write minimal implementation**

```go
package retrievaleval

import (
	"encoding/json"
	"fmt"
	"os"
)

// GoldenQuery is one labeled query → relevant-note(s) pair.
// Direction is one of: "ru->ru", "en->en", "ru->en", "en->ru".
type GoldenQuery struct {
	Query        string   `json:"query"`
	Lang         string   `json:"lang"`
	Direction    string   `json:"direction"`
	ExpectedURLs []string `json:"expected_urls"`
	Verified     bool     `json:"verified"`
}

type GoldenSet struct {
	Queries []GoldenQuery `json:"queries"`
}

func LoadGoldenSet(path string) (*GoldenSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read golden set: %w", err)
	}
	var gs GoldenSet
	if err := json.Unmarshal(data, &gs); err != nil {
		return nil, fmt.Errorf("parse golden set: %w", err)
	}
	if err := gs.Validate(); err != nil {
		return nil, err
	}
	return &gs, nil
}

// Validate ensures every query has text and at least one expected URL.
func (gs GoldenSet) Validate() error {
	for i, q := range gs.Queries {
		if q.Query == "" {
			return fmt.Errorf("query %d: empty query text", i)
		}
		if len(q.ExpectedURLs) == 0 {
			return fmt.Errorf("query %d (%q): no expected_urls", i, q.Query)
		}
	}
	return nil
}

// Verified returns only hand-verified queries (the trustworthy subset).
func (gs GoldenSet) Verified() []GoldenQuery {
	out := make([]GoldenQuery, 0, len(gs.Queries))
	for _, q := range gs.Queries {
		if q.Verified {
			out = append(out, q)
		}
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags dev ./internal/retrievaleval/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/retrievaleval/goldenset.go internal/retrievaleval/goldenset_test.go
git commit -m "feat(eval): add golden-set types and loader"
```

---

### Task 3: GraphQL search client

**Files:**
- Create: `internal/retrievaleval/client.go`
- Test: `internal/retrievaleval/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
package retrievaleval

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchClientReturnsOrderedURLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/_system/graphql", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"search":{"totalCount":2,"nodes":[
			{"url":"/first","score":0.9,"matchOrigin":"HYBRID"},
			{"url":"/second","score":0.4,"matchOrigin":"TEXT"}
		]}}}`))
	}))
	defer srv.Close()

	c := NewSearchClient(srv.URL+"/_system/graphql", "")
	res, err := c.Search(context.Background(), "anything")
	require.NoError(t, err)
	require.Equal(t, []string{"/first", "/second"}, res.URLs())
	require.Equal(t, "HYBRID", res.Nodes[0].MatchOrigin)
}

func TestSearchClientSurfacesGraphQLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))
	defer srv.Close()

	c := NewSearchClient(srv.URL+"/graphql", "")
	_, err := c.Search(context.Background(), "q")
	require.ErrorContains(t, err, "boom")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags dev ./internal/retrievaleval/ -run TestSearchClient -v`
Expected: FAIL — `undefined: NewSearchClient`

- [ ] **Step 3: Write minimal implementation**

```go
package retrievaleval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SearchNode struct {
	URL         string  `json:"url"`
	Score       float64 `json:"score"`
	MatchOrigin string  `json:"matchOrigin"`
}

type SearchResponse struct {
	Nodes []SearchNode
}

func (r SearchResponse) URLs() []string {
	out := make([]string, len(r.Nodes))
	for i, n := range r.Nodes {
		out[i] = n.URL
	}
	return out
}

type SearchClient struct {
	endpoint string
	bearer   string
	http     *http.Client
}

// NewSearchClient targets a GraphQL endpoint, e.g. "http://localhost:8081/graphql".
// bearer is optional (empty = anonymous; anonymous sees only live/free notes).
func NewSearchClient(endpoint, bearer string) *SearchClient {
	return &SearchClient{endpoint: endpoint, bearer: bearer, http: &http.Client{Timeout: 30 * time.Second}}
}

const searchQuery = `query($q: String!){ search(input:{query:$q}){ totalCount nodes { url score matchOrigin } } }`

func (c *SearchClient) Search(ctx context.Context, query string) (*SearchResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"query":     searchQuery,
		"variables": map[string]any{"q": query},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Data struct {
			Search struct {
				Nodes []SearchNode `json:"nodes"`
			} `json:"search"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", out.Errors[0].Message)
	}
	return &SearchResponse{Nodes: out.Data.Search.Nodes}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags dev ./internal/retrievaleval/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/retrievaleval/client.go internal/retrievaleval/client_test.go
git commit -m "feat(eval): add GraphQL search client for the harness"
```

---

### Task 4: Report aggregation + gate

**Files:**
- Create: `internal/retrievaleval/report.go`
- Test: `internal/retrievaleval/report_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
		{"/a", "/b"}, // q1: hit at rank 1
		{"/y", "/x"}, // q2: miss
	}

	rep := BuildReport("baseline", queries, retrievedByQuery, 10)

	require.InDelta(t, 0.5, rep.Overall.RecallAtK, 1e-9) // 1 of 2 found
	require.InDelta(t, 0.5, rep.Overall.MRR, 1e-9)       // (1 + 0)/2
	require.Contains(t, rep.ByDirection, "ru->ru")
	require.InDelta(t, 1.0, rep.ByDirection["ru->ru"].RecallAtK, 1e-9)
	require.InDelta(t, 0.0, rep.ByDirection["en->en"].RecallAtK, 1e-9)
}

func TestReportGate(t *testing.T) {
	rep := Report{Overall: Metrics{NDCGAtK: 0.80}}
	require.NoError(t, rep.Gate(0.75))
	require.Error(t, rep.Gate(0.85))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags dev ./internal/retrievaleval/ -run 'TestBuildReport|TestReportGate' -v`
Expected: FAIL — `undefined: BuildReport`

- [ ] **Step 3: Write minimal implementation**

```go
package retrievaleval

import "fmt"

type Metrics struct {
	Count     int     `json:"count"`
	RecallAtK float64 `json:"recall_at_k"`
	NDCGAtK   float64 `json:"ndcg_at_k"`
	MRR       float64 `json:"mrr"`
}

type QueryResult struct {
	Query      string   `json:"query"`
	Direction  string   `json:"direction"`
	Expected   []string `json:"expected_urls"`
	Retrieved  []string `json:"retrieved_urls"`
	RecallAtK  float64  `json:"recall_at_k"`
	NDCGAtK    float64  `json:"ndcg_at_k"`
	RR         float64  `json:"rr"`
}

type Report struct {
	Label      string             `json:"label"`
	K          int                `json:"k"`
	Overall    Metrics            `json:"overall"`
	ByDirection map[string]Metrics `json:"by_direction"`
	Queries    []QueryResult      `json:"queries"`
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags dev ./internal/retrievaleval/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/retrievaleval/report.go internal/retrievaleval/report_test.go
git commit -m "feat(eval): add report aggregation and regression gate"
```

---

### Task 5: CLI `cmd/evalretrieval`

**Files:**
- Create: `cmd/evalretrieval/main.go`

- [ ] **Step 1: Write the implementation**

```go
// Command evalretrieval runs the retrieval golden set against a running trip2g
// GraphQL endpoint and writes a JSON metrics artifact.
//
// Usage:
//   go run ./cmd/evalretrieval \
//     -golden testdata/eval/golden_set.json \
//     -endpoint http://localhost:8081/graphql \
//     -label baseline -k 10 \
//     -out docs/superpowers/eval-runs/00-baseline.json \
//     -fail-under-ndcg 0
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
	bearer := flag.String("bearer", "", "optional Authorization bearer token")
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
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./cmd/evalretrieval/`
Expected: builds with no output.

- [ ] **Step 3: Verify it runs and reports a clear error without a server**

Run: `go run ./cmd/evalretrieval -golden /nonexistent.json`
Expected: prints `error: read golden set: ...` and exits nonzero (confirms wiring; no server needed).

- [ ] **Step 4: Commit**

```bash
git add cmd/evalretrieval/main.go
git commit -m "feat(eval): add evalretrieval CLI"
```

---

### Task 6: Make target, runner script, and docs

**Files:**
- Create: `scripts/eval.sh`
- Modify: `Makefile` (add `eval` target near the other phony targets)
- Create: `docs/dev/retrieval_eval.md`

- [ ] **Step 1: Write `scripts/eval.sh`**

```bash
#!/usr/bin/env bash
# Run the retrieval benchmark against a running trip2g instance.
# Assumes the stack is up with bge-m3 and the demo vault synced + embeddings generated
# (see docs/dev/retrieval_eval.md). Pass a label and output path.
set -euo pipefail

LABEL="${1:-run}"
OUT="${2:-docs/superpowers/eval-runs/${LABEL}.json}"
ENDPOINT="${EVAL_ENDPOINT:-http://localhost:21081/_system/graphql}"

go run ./cmd/evalretrieval \
  -golden testdata/eval/golden_set.json \
  -endpoint "$ENDPOINT" \
  -label "$LABEL" \
  -k 10 \
  -out "$OUT" \
  -fail-under-ndcg "${EVAL_MIN_NDCG:-0}"
```

- [ ] **Step 2: Make it executable + add Make target**

Run: `chmod +x scripts/eval.sh`

Add to `Makefile`:

```makefile
.PHONY: eval
eval: ## Run retrieval benchmark (LABEL=baseline OUT=path); needs a running stack
	./scripts/eval.sh $(LABEL) $(OUT)
```

- [ ] **Step 3: Write `docs/dev/retrieval_eval.md`**

Document: prerequisites (stack up via `./scripts/test-e2e.sh` env or `make air` + `embedding-server/` with bge-m3 + `cd docs && obsidian-sync` to push the demo vault, then wait for embedding jobs to drain), how to run `make eval LABEL=baseline`, where artifacts land (`docs/superpowers/eval-runs/`), how to read the metrics, and how the CI gate works. State explicitly: the benchmark needs a reachable bge-m3 endpoint because the vector lane embeds the query live per request.

- [ ] **Step 4: Verify the doc references real paths**

Run: `go vet ./cmd/evalretrieval/ ./internal/retrievaleval/`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add scripts/eval.sh Makefile docs/dev/retrieval_eval.md
git commit -m "feat(eval): add eval make target, runner script, and docs"
```

---

# PHASE 1 — Golden set + bge-m3 baseline

> **Corpus decision (refined):** instead of syncing the demo vault into a dev instance, the benchmark runs against a **dedicated, isolated stack** — `docker-compose.vecbench.yml` + `scripts/vecbench.sh` + the purpose-built bilingual vault `testdata/vecbench/vault/` (6 topics × {en, ru}, all four retrieval directions). This stack already exists in the repo (committed before this phase). The GraphQL endpoint is `http://localhost:21081/_system/graphql` (`/graphql` is deprecated).

### Task 7: Bring up the isolated vecbench stack

**Files:** none (operational); produces a running benchmark instance.

- [ ] **Step 1: Bring it up**

Run: `./scripts/vecbench.sh up`
This builds + starts `docker-compose.vecbench.yml` (minio + bge-m3 embedding + one app on ports 21081/21082), auto-provisions an API key (dev sign-in code `111111`), pushes `testdata/vecbench/vault/` via obsidian-sync, and waits for embedding jobs to drain.
Expected: ends with `✓ vecbench up.` and a saved `.vecbench-api-key`.

- [ ] **Step 2: Smoke-test the search endpoint**

Run:
```bash
curl -s http://localhost:21081/_system/graphql -H 'Content-Type: application/json' \
  -d '{"query":"query($q:String!){search(input:{query:$q}){nodes{url score matchOrigin}}}","variables":{"q":"как ищут экзопланеты"}}'
```
Expected: JSON `nodes` with at least one `"matchOrigin":"VECTOR"` or `"HYBRID"`, and the exoplanets note (`/ru/ekzoplanety` or its real permalink) present. Record the exact `url` values — they populate `expected_urls` in Task 8.

---

### Task 8: Build and hand-verify the golden set

**Files:**
- Create: `testdata/eval/golden_set.json`

- [ ] **Step 1: Enumerate the corpus**

The vecbench vault has 12 content notes across 6 topics, each in EN (`en/…`) and RU (`ru/…`): exoplanets, sourdough, goroutines, vector-search, photosynthesis, green tea (see `testdata/vecbench/vault/README.md`). Every topic exists in both languages, so all four directions (ru→ru, en→en, ru→en, en→ru) are achievable.

- [ ] **Step 2: LLM-generate candidate queries (silver)**

For each note, prompt an LLM with the note text: "Generate 3 realistic search queries a user would type to find this note. Use synonyms; do NOT copy exact phrases. Output in {ru|en}. Add one 'need-based' query describing the underlying need." For cross-lingual pairs, also generate RU queries for EN notes and vice versa (the topics are parallel). Aim for ~60–80 candidates with even direction coverage.

- [ ] **Step 3: Hand-verify every pair (promote silver → gold)**

For each candidate: run it through the live endpoint (Task 7 smoke command). Confirm the intended note is genuinely the right answer; set `expected_urls` to the **actual** `url`(s) the endpoint returns for that note. Drop ambiguous/wrong pairs (expect ~20% loss). Keep `verified: true` only for confirmed pairs. Target ≥50 verified, roughly balanced across the four directions.

- [ ] **Step 4: Write `testdata/eval/golden_set.json`**

```json
{
  "queries": [
    {"query": "как астрономы находят планеты у далёких звёзд", "lang": "ru", "direction": "ru->ru", "expected_urls": ["/ru/ekzoplanety"], "verified": true},
    {"query": "transit method for detecting planets", "lang": "en", "direction": "en->en", "expected_urls": ["/en/exoplanets"], "verified": true},
    {"query": "как заваривать зелёный чай без горечи", "lang": "ru", "direction": "ru->en", "expected_urls": ["/en/green-tea", "/ru/zelenyy-chay"], "verified": true},
    {"query": "goroutines and channels concurrency", "lang": "en", "direction": "en->ru", "expected_urls": ["/ru/goroutines", "/en/goroutines"], "verified": true}
  ]
}
```
(Replace `expected_urls` with the real permalinks observed in Step 3; add the rest of the verified pairs. For cross-lingual entries, the same-topic note in the *other* language is the relevant target.)

- [ ] **Step 5: Validate the file loads**

Run: `go run ./cmd/evalretrieval -golden testdata/eval/golden_set.json -endpoint http://localhost:21081/_system/graphql -label validate-only -k 10`
Expected: prints metrics for all verified queries with no load/parse error.

- [ ] **Step 6: Commit**

```bash
git add testdata/eval/golden_set.json
git commit -m "test(eval): add hand-verified RU/EN retrieval golden set"
```

---

### Task 9: Establish the bge-m3 baseline

**Files:**
- Create: `docs/superpowers/eval-runs/00-baseline-bgem3.json`

- [ ] **Step 1: Run the benchmark**

Run: `./scripts/vecbench.sh eval 00-baseline-bgem3`
Expected: prints overall + per-direction Recall@10 / nDCG@10 / MRR and writes `docs/superpowers/eval-runs/00-baseline-bgem3.json`.

- [ ] **Step 2: Record the headline numbers**

Append the overall + per-direction numbers to a running scratch table in `docs/dev/retrieval_eval.md` under a "Run history" heading (this becomes the article's data spine).

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/eval-runs/00-baseline-bgem3.json docs/dev/retrieval_eval.md
git commit -m "test(eval): record bge-m3 baseline retrieval metrics"
```

---

# PHASE 2 — Apply fixes, measure each

Each fix below: (a) add/adjust a focused test, (b) make the change, (c) re-run the benchmark with a new label, (d) compare to the previous artifact, (e) commit code + artifact together. If a fix **regresses** overall nDCG@10, keep the artifact, note it in the run history, and decide (revert / re-tune) before moving on.

### Task 10: F1 — Stop truncating the vector list before RRF

**Files:**
- Modify: `internal/case/sitesearch/resolve.go:118`
- Modify: `internal/case/mcp/resolve.go:26`
- Test: `internal/case/sitesearch/resolve_test.go`
- Artifact: `docs/superpowers/eval-runs/10-wide-fusion.json`

- [ ] **Step 1: Write a failing test proving more vector candidates reach fusion**

Add to `internal/case/sitesearch/resolve_test.go` a test that calls `mergeResults` with a text list and a vector list of length 30 where a note at vector-rank 15 is also high in text, and assert it appears in the merged output (today it would be absent if the caller truncated to 5). Since `mergeResults` itself doesn't truncate, assert the **constant** guards the right window:

```go
func TestVectorTopKWideEnoughForFusion(t *testing.T) {
	// Guard against regressing back to the pre-fusion truncation bug.
	require.GreaterOrEqual(t, vectorTopK, 50)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags dev ./internal/case/sitesearch/ -run TestVectorTopKWideEnoughForFusion -v`
Expected: FAIL — `vectorTopK` is 5.

- [ ] **Step 3: Make the change**

In `internal/case/sitesearch/resolve.go:118`: `const vectorTopK = 50` (was 5). Keep `rrfK = 60` and the post-merge `[:20]` cap (resolve.go:257) unchanged.
In `internal/case/mcp/resolve.go:26`: `DefaultVectorSearchLimit = 50` (was 10). Keep `MaxMergedResults = 20` and `DefaultDisplayLimit = 10` unchanged.

- [ ] **Step 4: Run tests**

Run: `go test -tags dev ./internal/case/sitesearch/ ./internal/case/mcp/ -v`
Expected: PASS.

- [ ] **Step 5: Re-embed not needed; re-run benchmark**

Restart the app (constants are compile-time), then:
Run: `make eval LABEL=10-wide-fusion OUT=docs/superpowers/eval-runs/10-wide-fusion.json`
Expected: overall Recall@10 / nDCG@10 ≥ baseline (especially on multi-term queries).

- [ ] **Step 6: Commit**

```bash
git add internal/case/sitesearch/resolve.go internal/case/sitesearch/resolve_test.go internal/case/mcp/resolve.go docs/superpowers/eval-runs/10-wide-fusion.json
git commit -m "fix(search): feed full candidate pool into RRF (vectorTopK 5->50)"
```

---

### Task 11: F2 — Per-language bleve analyzer + lang-via-frontmatter docs

**Files:**
- Modify: `internal/noteloader/search.go` (analyzer setup ~14/32/38/45, indexing loop ~82, `Search` ~140)
- Test: `internal/noteloader/search_test.go`
- Modify: `docs/dev/search.md`, `docs/dev/frontmatter_patches.md` (lang-per-folder recommendation)
- Artifact: `docs/superpowers/eval-runs/11-per-lang-analyzer.json`

- [ ] **Step 1: Write a failing test for English stemming**

Add to `internal/noteloader/search_test.go` an index built from one English note (`Lang: "en"`, body contains "running races") and assert a query "run race" matches it (Russian analyzer would not stem English correctly). Use the existing test harness pattern in that file (it already builds indexes via the loader).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags dev ./internal/noteloader/ -run TestEnglishStemming -v`
Expected: FAIL — English query misses under the `"ru"` analyzer.

- [ ] **Step 3: Implement per-language fields**

In `createSearchIndex()` register both analyzers and index Title/Body into dual fields — `Title`/`Body` (ru-analyzed) and `Title_en`/`Body_en` (en-analyzed) — by adding `en`-analyzed field mappings alongside the existing `ru` ones. In `buildSearchIndex` (the indexing loop ~line 82), continue indexing every note into both field sets (cheap at this corpus size) OR route by `note.Lang` when populated, defaulting to `ru` when empty. In `Search()` (line 140), issue a `DisjunctionQuery` over both the ru and en fields so a query matches regardless of the note's language. Keep `MemOnly` rebuild flow and `searchRequest.Size = 20`.

- [ ] **Step 4: Run tests**

Run: `go test -tags dev ./internal/noteloader/ -v`
Expected: PASS (English query now matches; existing RU tests still pass).

- [ ] **Step 5: Document lang-via-frontmatter-patches**

In `docs/dev/search.md` (and a cross-link in `docs/dev/frontmatter_patches.md`), add a recommendation: the analyzer keys off `note.Lang`, which comes from the `lang:` frontmatter field. Operators can bulk-set it per folder with a frontmatter patch — no per-note edits:

```
include: ["en/**"]   jsonnet: { lang: "en" }
include: ["ru/**"]   jsonnet: { lang: "ru" }
```
or path-conditional in one rule:
```
include: ["*"]
jsonnet: if std.startsWith(path, "en/") then { lang: "en" }
         else if std.startsWith(path, "ru/") then { lang: "ru" } else {}
```
Note vault-based patches (`type: frontmatter-patch`) work too. This makes the per-language analyzer effective on existing vaults that lack explicit `lang:`.

- [ ] **Step 6: Re-run benchmark**

Restart the app (index rebuilds on load), then:
Run: `make eval LABEL=11-per-lang-analyzer OUT=docs/superpowers/eval-runs/11-per-lang-analyzer.json`
Expected: EN-direction and cross-lingual buckets improve; RU unchanged-or-better.

- [ ] **Step 7: Commit**

```bash
git add internal/noteloader/search.go internal/noteloader/search_test.go docs/dev/search.md docs/dev/frontmatter_patches.md docs/superpowers/eval-runs/11-per-lang-analyzer.json
git commit -m "fix(search): per-language bleve analyzer; document lang via frontmatter patches"
```

---

### Task 12: F3 — Cross-encoder reranker after RRF

**Files:**
- Create: `internal/reranker/client.go`, `internal/reranker/client_test.go`
- Modify: `internal/features/vector_search.go` (add `RerankerConfig`)
- Modify: `internal/case/sitesearch/resolve.go` (rerank step after `mergeResults`)
- Modify: `internal/case/mcp/resolve.go` (same step) and the sitesearch `Env` (expose reranker)
- Artifact: `docs/superpowers/eval-runs/12-reranker.json`

- [ ] **Step 1: Write a failing test for the reranker client**

```go
package reranker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRerankerReordersByScore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// bge-reranker-style response: results with index + relevance_score
		_, _ = w.Write([]byte(`{"results":[{"index":2,"relevance_score":0.9},{"index":0,"relevance_score":0.5},{"index":1,"relevance_score":0.1}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/rerank", "BAAI/bge-reranker-v2-m3")
	order, err := c.Rerank(context.Background(), "q", []string{"doc0", "doc1", "doc2"})
	require.NoError(t, err)
	require.Equal(t, []int{2, 0, 1}, order)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags dev ./internal/reranker/ -v`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement the reranker client**

```go
// Package reranker calls an OpenAI-compatible / TEI-style cross-encoder rerank endpoint.
package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

type Client struct {
	endpoint string
	model    string
	http     *http.Client
}

func New(endpoint, model string) *Client {
	return &Client{endpoint: endpoint, model: model, http: &http.Client{Timeout: 10 * time.Second}}
}

// Rerank returns document indices ordered best-first for the given query.
func (c *Client) Rerank(ctx context.Context, query string, docs []string) ([]int, error) {
	body, _ := json.Marshal(map[string]any{"model": c.model, "query": query, "documents": docs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rerank request: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Results []struct {
			Index int     `json:"index"`
			Score float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}
	sort.SliceStable(out.Results, func(i, j int) bool { return out.Results[i].Score > out.Results[j].Score })
	order := make([]int, len(out.Results))
	for i, r := range out.Results {
		order[i] = r.Index
	}
	return order, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags dev ./internal/reranker/ -v`
Expected: PASS.

- [ ] **Step 5: Add `RerankerConfig` to features**

In `internal/features/vector_search.go`, add to `VectorSearchConfig`:

```go
Reranker RerankerConfig `json:"reranker"`
```
```go
type RerankerConfig struct {
	Enabled  bool   `json:"enabled"`
	BaseURL  string `json:"base_url"` // e.g. "http://reranker:8001/rerank"
	Model    string `json:"model"`    // e.g. "BAAI/bge-reranker-v2-m3"
	TopN     int    `json:"top_n"`    // candidates to rerank (default 50)
	OutputK  int    `json:"output_k"` // results to keep after rerank (default 10)
}
```
Add defaults in `Parse()`/validation (TopN=50, OutputK=10 when zero).

- [ ] **Step 6: Integrate the rerank step (flag-gated)**

In `internal/case/sitesearch/resolve.go`, after `mergeResults` and before returning, if `env.Features().VectorSearch.Reranker.Enabled`: take the top `TopN` fused results, send `{query, [result.HighlightedContent or chunk text]}` to the reranker, reorder, truncate to `OutputK`. Add a `Reranker()` accessor to the sitesearch `Env` interface and implement it on `*app` in `cmd/server/main.go` (build the client once from config). On reranker error, log a Warn and keep the RRF order (graceful degradation, matching the existing vector-fail behavior).

- [ ] **Step 7: Run tests + build**

Run: `go test -tags dev ./internal/... && go build ./...`
Expected: PASS / builds.

- [ ] **Step 8: Re-run benchmark with reranker enabled**

Start a bge-reranker-v2-m3 sidecar; set `FEATURES.vector_search.reranker = {"enabled":true,"base_url":...,"model":"BAAI/bge-reranker-v2-m3"}`. Restart app.
Run: `make eval LABEL=12-reranker OUT=docs/superpowers/eval-runs/12-reranker.json`
Expected: nDCG@10 / Recall@5 improvement (literature: Recall@5 ~0.70→0.82).

- [ ] **Step 9: Commit**

```bash
git add internal/reranker/ internal/features/vector_search.go internal/case/sitesearch/resolve.go internal/case/mcp/resolve.go cmd/server/main.go docs/superpowers/eval-runs/12-reranker.json
git commit -m "feat(search): add flag-gated cross-encoder reranker after RRF"
```

---

### Task 13: F4 — Heading-path chunk prefix + token-aware sizing

**Files:**
- Modify: `internal/mdchunk/chunk.go` (Split ~21, constants 6–8)
- Test: `internal/mdchunk/chunk_test.go`
- Artifact: `docs/superpowers/eval-runs/13-chunking.json`

- [ ] **Step 1: Write a failing test for the heading path**

```go
func TestChunkCarriesHeadingPath(t *testing.T) {
	md := []byte("# Intro\n\ntext a\n\n## Details\n\ntext b under details\n")
	chunks := Split("My Note", md)
	// The chunk covering "text b" should include the heading breadcrumb.
	var found bool
	for _, c := range chunks {
		if strings.Contains(c.Content, "text b under details") {
			require.Contains(t, c.Content, "My Note")
			require.Contains(t, c.Content, "Details")
			found = true
		}
	}
	require.True(t, found)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags dev ./internal/mdchunk/ -run TestChunkCarriesHeadingPath -v`
Expected: FAIL — only the note title is prefixed today.

- [ ] **Step 3: Implement heading-stack tracking + token-aware sizing**

In `internal/mdchunk/chunk.go`: while iterating blocks in `Split`, maintain a heading stack (push on `isHeadingBlock`, pop to the heading level). Build each chunk's content as `"{title} > {h1} > {h2}...\n\n{body}"`. Switch `chunkTargetSize`/`chunkMinSize`/`chunkOverlap` to a token budget: keep char constants but add a conservative per-language ratio (Cyrillic ≈ 2 chars/token, Latin ≈ 4 chars/token) so chunks stay under ~480 tokens (under E5/bge-m3's 512 limit); split oversized single blocks instead of emitting whole. Keep `StripFrontmatter`.

- [ ] **Step 4: Run tests**

Run: `go test -tags dev ./internal/mdchunk/ -v`
Expected: PASS (new test + existing incl. the Cyrillic test at chunk_test.go).

- [ ] **Step 5: Re-embed and re-run benchmark**

Chunk content changed → content hashes change → the async job re-embeds automatically. Restart app, re-sync demo vault if needed, wait for the job queue to drain (Task 7 Step 3).
Run: `make eval LABEL=13-chunking OUT=docs/superpowers/eval-runs/13-chunking.json`
Expected: section-specific queries improve; no overflow truncation on RU.

- [ ] **Step 6: Commit**

```bash
git add internal/mdchunk/chunk.go internal/mdchunk/chunk_test.go docs/superpowers/eval-runs/13-chunking.json
git commit -m "feat(chunking): carry heading path into chunks; token-aware sizing"
```

---

### Task 14 (OPTIONAL): F5 — AND→OR fallback + cosine norm precompute + observability

**Files:**
- Modify: `internal/noteloader/search.go` (`Search` ~146)
- Modify: `internal/case/sitesearch/resolve.go` (`vectorSearch` ~120, `cosineSimilarity` ~335)
- Test: `internal/noteloader/search_test.go`
- Artifact: `docs/superpowers/eval-runs/14-fallback-perf.json`

- [ ] **Step 1: Write a failing test for the OR fallback**

Add a test: a two-term query where one term is absent returns the note via the OR fallback (today AND yields zero text hits).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags dev ./internal/noteloader/ -run TestAndOrFallback -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

In `Search()`: run AND first; if results are below a small threshold, re-run with `MatchQueryOperatorOr` and merge. In `vectorSearch`/`cosineSimilarity`: precompute each passage's L2 norm once (add a `Norm float32` to `NoteChunk`, set at load in `noteloader.loadChunks` using `BytesToFloat32Slice`) and skip per-query norm recomputation — OR, if you confirm the bge-m3 server returns unit-normalized vectors, switch to a plain dot product. Add a Warn log of `len(chunks)` + scan duration in `vectorSearch`.

- [ ] **Step 4: Run tests + benchmark**

Run: `go test -tags dev ./internal/... && make eval LABEL=14-fallback-perf OUT=docs/superpowers/eval-runs/14-fallback-perf.json`
Expected: recall steady-or-up; lower per-query latency in the log.

- [ ] **Step 5: Commit**

```bash
git add internal/noteloader/search.go internal/case/sitesearch/resolve.go internal/model/chunk.go internal/noteloader/loader.go internal/noteloader/search_test.go docs/superpowers/eval-runs/14-fallback-perf.json
git commit -m "perf(search): AND->OR fallback, precomputed norms, vector-search metrics"
```

---

# PHASE 3 — Article + docs

### Task 15: Aggregate results, write the article, fix the stale doc

**Files:**
- Create: `docs/en/blog/improving-vector-search.md` + `docs/ru/blog/improving-vector-search.md` (bilingual, per project convention)
- Modify: `docs/dev/vector_search.md` (rewrite to match reality — closes pending task #2)

- [ ] **Step 1: Build the results table from artifacts**

From `docs/superpowers/eval-runs/*.json`, assemble one table: rows = runs (baseline → wide-fusion → analyzer → reranker → chunking → fallback), columns = Recall@10, nDCG@10, MRR (overall + by direction), with the delta vs the previous run.

- [ ] **Step 2: Write the article**

Lead with the answer (Minto, per `docs/CLAUDE.md` writing rules): what we built, the baseline, each change and its measured effect, what worked / what didn't, and the final lift. Use concrete numbers from Step 1, name the techniques (hybrid + RRF, per-language analyzer, cross-encoder rerank, contextual chunking), and link the eval harness so readers can reproduce. Write EN first, then the RU pair.

- [ ] **Step 3: Rewrite `docs/dev/vector_search.md`**

Replace the stale content: document the live hybrid pipeline (bleve BM25 + per-chunk cosine + RRF k=60, cap 20), per-chunk embeddings, the bge-m3 default, the eval harness, the reranker flag, and the per-language analyzer. Remove "hybrid search" from "Future Improvements". Verify every statement against current code before writing.

- [ ] **Step 4: Verify links/build**

Run: `go build ./... && go vet ./internal/retrievaleval/ ./cmd/evalretrieval/`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add docs/en/blog/improving-vector-search.md docs/ru/blog/improving-vector-search.md docs/dev/vector_search.md
git commit -m "docs(search): article on improving vector search; refresh vector_search.md"
```

---

## Self-Review

**Spec coverage:** benchmark first (Phase 0–1) ✓; apply fixes and measure (Phase 2, F1–F5 each with before/after artifact) ✓; article at the end (Phase 3) ✓; bge-m3 baseline ✓; demo-vault corpus ✓; LLM-generate + hand-verify golden set (Task 8) ✓; CLI + JSON artifact + CI gate (Tasks 5–6, `-fail-under-ndcg`) ✓; lang-via-frontmatter-patches recommendation (Task 11 Step 5) ✓.

**Placeholder scan:** metric/golden-set/client/report/CLI/reranker code is complete and runnable; constant edits cite exact file:line; data/writing tasks (8, 9, 15) give exact paths, formats, and verification commands rather than test code (appropriate for non-code deliverables).

**Type consistency:** `GoldenQuery`/`GoldenSet`, `SearchClient.Search`→`SearchResponse.URLs()`, `BuildReport(...) Report`, `Report.Gate(minNDCG)`, `reranker.New(endpoint, model).Rerank(...) []int`, `RerankerConfig` fields are used consistently across tasks. `vectorTopK`/`DefaultVectorSearchLimit`/`rrfK`/`MaxMergedResults` names match the verified source.

**Known limitation (flagged, not a gap):** the demo vault is RU-heavy, so cross-lingual buckets may be thin; Task 8 Step 7 optionally adds EN notes. The benchmark requires a reachable bge-m3 endpoint (vector lane embeds queries live) — documented in Task 6/7.
