package fleet

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/require"
)

// scopedGQL builds a graphql.Client backed by a gqlDoerFunc that returns the
// given data JSON string wrapped in a {"data": ...} envelope.
func scopedGQL(respond func(body string) string) graphql.Client {
	doer := gqlDoerFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(req.Body)
		data := respond(string(raw))
		env := `{"data":` + data + `}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(env)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})
	return graphql.NewClient("http://fake.local/_system/graphql", doer)
}

func TestRemoteKB_ReadOverlayHitNoClientCall(t *testing.T) {
	var calls int
	doer := gqlDoerFunc(func(_ *http.Request) (*http.Response, error) {
		calls++
		return nil, nil
	})
	gql := graphql.NewClient("http://fake.local/_system/graphql", doer)
	kb := newRemoteKB(gql, map[string]string{"boards/sprint.md": "board body"})
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
	gql := scopedGQL(func(_ string) string {
		return `{"notePaths":[{"content":"# Sprint\n@status:todo task A"}]}`
	})
	kb := newRemoteKB(gql, nil)
	got, err := kb.Read(context.Background(), "boards/other.md")
	require.NoError(t, err)
	// The returned string must be raw markdown, not rendered HTML.
	require.Equal(t, "# Sprint\n@status:todo task A", got)
	require.Contains(t, got, "@status:todo", "find-string must be present in raw markdown body")
	require.NotContains(t, got, "<h1>", "Read must not return rendered HTML")
}

func TestRemoteKB_ReadNotFoundReturnsError(t *testing.T) {
	// notePaths returns an empty list when the path is out of scope or missing.
	gql := scopedGQL(func(_ string) string {
		return `{"notePaths":[]}`
	})
	kb := newRemoteKB(gql, nil)
	_, err := kb.Read(context.Background(), "private/secret.md")
	require.Error(t, err)
	require.Contains(t, err.Error(), "note not found")
}

func TestRemoteKB_PatchIssuesUpdateNotesPatchVariant(t *testing.T) {
	gql := scopedGQL(func(_ string) string {
		return `{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["boards/sprint.md"]}}`
	})
	kb := newRemoteKB(gql, nil)
	require.NoError(t, kb.Patch(context.Background(), "boards/sprint.md", "@status:todo", "@status:doing"))
}

func TestRemoteKB_WriteIssuesUpsert(t *testing.T) {
	gql := scopedGQL(func(_ string) string {
		return `{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["x.md"]}}`
	})
	kb := newRemoteKB(gql, nil)
	require.NoError(t, kb.Write(context.Background(), "x.md", "body"))
}

func TestRemoteKB_PatchNotFoundReturnsError(t *testing.T) {
	gql := scopedGQL(func(_ string) string {
		return `{"updateNotes":{"__typename":"UpdateNotesPatchNotFoundPayload","path":"boards/sprint.md","find":"@status:todo"}}`
	})
	kb := newRemoteKB(gql, nil)
	err := kb.Patch(context.Background(), "boards/sprint.md", "@status:todo", "@status:doing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "patch find not found")
}

func TestRemoteKB_HashMismatchReturnsError(t *testing.T) {
	gql := scopedGQL(func(_ string) string {
		return `{"updateNotes":{"__typename":"UpdateNotesHashMismatchPayload","path":"boards/sprint.md","actualHash":"abc123"}}`
	})
	kb := newRemoteKB(gql, nil)
	err := kb.Write(context.Background(), "boards/sprint.md", "body")
	require.Error(t, err)
	require.Contains(t, err.Error(), "hash mismatch")
}

func TestRemoteKB_UpdateNotesErrorPayloadReturnsError(t *testing.T) {
	gql := scopedGQL(func(_ string) string {
		return `{"updateNotes":{"__typename":"ErrorPayload","message":"permission denied"}}`
	})
	kb := newRemoteKB(gql, nil)
	err := kb.Write(context.Background(), "out/scope.md", "body")
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}

// TestRemoteKB_WriteUpdatesOverlay is a regression for G6: Write must update the
// in-memory overlay so that a subsequent Read in the same run is served from cache
// without a remote round-trip, and returns the new content.
func TestRemoteKB_WriteUpdatesOverlay(t *testing.T) {
	var calls int
	doer := gqlDoerFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		body := `{"data":{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["notes/task.md"]}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})
	gql := graphql.NewClient("http://fake.local/_system/graphql", doer)
	kb := newRemoteKB(gql, nil)
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
	var calls int
	doer := gqlDoerFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		body := `{"data":{"updateNotes":{"__typename":"UpdateNotesSuccessPayload","paths":["boards/sprint.md"]}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})
	gql := graphql.NewClient("http://fake.local/_system/graphql", doer)
	initial := "# Sprint\n@status:todo task A\n@status:todo task B"
	kb := newRemoteKB(gql, map[string]string{"boards/sprint.md": initial})
	require.NoError(t, kb.Patch(context.Background(), "boards/sprint.md", "@status:todo", "@status:doing"))
	got, err := kb.Read(context.Background(), "boards/sprint.md")
	require.NoError(t, err)
	// Only the FIRST occurrence must be replaced (unique patch per G5 contract).
	require.Equal(t, "# Sprint\n@status:doing task A\n@status:todo task B", got)
	require.Equal(t, 1, calls, "only the Patch mutation must hit the client; Read must be overlay-served")
}
