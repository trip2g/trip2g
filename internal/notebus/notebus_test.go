package notebus_test

import (
	"testing"
	"time"

	"trip2g/internal/logger"
	"trip2g/internal/notebus"

	"github.com/stretchr/testify/require"
)

func newBus() *notebus.Bus {
	return notebus.New(&logger.DummyLogger{})
}

// 1. Subscribe → Publish → receive batch.
func TestSubscribePublishReceive(t *testing.T) {
	b := newBus()
	sub := b.Subscribe([]string{"**"}, nil, 8)
	defer b.Unsubscribe(sub)

	batch := notebus.Batch{Changes: []notebus.Change{
		{PathID: 1, Path: "blog/post.md", Event: "create"},
	}}
	b.Publish(batch)

	select {
	case got, ok := <-sub.Ch:
		require.True(t, ok)
		require.Equal(t, batch, got)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for batch")
	}
}

// 2. includePatterns filter — non-matching path not received.
func TestIncludePatternsFilter(t *testing.T) {
	b := newBus()
	sub := b.Subscribe([]string{"blog/**"}, nil, 8)
	defer b.Unsubscribe(sub)

	b.Publish(notebus.Batch{Changes: []notebus.Change{
		{PathID: 2, Path: "other/note.md", Event: "update"},
	}})

	select {
	case <-sub.Ch:
		t.Fatal("received batch that should have been filtered")
	case <-time.After(100 * time.Millisecond):
		// expected: nothing received
	}
}

// 3. excludePatterns filter.
func TestExcludePatternsFilter(t *testing.T) {
	b := newBus()
	sub := b.Subscribe([]string{"**"}, []string{"drafts/**"}, 8)
	defer b.Unsubscribe(sub)

	b.Publish(notebus.Batch{Changes: []notebus.Change{
		{PathID: 3, Path: "drafts/x.md", Event: "create"},
	}})

	select {
	case <-sub.Ch:
		t.Fatal("received batch that should have been excluded")
	case <-time.After(100 * time.Millisecond):
		// expected: nothing received
	}
}

// 4. Unsubscribe → channel closed.
func TestUnsubscribeChannelClosed(t *testing.T) {
	b := newBus()
	sub := b.Subscribe([]string{"**"}, nil, 8)
	b.Unsubscribe(sub)

	select {
	case _, ok := <-sub.Ch:
		require.False(t, ok, "channel should be closed after Unsubscribe")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout: channel not closed after Unsubscribe")
	}
}

// 5. Fan-out: 3 subscribers all receive.
func TestFanOut(t *testing.T) {
	b := newBus()
	subs := make([]*notebus.Subscriber, 3)
	for i := range subs {
		subs[i] = b.Subscribe([]string{"**"}, nil, 8)
		defer b.Unsubscribe(subs[i])
	}

	batch := notebus.Batch{Changes: []notebus.Change{
		{PathID: 10, Path: "shared/doc.md", Event: "update"},
	}}
	b.Publish(batch)

	for i, sub := range subs {
		select {
		case got, ok := <-sub.Ch:
			require.True(t, ok)
			require.Equal(t, batch, got, "subscriber %d", i)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout on subscriber %d", i)
		}
	}
}

// 6. Buffer full → event dropped, stats updated.
func TestBufferFullDropped(t *testing.T) {
	b := newBus()
	sub := b.Subscribe([]string{"**"}, nil, 1)
	defer b.Unsubscribe(sub)

	// Fill buffer
	b.Publish(notebus.Batch{Changes: []notebus.Change{
		{PathID: 20, Path: "a.md", Event: "create"},
	}})
	// This publish should be dropped (buffer full, not yet consumed)
	b.Publish(notebus.Batch{Changes: []notebus.Change{
		{PathID: 21, Path: "b.md", Event: "create"},
	}})

	stats := b.Stats()
	require.Positive(t, stats.Dropped, "expected at least one dropped event")
}

// 7. Partial batch filter — only matching changes forwarded.
func TestPartialBatchFilter(t *testing.T) {
	b := newBus()
	sub := b.Subscribe([]string{"blog/**"}, nil, 8)
	defer b.Unsubscribe(sub)

	b.Publish(notebus.Batch{Changes: []notebus.Change{
		{PathID: 30, Path: "blog/a.md", Event: "create"},
		{PathID: 31, Path: "other/b.md", Event: "update"},
	}})

	select {
	case got, ok := <-sub.Ch:
		require.True(t, ok)
		require.Len(t, got.Changes, 1)
		require.Equal(t, int64(30), got.Changes[0].PathID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for partial batch")
	}
}
