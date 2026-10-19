package runnotequery

import (
	"sync"

	"trip2g/internal/case/canreadnote"
)

// MaxCachedNotes bounds how many access classes one generation keeps.
const MaxCachedNotes = 16

// NotesCache keeps the `notes` JSON a query is given, built once per access
// class for the newest note generation it has seen.
type NotesCache struct {
	mu         sync.Mutex
	generation uint64
	entries    map[notesKey][]byte
}

type notesKey struct {
	class canreadnote.Class
	all   bool
}

func NewNotesCache() *NotesCache {
	return &NotesCache{entries: map[notesKey][]byte{}}
}

// CachedQueryNotes returns the notes JSON for the class, calling build on a
// miss. The result is shared: callers must not modify it.
func (c *NotesCache) CachedQueryNotes(generation uint64, class canreadnote.Class, all bool, build func() ([]byte, error)) ([]byte, error) {
	key := notesKey{class: class, all: all}

	c.mu.Lock()
	out, ok := c.entries[key]
	hit := ok && c.generation == generation
	c.mu.Unlock()
	if hit {
		return out, nil
	}

	out, err := build()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if generation > c.generation {
		clear(c.entries)
		c.generation = generation
	}
	if generation != c.generation {
		return out, nil
	}
	if len(c.entries) >= MaxCachedNotes {
		clear(c.entries)
	}
	c.entries[key] = out

	return out, nil
}
