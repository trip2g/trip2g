package fleet

import (
	"context"
	"encoding/json"
	"fmt"

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

const noteContentScopedQuery = `query NoteContent($path: String!) {
  note(input: {path: $path, referer: ""}) { content: html }
}`

const updateNotesMutation = `mutation Update($input: UpdateNotesInput!) {
  updateNotes(input: $input) {
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
	raw, err := k.client.GraphQLScoped(ctx, k.token, noteContentScopedQuery, map[string]any{"path": path})
	if err != nil {
		return "", err
	}
	var data struct {
		Note *struct {
			Content string `json:"content"`
		} `json:"note"`
	}
	if uerr := json.Unmarshal(raw, &data); uerr != nil {
		return "", uerr
	}
	if data.Note == nil {
		return "", fmt.Errorf("note not found: %s", path)
	}
	return data.Note.Content, nil
}

func (k *remoteKB) Write(ctx context.Context, path, content string) error {
	return k.update(ctx, []map[string]any{
		{"upsert": map[string]any{"path": path, "content": content}},
	})
}

func (k *remoteKB) Patch(ctx context.Context, path, find, replace string) error {
	return k.update(ctx, []map[string]any{
		{"patch": map[string]any{"path": path, "find": find, "replace": replace}},
	})
}

func (k *remoteKB) update(ctx context.Context, changes []map[string]any) error {
	_, err := k.client.GraphQLScoped(ctx, k.token, updateNotesMutation,
		map[string]any{"input": map[string]any{"changes": changes}})
	return err
}
