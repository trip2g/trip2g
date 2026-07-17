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
	// InsertNoteWithVersion writes the note and returns its path id and the id of
	// the note_versions row it inserted (0 when content was unchanged). The own
	// version id lets us report each save's OWN version (race-free) rather than
	// re-deriving it from the post-write reload, which under concurrent same-board
	// editing can be a peer's newer version.
	InsertNoteWithVersion(ctx context.Context, note appmodel.RawNote) (int64, int64, error)
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
			norm, errp := normalizeAndAuthorize(ctx, upsert.Path)
			if errp != nil {
				return errp, nil
			}
			upsert.Path = norm
			// ExpectedHash gates the upsert with optimistic concurrency.
			// actualHash defaults to "" for an absent note (the nv != nil block is skipped),
			// so expectedHash == "" is the create-only sentinel: it asserts "expect this note
			// to be absent" — absent → actualHash "" matches → create; an existing note always
			// hashes non-empty → HashMismatch (never overwritten). A non-empty expectedHash is
			// the usual optimistic update (matches only the exact current content).
			if upsert.ExpectedHash != nil {
				nv := nvs.PathMap[upsert.Path]
				var actualHash string
				if nv != nil {
					actualHash = hashContent(nv.Content)
				}
				if actualHash != *upsert.ExpectedHash {
					return model.UpdateNotesHashMismatchPayload{
						Path:       upsert.Path,
						ActualHash: actualHash,
					}, nil
				}
			}
			pathID, versionID, err := env.InsertNoteWithVersion(ctx, appmodel.RawNote{Path: upsert.Path, Content: upsert.Content})
			if err != nil {
				return nil, fmt.Errorf("updatenotes: insert upsert %s: %w", upsert.Path, err)
			}
			pathIDs = append(pathIDs, pathID)
			versionIDs = append(versionIDs, versionID)
			paths = append(paths, upsert.Path)
		case change.Patch != nil:
			patch := change.Patch
			norm, errp := normalizeAndAuthorize(ctx, patch.Path)
			if errp != nil {
				return errp, nil
			}
			patch.Path = norm
			nv := nvs.PathMap[patch.Path]
			if nv == nil {
				return &model.ErrorPayload{Message: fmt.Sprintf("note not found: %s", patch.Path)}, nil
			}
			if patch.ExpectedHash != nil {
				actualHash := hashContent(nv.Content)
				if actualHash != *patch.ExpectedHash {
					return model.UpdateNotesHashMismatchPayload{
						Path:       patch.Path,
						ActualHash: actualHash,
					}, nil
				}
			}
			content := string(nv.Content)
			idx := strings.Index(content, patch.Find)
			if idx == -1 {
				return model.UpdateNotesPatchNotFoundPayload{Path: patch.Path, Find: patch.Find}, nil
			}
			// Check for multiple occurrences
			if strings.Contains(content[idx+len(patch.Find):], patch.Find) {
				return model.UpdateNotesPatchNotFoundPayload{Path: patch.Path, Find: patch.Find}, nil
			}
			newContent := content[:idx] + patch.Replace + content[idx+len(patch.Find):]
			pathID, versionID, err := env.InsertNoteWithVersion(ctx, appmodel.RawNote{Path: patch.Path, Content: newContent})
			if err != nil {
				return nil, fmt.Errorf("updatenotes: insert patch %s: %w", patch.Path, err)
			}
			pathIDs = append(pathIDs, pathID)
			versionIDs = append(versionIDs, versionID)
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

	// Content changes reload NoteViews and run after-save handling.
	// Hide-only batches still reload NoteViews so the hidden paths stop
	// resolving on the public site (rendernotepage reads the in-memory cache),
	// but skip HandleLatestNotesAfterSave since no content was saved.
	var updated []model.NoteWriteResult
	if len(pathIDs) > 0 {
		reloaded, prepErr := env.PrepareLatestNotes(ctx, false)
		if prepErr != nil {
			return nil, fmt.Errorf("updatenotes: prepare latest notes: %w", prepErr)
		}
		if handleErr := env.HandleLatestNotesAfterSave(ctx, pathIDs); handleErr != nil {
			return nil, fmt.Errorf("updatenotes: handle latest notes after save: %w", handleErr)
		}
		// Surface each saved note's new version id (mirrors pushNotes' updated[].id) so a
		// client can advance its self-echo baseline to its OWN save's version.
		updated = collectUpdated(reloaded, pathIDs, versionIDs)
	} else if hid {
		if _, err := env.PrepareLatestNotes(ctx, false); err != nil {
			return nil, fmt.Errorf("updatenotes: prepare latest notes after hide: %w", err)
		}
	}

	return model.UpdateNotesSuccessPayload{Paths: paths, Updated: updated}, nil
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
