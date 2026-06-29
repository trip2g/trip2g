package fleet

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoteKB_ReadOverlayHitNoClientCall(t *testing.T) {
	var calls int
	client := &ClientMock{GraphQLScopedFunc: func(context.Context, string, string, map[string]any) (json.RawMessage, error) {
		calls++
		return nil, nil
	}}
	kb := newRemoteKB(client, "tok", map[string]string{"boards/sprint.md": "board body"})
	got, err := kb.Read(context.Background(), "boards/sprint.md")
	require.NoError(t, err)
	require.Equal(t, "board body", got)
	require.Zero(t, calls) // served from overlay
}

func TestRemoteKB_ReadFallsBackToScopedNote(t *testing.T) {
	// Read must use notePaths (raw markdown) not note{html} (rendered HTML).
	// Regression for F4: the old query aliased html as "content", so Read returned
	// rendered HTML — a Patch find-string derived from an ad-hoc Read would never
	// match the stored markdown.
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, q string, vars map[string]any) (json.RawMessage, error) {
		// Must use notePaths, not the HTML-returning note() query.
		require.Contains(t, q, "notePaths")
		require.NotContains(t, q, "html")
		// Variable must carry the path as a filter list.
		filter, ok := vars["filter"].(map[string]any)
		require.True(t, ok, "vars must contain a filter map")
		paths, ok := filter["paths"].([]string)
		require.True(t, ok, "filter must contain a paths []string")
		require.Equal(t, []string{"boards/other.md"}, paths)
		// Response mirrors notePaths: [{content: "..."}] with raw markdown.
		return json.RawMessage(`{"notePaths":[{"content":"# Sprint\n@status:todo task A"}]}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	got, err := kb.Read(context.Background(), "boards/other.md")
	require.NoError(t, err)
	// The returned string must be raw markdown, not rendered HTML.
	// A find-string present in markdown must survive a round-trip.
	require.Equal(t, "# Sprint\n@status:todo task A", got)
	require.Contains(t, got, "@status:todo", "find-string must be present in raw markdown body")
	require.NotContains(t, got, "<h1>", "Read must not return rendered HTML")
}

func TestRemoteKB_ReadNotFoundReturnsError(t *testing.T) {
	// notePaths returns an empty list when the path is out of scope or missing.
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, q string, _ map[string]any) (json.RawMessage, error) {
		require.Contains(t, q, "notePaths")
		return json.RawMessage(`{"notePaths":[]}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	_, err := kb.Read(context.Background(), "private/secret.md")
	require.Error(t, err)
	require.Contains(t, err.Error(), "note not found")
}

func TestRemoteKB_PatchIssuesUpdateNotesPatchVariant(t *testing.T) {
	var sentVars map[string]any
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, tok, q string, vars map[string]any) (json.RawMessage, error) {
		require.Equal(t, "tok", tok)
		require.Contains(t, q, "updateNotes")
		sentVars = vars
		return json.RawMessage(`{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["boards/sprint.md"]}}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	require.NoError(t, kb.Patch(context.Background(), "boards/sprint.md", "@status:todo", "@status:doing"))

	input := sentVars["input"].(map[string]any)
	changes := input["changes"].([]map[string]any)
	require.Len(t, changes, 1)
	patch := changes[0]["patch"].(map[string]any)
	require.Equal(t, "boards/sprint.md", patch["path"])
	require.Equal(t, "@status:todo", patch["find"])
	require.Equal(t, "@status:doing", patch["replace"])
	require.NotContains(t, changes[0], "upsert")
}

func TestRemoteKB_WriteIssuesUpsert(t *testing.T) {
	var q string
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, query string, vars map[string]any) (json.RawMessage, error) {
		q = query
		input := vars["input"].(map[string]any)
		changes := input["changes"].([]map[string]any)
		_, hasUpsert := changes[0]["upsert"]
		require.True(t, hasUpsert)
		return json.RawMessage(`{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["x.md"]}}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	require.NoError(t, kb.Write(context.Background(), "x.md", "body"))
	require.Contains(t, q, "updateNotes")
}

func TestRemoteKB_PatchNotFoundReturnsError(t *testing.T) {
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, _ string, _ map[string]any) (json.RawMessage, error) {
		return json.RawMessage(`{"updateNotes":{"__typename":"UpdateNotesPatchNotFoundPayload","path":"boards/sprint.md","find":"@status:todo"}}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	err := kb.Patch(context.Background(), "boards/sprint.md", "@status:todo", "@status:doing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "patch find not found")
}

func TestRemoteKB_HashMismatchReturnsError(t *testing.T) {
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, _ string, _ map[string]any) (json.RawMessage, error) {
		return json.RawMessage(`{"updateNotes":{"__typename":"UpdateNotesHashMismatchPayload","path":"boards/sprint.md","actualHash":"abc123"}}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	err := kb.Write(context.Background(), "boards/sprint.md", "body")
	require.Error(t, err)
	require.Contains(t, err.Error(), "hash mismatch")
}

func TestRemoteKB_UpdateNotesErrorPayloadReturnsError(t *testing.T) {
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, _ string, _ map[string]any) (json.RawMessage, error) {
		return json.RawMessage(`{"updateNotes":{"__typename":"ErrorPayload","message":"permission denied"}}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	err := kb.Write(context.Background(), "out/scope.md", "body")
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}

// TestRemoteKB_WriteUpdatesOverlay is a regression for G6: Write must update the
// in-memory overlay so that a subsequent Read in the same run is served from cache
// without a remote round-trip, and returns the new content.
func TestRemoteKB_WriteUpdatesOverlay(t *testing.T) {
	successResp := json.RawMessage(`{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["notes/task.md"]}}`)
	var calls int
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, q string, _ map[string]any) (json.RawMessage, error) {
		calls++
		if strings.Contains(q, "updateNotes") {
			return successResp, nil
		}
		// If Read falls through to the remote, the test must fail.
		t.Fatal("Read after Write must be served from overlay, got remote call for query:", q)
		return nil, nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	require.NoError(t, kb.Write(context.Background(), "notes/task.md", "new content"))
	got, err := kb.Read(context.Background(), "notes/task.md")
	require.NoError(t, err)
	require.Equal(t, "new content", got)
	require.Equal(t, 1, calls, "only the Write mutation must hit the client; Read must be overlay-served")
}

// TestRemoteKB_PatchUpdatesOverlay is a regression for G6: Patch must apply the
// find→replace to the in-memory overlay so that a subsequent Read returns the
// patched content without a remote round-trip.
func TestRemoteKB_PatchUpdatesOverlay(t *testing.T) {
	successResp := json.RawMessage(`{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["boards/sprint.md"]}}`)
	var calls int
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, q string, _ map[string]any) (json.RawMessage, error) {
		calls++
		if strings.Contains(q, "updateNotes") {
			return successResp, nil
		}
		t.Fatal("Read after Patch must be served from overlay, got remote call for query:", q)
		return nil, nil
	}}
	initial := "# Sprint\n@status:todo task A\n@status:todo task B"
	kb := newRemoteKB(client, "tok", map[string]string{"boards/sprint.md": initial})
	require.NoError(t, kb.Patch(context.Background(), "boards/sprint.md", "@status:todo", "@status:doing"))
	got, err := kb.Read(context.Background(), "boards/sprint.md")
	require.NoError(t, err)
	// Only the FIRST occurrence must be replaced (unique patch per G5 contract).
	require.Equal(t, "# Sprint\n@status:doing task A\n@status:todo task B", got)
	require.Equal(t, 1, calls, "only the Patch mutation must hit the client; Read must be overlay-served")
}
