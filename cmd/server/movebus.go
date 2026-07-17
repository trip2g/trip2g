package main

import (
	"time"

	"trip2g/internal/movebus"
)

func (a *app) SubscribeReaderMoves() *movebus.Subscriber {
	return a.moveBus.Subscribe(64)
}

func (a *app) UnsubscribeReaderMoves(sub *movebus.Subscriber) {
	a.moveBus.Unsubscribe(sub)
}

// ReaderMovesActive reports whether anyone watches the live movement stream.
// The render path checks it before doing any per-move work.
func (a *app) ReaderMovesActive() bool {
	return a.moveBus.HasSubscribers()
}

// PublishReaderMove emits an ephemeral reader movement event. The raw viewer
// id stays in-process: subscribers only ever see the anonymous hourly-rotated
// session key.
func (a *app) PublishReaderMove(fromPathID *int64, toPathID int64, viewerID string) {
	if !a.moveBus.HasSubscribers() {
		return
	}

	now := time.Now()
	a.moveBus.Publish(movebus.Move{
		FromPathID: fromPathID,
		ToPathID:   toPathID,
		At:         now,
		SessionKey: a.moveBus.SessionKey(viewerID, now),
	})
}
