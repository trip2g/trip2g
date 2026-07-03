package fleet

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// newDebugFleet builds a Fleet whose debug surface can run real bash blocks.
// Sandbox "off" keeps the tests runnable without Linux namespace privileges.
func newDebugFleet() *Fleet {
	cfg := Config{
		FleetID: "f1", FleetSecret: "seed",
		AllowedPrograms: []string{"bash"},
		Sandbox:         "off",
	}
	return NewFleet(cfg, nil, nil)
}

func postDebug(t *testing.T, f *Fleet, req debugRunBlockRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodPost, "/debug/run-block", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	f.DebugHandler().ServeHTTP(rec, r)
	return rec
}

func decodeDebug(t *testing.T, rec *httptest.ResponseRecorder) debugRunBlockResponse {
	t.Helper()
	var resp debugRunBlockResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

func TestDebugRunBlock_InlineBodyStdinPiped(t *testing.T) {
	f := newDebugFleet()
	rec := postDebug(t, f, debugRunBlockRequest{
		Body:  "```bash\ncat\n```\n```bash\necho second\n```",
		Block: 0,
		Stdin: "ping",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodeDebug(t, rec)
	require.True(t, resp.OK)
	require.Equal(t, "ping", resp.Stdout)
	require.Equal(t, 0, resp.Block)
	require.Equal(t, 2, resp.BlocksTotal)
	require.Equal(t, "bash", resp.Program)
}

func TestDebugRunBlock_SecondBlock(t *testing.T) {
	f := newDebugFleet()
	rec := postDebug(t, f, debugRunBlockRequest{
		Body:  "```bash\ncat\n```\n```bash\necho second\n```",
		Block: 1,
	})
	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodeDebug(t, rec)
	require.True(t, resp.OK)
	require.Equal(t, "second\n", resp.Stdout)
}

// TestDebugRunBlock_NonZeroExit asserts a failing block returns 200 with
// OK=false plus captured stderr — the whole point of step debugging.
func TestDebugRunBlock_NonZeroExit(t *testing.T) {
	f := newDebugFleet()
	rec := postDebug(t, f, debugRunBlockRequest{
		Body: "```bash\necho boom >&2; exit 3\n```",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodeDebug(t, rec)
	require.False(t, resp.OK)
	require.Contains(t, resp.Stderr, "boom")
	require.NotEmpty(t, resp.Error)
}

func TestDebugRunBlock_RoleLookup(t *testing.T) {
	f := newDebugFleet()
	f.SetRoles([]Role{{
		NotePath: "roles/pipe.md", Executor: executorCode,
		Body: "```bash\ncat\n```",
	}})
	rec := postDebug(t, f, debugRunBlockRequest{
		Role:  "roles/pipe.md",
		Stdin: "from-role",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodeDebug(t, rec)
	require.True(t, resp.OK)
	require.Equal(t, "from-role", resp.Stdout)
}

func TestDebugRunBlock_UnknownRole404(t *testing.T) {
	f := newDebugFleet()
	rec := postDebug(t, f, debugRunBlockRequest{Role: "roles/nope.md"})
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDebugRunBlock_BadIndex400(t *testing.T) {
	f := newDebugFleet()
	rec := postDebug(t, f, debugRunBlockRequest{
		Body:  "```bash\necho hi\n```",
		Block: 5,
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDebugRunBlock_ProgramNotAllowed400(t *testing.T) {
	f := newDebugFleet() // bash only
	rec := postDebug(t, f, debugRunBlockRequest{
		Body: "```python\nprint('x')\n```",
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "allowed")
}

func TestDebugRunBlock_MethodNotAllowed(t *testing.T) {
	f := newDebugFleet()
	r := httptest.NewRequest(http.MethodGet, "/debug/run-block", nil)
	rec := httptest.NewRecorder()
	f.DebugHandler().ServeHTTP(rec, r)
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// TestDebugRunBlock_SecretScrub proves the debug path reuses RunBlock's
// secret-scrubbed env: a parent env sentinel must be invisible to the block.
func TestDebugRunBlock_SecretScrub(t *testing.T) {
	const sentinel = "FLEET_DEBUG_TEST_SENTINEL"
	t.Setenv(sentinel, "PARENT_SECRET")
	f := newDebugFleet()
	rec := postDebug(t, f, debugRunBlockRequest{
		Body: "```bash\necho \"v=${" + sentinel + ":-absent}\"\n```",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodeDebug(t, rec)
	require.True(t, resp.OK)
	require.Equal(t, "v=absent\n", resp.Stdout)
}

func TestDebugBlocks_InlineBody(t *testing.T) {
	f := newDebugFleet()
	r := httptest.NewRequest(http.MethodGet, "/debug/blocks?body="+
		"%60%60%60bash%0Aecho+hi%0A%60%60%60%0A%60%60%60python%0Aprint('x')%0A%60%60%60", nil)
	rec := httptest.NewRecorder()
	f.DebugHandler().ServeHTTP(rec, r)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp debugBlocksResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Blocks, 2)
	require.Equal(t, 0, resp.Blocks[0].Index)
	require.Equal(t, "bash", resp.Blocks[0].Lang)
	require.Equal(t, "bash", resp.Blocks[0].Program)
	require.Equal(t, 1, resp.Blocks[1].Index)
	require.Equal(t, "python", resp.Blocks[1].Lang)
	require.Equal(t, "python", resp.Blocks[1].Program)
}

func TestDebugBlocks_RoleLookup(t *testing.T) {
	f := newDebugFleet()
	f.SetRoles([]Role{{
		NotePath: "roles/pipe.md", Executor: executorCode,
		Body: "```bash\ncat\n```\n```bash\necho second\n```",
	}})
	r := httptest.NewRequest(http.MethodGet, "/debug/blocks?path=roles/pipe.md", nil)
	rec := httptest.NewRecorder()
	f.DebugHandler().ServeHTTP(rec, r)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp debugBlocksResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Blocks, 2)
	require.Equal(t, "bash", resp.Blocks[0].Lang)
	require.Equal(t, "cat\n", resp.Blocks[0].Code)
}

func TestDebugBlocks_UnknownRole404(t *testing.T) {
	f := newDebugFleet()
	r := httptest.NewRequest(http.MethodGet, "/debug/blocks?path=roles/nope.md", nil)
	rec := httptest.NewRecorder()
	f.DebugHandler().ServeHTTP(rec, r)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDebugBlocks_NoContent400(t *testing.T) {
	f := newDebugFleet()
	r := httptest.NewRequest(http.MethodGet, "/debug/blocks", nil)
	rec := httptest.NewRecorder()
	f.DebugHandler().ServeHTTP(rec, r)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDebugUI_ServesHTML(t *testing.T) {
	f := newDebugFleet()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	f.DebugHandler().ServeHTTP(rec, r)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	require.Contains(t, rec.Body.String(), "debug/blocks")
}

func TestValidateLoopbackAddr(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:9091", "localhost:9091", "[::1]:9091", "127.1.2.3:0"} {
		require.NoError(t, ValidateLoopbackAddr(addr), addr)
	}
	for _, addr := range []string{"0.0.0.0:9091", ":9091", "192.168.1.5:9091", "example.com:9091", "127.0.0.1"} {
		require.Error(t, ValidateLoopbackAddr(addr), addr)
	}
}
