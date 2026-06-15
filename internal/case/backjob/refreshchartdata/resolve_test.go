package refreshchartdata

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"trip2g/internal/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEnv struct {
	versionID int64
	hash      string
	data      string
	called    bool

	errMsg    string
	errCalled bool
}

func (m *mockEnv) Logger() logger.Logger { return &logger.TestLogger{} }
func (m *mockEnv) SaveChartData(_ context.Context, versionID int64, hash, dataJSON string) error {
	m.versionID, m.hash, m.data, m.called = versionID, hash, dataJSON, true
	return nil
}
func (m *mockEnv) SaveChartDataError(_ context.Context, _ int64, _ string, errMsg string) error {
	m.errMsg, m.errCalled = errMsg, true
	return nil
}

func TestResolve_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`[{"day":"Mon","n":5}]`))
	}))
	defer srv.Close()

	env := &mockEnv{}
	err := Resolve(context.Background(), env, Params{VersionID: 42, Hash: "h", URL: srv.URL})
	require.NoError(t, err)
	require.True(t, env.called)
	require.Equal(t, int64(42), env.versionID)
	require.Equal(t, "h", env.hash)
	require.JSONEq(t, `[{"day":"Mon","n":5}]`, env.data)
	require.False(t, env.errCalled, "success must not record an error")
}

func TestResolve_POST_WithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		b, _ := io.ReadAll(r.Body)
		assert.JSONEq(t, `{"sql":"SELECT 1"}`, string(b))
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	env := &mockEnv{}
	err := Resolve(context.Background(), env, Params{URL: srv.URL, Body: `{"sql":"SELECT 1"}`})
	require.NoError(t, err)
	require.True(t, env.called)
	require.False(t, env.errCalled, "success must not record an error")
}

// Fetch problems are expected for external sources: the job must complete
// (nil error) without caching, so it is not retried and never poisons the
// queue. The TTL refresh tries again later.

func TestResolve_NonJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	env := &mockEnv{}
	err := Resolve(context.Background(), env, Params{VersionID: 1, Hash: "h", URL: srv.URL})
	require.NoError(t, err, "non-JSON response is not a job failure")
	require.False(t, env.called, "must not cache a non-JSON response")
	require.True(t, env.errCalled, "must record the error")
	require.Equal(t, "non-JSON response", env.errMsg)
}

func TestResolve_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	env := &mockEnv{}
	err := Resolve(context.Background(), env, Params{VersionID: 2, Hash: "h2", URL: srv.URL})
	require.NoError(t, err, "upstream HTTP error is not a job failure")
	require.False(t, env.called)
	require.True(t, env.errCalled, "must record the error")
	require.NotEmpty(t, env.errMsg)
}

func TestResolve_UnreachableHost(t *testing.T) {
	env := &mockEnv{}
	err := Resolve(context.Background(), env, Params{VersionID: 3, Hash: "h3", URL: "http://127.0.0.1:1/v1/query", Body: `{"sql":"SELECT 1"}`})
	require.NoError(t, err, "unreachable source is not a job failure")
	require.False(t, env.called)
	require.True(t, env.errCalled, "must record the error")
	require.NotEmpty(t, env.errMsg)
}
