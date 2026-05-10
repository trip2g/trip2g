package renderpreview

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

type Config struct {
	BufferSize int
}

func DefaultConfig() Config {
	return Config{
		BufferSize: 10,
	}
}

type PreviewEntry struct {
	ID       string
	HTML     string
	Warnings []string
	Version  int
}

type PreviewBuffer struct {
	mu      sync.Mutex
	entries []PreviewEntry
	head    int
	count   int
	version int
	notify  chan struct{}
}

func NewPreviewBuffer(config Config) *PreviewBuffer {
	size := config.BufferSize
	if size <= 0 {
		size = DefaultConfig().BufferSize
	}
	return &PreviewBuffer{
		entries: make([]PreviewEntry, size),
		notify:  make(chan struct{}),
	}
}

func (b *PreviewBuffer) Push(html string, warnings []string) PreviewEntry {
	id := randomID()
	b.mu.Lock()
	b.version++
	entry := PreviewEntry{ID: id, HTML: html, Warnings: warnings, Version: b.version}
	b.entries[b.head] = entry
	b.head = (b.head + 1) % len(b.entries)
	if b.count < len(b.entries) {
		b.count++
	}
	old := b.notify
	b.notify = make(chan struct{})
	b.mu.Unlock()
	close(old)
	return entry
}

func (b *PreviewBuffer) Latest() (PreviewEntry, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.count == 0 {
		return PreviewEntry{}, false
	}
	idx := (b.head - 1 + len(b.entries)) % len(b.entries)
	return b.entries[idx], true
}

func (b *PreviewBuffer) GetByID(id string) (PreviewEntry, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.count {
		e := b.entries[i]
		if e.ID == id {
			return e, true
		}
	}
	return PreviewEntry{}, false
}

// Poll returns the current version and a channel that is closed when a new entry is pushed.
func (b *PreviewBuffer) Poll() (int, <-chan struct{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.version, b.notify
}

func randomID() string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
