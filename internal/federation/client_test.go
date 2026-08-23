package federation_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
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

	_, err := client.Search(context.Background(), model.MCPSearchParams{Query: "q"})
	require.NoError(t, err)
	_, err = client.Similar(context.Background(), model.MCPSimilarParams{PID: model.PID{Value: 1}})
	require.NoError(t, err)
	_, err = client.NoteHTML(context.Background(), model.MCPNoteHTMLParams{PID: model.PID{Value: 1}})
	require.NoError(t, err)
	_, err = client.FederatedSearch(context.Background(), model.MCPSearchParams{Query: "q", KBID: "deep"})
	require.NoError(t, err)
	_, err = client.FederatedSimilar(context.Background(), model.MCPSimilarParams{KBID: "deep", PID: model.PID{Value: 1}})
	require.NoError(t, err)
	result, err := client.FederatedNoteHTML(context.Background(), model.MCPNoteHTMLParams{KBID: "deep", PID: model.PID{Value: 1}})
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

// A rotation whose response was lost leaves the peer holding the new key while
// this side still calls the old one current — or the reverse, if the call never
// arrived. Trying the previous key once is what turns that into a link that
// takes an extra attempt instead of a link that is down.
func TestClientRetriesWithThePreviousKey(t *testing.T) {
	var attempts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		require.NoError(t, readErr) //nolint:testifylint // require safe: handler called synchronously via httptest

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		w.Header().Set("Content-Type", "application/json")

		if verifyHS256(token, []byte("current-secret"), body) {
			attempts = append(attempts, "current")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"content":[{"type":"text","text":"ok"}]}}`))
			return
		}
		if verifyHS256(token, []byte("previous-secret"), body) {
			attempts = append(attempts, "previous")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"content":[{"type":"text","text":"ok"}]}}`))
			return
		}

		attempts = append(attempts, "rejected")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","error":{"code":-32001,"message":"Federation auth failed"}}`))
	}))
	defer server.Close()

	peer := model.FederationPeer{
		KBURL:      server.URL,
		KID:        "kid-1",
		Secret:     []byte("stale-secret"),
		PrevSecret: []byte("previous-secret"),
	}

	result, err := federation.NewClient(peer, &fasthttp.Client{}, true).
		Search(context.Background(), model.MCPSearchParams{Query: "x"})

	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Equal(t, []string{"rejected", "previous"}, attempts)
}

// Without a previous key there is nothing to fall back to, and a second request
// would be a blind retry of a call the peer has already answered.
func TestClientDoesNotRetryWithoutAPreviousKey(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","error":{"code":-32001,"message":"Federation auth failed"}}`))
	}))
	defer server.Close()

	peer := model.FederationPeer{KBURL: server.URL, KID: "kid-1", Secret: []byte("stale-secret")}

	_, err := federation.NewClient(peer, &fasthttp.Client{}, true).
		Search(context.Background(), model.MCPSearchParams{Query: "x"})

	require.Error(t, err)
	require.Equal(t, 1, calls)
}

// A JSON-RPC error is an answer — the peer heard the call and named a reason —
// where a transport failure proves nothing reached it. Callers that record
// opposite outcomes for the two (key rotation) need the code and message typed.
func TestClientTypesAJSONRPCErrorAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","error":{"code":-32601,"message":"Method not found: rotate_secret"}}`))
	}))
	defer server.Close()

	peer := model.FederationPeer{KBURL: server.URL, KID: "kid-1", Secret: []byte("secret")}

	_, err := federation.NewClient(peer, &fasthttp.Client{}, true).
		RotateSecret(context.Background(), model.MCPRotateSecretParams{SecretHex: "00"})

	var rpcErr *model.FederationRPCError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, -32601, rpcErr.Code)
	require.Contains(t, rpcErr.Message, "rotate_secret")
}

// The signature covers the body, so the arguments a peer verifies are the ones
// that were sent — the property a call carrying a key depends on.
func TestClientSignsTheBody(t *testing.T) {
	verified := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		require.NoError(t, readErr) //nolint:testifylint // require safe: handler called synchronously via httptest

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		verified = verifyHS256(token, []byte("secret"), body) &&
			!verifyHS256(token, []byte("secret"), append(body, ' '))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"content":[]}}`))
	}))
	defer server.Close()

	peer := model.FederationPeer{KBURL: server.URL, KID: "kid-1", Secret: []byte("secret")}

	_, err := federation.NewClient(peer, &fasthttp.Client{}, true).
		Search(context.Background(), model.MCPSearchParams{Query: "x"})

	require.NoError(t, err)
	require.True(t, verified, "the signature does not cover the body it was sent with")
}

// verifyHS256 checks a token the way a peer does, including the body digest, so
// a test asserts what the other side would accept rather than what this side
// happened to write.
func verifyHS256(token string, secret, body []byte) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return false
	}

	rawClaims, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Bh string `json:"bh"`
	}
	err = json.Unmarshal(rawClaims, &claims)
	if err != nil {
		return false
	}

	digest := sha256.Sum256(body)
	return claims.Bh == base64.RawURLEncoding.EncodeToString(digest[:])
}
