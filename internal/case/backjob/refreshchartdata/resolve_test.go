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
}

func (m *mockEnv) Logger() logger.Logger { return &logger.TestLogger{} }
func (m *mockEnv) SaveChartData(_ context.Context, versionID int64, hash, dataJSON string) error {
	m.versionID, m.hash, m.data, m.called = versionID, hash, dataJSON, true
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
}

func TestResolve_NonJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	env := &mockEnv{}
	err := Resolve(context.Background(), env, Params{URL: srv.URL})
	require.Error(t, err)
	require.False(t, env.called, "must not cache a non-JSON response")
}

func TestResolve_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	env := &mockEnv{}
	err := Resolve(context.Background(), env, Params{URL: srv.URL})
	require.Error(t, err)
	require.False(t, env.called)
}
