// Package reranker calls an OpenAI-/TEI-style cross-encoder rerank endpoint
// (e.g. a bge-reranker-v2-m3 sidecar) to score candidate documents by their
// relevance to a query — the second stage of retrieve-then-rerank.
//
// Unlike a pure reorder, this returns per-document scores so the caller can
// BLEND the cross-encoder signal with the first-stage (RRF) ranking instead of
// letting the cross-encoder override it. Blending is what makes the second
// stage safe on top of a strong hybrid first stage (see docs/dev/reranker.md).
package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultTimeout = 10 * time.Second

type Client struct {
	endpoint string
	model    string
	http     *http.Client
}

// New builds a client with the default per-request timeout.
func New(endpoint, model string) *Client {
	return NewWithTimeout(endpoint, model, 0)
}

// NewWithTimeout builds a client with a per-request timeout. A non-positive
// timeout falls back to the default. CPU cross-encoder inference over a wide
// candidate set can be slow, so callers should raise this from config.
func NewWithTimeout(endpoint, model string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		endpoint: endpoint,
		model:    model,
		// Transport is left nil so the client uses http.DefaultTransport, which
		// pools connections across all clients constructed per request from config.
		http: &http.Client{Timeout: timeout},
	}
}

// Result is a cross-encoder relevance score for one input document.
// Index refers into the docs slice passed to Rerank.
type Result struct {
	Index int
	Score float64
}

// Rerank returns a cross-encoder relevance score for each document, keyed by its
// index into docs. Results are not sorted — the caller blends these scores with
// its own first-stage ranking. An empty docs slice returns nil, nil.
func (c *Client) Rerank(ctx context.Context, query string, docs []string) ([]Result, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	body, _ := json.Marshal(map[string]any{"query": query, "texts": docs})
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rerank status %d", resp.StatusCode)
	}

	var out []struct {
		Index int     `json:"index"`
		Score float64 `json:"score"`
	}
	if decErr := json.NewDecoder(resp.Body).Decode(&out); decErr != nil {
		return nil, fmt.Errorf("decode rerank response: %w", decErr)
	}

	results := make([]Result, 0, len(out))
	for _, r := range out {
		if r.Index >= 0 && r.Index < len(docs) {
			results = append(results, Result{Index: r.Index, Score: r.Score})
		}
	}
	return results, nil
}
