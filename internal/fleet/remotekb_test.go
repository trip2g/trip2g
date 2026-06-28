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
	// note(path:) returns NoteView content via a content field; assert it routes.
	client := &ClientMock{GraphQLScopedFunc: func(_ context.Context, _ string, q string, _ map[string]any) (json.RawMessage, error) {
		require.Contains(t, q, "note(")
		return json.RawMessage(`{"note":{"content":"fetched body"}}`), nil
	}}
	kb := newRemoteKB(client, "tok", nil)
	got, err := kb.Read(context.Background(), "boards/other.md")
	require.NoError(t, err)
	require.Equal(t, "fetched body", got)
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
	require.True(t, strings.Contains(q, "updateNotes"))
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
