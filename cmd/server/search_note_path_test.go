package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func (m *freeNoteMatrix) requireSearchCarriesVaultPath(t *testing.T) {
	t.Helper()

	query := `{ search(input: {query: "matrixword"}) { nodes { document { ... on PublicNote { path notePath } } } } }`
	body := m.graphql(t, "admin", query, nil)

	var out struct {
		Data struct {
			Search struct {
				Nodes []struct {
					Document *struct {
						Path     string `json:"path"`
						NotePath string `json:"notePath"`
					} `json:"document"`
				} `json:"nodes"`
			} `json:"search"`
		} `json:"data"`
	}
	err := json.Unmarshal([]byte(body), &out)
	require.NoError(t, err, body)

	want := map[string]string{}
	for _, note := range m.notes {
		want[note.permalink] = note.path
	}

	got := map[string]string{}
	for _, node := range out.Data.Search.Nodes {
		require.NotNil(t, node.Document, body)
		got[node.Document.Path] = node.Document.NotePath
	}
	require.NotEmpty(t, got, body)

	for permalink, notePath := range got {
		require.Equal(t, want[permalink], notePath, "search hit %s", permalink)
	}
}
