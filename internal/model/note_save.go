package model

// NoteSaveResult is the outcome of writing one note. A write is not necessarily
// a change: pushing content identical to what is already stored creates no
// version, and the only thing that can still make such a write matter is that
// the path was hidden and the write brought it back.
type NoteSaveResult struct {
	PathID int64
	// VersionID is the note_versions row this write inserted, or 0 when the
	// content matched what was already stored.
	VersionID int64
	// Unhidden reports that the path was hidden before this write. Writing is the
	// only way back — there is no unhide mutation — so it happens on every push,
	// content change or not.
	Unhidden bool
}

// Versioned reports whether the write created a new version. Only then did the
// note's content change, so only then may change events be raised.
func (r NoteSaveResult) Versioned() bool {
	return r.VersionID != 0
}

// AffectsSnapshot reports whether the served note set must be reloaded: either
// the content changed, or a hidden note became visible again. An unhidden note
// is absent from the in-memory views, so without a reload the site keeps
// serving it as missing even though the row says otherwise.
func (r NoteSaveResult) AffectsSnapshot() bool {
	return r.Versioned() || r.Unhidden
}
