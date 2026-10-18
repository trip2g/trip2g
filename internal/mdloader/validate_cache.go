package mdloader

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"

	"trip2g/internal/model"
)

// ValidateCache keeps validator results between loads, so a sync re-runs
// jsonnet only for notes whose result can have changed.
//
// A result depends on the note's content, the patched frontmatter and the
// validators' content. The key hashes the note, the patch-set hash and every
// validator's path and content: any patch change misses every entry, and an
// edited validator misses only the notes that list it. Each load keeps only
// the notes it validated, so memory follows the current vault.
//
// ValidateCache is safe for concurrent use. A nil cache disables caching.
type ValidateCache struct {
	mu      sync.Mutex
	entries map[string]validateEntry
}

type validateEntry struct {
	key      [sha256.Size]byte
	warnings []model.NoteWarning
}

// NewValidateCache returns an empty cache.
func NewValidateCache() *ValidateCache {
	return &ValidateCache{entries: map[string]validateEntry{}}
}

func (c *ValidateCache) get(path string, key [sha256.Size]byte) ([]model.NoteWarning, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[path]
	if !ok || e.key != key {
		return nil, false
	}
	return e.warnings, true
}

func (c *ValidateCache) put(path string, key [sha256.Size]byte, warnings []model.NoteWarning) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[path] = validateEntry{key: key, warnings: append([]model.NoteWarning(nil), warnings...)}
}

func validateKey(note *model.NoteView, validators []*model.NoteView, patchSetHash string) [sha256.Size]byte {
	h := sha256.New()
	write := func(b []byte) {
		_ = binary.Write(h, binary.LittleEndian, uint64(len(b)))
		h.Write(b)
	}

	write([]byte(note.Path))
	write(note.Content)
	write([]byte(patchSetHash))
	for _, v := range validators {
		write([]byte(v.Path))
		write(v.Content)
	}

	var key [sha256.Size]byte
	h.Sum(key[:0])
	return key
}

// retain drops entries for notes the last load didn't validate: deleted,
// renamed, or no longer listing validators.
func (c *ValidateCache) retain(paths map[string]bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	for path := range c.entries {
		if !paths[path] {
			delete(c.entries, path)
		}
	}
}
