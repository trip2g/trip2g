package fleet

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"

	"trip2g/cmd/fleet/internal/agentruntime"
	"trip2g/cmd/fleet/internal/fleet/trip2ggql"
	"trip2g/internal/frontmatteredit"
)

// JSON map keys used in the hybrid UpdateNotesScoped variables.
const (
	jsonKeyPath         = "path"
	jsonKeyContent      = "content"
	jsonKeyFind         = "find"
	jsonKeyReplace      = "replace"
	jsonKeyExpectedHash = "expectedHash"
	jsonKeyChanges      = "changes"
	jsonKeySet          = "set"
	jsonKeyUnset        = "unset"
)

// remoteKB is the fleet's agentruntime.KB over the trip2g API, scoped to a
// single delivery's shortapitoken. attach_notes materialized in the payload
// seed the overlay; reads of attached paths cost no round-trip.
//
// The Bearer token is baked into gql (via scopedDoer) rather than passed
// per-call, so every request to trip2g carries the delivery's scoped token.
type remoteKB struct {
	gql     graphql.Client
	overlay map[string]string
}

// newRemoteKB builds a remoteKB. gql must be scoped to the delivery's Bearer
// token (via NewScopedGraphQLClient). overlay may be nil.
func newRemoteKB(gql graphql.Client, overlay map[string]string) *remoteKB {
	if overlay == nil {
		overlay = map[string]string{}
	}
	return &remoteKB{gql: gql, overlay: overlay}
}

var _ agentruntime.KB = (*remoteKB)(nil)

func (k *remoteKB) Search(ctx context.Context, query string) ([]agentruntime.Doc, error) {
	resp, err := trip2ggql.SearchScoped(ctx, k.gql, query)
	if err != nil {
		return nil, err
	}
	out := make([]agentruntime.Doc, 0, len(resp.Search.Nodes))
	for _, n := range resp.Search.Nodes {
		pn, ok := n.Document.(*trip2ggql.SearchScopedSearchSearchConnectionNodesSearchResultDocumentPublicNote)
		if ok && pn.NotePath != "" {
			out = append(out, agentruntime.Doc{Path: pn.NotePath})
		}
	}
	return out, nil
}

func (k *remoteKB) Read(ctx context.Context, path string) (string, error) {
	if body, ok := k.overlay[path]; ok {
		return body, nil
	}
	filter := trip2ggql.NotePathsFilter{Paths: []string{path}}
	resp, err := trip2ggql.NoteContentScoped(ctx, k.gql, filter)
	if err != nil {
		return "", err
	}
	if len(resp.NotePaths) == 0 {
		return "", fmt.Errorf("note not found: %s", path)
	}
	return resp.NotePaths[0].Content, nil
}

func (k *remoteKB) Write(ctx context.Context, path, content string) error {
	if err := k.update(ctx, []map[string]any{
		{"upsert": map[string]any{jsonKeyPath: path, jsonKeyContent: content}},
	}); err != nil {
		return err
	}
	k.overlay[path] = content
	return nil
}

func (k *remoteKB) Patch(ctx context.Context, path, find, replace string) error {
	if err := k.update(ctx, []map[string]any{
		{"patch": map[string]any{jsonKeyPath: path, jsonKeyFind: find, jsonKeyReplace: replace}},
	}); err != nil {
		return err
	}
	// Best-effort overlay sync: keep in-memory view consistent so a subsequent
	// Read in the same run is served from cache without a remote round-trip.
	if cur, ok := k.overlay[path]; ok {
		k.overlay[path] = replaceOnce(cur, find, replace)
	}
	return nil
}

