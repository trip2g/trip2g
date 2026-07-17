package codellmgql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/cmd/codellm/internal/coderun"
)

type testBlockRunner struct{}

type failingBlockRunner struct{}

func (failingBlockRunner) RunBlocks(context.Context, BlockRunRequest) (BlockRunResult, error) {
	return BlockRunResult{}, errors.New("coderun: block 2/3: non-zero exit: boom")
}

func (testBlockRunner) RunBlocks(ctx context.Context, req BlockRunRequest) (BlockRunResult, error) {
	out, debug, err := coderun.ExecBlocksDebug(ctx, coderun.CodeInput{
		Body: req.Body, Input: req.FleetInput, AllowedPrograms: []string{"bash", "jq"},
		Sandbox: coderun.SandboxPolicy{Mode: coderun.SandboxOff},
	}, req.MaxSteps)
	if err != nil {
		return BlockRunResult{}, err
	}
	results := make([]BlockResult, 0, len(debug))
	for _, d := range debug {
		results = append(results, BlockResult{Index: d.Index, ExitCode: d.ExitCode, Stdout: d.Stdout, Stderr: d.Stderr})
	}
	return BlockRunResult{Output: out, Results: results}, nil
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
	      ] }) { ... on RunBlocksPayload { output results { index stdout stderr } } ... on ErrorPayload { message } } }`
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
				Output  string `json:"output"`
				Results []struct {
					Index  int    `json:"index"`
					Stdout string `json:"stdout"`
					Stderr string `json:"stderr"`
					Pipe   string `json:"pipe"`
				} `json:"results"`
			} `json:"runBlocks"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Empty(t, got.Errors)
	require.Equal(t, "{}\n", got.Data.Run.Output)
	require.Len(t, got.Data.Run.Results, 2)
	require.Equal(t, 0, got.Data.Run.Results[0].Index)
	//nolint:testifylint // byte-exact stdout incl. trailing newline, not JSON-structural
	require.Equal(t, "{\"hello\":{}}\n", got.Data.Run.Results[0].Stdout)
}

func TestRunBlocks_MaxStepsUsesMarkdownIndex(t *testing.T) {
	h := NewHTTPHandler(nil, testBlockRunner{})
	query := `mutation { runBlocks(input: { input: { changedFiles: [], attachedNotes: [], depth: 1 }, maxSteps: 1, blocks: [
        { kind: CODE, language: "bash", content: "printf first" },
        { kind: PROSE, content: "\n\n" },
        { kind: CODE, language: "bash", content: "printf second" }
	      ] }) { ... on RunBlocksPayload { output results { index stdout stderr } } ... on ErrorPayload { message } } }`
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
				Output  string `json:"output"`
				Results []any  `json:"results"`
			} `json:"runBlocks"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Empty(t, got.Errors)
	require.Equal(t, "first", got.Data.Run.Output)
	require.Len(t, got.Data.Run.Results, 1)
}

func TestRunBlocks_ReturnsBlockErrorPayload(t *testing.T) {
	h := NewHTTPHandler(nil, failingBlockRunner{})
	query := `mutation { runBlocks(input: { input: { changedFiles: [], attachedNotes: [], depth: 1 }, blocks: [
        { kind: CODE, language: "bash", content: "true" }
      ] }) { ... on BlockErrorPayload { index message } } }`
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
				Index   int    `json:"index"`
				Message string `json:"message"`
			} `json:"runBlocks"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Empty(t, got.Errors)
	require.Equal(t, 1, got.Data.Run.Index)
	require.Equal(t, "coderun: block 2/3: non-zero exit: boom", got.Data.Run.Message)
}
