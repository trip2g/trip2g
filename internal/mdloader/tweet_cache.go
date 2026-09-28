package mdloader

import (
	"errors"
	"sync"
	"time"

	"github.com/quailyquaily/goldmark-enclave/object"
)

// tweetRetryAfter is how long a failed oEmbed fetch is remembered before the
// next render tries again.
const tweetRetryAfter = 10 * time.Minute

// TweetCache memoizes Twitter oEmbed HTML by (url, theme). The fetch is a
// synchronous HTTP call made during rendering; without the cache it repeats
// for every render variant (HTML, free, domain, partials) and every reload.
type TweetCache struct {
	fetch func(url, theme string) (string, error)
	now   func() time.Time

	mu      sync.Mutex
	entries map[tweetKey]tweetEntry
}

type tweetKey struct {
	url   string
	theme string
}

type tweetEntry struct {
	html      string
	err       error
	fetchedAt time.Time
}

func NewTweetCache() *TweetCache {
	return &TweetCache{
		fetch:   object.GetTweetOembedHtml,
		now:     time.Now,
		entries: make(map[tweetKey]tweetEntry),
	}
}

// errTweetNotCached is returned by Cached for a tweet no load has fetched yet.
var errTweetNotCached = errors.New("tweet not fetched yet")

// Tweet returns the oEmbed HTML, fetching it unless a success, or a failure
// younger than tweetRetryAfter, is cached.
func (c *TweetCache) Tweet(url, theme string) (string, error) {
	key := tweetKey{url: url, theme: theme}

	c.mu.Lock()
	entry, ok := c.entries[key]
	c.mu.Unlock()

	if ok && (entry.err == nil || c.now().Sub(entry.fetchedAt) < tweetRetryAfter) {
		return entry.html, entry.err
	}

	html, err := c.fetch(url, theme)
	if err == nil && html == "" {
		// The upstream client ignores the HTTP status: a rate-limit or error
		// response decodes to empty HTML without an error.
		err = errors.New("empty oEmbed response")
	}

	c.mu.Lock()
	c.entries[key] = tweetEntry{html: html, err: err, fetchedAt: c.now()}
	c.mu.Unlock()

	return html, err
}

// Cached returns what the last fetch got without fetching again. Request-time
// partial renders use it: they hold the shared render lock and must not wait
// on the network.
func (c *TweetCache) Cached(url, theme string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[tweetKey{url: url, theme: theme}]
	if !ok {
		return "", errTweetNotCached
	}
	return entry.html, entry.err
}