// PatchIfUnchanged applies a patch only if the note still hashes to
// expectedHash, letting trip2g compare against the live content inside the same
// atomic mutation (internal/case/updatenotes applies the check and the edit
// together). That closes the window between the caller verifying a note and the
// edit landing — including the case where Read was served from the overlay,
// which is seeded once per delivery and never refreshed: stale bytes now produce
// a loud hash mismatch instead of an unverified write.
func (k *remoteKB) PatchIfUnchanged(ctx context.Context, path, find, replace, expectedHash string) error {
	if err := k.update(ctx, []map[string]any{
		{"patch": map[string]any{
			jsonKeyPath:         path,
			jsonKeyFind:         find,
			jsonKeyReplace:      replace,
			jsonKeyExpectedHash: expectedHash,
		}},
	}); err != nil {
		return err
	}
	if cur, ok := k.overlay[path]; ok {
		k.overlay[path] = replaceOnce(cur, find, replace)
	}
	return nil
}

// SetMeta sends a meta change: trip2g edits the frontmatter keys against the live
// content, so nothing is read here.
func (k *remoteKB) SetMeta(ctx context.Context, path string, set map[string]any, unset []string) error {
	return k.setMeta(ctx, path, set, unset, nil)
}

// SetMetaIfUnchanged is PatchIfUnchanged's twin: the edit lands only if the note
// still hashes to expectedHash.
func (k *remoteKB) SetMetaIfUnchanged(ctx context.Context, path string, set map[string]any, unset []string, expectedHash string) error {
	return k.setMeta(ctx, path, set, unset, &expectedHash)
}

// setMeta keeps the overlay in step with the edit trip2g made. An overlay entry
// the edit cannot be replayed on is dropped, so the next Read goes to trip2g.
func (k *remoteKB) setMeta(ctx context.Context, path string, set map[string]any, unset []string, expectedHash *string) error {
	meta := map[string]any{jsonKeyPath: path, jsonKeySet: set, jsonKeyUnset: unset}
	if expectedHash != nil {
		meta[jsonKeyExpectedHash] = *expectedHash
	}
	err := k.update(ctx, []map[string]any{{"meta": meta}})
	if err != nil {
		return err
	}
	cur, ok := k.overlay[path]
	if !ok {
		return nil
	}
	edited, err := frontmatteredit.Apply(cur, set, unset)
	if err != nil {
		delete(k.overlay, path)
		return nil //nolint:nilerr // trip2g applied the edit; only the cached copy is stale.
	}
	k.overlay[path] = edited
	return nil
}

// replaceOnce replaces the first occurrence of find in s with replace.
func replaceOnce(s, find, replace string) string {
	idx := strings.Index(s, find)
	if idx == -1 {
		return s
	}
	return s[:idx] + replace + s[idx+len(find):]
}

// update sends an UpdateNotesScoped mutation. The variables are built as
// map[string]any so that each NoteChangeInput only includes the active variant
// (upsert xor patch xor meta xor hide), matching the server's nil-based dispatch.
// The response type is the genqlient-generated union so __typename dispatch
// and field access are type-safe even though the input is free-form.
func (k *remoteKB) update(ctx context.Context, changes []map[string]any) error {
	req := &graphql.Request{
		OpName: "UpdateNotesScoped",
		Query:  trip2ggql.UpdateNotesScoped_Operation,
		Variables: map[string]any{
			"input": map[string]any{jsonKeyChanges: changes},
		},
	}
	var data trip2ggql.UpdateNotesScopedResponse
	resp := &graphql.Response{Data: &data}
	if err := k.gql.MakeRequest(ctx, req, resp); err != nil {
		return err
	}
	switch r := data.UpdateNotes.(type) {
	case *trip2ggql.UpdateNotesScopedUpdateNotesUpdateNotesSuccessPayload:
		return nil
	case *trip2ggql.UpdateNotesScopedUpdateNotesUpdateNotesPatchNotFoundPayload:
		return fmt.Errorf("patch find not found in %s: %q", r.Path, r.Find)
	case *trip2ggql.UpdateNotesScopedUpdateNotesUpdateNotesHashMismatchPayload:
		return fmt.Errorf("hash mismatch on %s (actual: %s)", r.Path, r.ActualHash)
	case *trip2ggql.UpdateNotesScopedUpdateNotesErrorPayload:
		return errors.New(r.Message)
	default:
		return fmt.Errorf("updateNotes: unexpected type %T", r)
	}
}
