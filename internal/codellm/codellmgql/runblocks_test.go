package codellmgql

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/coderun"
)

type testBlockRunner struct{}

func (testBlockRunner) RunBlocks(ctx context.Context, req BlockRunRequest) (BlockRunResult, error) {
	out, debug, err := coderun.ExecBlocksDebug(ctx, coderun.CodeInput{
		Body: req.Body, Input: req.FleetInput, AllowedPrograms: []string{"bash", "jq"},
		Sandbox: coderun.SandboxPolicy{Mode: coderun.SandboxOff},
	}, req.MaxSteps)
	if err != nil {
		return BlockRunResult{}, err
	}
	pipes := make([]BlockPipe, 0, max(0, len(debug)-1))
	for _, d := range debug[:max(0, len(debug)-1)] {
		pipes = append(pipes, BlockPipe{Index: d.Index, Content: d.PipeBuffer})
	}
	return BlockRunResult{Output: out, Pipes: pipes}, nil
}

func TestRunBlocks_PipesOnlyBetweenCodeBlocks(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required")
	}
	h := NewHTTPHandler(nil, testBlockRunner{})
	query := `mutation { runBlocks(input: { input: { changedFiles: [], attachedNotes: [], depth: 1 }, maxSteps: 4, blocks: [
        { kind: CODE, language: "bash", content: "echo '{\"hello\":{}}'" },
        { kind: PROSE, content: "\\n\\n" },
        { kind: CODE, language: "bash", content: "jq .hello" }
      ] }) { ... on RunBlocksPayload { output pipes { index content } } ... on ErrorPayload { message } } }`
	reqBody, err := json.Marshal(map[string]string{"query": query})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got struct {
		Data struct {
			Run struct {
				Output string `json:"output"`
				Pipes  []struct {
					Index   int    `json:"index"`
					Content string `json:"content"`
				} `json:"pipes"`
			} `json:"runBlocks"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Empty(t, got.Errors)
	require.Equal(t, "{}\n", got.Data.Run.Output)
	require.Len(t, got.Data.Run.Pipes, 1)
	require.Equal(t, 0, got.Data.Run.Pipes[0].Index)
	require.JSONEq(t, `{"hello":{}}`, got.Data.Run.Pipes[0].Content)
}
