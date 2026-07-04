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
	endpoint   string
	bearer     string
	cookieName string
	http       *http.Client
}

// NewSearchClient targets a GraphQL endpoint, e.g. "http://localhost:21081/_system/graphql".
// bearer is optional (empty = anonymous; anonymous sees only live/free notes).
func NewSearchClient(endpoint, bearer string) *SearchClient {
	return &SearchClient{endpoint: endpoint, bearer: bearer, http: &http.Client{Timeout: 30 * time.Second}}
}

// WithCookieAuth sends the session token as a cookie instead of a Bearer header.
// A session-token JWT sent via Authorization: Bearer is misclassified by the
// scoped-shortapitoken stamp and yields deny-all in search; the session cookie
// is the reliable admin path for the benchmark.
func (c *SearchClient) WithCookieAuth(cookieName string) *SearchClient {
	c.cookieName = cookieName
	return c
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
		if c.cookieName != "" {
			req.Header.Set("Cookie", c.cookieName+"="+c.bearer)
		} else {
			req.Header.Set("Authorization", "Bearer "+c.bearer)
		}
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
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", out.Errors[0].Message)
	}
	return &SearchResponse{Nodes: out.Data.Search.Nodes}, nil
}
