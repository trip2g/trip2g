// Package assetindex maintains an in-memory index from asset sha256 hash to
// the notes that reference it and whether it is publicly reachable. The
// content-addressed asset route (/_system/assets/{sha256}/{fileName}) uses it
// to decide between anonymous serving (public asset) and per-user access
// checks. The index is rebuilt lazily and invalidated on every note reload.
package assetindex

import (
	"sync"

	"trip2g/internal/model"
	"trip2g/internal/rssfeed"
)

type Env interface {
	// LiveNoteViews is the currently served note snapshot — publicness must be
	// decided against what is actually published, never against drafts.
	LiveNoteViews() *model.NoteViews
	// Layouts are the site chrome; assets referenced by layouts render on
	// public pages and are therefore public.
	Layouts() *model.Layouts
}

// Ownership describes who may read an asset.
type Ownership struct {
	// Public is true when the asset is reachable anonymously: referenced by a
	// layout or by at least one publicly readable note.
	Public bool
	// Notes are the live notes referencing the asset; a signed-in user may
	// read the asset if they can read at least one of them.
	Notes []*model.NoteView
}

type AssetIndex struct {
	env Env

	mu     sync.Mutex
	byHash map[string]*Ownership
}

func New(env Env) *AssetIndex {
	return &AssetIndex{env: env}
}

// AssetOwnership returns the ownership entry for an asset hash. ok=false means
// the hash is not referenced by any live note or layout — callers must fail
// closed (admin-only access). Safe on a nil receiver (fails closed).
func (x *AssetIndex) AssetOwnership(sha256Hash string) (Ownership, bool) {
	if x == nil {
		return Ownership{}, false
	}

	x.mu.Lock()
	defer x.mu.Unlock()

	if x.byHash == nil {
		x.byHash = build(x.env)
	}

	entry, ok := x.byHash[sha256Hash]
	if !ok {
		return Ownership{}, false
	}
	return *entry, true
}

// InvalidateAssetIndex drops the index so the next lookup rebuilds it from the
// current note snapshot. Call after every note reload. Safe on a nil receiver
// (partially wired apps in tests).
func (x *AssetIndex) InvalidateAssetIndex() {
	if x == nil {
		return
	}

	x.mu.Lock()
	defer x.mu.Unlock()
	x.byHash = nil
}

func build(env Env) map[string]*Ownership {
	byHash := make(map[string]*Ownership)

	entry := func(hash string) *Ownership {
		e, ok := byHash[hash]
		if !ok {
			e = &Ownership{}
			byHash[hash] = e
		}
		return e
	}

	if nvs := env.LiveNoteViews(); nvs != nil {
		for _, nv := range nvs.List {
			for _, ar := range nv.AssetReplaces {
				if ar == nil || ar.Hash == "" {
					continue
				}
				e := entry(ar.Hash)
				e.Notes = append(e.Notes, nv)
				if rssfeed.IsPubliclyReadable(nv) {
					e.Public = true
				}
			}
		}
	}

	if layouts := env.Layouts(); layouts != nil {
		for _, ar := range layouts.AssetReplaces {
			if ar == nil || ar.Hash == "" {
				continue
			}
			entry(ar.Hash).Public = true
		}
	}

	return byHash
}
