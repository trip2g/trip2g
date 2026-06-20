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

	c := NewSearchClient(srv.URL+"/_system/graphql", "")
	_, err := c.Search(context.Background(), "q")
	require.ErrorContains(t, err, "boom")
}
