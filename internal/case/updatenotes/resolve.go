package updatenotes

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	pathpkg "path"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/webhookutil"
)

type Env interface {
	LatestNoteViews() *appmodel.NoteViews
	// InsertNote writes the note and reports what the write did: its path id, the
	// version it inserted (0 when content was unchanged) and whether it unhid the
	// path. The own version id lets us report each save's OWN version (race-free)
	// rather than re-deriving it from the post-write reload, which under concurrent
	// same-board editing can be a peer's newer version.
	InsertNote(ctx context.Context, note appmodel.RawNote) (appmodel.NoteSaveResult, error)
	HideNotePath(ctx context.Context, params db.HideNotePathParams) error
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
	HandleLatestNotesAfterSave(ctx context.Context, pathIDs []int64) error
}

func hashContent(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// normalizeNotePath mirrors agentruntime's scope-path normalization: clients
// (especially small LLMs) sometimes prepend "/" or "./" to a path, but
// write_patterns and nvs.PathMap keys are both slash-less. Strips leading "/"
// and "./", then path.Clean-resolves "."/".." segments. Returns "" when the
// path resolves outside the vault root — callers treat that as invalid.
func normalizeNotePath(p string) string {
	p = strings.TrimSpace(p)
	for strings.HasPrefix(p, "./") || strings.HasPrefix(p, "/") {
		p = strings.TrimPrefix(strings.TrimPrefix(p, "./"), "/")
	}
	if p == "" {
		return ""
	}
	c := pathpkg.Clean(p)
	if c == "." || c == ".." || strings.HasPrefix(c, "../") {
		return ""
	}
	return c
}

func webhookWriteDenied(ctx context.Context, path string) *model.ErrorPayload {
	if webhookutil.WriteScopeDenied(ctx, path) {
		return &model.ErrorPayload{Message: "write denied for path: " + path}
	}
	return nil
}

// normalizeAndAuthorize normalizes a change path and checks it against the
// request's write scope. Returns the canonical path, or an ErrorPayload.
func normalizeAndAuthorize(ctx context.Context, raw string) (string, *model.ErrorPayload) {
	norm := normalizeNotePath(raw)
	if norm == "" {
		return "", &model.ErrorPayload{Message: "invalid path: " + raw}
	}
	if denied := webhookWriteDenied(ctx, norm); denied != nil {
		return "", denied
	}
	return norm, nil
}

//nolint:gocognit // per-change branches share state with the outer loop; extraction would require threading paths/pathIDs and a multi-return payload sentinel through helpers.
func Resolve(ctx context.Context, env Env, input model.UpdateNotesInput) (model.UpdateNotesOrErrorPayload, error) {
	nvs := env.LatestNoteViews()
	var paths []string
	var pathIDs []int64
	// versionIDs is aligned with pathIDs: each entry is the write's OWN inserted
	// version id (0 when content was unchanged).
	var versionIDs []int64
	// changedPathIDs is pathIDs minus the writes that stored nothing new: only
	// those may raise change events. reload is wider — an unhidden path has no new
	// version but must still reach the served snapshot.
	var changedPathIDs []int64
	reload := false
	hid := false

	// nvs.PathMap is keyed by note.Path (filesystem path, e.g. "todo.md").
	// NoteViews.GetByPath uses the Permalink map (URL path), so PathMap is the correct lookup here.
	//
	// Note: if the same path appears more than once in changes, the second
	// operation reads stale content — callers must not patch the same path twice per call.
	for _, change := range input.Changes {
		switch {
		case change.Upsert != nil:
			upsert := change.Upsert
			saved, payload, err := applyUpsert(ctx, env, nvs, upsert)
			if err != nil {
				return nil, err
			}
			if payload != nil {
				return payload, nil
			}
			pathIDs = append(pathIDs, saved.PathID)
			versionIDs = append(versionIDs, saved.VersionID)
			if saved.Versioned() {
				changedPathIDs = append(changedPathIDs, saved.PathID)
			}
			if saved.AffectsSnapshot() {
				reload = true
			}
			paths = append(paths, upsert.Path)
		case change.Patch != nil:
			patch := change.Patch
			saved, payload, err := applyPatch(ctx, env, nvs, patch)
			if err != nil {
				return nil, err
			}
			if payload != nil {
				return payload, nil
			}
			pathIDs = append(pathIDs, saved.PathID)
			versionIDs = append(versionIDs, saved.VersionID)
			if saved.Versioned() {
				changedPathIDs = append(changedPathIDs, saved.PathID)
			}
			if saved.AffectsSnapshot() {
				reload = true
			}
			paths = append(paths, patch.Path)
		case change.Hide != nil:
			// Hide is a metadata operation. Unlike the standalone hideNotes mutation,
			// this does not trigger webhooks — extend Env if webhook support is needed.
			hide := change.Hide
			norm, errp := normalizeAndAuthorize(ctx, hide.Path)
			if errp != nil {
				return errp, nil
			}
			hide.Path = norm
			err := env.HideNotePath(ctx, db.HideNotePathParams{
				HiddenBy: &input.ApiKey.CreatedBy,
				Value:    hide.Path,
			})
			if err != nil {
				return nil, fmt.Errorf("updatenotes: hide %s: %w", hide.Path, err)
			}
			paths = append(paths, hide.Path)
			hid = true
		default:
			continue
		}
	}

	// A hide changes what the site serves just like a content write does, so it
	// reloads too — rendernotepage reads the in-memory cache.
	if hid {
		reload = true
	}

	// The reload covers everything that changed what is served: new versions,
	// unhidden paths, hides. The post-save handler is narrower — it raises change
	// events, and a write that stored nothing new is not a change. Without that
	// split an idempotent agent write re-triggers the webhooks that caused it, at
	// full LLM cost, on every cycle.
	views := nvs
	if reload {
		reloaded, prepErr := env.PrepareLatestNotes(ctx, false)
		if prepErr != nil {
			return nil, fmt.Errorf("updatenotes: prepare latest notes: %w", prepErr)
		}
		views = reloaded
	}
	if len(changedPathIDs) > 0 {
		if handleErr := env.HandleLatestNotesAfterSave(ctx, changedPathIDs); handleErr != nil {
			return nil, fmt.Errorf("updatenotes: handle latest notes after save: %w", handleErr)
		}
	}

	// Surface each saved note's own version id (mirrors pushNotes' updated[].id) so a
	// client can advance its self-echo baseline to its OWN save's version. Every
	// written path is reported, including the ones that stored nothing new.
	var updated []model.NoteWriteResult
	if len(pathIDs) > 0 {
		updated = collectUpdated(views, pathIDs, versionIDs)
	}

	return model.UpdateNotesSuccessPayload{Paths: paths, Updated: updated}, nil
}

// applyUpsert normalizes and authorizes the path, enforces the optimistic
// concurrency check, and writes. A non-nil payload is a caller-visible refusal
// (unauthorized path, hash mismatch) that ends the whole batch.
//
// ExpectedHash gates the upsert with optimistic concurrency. actualHash defaults
// to "" for an absent note, so expectedHash == "" is the create-only sentinel: it
// asserts "expect this note to be absent" — absent → actualHash "" matches →
// create; an existing note always hashes non-empty → HashMismatch (never
// overwritten). A non-empty expectedHash is the usual optimistic update (matches
// only the exact current content).
func applyUpsert(
	ctx context.Context,
	env Env,
	nvs *appmodel.NoteViews,
	upsert *model.NoteChangeUpsertInput,
) (appmodel.NoteSaveResult, model.UpdateNotesOrErrorPayload, error) {
	norm, errp := normalizeAndAuthorize(ctx, upsert.Path)
	if errp != nil {
		return appmodel.NoteSaveResult{}, errp, nil
	}
	upsert.Path = norm

	if upsert.ExpectedHash != nil {
		nv := nvs.PathMap[upsert.Path]
		var actualHash string
		if nv != nil {
			actualHash = hashContent(nv.Content)
		}
		if actualHash != *upsert.ExpectedHash {
			return appmodel.NoteSaveResult{}, model.UpdateNotesHashMismatchPayload{
				Path:       upsert.Path,
				ActualHash: actualHash,
			}, nil
		}
	}

	saved, err := env.InsertNote(ctx, appmodel.RawNote{Path: upsert.Path, Content: upsert.Content})
	if err != nil {
		return appmodel.NoteSaveResult{}, nil, fmt.Errorf("updatenotes: insert upsert %s: %w", upsert.Path, err)
	}
	return saved, nil, nil
}

// applyPatch resolves the single occurrence of Find in the stored content and
// writes the result. A non-nil payload is a caller-visible refusal (unauthorized
// path, missing note, hash mismatch, absent or ambiguous Find) that ends the
// whole batch.
func applyPatch(
	ctx context.Context,
	env Env,
	nvs *appmodel.NoteViews,
	patch *model.NoteChangePatchInput,
) (appmodel.NoteSaveResult, model.UpdateNotesOrErrorPayload, error) {
	norm, errp := normalizeAndAuthorize(ctx, patch.Path)
	if errp != nil {
		return appmodel.NoteSaveResult{}, errp, nil
	}
	patch.Path = norm

	nv := nvs.PathMap[patch.Path]
	if nv == nil {
		return appmodel.NoteSaveResult{}, &model.ErrorPayload{Message: fmt.Sprintf("note not found: %s", patch.Path)}, nil
	}
	if patch.ExpectedHash != nil {
		actualHash := hashContent(nv.Content)
		if actualHash != *patch.ExpectedHash {
			return appmodel.NoteSaveResult{}, model.UpdateNotesHashMismatchPayload{
				Path:       patch.Path,
				ActualHash: actualHash,
			}, nil
		}
	}

	content := string(nv.Content)
	idx := strings.Index(content, patch.Find)
	if idx == -1 {
		return appmodel.NoteSaveResult{}, model.UpdateNotesPatchNotFoundPayload{Path: patch.Path, Find: patch.Find}, nil
	}
	// A second occurrence makes the edit ambiguous, so it is refused rather than
	// applied to the first one.
	if strings.Contains(content[idx+len(patch.Find):], patch.Find) {
		return appmodel.NoteSaveResult{}, model.UpdateNotesPatchNotFoundPayload{Path: patch.Path, Find: patch.Find}, nil
	}

	newContent := content[:idx] + patch.Replace + content[idx+len(patch.Find):]
	saved, err := env.InsertNote(ctx, appmodel.RawNote{Path: patch.Path, Content: newContent})
	if err != nil {
		return appmodel.NoteSaveResult{}, nil, fmt.Errorf("updatenotes: insert patch %s: %w", patch.Path, err)
	}
	return saved, nil, nil
}

// collectUpdated reports each saved note with the version id the WRITE itself
// inserted (versionIDs[i], aligned with pathIDs[i]). Reporting the write's own
// version — rather than re-deriving it from the reloaded NoteViews — is race-free:
// under concurrent same-board editing the reload's latest version can be a peer's,
// which would briefly suppress a genuine peer event on the saving client. The
// reload is still consulted for the note's Path and to skip notes no longer present
// (e.g. hidden). When the write created no new version (versionID 0, content
// unchanged) the reload's current version is used as a fallback.
func collectUpdated(nvs *appmodel.NoteViews, pathIDs, versionIDs []int64) []model.NoteWriteResult {
	updated := make([]model.NoteWriteResult, 0, len(pathIDs))
	for i, id := range pathIDs {
		nv := nvs.GetByPathID(id)
		if nv == nil {
			continue
		}
		versionID := versionIDs[i]
		if versionID == 0 {
			versionID = nv.VersionID
		}
		updated = append(updated, model.NoteWriteResult{Path: nv.Path, VersionID: versionID})
	}
	return updated
}
