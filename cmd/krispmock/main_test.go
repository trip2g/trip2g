package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHealth verifies GET /health returns 200 and body "ok".
func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handleHealth(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "ok", w.Body.String())
}

// TestMeetingsList verifies POST /v2/meetings/list returns the expected synthetic rows.
func TestMeetingsList(t *testing.T) {
	body := strings.NewReader(`{"sort":"desc","sortKey":"created_at","page":1,"limit":200}`)
	req := httptest.NewRequest(http.MethodPost, "/v2/meetings/list", body)
	w := httptest.NewRecorder()
	handleMeetingsList(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "response must have data object")

	rows, ok := data["rows"].([]any)
	require.True(t, ok, "data must have rows array")
	require.Len(t, rows, 3, "expected exactly 3 synthetic meetings")

	// Verify required fields are present on first row.
	row, ok := rows[0].(map[string]any)
	require.True(t, ok, "row must be a JSON object")

	for _, field := range []string{"id", "name", "started_at", "duration", "speakers", "is_demo"} {
		_, exists := row[field]
		require.True(t, exists, "missing required field %q in row", field)
	}

	// Speakers must have first_name for the Python ingest's speaker_names().
	speakers, ok := row["speakers"].([]any)
	require.True(t, ok, "speakers must be an array")
	require.NotEmpty(t, speakers, "speakers must not be empty")

	spk, ok := speakers[0].(map[string]any)
	require.True(t, ok, "speaker must be a JSON object")
	_, hasFirstName := spk["first_name"]
	require.True(t, hasFirstName, "speaker must have first_name")

	// is_demo must be false so the ingest does not skip synthetic meetings.
	require.Equal(t, false, row["is_demo"])
}

// TestMeetingsListPage2ReturnsEmpty verifies that page > 1 returns no rows.
func TestMeetingsListPage2ReturnsEmpty(t *testing.T) {
	body := strings.NewReader(`{"page":2,"limit":200}`)
	req := httptest.NewRequest(http.MethodPost, "/v2/meetings/list", body)
	w := httptest.NewRecorder()
	handleMeetingsList(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)

	rows, ok := data["rows"].([]any)
	require.True(t, ok)
	require.Empty(t, rows, "page 2 must return no rows")

	// count still reflects total; JSON numbers decode as float64.
	count, ok := data["count"].(float64)
	require.True(t, ok)
	require.Equal(t, 3, int(count))
}

// TestBlockTree verifies GET /v2/block/{id}/tree returns a walkable transcript.
func TestBlockTree(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v2/block/"+meetingID1+"/tree", nil)
	req.SetPathValue("id", meetingID1)
	w := httptest.NewRecorder()
	handleBlockTree(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var tree map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tree))

	rows := extractRows(tree)
	require.NotEmpty(t, rows, "expected at least one transcript row from block tree")

	for i, r := range rows {
		require.NotEmpty(t, r.text, "transcript row %d has empty text", i)
		require.Positive(t, r.spk, "transcript row %d has non-positive speakerIndex", i)
	}
}

// TestBlockTreeAllMeetings verifies all three synthetic meeting IDs produce walkable transcripts.
func TestBlockTreeAllMeetings(t *testing.T) {
	ids := []string{meetingID1, meetingID2, meetingID3}
	for _, id := range ids {
		req := httptest.NewRequest(http.MethodGet, "/v2/block/"+id+"/tree", nil)
		req.SetPathValue("id", id)
		w := httptest.NewRecorder()
		handleBlockTree(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var tree map[string]any
		require.NoError(t, json.NewDecoder(w.Body).Decode(&tree))

		rows := extractRows(tree)
		require.NotEmpty(t, rows, "meeting %s produced no transcript rows", id)
	}
}

// transcriptRow mirrors the output of extract_rows in extract_krisp_transcripts.py.
type transcriptRow struct {
	start float64
	spk   int
	text  string
}

// extractRows is a Go port of extract_rows from extract_krisp_transcripts.py.
// It walks the block tree and collects all speakerIndex+speech pairs.
func extractRows(x any) []transcriptRow {
	var rows []transcriptRow
	walkForRows(x, &rows)
	return rows
}

func walkForRows(x any, rows *[]transcriptRow) {
	sl, isList := x.([]any)
	if isList {
		for _, item := range sl {
			walkForRows(item, rows)
		}
		return
	}
	m, isMap := x.(map[string]any)
	if !isMap {
		return
	}
	collectUtterance(m, rows)
	for _, val := range m {
		walkForRows(val, rows)
	}
}

func collectUtterance(m map[string]any, rows *[]transcriptRow) {
	speakerIndex, hasSI := m["speakerIndex"]
	speech, hasSP := m["speech"]
	if !hasSI || !hasSP {
		return
	}
	spMap, isMap := speech.(map[string]any)
	if !isMap {
		return
	}
	text, _ := spMap["text"].(string)
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	var start float64
	if s, startOK := spMap["start"].(float64); startOK {
		start = s
	}
	var spk int
	if idx, idxOK := speakerIndex.(float64); idxOK {
		spk = int(idx)
	}
	*rows = append(*rows, transcriptRow{start: start, spk: spk, text: text})
}
