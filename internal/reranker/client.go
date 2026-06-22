// Package reranker calls an OpenAI-/TEI-style cross-encoder rerank endpoint
// (e.g. a bge-reranker-v2-m3 sidecar) to reorder candidate documents by their
// relevance to a query — the second stage of retrieve-then-rerank.
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

// sharedHTTP is reused across clients so connections pool even when callers
// construct a Client per request from config.
var sharedHTTP = &http.Client{Timeout: 10 * time.Second} //nolint:gochecknoglobals // intentional shared pool

type Client struct {
	endpoint string
	model    string
}

func New(endpoint, model string) *Client {
	return &Client{endpoint: endpoint, model: model}
}

// Rerank returns document indices (into docs) ordered best-first for the query.
// The returned slice may be shorter than docs if the server drops entries.
func (c *Client) Rerank(ctx context.Context, query string, docs []string) ([]int, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	body, _ := json.Marshal(map[string]any{"model": c.model, "query": query, "documents": docs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := sharedHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rerank request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rerank status %d", resp.StatusCode)
	}

	var out struct {
		Results []struct {
			Index int     `json:"index"`
			Score float64 `json:"relevance_score"`
		} `json:"results"`
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}

	// Sort defensively in case the server returns unsorted results.
	sort.SliceStable(out.Results, func(i, j int) bool { return out.Results[i].Score > out.Results[j].Score })
	order := make([]int, 0, len(out.Results))
	for _, r := range out.Results {
		if r.Index >= 0 && r.Index < len(docs) {
			order = append(order, r.Index)
		}
	}
	return order, nil
}
