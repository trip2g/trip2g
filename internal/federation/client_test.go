package federation_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"trip2g/internal/federation"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestClientImplementsFederationInterface(t *testing.T) {
	var _ model.Federation = (*federation.Client)(nil)
}

func TestClientCallsSixFederationTools(t *testing.T) {
	var names []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)                            //nolint:testifylint // require safe: handler called synchronously via httptest
		require.Equal(t, "2", r.Header.Get("X-Mcp-Federation-Depth")) //nolint:testifylint // require safe: handler called synchronously via httptest
		requireJWTKid(t, r.Header.Get("Authorization"), "kid-1")

		var req struct {
			Method string `json:"method"`
			Params struct {
				Name string `json:"name"`
			} `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req)) //nolint:testifylint // require safe: handler called synchronously via httptest
		require.Equal(t, "tools/call", req.Method)               //nolint:testifylint // require safe: handler called synchronously via httptest
		names = append(names, req.Params.Name)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"ok"}],"structuredContent":{"ok":true}},"id":"rid"}`))
	}))
	defer server.Close()

	client := federation.NewClient(model.FederationPeer{
		KBURL:  server.URL,
		KID:    "kid-1",
		Secret: []byte("12345678901234567890123456789012"),
		Issuer: "https://hub.local",
		Depth:  1,
	}, &fasthttp.Client{}, false)

	_, err := client.Search(context.Background(), model.FederationSearchParams{Query: "q"})
	require.NoError(t, err)
	_, err = client.Similar(context.Background(), model.FederationSimilarParams{PID: 1})
	require.NoError(t, err)
	_, err = client.NoteHTML(context.Background(), model.FederationNoteHTMLParams{PID: 1})
	require.NoError(t, err)
	_, err = client.FederatedSearch(context.Background(), model.FederationSearchParams{Query: "q", KBID: "deep"})
	require.NoError(t, err)
	_, err = client.FederatedSimilar(context.Background(), model.FederationSimilarParams{KBID: "deep", PID: 1})
	require.NoError(t, err)
	result, err := client.FederatedNoteHTML(context.Background(), model.FederationNoteHTMLParams{KBID: "deep", PID: 1})
	require.NoError(t, err)

	require.Equal(t, []string{
		"search",
		"similar",
		"note_html",
		"federated_search",
		"federated_similar",
		"federated_note_html",
	}, names)
	require.Equal(t, "ok", result.Content[0].Text)
	require.JSONEq(t, `{"ok":true}`, string(result.StructuredContent))
}

func requireJWTKid(t *testing.T, authorization string, wantKID string) {
	t.Helper()

	require.True(t, strings.HasPrefix(authorization, "Bearer "))
	token := strings.TrimPrefix(authorization, "Bearer ")
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	var header struct {
		KID string `json:"kid"`
	}
	require.NoError(t, json.Unmarshal(headerJSON, &header))
	require.Equal(t, wantKID, header.KID)
}
