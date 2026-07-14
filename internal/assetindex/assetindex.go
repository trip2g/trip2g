// Package assetindex maintains an in-memory index from an exact asset
// identity (sha256 hash + served file name) to the notes that reference it
// and whether it is publicly reachable. The content-addressed asset route
// (/_system/assets/{sha256}/{fileName}) uses it to decide between anonymous
// serving (public asset) and per-user access checks.
//
// Ownership is keyed by (hash, fileName), not by hash alone: two distinct
// note_assets rows can share a hash (identical bytes) while carrying
// different file names, and only one of them may be publicly referenced. Two
// rows that share BOTH hash and fileName are byte-for-byte and
// content-type-for-content-type indistinguishable to a client, so merging
// their owning notes for that key is safe.
//
// The index is generation-checked on every lookup: it compares
// env.AssetIndexGeneration() against the generation it was built from and
// rebuilds synchronously on a mismatch. This closes the TOCTOU window an
// explicit "invalidate after reload" call would leave open — a reload
// publishes its new snapshot and bumps the generation atomically (see
// noteloader.Loader), so there is no instant where a fresh snapshot is live
// but a stale index is still trusted.
package assetindex

import (
	"sync"

	"trip2g/internal/model"
)

type Env interface {
	// LiveNoteViews is the currently served note snapshot — publicness must be
	// decided against what is actually published, never against drafts.
	LiveNoteViews() *model.NoteViews
	// Layouts are the site chrome; assets referenced by layouts render on
	// public pages and are therefore public.
	Layouts() *model.Layouts
	// AssetIndexGeneration changes whenever LiveNoteViews/Layouts may have
	// changed (i.e. after every note reload). Used to detect a stale cache.
	AssetIndexGeneration() uint64
}

// Key identifies one servable asset identity.
type Key struct {
	Hash     string
	FileName string
}

// Ownership describes who may read an asset.
type Ownership struct {
	// Public is true when the asset is reachable anonymously: referenced by a
	// layout or by at least one publicly readable note.
	Public bool
	// Notes are the live notes referencing this exact (hash, fileName); a
	// signed-in user may read the asset if they can read at least one of them.
	Notes []*model.NoteView
}

type AssetIndex struct {
	env Env

	mu      sync.Mutex
	byKey   map[Key]*Ownership
	builtOn uint64 // generation the current byKey was built from; 0 = never built
}

func New(env Env) *AssetIndex {
	return &AssetIndex{env: env}
}

// AssetOwnership returns the ownership entry for an exact (hash, fileName)
// asset identity. ok=false means it is not referenced by any live note or
// layout — callers must fail closed (admin-only access). Safe on a nil
// receiver (fails closed).
func (x *AssetIndex) AssetOwnership(sha256Hash, fileName string) (Ownership, bool) {
	if x == nil {
		return Ownership{}, false
	}

	x.mu.Lock()
	defer x.mu.Unlock()

	if gen := x.env.AssetIndexGeneration(); x.byKey == nil || gen != x.builtOn {
		x.byKey = build(x.env)
		x.builtOn = gen
	}

	entry, ok := x.byKey[Key{Hash: sha256Hash, FileName: fileName}]
	if !ok {
		return Ownership{}, false
	}
	return *entry, true
}

func build(env Env) map[Key]*Ownership {
	byKey := make(map[Key]*Ownership)

	entry := func(k Key) *Ownership {
		e, ok := byKey[k]
		if !ok {
			e = &Ownership{}
			byKey[k] = e
		}
		return e
	}

	if nvs := env.LiveNoteViews(); nvs != nil {
		for _, nv := range nvs.List {
			for _, ar := range nv.AssetReplaces {
				if ar == nil || ar.Hash == "" || ar.FileName == "" {
					continue
				}
				k := Key{Hash: ar.Hash, FileName: ar.FileName}
				e := entry(k)
				e.Notes = append(e.Notes, nv)
				if nv.IsPubliclyReadable() {
					e.Public = true
				}
			}
		}
	}

	if layouts := env.Layouts(); layouts != nil {
		for _, ar := range layouts.AssetReplaces {
			if ar == nil || ar.Hash == "" || ar.FileName == "" {
				continue
			}
			entry(Key{Hash: ar.Hash, FileName: ar.FileName}).Public = true
		}
	}

	return byKey
}
