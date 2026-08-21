package webhookutil

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseAgentResponse_Empty(t *testing.T) {
	resp, err := ParseAgentResponse(nil)
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestParseAgentResponse_InvalidJSON(t *testing.T) {
	resp, err := ParseAgentResponse([]byte("not json"))
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestParseAgentResponse_NoChanges(t *testing.T) {
	body := []byte(`{"status":"ok","message":"nothing to do"}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "ok", resp.Status)
	require.Empty(t, resp.Changes)
}

func TestParseAgentResponse_WithChanges(t *testing.T) {
	body := []byte(`{
		"status": "ok",
		"message": "fixed 1 file",
		"changes": [
			{"path": "blog/post.md", "content": "# Fixed"}
		]
	}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Changes, 1)
	require.Equal(t, "blog/post.md", resp.Changes[0].Path)
	require.Equal(t, "# Fixed", resp.Changes[0].Content)
}

func TestParseAgentResponse_MissingPath(t *testing.T) {
	body := []byte(`{"changes": [{"content": "no path"}]}`)
	_, err := ParseAgentResponse(body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid change")
}

func TestParseAgentResponse_MissingContent(t *testing.T) {
	body := []byte(`{"changes": [{"path": "test.md"}]}`)
	_, err := ParseAgentResponse(body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid change")
}

func TestParseAgentResponse_WithExpectedHash(t *testing.T) {
	hash := "abc123"
	body := []byte(`{
		"changes": [
			{"path": "test.md", "content": "# Test", "expected_hash": "abc123"}
		]
	}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, &hash, resp.Changes[0].ExpectedHash)
}

func TestParseAgentResponse_ParsesCosts(t *testing.T) {
	body := []byte(`{"status":"ok","costs":{"tokens":1234,"steps":5},"changes":[]}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, map[string]float64{"tokens": 1234, "steps": 5}, resp.Costs)
}

// The unit set is open, so a cost object is stored as given — including units
// trip2g has never heard of.
func TestMarshalCosts_KeepsUnknownUnits(t *testing.T) {
	raw := MarshalCosts(map[string]float64{"usd": 0.004})
	require.NotNil(t, raw)
	require.JSONEq(t, `{"usd":0.004}`, *raw)
}

// A careless agent must not poison later sums, and an empty report must leave the
// delivery's costs untouched rather than storing an empty object.
func TestMarshalCosts_DropsUnusableValues(t *testing.T) {
	require.Nil(t, MarshalCosts(nil))
	require.Nil(t, MarshalCosts(map[string]float64{}))
	require.Nil(t, MarshalCosts(map[string]float64{"tokens": math.NaN()}))
	require.Nil(t, MarshalCosts(map[string]float64{"": 1}))

	raw := MarshalCosts(map[string]float64{"tokens": 5, "broken": math.Inf(1)})
	require.NotNil(t, raw)
	require.JSONEq(t, `{"tokens":5}`, *raw)
}

func TestParseAgentResponse_PatchChangeNoContentOK(t *testing.T) {
	body := []byte(`{"changes":[{"path":"boards/sprint.md","find":"todo","replace":"doing","kind":"patch"}]}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Changes, 1)
	require.Equal(t, "patch", resp.Changes[0].Kind)
	require.Equal(t, "todo", resp.Changes[0].Find)
	require.Equal(t, "doing", resp.Changes[0].Replace)
}

func TestParseAgentResponse_PatchChangeMissingFind(t *testing.T) {
	body := []byte(`{"changes":[{"path":"boards/sprint.md","kind":"patch"}]}`)
	_, err := ParseAgentResponse(body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid change")
}

// G9 regression: IsPatch must exist and return true only for patch kind.
func TestAgentChangeIsPatch(t *testing.T) {
	cases := []struct {
		kind string
		want bool
	}{
		{AgentChangeKindPatch, true},
		{AgentChangeKindWrite, false},
		{"", false},       // backward-compat: empty string = write
		{"upsert", false}, // legacy alias also not patch
	}
	for _, tc := range cases {
		c := AgentChange{Path: "x.md", Kind: tc.kind}
		if tc.kind == AgentChangeKindPatch {
			c.Find = "x"
		} else {
			c.Content = "y"
		}
		require.Equalf(t, tc.want, c.IsPatch(), "IsPatch() for kind=%q", tc.kind)
	}
}

func TestMarshalLogs_Empty(t *testing.T) {
	require.Nil(t, MarshalLogs(nil))
	require.Nil(t, MarshalLogs([]AgentLog{}))
}

func TestMarshalLogs_KeepsEntries(t *testing.T) {
	out := MarshalLogs([]AgentLog{
		{TS: time.Unix(0, 0).UTC(), Level: "info", Msg: "read_note a.md", Data: json.RawMessage(`{"tool":"read_note"}`)},
	})
	require.NotNil(t, out)
	require.Contains(t, *out, `"msg":"read_note a.md"`)
	require.Contains(t, *out, `"tool":"read_note"`)
}

func TestMarshalLogs_CapsEntryCount(t *testing.T) {
	logs := make([]AgentLog, MaxAgentLogs+40)
	for i := range logs {
		logs[i] = AgentLog{Level: "info", Msg: "entry"}
	}

	out := MarshalLogs(logs)
	require.NotNil(t, out)

	var got []AgentLog
	require.NoError(t, json.Unmarshal([]byte(*out), &got))
	require.Len(t, got, MaxAgentLogs+1, "the cap plus the line saying what was dropped")
	require.Contains(t, got[len(got)-1].Msg, "40", "the operator has to learn how much is missing")
}

func TestMarshalLogs_CapsTotalSize(t *testing.T) {
	// One entry whose Data alone blows the size ceiling.
	fat := make([]byte, MaxAgentLogsBytes*2)
	for i := range fat {
		fat[i] = 'x'
	}
	logs := []AgentLog{
		{Level: "info", Msg: "small"},
		{Level: "info", Msg: "fat", Data: json.RawMessage(`"` + string(fat) + `"`)},
	}

	out := MarshalLogs(logs)
	require.NotNil(t, out)
	require.LessOrEqual(t, len(*out), MaxAgentLogsBytes*2,
		"an agent must not be able to store an unbounded blob for 90 days")

	var got []AgentLog
	require.NoError(t, json.Unmarshal([]byte(*out), &got))
	require.Equal(t, "small", got[0].Msg)
	require.Contains(t, got[len(got)-1].Msg, "dropped")
}

func TestParseAgentResponse_ParsesLogs(t *testing.T) {
	body := []byte(`{"status":"completed","logs":[` +
		`{"ts":"2026-08-21T10:00:01Z","level":"warn","msg":"denied",` +
		`"data":{"tool":"write_note","outcome":"denied"}}]}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.Len(t, resp.Logs, 1)
	require.Equal(t, "warn", resp.Logs[0].Level)
	require.JSONEq(t, `{"tool":"write_note","outcome":"denied"}`, string(resp.Logs[0].Data))
}

// A fleet response, through storage, back to the admin decoder.
func TestAgentLogRoundTrip(t *testing.T) {
	body, err := json.Marshal(AgentResponse{
		Status: "completed",
		Costs:  map[string]float64{"tokens": 5186},
		Logs: []AgentLog{{
			TS:    time.Date(2026, 8, 21, 10, 0, 1, 0, time.UTC),
			Level: "warn",
			Msg:   "write_note segments/x.md: denied",
			Data:  json.RawMessage(`{"tool":"write_note","outcome":"denied","reason":"write outside scope"}`),
		}},
	})
	require.NoError(t, err)

	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)

	stored := MarshalLogs(resp.Logs)
	require.NotNil(t, stored)
	require.Contains(t, *stored, "write outside scope")
	require.NotContains(t, *stored, "content")
}
