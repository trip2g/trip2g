package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"trip2g/internal/agentruntime"
)

// remoteKB is the fleet's agentruntime.KB over the trip2g API, scoped to a
// single delivery's shortapitoken. attach_notes materialized in the payload
// seed the overlay; reads of attached paths cost no round-trip.
type remoteKB struct {
	client  Client
	token   string
	overlay map[string]string
}

// newRemoteKB builds a remoteKB. overlay may be nil.
func newRemoteKB(client Client, token string, overlay map[string]string) *remoteKB {
	if overlay == nil {
		overlay = map[string]string{}
	}
	return &remoteKB{client: client, token: token, overlay: overlay}
}

var _ agentruntime.KB = (*remoteKB)(nil)

const searchScopedQuery = `query Search($q: String!) {
  search(input: {query: $q}) { nodes { document { ... on PublicNote { path } } } }
}`

// noteContentScopedQuery fetches raw markdown via notePaths, not the note()
// query which returns rendered HTML. Using notePaths ensures the returned
// string is the same raw markdown that Write/Patch operate on, so a
// find-string derived from Read round-trips correctly through Patch.
// notePaths is scope-enforced by filterNotePathsByScope (F1 fix), so
// read_patterns are respected — an out-of-scope path returns an empty list.
const noteContentScopedQuery = `query NoteContent($filter: NotePathsFilter!) {
  notePaths(filter: $filter) { content }
}`

const updateNotesMutation = `mutation Update($input: UpdateNotesInput!) {
  updateNotes(input: $input) {
    __typename
    ... on UpdateNotesSuccessPayload { paths }
    ... on UpdateNotesPatchNotFoundPayload { path find }
    ... on UpdateNotesHashMismatchPayload { path actualHash }
    ... on ErrorPayload { message }
  }
}`

func (k *remoteKB) Search(ctx context.Context, query string) ([]agentruntime.Doc, error) {
	raw, err := k.client.GraphQLScoped(ctx, k.token, searchScopedQuery, map[string]any{"q": query})
	if err != nil {
		return nil, err
	}
	var data struct {
		Search struct {
			Nodes []struct {
				Document struct {
					Path string `json:"path"`
				} `json:"document"`
			} `json:"nodes"`
		} `json:"search"`
	}
	if uerr := json.Unmarshal(raw, &data); uerr != nil {
		return nil, uerr
	}
	out := make([]agentruntime.Doc, 0, len(data.Search.Nodes))
	for _, n := range data.Search.Nodes {
		if n.Document.Path != "" {
			out = append(out, agentruntime.Doc{Path: n.Document.Path})
		}
	}
	return out, nil
}

func (k *remoteKB) Read(ctx context.Context, path string) (string, error) {
	if body, ok := k.overlay[path]; ok {
		return body, nil
	}
	vars := map[string]any{
		"filter": map[string]any{"paths": []string{path}},
	}
	raw, err := k.client.GraphQLScoped(ctx, k.token, noteContentScopedQuery, vars)
	if err != nil {
		return "", err
	}
	var data struct {
		NotePaths []struct {
			Content string `json:"content"`
		} `json:"notePaths"`
	}
	if uerr := json.Unmarshal(raw, &data); uerr != nil {
		return "", uerr
	}
	if len(data.NotePaths) == 0 {
		return "", fmt.Errorf("note not found: %s", path)
	}
	return data.NotePaths[0].Content, nil
}

func (k *remoteKB) Write(ctx context.Context, path, content string) error {
	if err := k.update(ctx, []map[string]any{
		{"upsert": map[string]any{"path": path, "content": content}},
	}); err != nil {
		return err
	}
	k.overlay[path] = content
	return nil
}

func (k *remoteKB) Patch(ctx context.Context, path, find, replace string) error {
	if err := k.update(ctx, []map[string]any{
		{"patch": map[string]any{"path": path, "find": find, "replace": replace}},
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

// replaceOnce replaces the first occurrence of find in s with replace.
func replaceOnce(s, find, replace string) string {
	idx := strings.Index(s, find)
	if idx == -1 {
		return s
	}
	return s[:idx] + replace + s[idx+len(find):]
}

// updateNotesResult mirrors the updateNotes union. All fields are optional so
// that any discriminant variant populates exactly the fields it selects.
type updateNotesResult struct {
	Typename   string   `json:"__typename"`
	Paths      []string `json:"paths"`      // UpdateNotesSuccessPayload
	Path       string   `json:"path"`       // UpdateNotesPatchNotFoundPayload / UpdateNotesHashMismatchPayload
	Find       string   `json:"find"`       // UpdateNotesPatchNotFoundPayload
	ActualHash string   `json:"actualHash"` // UpdateNotesHashMismatchPayload
	Message    string   `json:"message"`    // ErrorPayload
}

func (k *remoteKB) update(ctx context.Context, changes []map[string]any) error {
	raw, err := k.client.GraphQLScoped(ctx, k.token, updateNotesMutation,
		map[string]any{"input": map[string]any{"changes": changes}})
	if err != nil {
		return err
	}
	var data struct {
		UpdateNotes updateNotesResult `json:"updateNotes"`
	}
	if uerr := json.Unmarshal(raw, &data); uerr != nil {
		return uerr
	}
	r := data.UpdateNotes
	switch r.Typename {
	case "UpdateNotesSuccessPayload", "":
		// "" keeps backward-compat with mocks that omit __typename.
		return nil
	case "UpdateNotesPatchNotFoundPayload":
		return fmt.Errorf("patch find not found in %s: %q", r.Path, r.Find)
	case "UpdateNotesHashMismatchPayload":
		return fmt.Errorf("hash mismatch on %s (actual: %s)", r.Path, r.ActualHash)
	case "ErrorPayload":
		return fmt.Errorf("%s", r.Message)
	default:
		return fmt.Errorf("updateNotes: unexpected type %q", r.Typename)
	}
}
