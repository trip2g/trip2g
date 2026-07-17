package movebus_test

import (
	"testing"
	"time"

	"trip2g/internal/logger"
	"trip2g/internal/movebus"

	"github.com/stretchr/testify/require"
)

func newBus() *movebus.Bus {
	return movebus.New(&logger.DummyLogger{})
}

func TestSubscribePublishReceive(t *testing.T) {
	b := newBus()
	sub := b.Subscribe(8)
	defer b.Unsubscribe(sub)

	from := int64(1)
	move := movebus.Move{FromPathID: &from, ToPathID: 2, At: time.Now(), SessionKey: "abc"}
	b.Publish(move)

	select {
	case got, ok := <-sub.Ch:
		require.True(t, ok)
		require.Equal(t, move, got)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for move")
	}
}

// Publish with no subscribers must be a cheap no-op that never blocks.
func TestPublishNoSubscribers(t *testing.T) {
	b := newBus()
	require.False(t, b.HasSubscribers())

	done := make(chan struct{})
	go func() {
		b.Publish(movebus.Move{ToPathID: 1})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked with no subscribers")
	}
	require.Equal(t, int64(0), b.Stats().Dropped)
}

// A full subscriber buffer drops the move instead of blocking the publisher.
func TestPublishDropsOnFullBuffer(t *testing.T) {
	b := newBus()
	sub := b.Subscribe(1)
	defer b.Unsubscribe(sub)

	done := make(chan struct{})
	go func() {
		b.Publish(movebus.Move{ToPathID: 1})
		b.Publish(movebus.Move{ToPathID: 2}) // buffer full — must drop, not block
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on a full subscriber buffer")
	}
	require.Equal(t, int64(1), b.Stats().Dropped)
}

func TestHasSubscribers(t *testing.T) {
	b := newBus()
	require.False(t, b.HasSubscribers())

	sub := b.Subscribe(1)
	require.True(t, b.HasSubscribers())

	b.Unsubscribe(sub)
	require.False(t, b.HasSubscribers())
}

func TestUnsubscribedNotDelivered(t *testing.T) {
	b := newBus()
	sub := b.Subscribe(1)
	b.Unsubscribe(sub)

	b.Publish(movebus.Move{ToPathID: 1})

	select {
	case <-sub.Ch:
		t.Fatal("received move after unsubscribe")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSessionKeyStableWithinHourAndAnonymous(t *testing.T) {
	b := newBus()
	now := time.Now()

	k1 := b.SessionKey("42", now)
	k2 := b.SessionKey("42", now.Add(time.Second))
	k3 := b.SessionKey("43", now)

	require.Equal(t, k1, k2, "same viewer within the hour keeps the same token")
	require.NotEqual(t, k1, k3, "different viewers get different tokens")
	require.NotContains(t, k1, "42", "token must not contain the raw viewer id")
	require.Len(t, k1, 16, "8-byte hex token")

	// A different hour rotates the key, so the token changes.
	k4 := b.SessionKey("42", now.Add(2*time.Hour))
	require.NotEqual(t, k1, k4, "hourly rotation breaks cross-hour linkage")
}
