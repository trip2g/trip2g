package mdloader

import (
	"errors"
	"testing"
	"time"

	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
)

func TestTweetCacheAcrossRendersAndLoads(t *testing.T) {
	calls := 0
	cache := NewTweetCache()
	cache.fetch = func(_, _ string) (string, error) {
		calls++
		return "<p>tweet</p>", nil
	}

	load := func() {
		notes, err := Load(Options{
			Log:        &logger.TestLogger{},
			TweetCache: cache,
			Sources: []SourceFile{{Path: "a.md", Content: []byte(`---
free_paragraphs: 1
---
![](https://twitter.com/user/status/123)`)}},
		})
		require.NoError(t, err)

		a := notes.PathMap["a.md"]
		require.Contains(t, string(a.HTML), "tweet")
		a.PartialRenderer.Introduce()
		a.PartialRenderer.Introduce()
	}

	load()
	load()
	require.Equal(t, 1, calls)
}

func TestTweetCacheRetriesFailures(t *testing.T) {
	now := time.Unix(0, 0)
	calls := 0
	cache := NewTweetCache()
	cache.now = func() time.Time { return now }
	cache.fetch = func(_, _ string) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("timeout")
		}
		return "<p>tweet</p>", nil
	}

	_, err := cache.Tweet("u", "light")
	require.Error(t, err)

	_, err = cache.Tweet("u", "light")
	require.Error(t, err)
	require.Equal(t, 1, calls, "failure is remembered for a while")

	now = now.Add(tweetRetryAfter)
	html, err := cache.Tweet("u", "light")
	require.NoError(t, err)
	require.Equal(t, "<p>tweet</p>", html)
	require.Equal(t, 2, calls)

	_, err = cache.Tweet("u", "dark")
	require.NoError(t, err)
	require.Equal(t, 3, calls, "theme is part of the key")
}

func TestTweetCacheRetriesEmptyHTML(t *testing.T) {
	calls := 0
	cache := NewTweetCache()
	cache.fetch = func(_, _ string) (string, error) {
		calls++
		return "", nil
	}

	_, err := cache.Tweet("u", "light")
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

func TestTweetNotFetchedAfterLoad(t *testing.T) {
	calls := 0
	cache := NewTweetCache()
	cache.fetch = func(_, _ string) (string, error) {
		calls++
		return "", errors.New("timeout")
	}
	now := time.Unix(0, 0)
	cache.now = func() time.Time { return now }

	notes, err := Load(Options{
		Log:        &logger.TestLogger{},
		TweetCache: cache,
		Sources:    []SourceFile{{Path: "a.md", Content: []byte("![](https://twitter.com/user/status/123)")}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, calls)

	// Request-time partials hold the shared render lock; they must not wait on X.
	now = now.Add(tweetRetryAfter)
	notes.PathMap["a.md"].PartialRenderer.Introduce()
	require.Equal(t, 1, calls)
}
