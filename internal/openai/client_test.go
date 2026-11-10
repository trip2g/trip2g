package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateEmbeddingsTimesOut(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	c := newWithTimeout("k", "m", srv.URL+"/v1", 50*time.Millisecond)

	done := make(chan error, 1)
	go func() {
		_, err := c.CreateEmbeddings(context.Background(), []string{"text"}, KindDebug)
		done <- err
	}()

	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("embedding request did not time out")
	}
}
